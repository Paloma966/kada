from datetime import datetime
from langchain_core.tools import tool

@tool
def get_current_time()->str:
    """ 获取当前系统的时间， 当用户文 “现在几点” “ 今天几号的时候调用。”"""
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")

@tool
def add(a:int,b:int)->int:
    """两个整数相加返回和，当用户问加法计算的时候调用"""
    return a+b
ALL_TOOLS=[get_current_time,add]
TOOL_MAP={t.name:t for t in ALL_TOOLS}