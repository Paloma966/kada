import asyncio
import json
import time
from pydoc import text


def _see(event:str,data:dict)->str:
    return f"event:{event}\n data:{json.dumps(data,ensure_ascii=False)}"
#假流式输出
async def mock_stream(message:str):
    text=(
        f"你刚刚说了{message}"
        "我主要是来验证一下SSE链路的后续我会改为真正的大模型来输出"
    )
    for ch in text:
        yield _see("token",{"delta":ch})
        await  asyncio.sleep(0.3)