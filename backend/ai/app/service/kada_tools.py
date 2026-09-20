"""Kada 业务工具：在 AI 服务进程内，以"当前登录用户"的身份回调 Go 后端。

为什么这两个工具不放在 MCP 子服务里：它们必须代表本次聊天的用户执行，而 MCP stdio
子进程是跨请求共享的长连接，工具参数又由模型生成，拿不到请求级的身份
（langchain-mcp-adapters 只对 streamable_http 传输支持按调用注入 header）。所以业务
工具放在本进程内，用 contextvars 携带本次请求的登录凭据；MCP 子服务留给"外部通用
工具"（搜索、文件等）这类不需要用户身份的扩展。

凭据就是前端登录后拿到的那个 JWT：Go 网关已经把它原样转发进来（Authorization 头），
本服务不解析、不校验、不缓存，只把它转交给 Go —— 权限判定始终只有 Go 一处。用户被
停用或 token 过期时，工具的结果与用户自己调 API 完全一致，不需要重启本服务。
"""

import contextvars

import httpx
from langchain_core.tools import tool

from app.config import settings

# 本次聊天请求的登录凭据（Authorization 里 Bearer 后面那串）。
#
# 必须是 ContextVar，不能是模块级变量：每个请求在 asyncio 里是独立的 Task，Task 会
# 复制一份上下文，因此并发请求天然隔离；换成模块级变量就会把别人的凭据发给后端。
_request_token: contextvars.ContextVar[str] = contextvars.ContextVar(
    "kada_request_token", default=""
)

# 没有凭据时的回话：返回文本而不是抛异常，让模型能向用户解释，也不打断整段回答。
_NO_CREDENTIAL = "无法执行：本次请求没有携带登录凭据。请通过平台页面登录后重试。"


def set_request_token(token: str) -> None:
    """绑定本次请求的登录凭据；由 chat 路由在开始生成前调用。"""
    _request_token.set(token)


def _headers() -> dict:
    return {
        "Authorization": f"Bearer {_request_token.get()}",
        "Content-Type": "application/json",
    }


@tool
async def get_link_overview() -> str:
    """查询当前账号的短链总览数据（总短链数、总点击数）。
    当用户问"我一共有多少短链""总点击多少次"时调用。"""
    if not _request_token.get():
        return _NO_CREDENTIAL
    # trust_env=False：忽略系统代理，回调本机 Go 服务必须直连。
    async with httpx.AsyncClient(trust_env=False, timeout=10) as client:
        resp = await client.get(
            f"{settings.KADA_API_BASE}/api/analytics/overview", headers=_headers()
        )
    if resp.status_code != 200:
        return f"查询失败（HTTP {resp.status_code}）：{resp.text}"
    return resp.text


@tool
async def create_short_link(url: str) -> str:
    """为用户创建一条短链接（随机短码、默认域名、立即生效）。
    仅当用户明确发来一个完整的 http(s) 长链接，并要求"生成短链/缩短链接/转成短链/创建短链"时调用。
    参数 url 必须以 http:// 或 https:// 开头。"""
    if not _request_token.get():
        return _NO_CREDENTIAL
    async with httpx.AsyncClient(trust_env=False, timeout=15) as client:
        resp = await client.post(
            f"{settings.KADA_API_BASE}/api/links",
            headers=_headers(),
            json={"original_url": url},
        )
    if resp.status_code == 201:
        link = resp.json()["link"]
        return (
            "短链接创建成功：\n"
            f"短链接：{link['short_url']}\n"
            f"原链接：{link['original_url']}\n"
            f"短码：{link['short_code']}"
        )
    return f"短链接创建失败（HTTP {resp.status_code}）：{resp.text}"


# 注册表：由 tools.py 并入 TOOLS / TOOL_MAP。
KADA_TOOLS = [get_link_overview, create_short_link]
