import asyncio
from langchain_postgres import PGVector

from app.config import settings
from app.service.embedding import get_embeddings

#向量存储客户端
_store:PGVector|None=None
#连接到向量数据库
_PG_URL=settings.PostgreSQL_URL.replace("+asyncpg", "+psycopg")
#获取向量数据库连接
def get_store()->PGVector:
    global _store
    if _store is None:
        _store=PGVector(
            embeddings=get_embeddings(),
            connection=_PG_URL,
            collection_name=settings.DOC_COLLECTION_NAME
        )
    return _store
# 检索最相关的四个文档片段
async def retrieve(query:str,k:int=4)->list[str]:
    docs=await asyncio.to_thread(get_store().similarity_search,query,k)
    return [d.page_content for d in docs]