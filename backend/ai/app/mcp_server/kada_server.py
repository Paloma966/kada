"""Kada MCP 服务：外部通用工具的接入点（stdio）。

**这里目前没有工具，这是有意的。** 业务工具（查短链总览、建短链）不在这里，因为它们
必须以"当前聊天用户"的身份执行，而本子进程是跨请求共享的长连接，拿不到请求级的登录
凭据（langchain-mcp-adapters 只对 streamable_http 传输支持按调用注入 header）。那两个
工具在 `app/service/kada_tools.py` 里随 AI 服务进程运行。

保留这条 MCP 通道，是为了接不依赖用户身份的通用能力（网页搜索、文件、外部 API）：
在下面用 `@mcp.tool()` 注册，AI 服务启动时会自动把它们加载成 LangChain 工具，不需要
改动对话逻辑。
"""

from mcp.server.fastmcp import FastMCP

mcp = FastMCP("kada-server")


if __name__ == "__main__":
    mcp.run()
