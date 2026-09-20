"""AI 服务唯一的配置入口。

普通参数直接写死默认值（本地开发即用）；只有密钥从环境变量读取，
这样同一份代码在本地和服务器都能跑，且密钥不会进仓库。
"""

import os

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(extra="ignore")

    # 对话大模型（DeepSeek 兼容 OpenAI 接口）
    CHAT_MODEL: str = "deepseek-flash"
    DEEPSEEK_BASE_URL: str = "https://api.deepseek.com"
    DEEPSEEK_API_KEY: str = os.getenv("DEEPSEEK_API_KEY", "")
    AI_MAX_TOKENS: int = 8192

    # Embedding 用阿里云百炼；本机环境变量名约定为 aliyun
    EMBEDDING_MODEL: str = "text-embedding-v3"
    DASHSCOPE_API_KEY: str = os.getenv("aliyun", "")

    # 知识库（pgvector）
    DOC_COLLECTION_NAME: str = "knowledge_docs"
    DOC_DIR: str = "./docs"
    CHUNK_SIZE: int = 500
    CHUNK_OVERLAP: int = 100

    # 会话存储：Redis 做热缓存，PostgreSQL 是最终数据源
    REDIS_URL: str = "redis://127.0.0.1:6379/0"
    POSTGRES_URL: str = "postgresql+asyncpg://kada:kada123@127.0.0.1:5432/kada_ai"
    CACHE_TTL_SECONDS: int = 3600  # 热缓存存活 1 小时，过期后下次读取自动回填
    MAX_MESSAGES: int = 50  # 只喂最近 N 条给模型，避免对话过长撑爆上下文窗口

    # MCP 子服务回调 Go 后端的地址与长效 API Token
    KADA_API_BASE: str = "http://localhost:8080"
    KADA_API_TOKEN: str ="kada_4d6c97c34b195f6d44adba6cef986d2ad59ebe76a2d7b251"


settings = Settings()
