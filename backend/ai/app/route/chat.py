"""AI 对话接口：SSE 流式问答，支持 RAG 检索和工具调用（本地工具 + MCP 工具）。"""

import json

from fastapi import APIRouter, Request
from fastapi.responses import StreamingResponse
from langchain_core.messages import AIMessage, HumanMessage, ToolMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from pydantic import BaseModel

from app.service import llm, mcp_client, rag, session
from app.service.tools import TOOL_MAP, TOOLS

router = APIRouter()

# 工具调用最多循环几轮：模型可能"调工具→拿结果→再调工具"，
# 上限是为了在模型不收敛时也能停下来，不会无限循环。
MAX_AGENT_ROUNDS = 5
prompt = ChatPromptTemplate.from_messages(
    [
        (
            "system",
            "你是 kada 平台的 AI 助手，帮助用户分析和管理短链接数据。请用简体中文回答，"
            "简洁准确。优先使用参考资料；需要实时信息或计算时，可以调用提供的工具。",
        ),
        MessagesPlaceholder("history"),
        ("human", "【参考资料】\n{context}\n\n【用户问题】{input}"),
    ]
)


class ChatRequest(BaseModel):
    conversation_id: str | None = None
    message: str


def to_langchain(history: list) -> list:
    """把库里存的 [{"role", "content"}] 转成 LangChain 消息对象。"""
    result = []
    for message in history:
        if message["role"] == "user":
            result.append(HumanMessage(content=message["content"]))
        else:
            result.append(AIMessage(content=message["content"]))
    return result


def _sse(event: str, data: dict) -> str:
    """拼一条 SSE 消息（event + data，以空行结尾）。"""
    return f"event: {event}\ndata: {json.dumps(data, ensure_ascii=False)}\n\n"


async def _dispatch_tool(tool_call: dict):
    """执行一次工具调用：本地工具是同步 invoke，MCP 工具是异步 ainvoke。"""
    name = tool_call["name"]
    if name in TOOL_MAP:
        return TOOL_MAP[name].invoke(tool_call["args"])

    mcp_tool = next(
        (tool for tool in mcp_client.get_tools() if tool.name == name), None
    )
    if mcp_tool is None:
        return f"工具 {name} 不可用"
    return await mcp_tool.ainvoke(tool_call["args"])


async def real_stream(user_id: str, conv_id: str, history: list, user_message: str):
    # 用户消息先落库：即使后面生成失败，用户说的话也不会丢。
    await session.append_message(user_id, conv_id, "user", user_message)

    pieces = await rag.retrieve(user_message, k=4)
    context = "\n\n---\n\n".join(pieces) if pieces else "知识库中没有相关资料。"
    messages = prompt.format_messages(
        history=to_langchain(history),
        context=context,
        input=user_message,
    )
    model = llm.get_model().bind_tools(TOOLS + mcp_client.get_tools())

    full_text = ""
    try:
        for _ in range(MAX_AGENT_ROUNDS):
            collected = None
            # 边流式接收边转发：正文逐 token 推给前端形成打字效果；
            # 同时把 chunk 累加，流结束后才能从完整消息里读到 tool_calls。
            async for chunk in model.astream(messages):
                collected = chunk if collected is None else collected + chunk
                if chunk.content:
                    full_text += chunk.content
                    yield _sse("token", {"delta": chunk.content})

            if collected is None:
                break
            messages.append(collected)

            # 没有工具调用 = 这一轮就是最终回答，结束。
            if not collected.tool_calls:
                break

            # 有工具调用：执行后把 ToolMessage 塞回消息，进入下一轮让模型总结。
            for tool_call in collected.tool_calls:
                result = await _dispatch_tool(tool_call)
                messages.append(
                    ToolMessage(
                        content=str(result), tool_call_id=tool_call["id"]
                    )
                )
    except Exception as exc:
        yield _sse("error", {"message": f"AI 服务异常：{exc}"})
        return

    if full_text:
        await session.append_message(user_id, conv_id, "assistant", full_text)
    yield _sse(
        "done",
        {"conversation_id": conv_id, "usage": {"tokens": len(full_text)}},
    )


@router.post("/v1/chat")
async def chat(req: ChatRequest, request: Request):
    # X-Kada-User-ID 由 Go 网关注入（并覆盖客户端伪造的同名头），Python 不直连外网；
    # 默认值 demo-user 仅用于绕过网关、本地直连 Python 调试。
    user_id = request.headers.get("X-Kada-User-ID", "demo-user")
    conv_id = req.conversation_id or await session.new_conversation(user_id)
    history = await session.load_messages(user_id, conv_id)

    # X-Accel-Buffering: no 告诉 nginx 不要缓冲 SSE，否则文字会攒成一坨再发。
    return StreamingResponse(
        real_stream(user_id, conv_id, history, req.message),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )


@router.get("/v1/session/current")
async def current_session(request: Request):
    user_id = request.headers.get("X-Kada-User-ID", "demo-user")
    return await session.get_current_session(user_id)


@router.post("/v1/session/restart")
async def restart_session(request: Request):
    user_id = request.headers.get("X-Kada-User-ID", "demo-user")
    new_id = await session.restart_session(user_id)
    return {"conversation_id": new_id}
