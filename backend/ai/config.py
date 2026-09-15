import os

from conda.gateways.connection.download import CHUNK_SIZE

DEEPSEEK_API_KEY=os.getenv("DEEPSEEK_API_KEY")
CHAT_MODEL="deepseek-flash"
EMBEDDING_MODEL="text"
DASHSCOPE_API_KEY =os.getenv("aliyun")
MILVUS_URL="127.0.0.1:19530"
DOC_COLLECTION_NAME="konwledge_docs"
REDIS_URL="127.0.0.1:6379"
PostgreSQL_URL="127.0.0.1:5432"

#数据处理配置
DOC_DIR="./docs"
CHUNK_SIZE=500
CHUNK_OVERLAP=100