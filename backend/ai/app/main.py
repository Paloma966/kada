from fastapi import FastAPI
from contextlib import asynccontextmanager
from app.route import  chat
from fastapi.responses import StreamingResponse
from app.service.db import init_db
from app.config import settings
from fastapi.middleware.cors import CORSMiddleware
@asynccontextmanager
#确保sql表存在
async def lifespan(app:FastAPI):
    await init_db()
    yield


app = FastAPI(title="Kada AI Service", version="0.1.0",lifespan=lifespan)

@app.get("/healthz")
def healtjz():
    return {"status":"ok","service":"kada-ai","mock":settings.CHAT_MODEL}

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000"],
    allow_methods=["*"],
    allow_headers=["*"],

)

#把chat.py里定义的所有接口挂到app上
app.include_router(chat.router)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host="127.0.0.1", port=8000, reload=True)