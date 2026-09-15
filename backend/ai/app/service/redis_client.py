
import redis.asyncio as aioredis
from app.config import settings
#检验是否建立连接
_redis=None
#取redis客户端 ，连不上自行降级
async  def get_redis():
    global _redis
    if _redis is None:
        try:
            _redis=aioredis.from_url(
                settings.REDIS_URL,
                decode_respose=True
            )
            await _redis.ping()
        except Exception:
            return None
    return _redis