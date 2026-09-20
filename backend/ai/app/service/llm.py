"""对话大模型客户端单例。"""

from langchain_openai import ChatOpenAI

from app.config import settings

_model = None


def get_model() -> ChatOpenAI:
    """全局复用同一个模型客户端。

    单例是为了复用底层 HTTP 连接池，避免每个请求都重新构造客户端；
    streaming=True 让上层可以逐 token 流式输出（SSE 打字效果）。
    """
    global _model
    if _model is None:
        _model = ChatOpenAI(
            model=settings.CHAT_MODEL,
            api_key=settings.DEEPSEEK_API_KEY,
            base_url=settings.DEEPSEEK_BASE_URL,
            max_tokens=settings.AI_MAX_TOKENS,
            temperature=0.7,
            streaming=True,
        )
    return _model
