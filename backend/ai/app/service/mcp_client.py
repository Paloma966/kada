import sys
from  langchain_mcp_adapters.client import MultiServerMCPClient


#MCP配置
MCP_SERVER_CONFIG={
    "kada":{
        "command":sys.executable,
        "args":["-m","app.mcp_server.kada_server"],
        "transport":"stdio",
        "cwd":r"E:\kada\backend\ai"
    }
}

mcp_tool:list=[]