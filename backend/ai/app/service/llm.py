
from app.config import settings
from langchain_openai import ChatOpenAI

#创建模型实例
_model = None

def get_model() -> ChatOpenAI:
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
