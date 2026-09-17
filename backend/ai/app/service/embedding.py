import asyncio
from langchain_community.embeddings import DashScopeEmbeddings


from app.config import settings
#全局embedding客户端
def get_embeddings()->DashScopeEmbeddings:
    global get_embeddings
    if get_embeddings is None:
        get_embeddings=DashScopeEmbeddings(
            model=settings.EMBEDDING_MODEL,
            dashscope_api_key=settings.DASHSCOPE_API_KEY,
        )
    return get_embeddings

async def aembed_query(text:str)->list[float]:
    return await asyncio.to_thread(get_embeddings().embed_query,text)

async def amebed_documents(texts:list[str])->list[list[float]]:
    return await asyncio.to_thread(get_embeddings().embed_documents,texts)