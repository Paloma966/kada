import  os
import httpx
from mcp.server.fastmcp import FastMCP

KADA_API_BASE=os.getenv("KADA_API_BASE","http://localhost:8080")
KADA_API_TOKEN=os.getenv(
    "KADA_API_TOKEN",
    "kada_4d6c97c34b195f6d44adba6cef986d2ad59ebe76a2d7b251",
)
mcp=FastMCP("kada-server")
def _headers()->dict:
    """构造带认证的请求头"""
    return {
        "Authorization":f"Bearer {KADA_API_TOKEN}",
        "Content-Type":"application/json",
    }

@mcp.tool()
def get_link_overview()->str:
    """查询当前账号的短链总数：总短链数、总点击数、"当用户问我一共有多少短链" "总点击多少次" """
    resp=httpx.get(
        f"{KADA_API_BASE}/api/analytics/overview",
        headers=_headers(),
        timeout=10,
        trust_env=False,

    )
    resp.raise_for_status()
    return resp.text

@mcp.tool()
def create__short_link(url:str)->str:
    """为用户创建一条短链接（默认配置：随机短码、默认域名、立即生效）。
    仅当用户明确发来一个完整的http(s)长链接、并要求"生成短链/缩短链接/转成短链/创建短链"等词时调用
    参数url 必须是完整链接，以http://或https://开头。"""
    resp=httpx.post(
        f"{KADA_API_BASE}/api/links",
        headers=_headers(),
        json={"original_url":url},
        timeout=15,
        trust_env=False,
    )
    if resp.status_code==201:
        link=resp.json()["link"]
        return (
            f"短链接创建成功：\n"
            f"短链接：{link['short_url']}\n"
            f"原链接: {link['original_url']}\n"
            f"短码：{link['short_code']}"
        )
    return f"短链接创建失败(HTTP{resp.status_code}):{resp.text})"

if __name__=="__main__":
    mcp.run()