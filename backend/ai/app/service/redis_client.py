"""Redis 客户端单例。

Redis 只是会话历史的热缓存，不是必需品：连不上时返回 None，
业务降级为直接查 PostgreSQL，AI 对话不受影响（fail-open）。
"""

import redis.asyncio as aioredis

from app.config import settings

_redis = None


async def get_redis():
    """返回 Redis 客户端；连接不可用时返回 None。

    首次连接失败后不缓存失败状态，下次调用会重试，方便 Redis 中途恢复。
    """
    global _redis
    if _redis is None:
        try:
            # decode_responses：取回来直接是 str，否则还要手动 decode。
            client = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
            await client.ping()
            _redis = client
        except Exception:
            return None
    return _redis
