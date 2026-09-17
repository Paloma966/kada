import json

from fastapi import APIRouter,Request
from fastapi.responses import StreamingResponse
from langchain_core.messages import HumanMessage, AIMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
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
#提示词
prompt=ChatPromptTemplate.from_messages([
    ("system","你是kada平台的ai助手，帮助用户分析和管理短链接数据。请用简体中文来回答要求简洁准确"),
    MessagesPlaceholder("history"),
    ("human","{input}"),
])
#把数据库里面的[{"role","content"}]转成langchain消息对象
def to_langchain(history:list)->list:
    result=[]
    for m in history:
        if m["role"]=="user":
            result.append(HumanMessage(content=m["content"]))
        else:
            result.append(AIMessage(content=m["content"]))
    return result

#SSE工具函数
def _see(event:str,data:dict)->str:
    return f"event:{event}\ndata:{json.dumps(data,ensure_ascii=False)}\n\n"
#真模型
async def real_stream(user_id: str,conv_id: str,history,user_message:str):
    history_message=to_langchain(history)
    chain=prompt|llm.get_model()
    full_text=""
    async for chunk in chain.astream({"history":history_message,"input":user_message}):
        text=chunk.content if isinstance(chunk.content,str)else str(chunk.content)
        full_text+=text
        yield _see("token",{"delta":text})
    #会话记忆
    await session.append_message(user_id,conv_id,"user",user_message)
    if full_text:
        await session.append_message(user_id,conv_id,"assistant",full_text)
    yield _see("done",{"conversation_id":conv_id,"usage":{"tokens":len(full_text)}})



#主接口
@router.post("/v1/chat")
async def chat(req:ChatRequest,request:Request):
    user_id=request.headers.get("X-Kada-User-ID","demo-user")
#会话
    conv_id=req.conversation_id or await session.new_conversation(user_id)
    history=await session.load_messages(user_id,conv_id)
    gen=real_stream(user_id,conv_id,history,req.message)
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
