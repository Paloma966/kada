import asyncio
from langchain_community.embeddings import DashScopeEmbeddings


from app.config import settings
#全局embedding客户端
_embeddings:DashScopeEmbeddings|None=None

def get_embeddings()->DashScopeEmbeddings:
    global _embeddings
    if _embeddings is None:
        _embeddings=DashScopeEmbeddings(
            model=settings.EMBEDDING_MODEL,
            dashscope_api_key=settings.DASHSCOPE_API_KEY,
        )
    return _embeddings

async def aembed_query(text:str)->list[float]:
    return await asyncio.to_thread(get_embeddings().embed_query,text)

async def aembed_documents(texts:list[str])->list[list[float]]:
    return await asyncio.to_thread(get_embeddings().embed_documents,texts)