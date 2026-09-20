"""文本向量化（embedding）客户端单例，供知识库入库和检索使用。"""

from langchain_community.embeddings import DashScopeEmbeddings

from app.config import settings

_embeddings: DashScopeEmbeddings | None = None


def get_embeddings() -> DashScopeEmbeddings:
    """全局复用同一个 embedding 客户端（DashScope SDK 是同步实现）。"""
    global _embeddings
    if _embeddings is None:
        _embeddings = DashScopeEmbeddings(
            model=settings.EMBEDDING_MODEL,
            dashscope_api_key=settings.DASHSCOPE_API_KEY,
        )
    return _embeddings
