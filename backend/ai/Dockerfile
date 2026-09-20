# Kada AI service image (FastAPI + LangChain).
# The container stays on the internal compose network and is only reached
# through the Go gateway (Go proxies /api/ai/* to http://ai:8000).
FROM python:3.11-slim

# Default to the China mirror for builds on domestic servers; override with
# --build-arg PIP_INDEX_URL=https://pypi.org/simple when building abroad.
ARG PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple
ENV PIP_INDEX_URL=${PIP_INDEX_URL} \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

# Install dependencies first so code changes do not invalidate the pip layer.
COPY requirements.txt ./
RUN pip install -r requirements.txt

# Application code and the RAG knowledge base documents ingested at deploy time.
COPY app ./app
COPY docs ./docs

# Run as a non-root user, consistent with the Go image.
RUN useradd -r -u 10001 kada && chown -R kada:kada /app
USER kada

EXPOSE 8000

# Must bind 0.0.0.0 inside the container; --reload is a local-dev feature only.
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
