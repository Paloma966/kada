from fastapi import FastAPI
from pymilvus import settings
from contextlib import asynccontextmanager
from backend.ai.app.route import  chat
from fastapi.responses import StreamingResponse
from app.service.db import init_db

@asynccontextmanager
#确保sql表存在
async def lifespan(app:FastAPI):
    await init_db()
    yield


app = FastAPI(title="Kada AI Service", version="0.1.0",lifespan=lifespan)

@app.get("/healthz")
def healtjz():
    return {"status":"ok","service":"kada-ai","mock":settings.ai_mock}

#把chat.py里定义的所有接口挂到app上
app.include_router(chat.router)

