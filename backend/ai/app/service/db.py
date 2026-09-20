"""会话相关的 ORM 模型与数据库引擎。

只有会话两张表（ai_conversations / ai_messages）由 SQLModel 管理；
知识库的 langchain_pg_* 向量表由 langchain-postgres 自行创建维护，不在这里建。
"""

from datetime import datetime
from uuid import UUID, uuid4

from sqlalchemy.ext.asyncio import create_async_engine
from sqlmodel import Field, SQLModel

from app.config import settings

# pool_pre_ping：取连接前先探活，避免拿到被数据库/容器断开的死连接。
engine = create_async_engine(settings.POSTGRES_URL, echo=False, pool_pre_ping=True)


class Conversation(SQLModel, table=True):
    # 显式指定表名，避免 ORM 默认生成的复数表名不可控。
    __tablename__ = "ai_conversations"

    # 主键由数据库生成，不暴露自增 id，对外统一用无序的 UUID。
    id: UUID = Field(default_factory=uuid4, primary_key=True)
    user_id: int = Field(index=True)
    created_at: datetime = Field(default_factory=datetime.utcnow)


class Message(SQLModel, table=True):
    __tablename__ = "ai_messages"

    id: UUID = Field(default_factory=uuid4, primary_key=True)
    # 父会话删除时消息一并删除（数据库层 ON DELETE CASCADE），不留孤儿数据。
    conversation_id: UUID = Field(
        foreign_key="ai_conversations.id", ondelete="CASCADE", index=True
    )
    role: str  # user / assistant
    content: str
    created_at: datetime = Field(default_factory=datetime.utcnow)


async def init_db() -> None:
    """服务启动时建表；SQLModel.metadata.create_all 是幂等的，表已存在则跳过。"""
    async with engine.begin() as conn:
        await conn.run_sync(SQLModel.metadata.create_all)
