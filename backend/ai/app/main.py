from fastapi import FastAPI
from contextlib import asynccontextmanager

from langchain_mcp_adapters.client import MultiServerMCPClient

from app.route import chat
from app.service import mcp_client
from app.service.db import init_db
from app.config import settings

@asynccontextmanager
#确保sql表存在
async def lifespan(app:FastAPI):
    await init_db()
    try:
        client=MultiServerMCPClient(mcp_client.MCP_SERVER_CONFIG)
        mcp_client.mcp_tool=await client.get_tools()
        print(f"[MCP]已经加载{len(mcp_client.mcp_tool)}个工具"
              f"{[t.name for t in mcp_client.mcp_tool]}")
        yield
    except Exception as e:
        print(f"[MCP]连接失败，降级运行:{e}")
        mcp_client.mcp_tool=[]
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
