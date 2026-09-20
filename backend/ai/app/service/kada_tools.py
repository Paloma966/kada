"""Kada 业务工具：AI 服务进程内直接回调 Go 内网接口的 LangChain 工具。

为什么不放在 MCP 子服务里：建短链、查统计都要以"当前聊天用户"的身份执行，
而 MCP stdio 子进程是跨请求共享的长连接，工具参数由模型生成，拿不到请求级的
用户身份。因此业务工具在本进程内实现，用 contextvars 携带本次请求的 user_id，
通过 Go 的 /internal/ai/* 内网回调路由（共享密钥 + 内网来源校验）以真实用户
身份执行。外部通用工具（文件系统、搜索等）仍走 mcp_client.py 接入。
"""

import contextvars

import httpx
from langchain_core.tools import tool

from app.config import settings

# 本次聊天请求的用户 id：chat 路由在生成前 set，工具执行时 get。
# ContextVar 在同一请求的 async 任务链内可见，且不同请求之间天然隔离。
_current_user_id: contextvars.ContextVar[str] = contextvars.ContextVar(
    "kada_user_id", default=""
)


def set_current_user_id(user_id: str) -> None:
    """绑定本次请求的真实用户身份（由 chat 路由在生成前调用）。"""
    _current_user_id.set(user_id)


def _headers() -> dict:
    return {
        # 共享密钥证明调用方是内部 AI 服务；X-Kada-User-ID 标明真实用户。
        "X-Internal-Secret": settings.KADA_INTERNAL_SECRET,
        "X-Kada-User-ID": _current_user_id.get(),
        "Content-Type": "application/json",
    }


@tool
async def get_link_overview() -> str:
    """查询当前账号的短链总览数据（总短链数、总点击数）。
    当用户问"我一共有多少短链""总点击多少次"时调用。"""
    async with httpx.AsyncClient(trust_env=False, timeout=10) as client:
        resp = await client.get(
            f"{settings.KADA_API_BASE}/internal/ai/analytics/overview",
            headers=_headers(),
        )
        resp.raise_for_status()
        return resp.text


@tool
async def create_short_link(url: str) -> str:
    """为当前用户创建一条短链接（随机短码、默认域名、立即生效）。
    仅当用户明确发来一个完整的 http(s) 长链接，并要求"生成短链/缩短链接/转成短链/创建短链"时调用。
    参数 url 必须以 http:// 或 https:// 开头。"""
    async with httpx.AsyncClient(trust_env=False, timeout=15) as client:
        resp = await client.post(
            f"{settings.KADA_API_BASE}/internal/ai/links",
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


# 注册表：KADA_TOOLS 用于 bind_tools，KADA_TOOL_MAP 按名字取工具执行。
KADA_TOOLS = [get_link_overview, create_short_link]
KADA_TOOL_MAP = {tool_item.name: tool_item for tool_item in KADA_TOOLS}
