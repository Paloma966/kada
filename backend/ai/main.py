from fastapi import FastAPI
from pymilvus import settings
from streamlit.elements import media

from route.chat import  mock_stream
from fastapi.responses import StreamingResponse
app = FastAPI(title="Kada AI Service", version="0.1.0")

@app.get("/healthz")
def healtjz():
    return {"status":"ok","service":"kada-ai","mock":settings.ai_mock}
@app.get("/chat")
def chat():
    return StreamingResponse(
        mock_stream("真的假的"),
        media_type="text/event-steam"
    )

if __name__=="__main__":
    import uvicorn
    uvicorn.run(app,host="0.0.0.0",port=8000)

