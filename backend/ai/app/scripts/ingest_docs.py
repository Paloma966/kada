#文档入库脚本
import logging
import sys
from pathlib import Path
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))

from langchain_postgres import PGVector
from langchain_text_splitters import RecursiveCharacterTextSplitter

from app.config import settings
from app.service.embedding import get_embeddings
#docs目录
DOCS_DIR = Path(settings.DOC_DIR)
#向量数据库表名
COLLECTION_NAME=settings.DOC_COLLECTION_NAME
#连接到postgreasql
PG_URL=settings.PostgreSQL_URL.replace("postgresql://", "postgresql+psycopg://")
def load_document()->list[str]:
    texts=[]
    for f in sorted(DOCS_DIR.glob("*.md"))+sorted(DOCS_DIR.glob("*.txt")):
        content=f.read_text(encoding="utf-8")
        texts.append(content)
    return texts

#文档向量化
def main():
    logging.info("读取文档")
    docs=load_document()
    if not docs:
        logging.info("docs 目录下没有 合规文件")
        return
    logging.info("将数据切块")
    splitter=RecursiveCharacterTextSplitter(
        chunk_size=settings.CHUNK_SIZE,
        chunk_overlap=settings.CHUNK_OVERLAP
    )
    chunks=[]
    for d in docs:
        chunks.extend(splitter.split_text(d))
        logging.info(f"共切成{len(chunks)}块")
        logging.info("向量化和入库")
        store=PGVector(
            embeddings=get_embeddings(),
            connection=PG_URL,
            collection_name=COLLECTION_NAME
        )
        try:
            store.delete_collection()
            logging.info("清除旧数据成功")
        except Exception as e:
            logging.info("首次入库")
        store.add_texts(chunks)
        print(f"  ✅ 写入完成：表 {COLLECTION_NAME}，共 {len(chunks)} 条")