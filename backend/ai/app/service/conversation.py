"""会话历史的读写：Redis 缓存 + PostgreSQL 持久化（cache-aside）。

读：先查 Redis，没有再查 PostgreSQL 并回填缓存。
写：先写 PostgreSQL（唯一真相源），成功后再刷新 Redis。

入参 user_id 统一是字符串（来自网关注入的 X-Kada-User-ID 请求头），
落库查询时再转成 int（数据库列是整数）。
"""

import json
import uuid

from sqlmodel import select
from sqlmodel.ext.asyncio.session import AsyncSession

# 必须用 SQLModel 自带的 AsyncSession（有 .exec()，和 select() 搭配）；
# 误用 sqlalchemy.ext.asyncio.AsyncSession 会报没有 exec 方法。
from app.config import settings
from app.service.db import Conversation, Message, engine
from app.service.redis_client import get_redis


def _cache_key(user_id: str, conversation_id: str) -> str:
    # key 里带 user_id，不同用户的会话天然隔离。
    return f"ai:conv:{user_id}:{conversation_id}"


async def load_messages(user_id: str, conversation_id: str) -> list:
    """取某会话的历史；任何一层都没有（新会话）时返回空列表。"""
    key = _cache_key(user_id, conversation_id)
    cid = uuid.UUID(conversation_id)

    # 1) 先查缓存，命中则一条数据库查询都不用跑。
    redis = await get_redis()
    if redis is not None:
        data = await redis.get(key)
        if data:
            return json.loads(data)

    # 2) 缓存未命中，查 PostgreSQL。会话归属条件直接写进 WHERE：
    #    不属于该用户的会话根本查不出来（所有权在查询层保证）。
    async with AsyncSession(engine) as session:
        conversation = (
            await session.exec(
                select(Conversation).where(
                    Conversation.id == cid,
                    Conversation.user_id == int(user_id),
                )
            )
        ).first()
        if conversation is None:
            return []

        rows = (
            await session.exec(
                select(Message)
                .where(Message.conversation_id == cid)
                .order_by(Message.created_at)
            )
        ).all()

    # 只保留最近 MAX_MESSAGES 条，和缓存口径保持一致。
    messages = [
        {"role": m.role, "content": m.content} for m in rows
    ][-settings.MAX_MESSAGES :]

    # 3) 回填缓存，下次读取直接命中。
    if messages and redis is not None:
        await redis.set(
            key,
            json.dumps(messages, ensure_ascii=False),
            ex=settings.CACHE_TTL_SECONDS,
        )
    return messages


async def new_conversation(user_id: str) -> str:
    """新建一条会话记录，返回会话 id。"""
    async with AsyncSession(engine) as session:
        conversation = Conversation(user_id=int(user_id))
        session.add(conversation)
        await session.commit()
        await session.refresh(conversation)
        return str(conversation.id)


async def append_message(
    user_id: str, conversation_id: str, role: str, content: str
) -> None:
    """追加一条消息。先写库再刷缓存；会话不属于该用户时直接忽略。"""
    cid = uuid.UUID(conversation_id)

    async with AsyncSession(engine) as session:
        conversation = (
            await session.exec(
                select(Conversation).where(
                    Conversation.id == cid,
                    Conversation.user_id == int(user_id),
                )
            )
        ).first()
        if conversation is None:
            return

        session.add(Message(conversation_id=cid, role=role, content=content))
        await session.commit()

    # 缓存里有就就地追加（省一次全量回查）；没有则不主动写，
    # 留给下次读取时回填，避免在写路径上重复查库。
    redis = await get_redis()
    if redis is not None:
        key = _cache_key(user_id, conversation_id)
        data = await redis.get(key)
        if data:
            messages = json.loads(data)
            messages.append({"role": role, "content": content})
            await redis.set(
                key,
                json.dumps(messages[-settings.MAX_MESSAGES :], ensure_ascii=False),
                ex=settings.CACHE_TTL_SECONDS,
            )


async def list_conversations(user_id: str) -> list:
    """列出该用户的会话（最新在前），供单会话模式取"当前会话"。"""
    async with AsyncSession(engine) as session:
        result = await session.exec(
            select(Conversation)
            .where(Conversation.user_id == int(user_id))
            .order_by(Conversation.created_at.desc())
        )
        conversations = result.all()
    return [
        {
            "conversation_id": str(c.id),
            "created_at": c.created_at.isoformat(),
        }
        for c in conversations
    ]


async def delete_conversation(user_id: str, conversation_id: str) -> bool:
    """删除单个会话（消息靠数据库 ON DELETE CASCADE 连带删除）并清缓存。"""
    cid = uuid.UUID(conversation_id)
    async with AsyncSession(engine) as session:
        conversation = (
            await session.exec(
                select(Conversation).where(
                    Conversation.id == cid,
                    Conversation.user_id == int(user_id),
                )
            )
        ).first()
        if conversation is None:
            return False
        await session.delete(conversation)
        await session.commit()

    redis = await get_redis()
    if redis is not None:
        await redis.delete(_cache_key(user_id, conversation_id))
    return True


async def get_current_conversation(user_id: str) -> dict:
    """单会话模式：返回该用户最新一个会话及其消息；没有则返回空。"""
    conversations = await list_conversations(user_id)
    if not conversations:
        return {"conversation_id": None, "messages": []}
    cid = conversations[0]["conversation_id"]
    messages = await load_messages(user_id, cid)
    return {"conversation_id": cid, "messages": messages}


async def restart_conversation(user_id: str) -> str:
    """重新开始：删掉该用户所有旧会话（含缓存），返回全新空会话 id。"""
    conversations = await list_conversations(user_id)
    for conversation in conversations:
        await delete_conversation(user_id, conversation["conversation_id"])
    return await new_conversation(user_id)
