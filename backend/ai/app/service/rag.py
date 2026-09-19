"""知识库检索（RAG）：基于 pgvector 的相似度搜索。"""

import asyncio

from langchain_postgres import PGVector

from app.config import settings
from app.service.embedding import get_embeddings

_store: PGVector | None = None

# langchain-postgres 内部是同步实现，只认 psycopg 驱动；
# 而会话表走异步 asyncpg，所以这里把连接串里的驱动替换掉，两套互不影响。
_PG_URL = settings.POSTGRES_URL.replace("+asyncpg", "+psycopg")


def get_store() -> PGVector:
    """向量库客户端单例。"""
    global _store
    if _store is None:
        _store = PGVector(
            embeddings=get_embeddings(),
            connection=_PG_URL,
            collection_name=settings.DOC_COLLECTION_NAME,
        )
    return _store


async def retrieve(query: str, k: int = 4) -> list[str]:
    """检索与问题最相关的 k 个文档片段。

    similarity_search 是同步阻塞调用（含一次 embedding 请求 + 数据库查询），
    用 to_thread 丢到线程池执行，避免卡住 FastAPI 的事件循环。
    """
    docs = await asyncio.to_thread(get_store().similarity_search, query, k)
    return [doc.page_content for doc in docs]
