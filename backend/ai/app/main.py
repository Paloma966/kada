from fastapi import FastAPI
from contextlib import asynccontextmanager
from app.route import chat
from app.service.db import init_db
from app.config import settings

@asynccontextmanager
#确保sql表存在
async def lifespan(app:FastAPI):
    await init_db()
    yield


app = FastAPI(title="Kada AI Service", version="0.1.0",lifespan=lifespan)

@app.get("/healthz")
def healthz():
    return {"status":"ok","service":"kada-ai","mock":settings.CHAT_MODEL}

# 注：CORS 已移除 —— 现在前端通过 Go 网关（同源 /api/ai/*）访问，
# Python 只监听 127.0.0.1 内网，不再需要跨域放行。

#把chat.py里定义的所有接口挂到app上
app.include_router(chat.router)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host="127.0.0.1", port=8000, reload=True)
