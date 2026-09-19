"""本地工具：随 AI 服务进程运行、不依赖外部服务的 LangChain @tool。

注意：每个工具的 docstring 不是普通注释，而是给大模型看的工具说明，
模型靠它判断"什么时候该调用这个工具"，要写清触发场景。
"""

from datetime import datetime

from langchain_core.tools import tool


@tool
def get_current_time() -> str:
    """获取当前的日期和时间。当用户问"现在几点""今天几号""当前时间"时调用。"""
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


@tool
def add(a: float, b: float) -> float:
    """计算两个数字的和。当用户要求做加法、求和时调用。"""
    return a + b


# 注册表：TOOLS 用于 bind_tools，TOOL_MAP 按名字取工具执行结果。
TOOLS = [get_current_time, add]
TOOL_MAP = {tool.name: tool for tool in TOOLS}
