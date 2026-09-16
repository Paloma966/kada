
from app.config import settings
from langchain_openai import ChatOpenAI
#取大模型客户端
_client=None
def get_client()-> ChatOpenAI:
    global _client
    if _client is None:
        _client=ChatOpenAI(
            model=settings.CHAT_MODEL,
            api_key=settings.DEEPSEEK_API_KEY,
            base_url=settings.DEEPSEEK_BASE_URL,
            temperature=1.0,
            streaming=True
        )
    return _client
#流式输出
async def stream_chat(messages:list):
    client=get_client()
    stream=await client.chat.completions.create(
        model=settings.CHAT_MODEL,
        messages=messages,
        max_tokens=settings.AI_MAX_TOKENS,
        stream=True
    )
    #取每次新生成的值
    async for chunk in stream:
        delta=chunk.choices[0].delta.content if chunk.choices else None
        if delta:
            yield delta
