import os

from pydantic_settings import BaseSettings

class Settings():
    DEEPSEEK_API_KEY=os.getenv("DEEPSEEK_API_KEY")
    CHAT_MODEL="deepseek-flash"
    AI_MAX_TOKENS = 1024  # 一次回答最多 token 数（控制长度=控制成本）
    AI_QUOTA_DAILY = 50
    DEEPSEEK_BASE_URL="https://api.deepseek.com"

    EMBEDDING_MODEL="text-embedding-v3"
    DASHSCOPE_API_KEY =os.getenv("aliyun")
    QWEN_BASE_URL="https://dashscope-intl.aliyuncs.com/compatible-mode/v1"

    MILVUS_URL="127.0.0.1:19530"
    DOC_COLLECTION_NAME="knowledge_docs"
    REDIS_URL="redis://127.0.0.1:6379/0"
    PostgreSQL_URL="postgresql+asyncpg://kada:kada123@127.0.0.1:5432/kada_ai"
    #数据处理配置
    DOC_DIR="./docs"
    CHUNK_SIZE=500
    CHUNK_OVERLAP=100

    CACHE_TTL_SECONDS = 3600
    MAX_MESSAGES = 50
# 创建唯一的设置实例。
settings=Settings()