import os

from conda.gateways.connection.download import CHUNK_SIZE
from pydantic_settings import BaseSettings

class Settngs(BaseSettings):
    DEEPSEEK_API_KEY=os.getenv("DEEPSEEK_API_KEY")
    CHAT_MODEL="deepseek-flash"
    EMBEDDING_MODEL="text-embedding-v3"
    DASHSCOPE_API_KEY =os.getenv("aliyun")
    DEEPSEEK_BASE_URL="https://api.deepseek.com"
    QWEN_BASE_URL="https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
    MILVUS_URL="127.0.0.1:19530"
    DOC_COLLECTION_NAME="konwledge_docs"
    REDIS_URL="redis://127.0.0.1:6379/0"
    PostgreSQL_URL="postgresql://kada:kada@127.0.0.1:5432/kada_ai"
    #数据处理配置
    DOC_DIR="./docs"
    CHUNK_SIZE=500
    CHUNK_OVERLAP=100
# 创建唯一的设置实例。
settings=Settngs()