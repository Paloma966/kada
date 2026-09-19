"""Kada MCP 服务：把 Go 后端的能力以 MCP 工具形式暴露给 AI。

以 stdio 方式被 AI 服务作为子进程拉起，内部用 httpx 同步回调 Go 的 HTTP API。
"""

import httpx
from mcp.server.fastmcp import FastMCP

from app.config import settings

mcp = FastMCP("kada-server")


def _headers() -> dict:
    # Token 只从环境变量经 settings 读取，绝不写进代码，否则提交仓库即泄露。
    return {
        "Authorization": f"Bearer {settings.KADA_API_TOKEN}",
        "Content-Type": "application/json",
    }


def _require_token() -> None:
    if not settings.KADA_API_TOKEN:
        raise RuntimeError(
            "KADA_API_TOKEN 未配置：请先在环境变量中设置 Kada API Token 再启动 AI 服务"
        )


@mcp.tool()
def get_link_overview() -> str:
    """查询当前账号的短链总览数据（总短链数、总点击数）。
    当用户问"我一共有多少短链""总点击多少次"时调用。"""
    _require_token()
    resp = httpx.get(
        f"{settings.KADA_API_BASE}/api/analytics/overview",
        headers=_headers(),
        timeout=10,
        # trust_env=False：忽略系统代理。开发机的抓包/翻墙代理会拦截 localhost
        # 请求并报 SSL WRONG_VERSION_NUMBER，回调本机 Go 服务必须直连。
        trust_env=False,
    )
    resp.raise_for_status()
    return resp.text


@mcp.tool()
def create_short_link(url: str) -> str:
    """为用户创建一条短链接（随机短码、默认域名、立即生效）。
    仅当用户明确发来一个完整的 http(s) 长链接，并要求"生成短链/缩短链接/转成短链/创建短链"时调用。
    参数 url 必须以 http:// 或 https:// 开头。"""
    _require_token()
    resp = httpx.post(
        f"{settings.KADA_API_BASE}/api/links",
        headers=_headers(),
        json={"original_url": url},
        timeout=15,
        trust_env=False,
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


if __name__ == "__main__":
    mcp.run()
