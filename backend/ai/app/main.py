"""Kada AI 服务入口（FastAPI）。"""

from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.config import settings
from app.route import chat
from app.service import mcp_client
from app.service.db import init_db


@asynccontextmanager
async def lifespan(app: FastAPI):
    # 幂等创建会话表；知识库的向量表由 langchain-postgres 自行管理。
    await init_db()
    # MCP 是增强能力而非硬依赖：连不上就让工具列表为空，AI 仍能纯对话。
    # 注意 try 只包住启动连接，不能包住 yield，否则关停阶段抛的异常会被误报成"连接失败"。
    try:
        tools = await mcp_client.load_tools()
        print(f"[MCP] 已加载 {len(tools)} 个工具 {[t.name for t in tools]}")
    except Exception as exc:
        print(f"[MCP] 连接失败，降级运行：{exc}")
    yield


app = FastAPI(title="Kada AI Service", version="0.1.0", lifespan=lifespan)


@app.get("/healthz")
def healthz():
    return {"status": "ok", "service": "kada-ai", "model": settings.CHAT_MODEL}


# 生产环境前端经 Go 网关同源访问 /api/ai/*，Python 只监听 127.0.0.1，不需要 CORS。
app.include_router(chat.router)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run("app.main:app", host="127.0.0.1", port=8000, reload=True)
