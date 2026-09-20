"""知识库入库脚本：读取 docs 目录文档 → 切块 → 向量化 → 写入 pgvector。

在 backend/ai 目录下运行：
    python -m app.scripts.ingest_docs
也可以在 IDE 里直接运行本文件（下方的 sys.path 处理兼容这两种方式）。
每次运行都会先清空同名 collection 再全量重建，保证可重复执行。
"""

import logging
import sys
from pathlib import Path

# 直接运行本文件时 sys.path[0] 是 scripts 目录，import 不到 app 包；
# 本文件在 backend/ai/app/scripts/ 下，向上三级才是 backend/ai，手动加进去。
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from langchain_postgres import PGVector
from langchain_text_splitters import RecursiveCharacterTextSplitter

from app.config import settings
from app.service.embedding import get_embeddings

# 脚本默认不输出 INFO 级别以下日志，这里显式配置，否则跑完没有任何提示。
logging.basicConfig(level=logging.INFO, format="%(message)s")

DOCS_DIR = Path(settings.DOC_DIR)
# langchain-postgres 用同步 psycopg 驱动，替换掉会话库的 asyncpg。
PG_URL = settings.POSTGRES_URL.replace("+asyncpg", "+psycopg")


def load_documents() -> list[str]:
    """读取 docs 目录下所有 markdown 和 txt 文件的全文。"""
    texts = []
    files = sorted(DOCS_DIR.glob("*.md")) + sorted(DOCS_DIR.glob("*.txt"))
    for file in files:
        texts.append(file.read_text(encoding="utf-8"))
    return texts


def main() -> None:
    docs = load_documents()
    if not docs:
        logging.info("docs 目录下没有可入库的文件")
        return

    splitter = RecursiveCharacterTextSplitter(
        chunk_size=settings.CHUNK_SIZE,
        chunk_overlap=settings.CHUNK_OVERLAP,
    )
    chunks = []
    for doc in docs:
        chunks.extend(splitter.split_text(doc))
    logging.info("文档共切成 %d 块，开始向量化入库...", len(chunks))

    store = PGVector(
        embeddings=get_embeddings(),
        connection=PG_URL,
        collection_name=settings.DOC_COLLECTION_NAME,
    )
    store.create_tables_if_not_exists()
    # 全量重建：先删旧 collection（首次运行不存在会抛异常，忽略即可），再新建。
    try:
        store.delete_collection()
    except Exception:
        pass
    store.create_collection()
    store.add_texts(chunks)
    print(f"写入完成：collection={settings.DOC_COLLECTION_NAME}，共 {len(chunks)} 条")


if __name__ == "__main__":
    main()
