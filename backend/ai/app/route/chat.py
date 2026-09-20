"""AI 对话接口：SSE 流式问答，支持 RAG 检索和工具调用（业务工具 + 通用工具 + MCP 工具）。"""

import json

from fastapi import APIRouter, Depends, Request
from fastapi.responses import StreamingResponse
from langchain_core.messages import AIMessage, HumanMessage, ToolMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from pydantic import BaseModel

from app.route.deps import require_gateway
from app.service import conversation, kada_tools, llm, mcp_client, rag
from app.service.tools import TOOL_MAP, TOOLS

# 整个路由组都要求请求来自 Go 网关（见 deps.require_gateway）：
# 只有确认了调用方是网关，网关注入的 X-Kada-User-ID 才有资格当身份断言。
router = APIRouter(dependencies=[Depends(require_gateway)])

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
    """执行一次工具调用。

    进程内工具统一走 ainvoke：异步工具（业务工具要发 HTTP）直接执行，同步工具
    （查时间、加法）由 ainvoke 自动丢到线程池，不必在这里分支。
    """
    name = tool_call["name"]
    if name in TOOL_MAP:
        return await TOOL_MAP[name].ainvoke(tool_call["args"])

    mcp_tool = next(
        (tool for tool in mcp_client.get_tools() if tool.name == name), None
    )
    if mcp_tool is None:
        return f"工具 {name} 不可用"
    return await mcp_tool.ainvoke(tool_call["args"])


async def real_stream(
    user_id: str, conv_id: str, history: list, user_message: str, token: str
):
    # 把本次请求的登录凭据放进 contextvars，供业务工具回调 Go 时使用。
    # 绑定写在生成器里而不是路由里，是因为工具调用发生在生成器所在的任务上下文中；
    # ContextVar 按 Task 隔离，并发请求之间不会串号。
    kada_tools.set_request_token(token)

    # 用户消息先落库：即使后面生成失败，用户说的话也不会丢。
    await conversation.append_message(user_id, conv_id, "user", user_message)

    # 知识库是增强能力而非硬依赖：检索失败（文档没入库、embedding 密钥缺失、
    # pgvector 不可用等）降级为"没有参考资料"，对话本身照常继续。
    try:
        pieces = await rag.retrieve(user_message, k=4)
    except Exception as exc:
        print(f"[RAG] 检索失败，降级为无参考资料：{exc}")
        pieces = []

    context = "\n\n---\n\n".join(pieces) if pieces else "知识库中没有相关资料。"
    messages = prompt.format_messages(
        history=to_langchain(history),
        context=context,
        input=user_message,
    )

    # 模型构造失败就没有可用的大模型了，明确回一个 error 事件再结束。
    # 这一步若留在下面的 try 之外，失败会让流无声中断：响应头已经发出，
    # 前端既收不到 error 事件也收不到 done，只能看到一个空回复。
    try:
        model = llm.get_model().bind_tools(TOOLS + mcp_client.get_tools())
    except Exception as exc:
        yield _sse("error", {"message": f"AI 服务异常：{exc}"})
        return

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
        await conversation.append_message(user_id, conv_id, "assistant", full_text)
    yield _sse(
        "done",
        {"conversation_id": conv_id, "usage": {"tokens": len(full_text)}},
    )


def _bearer_token(request: Request) -> str:
    """取出本次请求的登录凭据（Go 网关原样转发的用户 JWT）。

    Python 不解析也不校验它：它只被继续转发给 Go 的 /api/*，权限判定始终只有 Go 一处。
    头缺失时返回空串，业务工具会明确回一句"没有携带登录凭据"，而不是换成别的身份去调后端。
    """
    authorization = request.headers.get("Authorization", "")
    prefix = "Bearer "
    if authorization.startswith(prefix):
        return authorization[len(prefix) :].strip()
    return ""


# （见 backend/internal/handler/ai），所以公开面是 /api/ai/chat、/api/ai/conversations/*。

@router.post("/chat")
async def chat(req: ChatRequest, request: Request):
    # X-Kada-User-ID 由 Go 网关注入（并覆盖客户端伪造的同名头），Python 不直连外网；
    # 默认值 demo-user 仅用于绕过网关、本地直连 Python 调试。
    user_id = request.headers.get("X-Kada-User-ID", "demo-user")
    token = _bearer_token(request)
    if not token:
        # 正常链路不会走到这里：网关一定会带上 Authorization。为空说明请求绕过了网关
        # （本地直连调试），此时工具不能以用户身份执行，只降级不冒充。
        print("[AUTH] 请求未携带 Authorization，业务工具本次不可用")

    conv_id = req.conversation_id or await conversation.new_conversation(user_id)
    history = await conversation.load_messages(user_id, conv_id)

    # X-Accel-Buffering: no 告诉 nginx 不要缓冲 SSE，否则文字会攒成一坨再发。
    return StreamingResponse(
        real_stream(user_id, conv_id, history, req.message, token),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )


@router.get("/conversations/current")
async def current_conversation(request: Request):
    user_id = request.headers.get("X-Kada-User-ID", "demo-user")
    return await conversation.get_current_conversation(user_id)


@router.post("/conversations/restart")
async def restart_conversation(request: Request):
    user_id = request.headers.get("X-Kada-User-ID", "demo-user")
    new_id = await conversation.restart_conversation(user_id)
    return {"conversation_id": new_id}
