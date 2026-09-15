import uuid
from datetime import datetime, timezone

from langchain_classic.chains.sql_database.prompt import POSTGRES_PROMPT
from psycopg import conninfo
from sqlalchemy import Column, DateTime, ForeignKey
from sqlalchemy.ext.asyncio import create_async_engine
from sqlmodel import SQLModel, Field

from backend.ai.app.config import settings


# 会话表生成会话id进行多轮聊天
class Conversation(SQLModel,table=True):
    __tablename__ = "ai_converations"
    id:uuid.UUID=Field(default_factory=uuid.uuid4(),primary_key=True)
    user_id:str=Field(index=True)
    #创建时间 timezone是时区列
    create_at:datetime=Field(
        sa_column=Column(DateTime(timezone=True),nullable=False),
        default_factory=lambda :datetime.now(timezone.utc)

    )


#消息表
class Message(SQLModel,table=True):
    __tablename__ = "ai_messages"

    id :int |None=Field(default=None,primary_key=True)
    #外键 消息关联
    conversation_id:uuid.UUID=Field(
        sa_column=Column(
            ForeignKey("ai_conversation",ondelete="CASCADE"),
            index=True,
            nullable=False,
        ),
    )
    role:str
    content:str
    created_at:datetime=Field(
        sa_column=Column(DateTime(timezone=True),nullable=False),
        default_factory=lambda :datetime.now(timezone.utc)
    )
#连接
enging=create_async_engine(
    POSTGRES_PROMPT,
    echo=False,
)
#建表
async  def init_db():
    async with enging.begin() as conn:
        await conn.run_sync(SQLModel.metadata.create_all())