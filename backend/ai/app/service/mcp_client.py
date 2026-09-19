"""MCP 客户端：以 stdio 方式拉起并连接本项目的 MCP 子服务。

langchain-mcp-adapters 0.1.0 起 MultiServerMCPClient 不能再用 async with，
正确用法是先实例化，再 await client.get_tools()。
"""

import sys
from pathlib import Path

from langchain_mcp_adapters.client import MultiServerMCPClient

# 本文件位于 backend/ai/app/service/mcp_client.py，向上三级是项目根 backend/ai。
# 用 __file__ 推导而不是写死盘符路径，部署到 Linux 服务器后才能正常启动子进程。
_PROJECT_ROOT = Path(__file__).resolve().parents[2]

SERVER_CONFIG = {
    "kada": {
        # 用当前解释器启动子进程，保证 MCP 服务和 AI 服务在同一个 Python 环境里。
        "command": sys.executable,
        "args": ["-m", "app.mcp_server.kada_server"],
        "transport": "stdio",
        "cwd": str(_PROJECT_ROOT),
    }
}

# 启动时加载一次的 MCP 工具；连接失败时保持为空，AI 降级为纯对话。
_mcp_tools: list = []


async def load_tools() -> list:
    """连接 MCP 子服务并拉取工具列表，返回加载到的工具。"""
    global _mcp_tools
    client = MultiServerMCPClient(SERVER_CONFIG)
    _mcp_tools = await client.get_tools()
    return _mcp_tools


def get_tools() -> list:
    """读取已加载的 MCP 工具（路由每轮调用，拿到的是启动时加载的结果）。"""
    return _mcp_tools
