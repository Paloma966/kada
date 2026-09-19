import json

from fastapi import APIRouter,Request
from fastapi.responses import StreamingResponse
from langchain_core.messages import HumanMessage, AIMessage,ToolMessage,SystemMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from pydantic import BaseModel
from starlette.responses import JSONResponse
from app.service import mcp_client
from app.service.mcp_client import mcp_tool

from app.service.tools import ALL_TOOLS,TOOL_MAP
from app.config import settings
from app.service import  llm,session,rag

router=APIRouter()
#定义请求
class ChatRequest(BaseModel):
    conversation_id: str | None=None
    message: str
    model:str |None=None
#提示词
prompt=ChatPromptTemplate.from_messages([
    ("system","你是kada平台的ai助手，帮助用户分析和管理短链接数据。请用简体中文来回答要求简洁准确。参考资料优先用："
              "需要查实时信息或做计算时，可以提供调用的工具"),
    MessagesPlaceholder("history"),
    ("human","【参考资料】\n{context}\n\n【用户问题】{input}"),
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
    pieces=await rag.retrieve(user_message,k=4)

    context="\n\n---\n\n".join(pieces) if pieces else"知识库中没有相关资料"
    messages=prompt.format_messages(
        history=history_message,
        context=context,
        input=user_message,
    )
    model=llm.get_model().bind_tools(ALL_TOOLS+mcp_client.mcp_tool)
    full_text=""
    for _ in range(5):
        ai_msg=await model.ainvoke(messages)
        messages.append(ai_msg)
        if not ai_msg.tool_calls:
            if ai_msg.content:
                full_text+=ai_msg.content
                yield _see("token",{"delta":ai_msg.content})
            break
        for tc in ai_msg.tool_calls:
            if tc["name"] in TOOL_MAP:
                result =TOOL_MAP[tc["name"].invoke(tc["args"])]
            else:
                mcp_tool=next((t for t in mcp_client.mcp_tool if t.name==tc["name"]),None)
                if mcp_tool is None:
                    result=f"未加工具：{tc['name']}"
                else :
                    result=await mcp_tool.ainvoke(tc)

            messages.append(
                ToolMessage(content=str(result),tool_call_id=tc["id"])
            )
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

#当前会话
@router.get("/v1/session/current")
async def current_session(requset:Request):
    user_id=requset.headers.get("X-Kada-User-ID","demo-user")
    return await session.get_current_session(user_id)
#重新开始
@router.post("v1/session/restart")
async def restart_session(request:Request):
    user_id=request.headers.get("X-Kada-User-ID","demo-user")
    new_id=await session.restart_session(user_id)
    return {"conversation_id":new_id}

