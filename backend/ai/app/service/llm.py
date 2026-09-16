from openai import AsyncOpenAI, base_url

from app.config import settings

#取大模型客户端
_client=None
def get_client()-> AsyncOpenAI:
    global _client
    if _client is None:
        _client=AsyncOpenAI(
            api_key=settings.DEEPSEEK_API_KEY,
            base_url=settings.DEEPSEEK_BASE_URL,

        )
        return _client
#流式输出
async def stream_chat(messages:list):
    client=get_client()
    stream=await client.chat.completions.create(
        model=settings.CHAT_MODEL,
        messages=messages,
        max_tokens=settings.AI_MAX_TOKENS
    )
    #取每次新生成的值
    async for chunk in stream:
        delta=chunk.choices[0].delta.content if chunk.choice else None
        if delta:
            yield delta
