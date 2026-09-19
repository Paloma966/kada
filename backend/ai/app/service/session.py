import json
import uuid

from langchain_community.embeddings import awa
from sqlmodel import select
from sqlmodel.ext.asyncio.session import AsyncSession
from app.config import settings
from app.service.db import Conversation,Message,engine
from app.service.redis_client import get_redis

#把用户的id放进key
def cache_key(user_id:str,conversation_id:str)->str:
    return f"ai:conv:{user_id}:{conversation_id}"
#取会话历史信息
async  def load_messages(user_id:str,conversation_id:str) ->list:
    key=cache_key(user_id,conversation_id)
    cid=uuid.UUID(conversation_id)
    #先去reids找
    r=await get_redis()
    if r is not None:
        data=await  r.get(key)
        if data:
            return json.loads(data)
    #没找到去PostgreSQL找
    async with AsyncSession(engine) as session:
        conv=(
            await  session.exec(
                select(Conversation).where(
                    Conversation.id==cid,
                    Conversation.user_id==user_id
                )
            )
        ).first()
        if conv is None:
            return []
        #查出所有的消息并按时间排序
        result=await session.exec(
            select(Message)
            .where(Message.conversation_id==cid)
            .order_by(Message.created_at)
        )
        rows=result.all()
    #把ORM对象变成前端要的格式：[{"role"...","content"},...]
    messages=[{"role":m.role,"content":m.content}for m in rows]
    if messages and r is not None:
        await r.set(key,json.dumps(messages,ensure_ascii=False),ex=settings.CACHE_TTL_SECONDS)
    return messages

#写入会话记录
async def new_conversation(user_id:str)->str:
    async with AsyncSession(engine) as session:
        conv=Conversation(user_id=user_id)
        session.add(conv)
        await session.commit()
        await session.refresh(conv)
        return  str(conv.id)


#追加一条消息 先PostgreSQL再redis
async def append_message(user_id:str,conversation_id:str,role:str,content:str):
    cid=uuid.UUID(conversation_id)
    #写PostgreSQL
    async  with AsyncSession(engine) as session:
        conv=(
            await session.exec(
                select(Conversation).where(
                    Conversation.id==cid,
                    Conversation.user_id==user_id,
                )
            )
        ).first()
        if conv is None:
            return
        session.add(Message(conversation_id=cid,role=role,content=content))
        await session.commit()

    #刷新Redis缓存
    r=await  get_redis()
    if r is not None:
        key=cache_key(user_id,conversation_id)
        data=await r.get(key)
        if data:
            messages=json.loads(data)
            messages.append({"role":role,"content":content})
            messages=messages[-settings.MAX_MESSAGES:]
            await r.set(key,json.dumps(messages,ensure_ascii=False),ex=settings.CACHE_TTL_SECONDS)
#会话列表
async def list_conversations(user_id:str)->list:
    async with AsyncSession(engine)as session:
        result=await session.exec(
            select(Conversation)
            .where(Conversation.user_id==user_id)
            .order_by(Conversation.created_at.desc())
        )
        convs=result.all()
    return[
        {"conversation_id":str(c.id),"created_at":c.created_at.isoformat()}
        for c in convs
    ]
#删除会话
async def delete_conversation(user_id:str,conversation_id:str)->bool:
    cid=uuid.UUID(conversation_id)
    async with AsyncSession(engine) as session:
        conv=(
            await  session.exec(
                select(Conversation).where(
                    Conversation.id==cid,
                    Conversation.user_id==user_id,
                )
            )
        ).first()
        if conv is None:
            return False
        await session.delete(conv)
        await session.commit()
    r=await get_redis()
    if r is not   None:
        await  r.delete(cache_key(user_id,conversation_id))
    return True
#取当前会话+全部消息
async def get_current_session(user_id:str)->dict:
    convs=await list_conversations(user_id)
    if not convs:
        return {"conversation_id":None,"messages":[]}
    cid=convs[0]["conversation_id"]
    msgs=await load_messages(user_id,cid)
    return {"conversation_id":cid,"messages":msgs}


#重新开始：删掉该用户所有的旧会话
async def restart_session(user_id:str)->str:
    convs=await list_conversations(user_id)
    for c in convs:
        await delete_conversation(user_id,c["conversation_id"])
    return await new_conversation(user_id)
