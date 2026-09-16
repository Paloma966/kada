import asyncio
import json
import time
from fastapi import APIRouter,Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from starlette.responses import JSONResponse

from app.config import settings
from app.service import  llm,session

router=APIRouter()
#定义请求
class ChatRequest(BaseModel):
    conversation_id: str | None=None
    message: str
    model:str |None=None

#SSE工具函数
def _see(event:str,data:dict)->str:
    return f"event:{event}\ndata:{json.dumps(data,ensure_ascii=False)}\n\n"
#假流式输出
async def mock_stream(message:str):
    text=(
        f"你刚刚说了{message}"
        "我主要是来验证一下SSE链路的后续我会改为真正的大模型来输出"
    )
    for ch in text:
        yield _see("token",{"delta":ch})
        await  asyncio.sleep(0.3)
#真模型
async def real_stream(user_id: str,conv_id: str,messages:list):
    full_text=""
    await  session.append_message(user_id,conv_id,"user",messages[-1]["content"])
    async for delta in llm.stream_chat(messages):
        full_text+=delta
        yield _see("token",{"delta":delta})
    #会话记忆
    if full_text:
        await  session.append_message(user_id,conv_id,"assistant",full_text)
    #结束事件

    yield _see("done",{"conversation_id":conv_id,"usage":{"tokens":len(full_text)}})
#错误事件生成器
async def _error_event(mes:str):
    yield _see("error",{"message":mes})


#主接口
@router.post("/v1/chat")
async def chat(req:ChatRequest,request:Request):
    user_id=request.headers.get("X-Kada-User-ID","demo-user")
#会话
    conv_id=req.conversation_id or await session.new_conversation(user_id)
    history=await session.load_messages(user_id,conv_id)
    messages=history+[{"role":"user","content":req.message}]
    gen=real_stream(user_id,conv_id,messages)
    return StreamingResponse(gen,media_type="text/event-stream")

#会话列表
@router.get("/v1/conversations")
async def list_convs(request:Request):
    user_id=request.headers.get("X-Kada-User-ID", "demo-user")
    convs=await session.list_conversations(user_id)
    return {"conversations":convs}
#单会话历史
@router.get("/v1/conversations/{conversation_id}/messages")
async def get_conv(conversation_id:str ,request:Request):
    user_id=request.headers.get("X-Kada-User-ID", "demo-user")
    msgs=await session.load_messages(user_id,conversation_id)
    return {"messages":msgs}

@router.delete("/v1/conversations/{conversation_id}")
async def delete_conv(conversation_id:str ,requset:Request):
    user_id=requset.headers.get("X-Kada-User-ID", "demo-user")
    ok=await session.delete_conversation(user_id,conversation_id)
    if not ok:
        return JSONResponse(status_code=404,content={"error":"会话不存在"})
    return {"ok":True}

