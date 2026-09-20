"""进程内工具：随 AI 服务进程运行、不经过 MCP 子进程的 LangChain @tool。

注意：每个工具的 docstring 不是普通注释，而是给大模型看的工具说明，
模型靠它判断"什么时候该调用这个工具"，要写清触发场景。

这里只放"不需要用户身份"的通用工具；Kada 业务工具在 kada_tools.py，
它们要带上本次请求的登录凭据才能回调 Go。
"""

from datetime import datetime

from langchain_core.tools import tool

from app.service.kada_tools import KADA_TOOLS


@tool
def get_current_time() -> str:
    """获取当前的日期和时间。当用户问"现在几点""今天几号""当前时间"时调用。"""
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


@tool
def add(a: float, b: float) -> float:
    """计算两个数字的和。当用户要求做加法、求和时调用。"""
    return a + b


# 注册表：TOOLS 用于 bind_tools，TOOL_MAP 按名字取工具执行。
# Kada 业务工具同样是进程内工具，只是执行时要用当前用户凭据；MCP 工具另算，
# 由 mcp_client 在启动时加载。
TOOLS = [get_current_time, add, *KADA_TOOLS]
TOOL_MAP = {tool.name: tool for tool in TOOLS}
