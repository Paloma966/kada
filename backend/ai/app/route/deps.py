"""进入这一侧的守卫：本服务只接受来自 Go 网关的请求。

网关代理 /api/ai/* 时会注入 X-Internal-Secret（并先删掉客户端伪造的同名头），
本服务据此确认"调用方确实是网关"，网关注入的 X-Kada-User-ID 才有资格被当成身份断言。
否则任何能连到本机回环的进程（同机服务、容器、被拿下的前端进程）都能伪造那个头，
去读或删除别人的对话历史。

AI_INTERNAL_SECRET 为空时不做校验，仅限本机开发；启动时会打印一行提示，生产环境
必须在网关侧（backend/.env）和本服务侧（deploy/ai.env）配同一个值。
"""

import secrets

from fastapi import Header, HTTPException, status

from app.config import settings

# 与 Go 侧 handler.aiInternalSecretHeader 保持一致。
INTERNAL_SECRET_HEADER = "X-Internal-Secret"


def require_gateway(
    x_internal_secret: str = Header(default="", alias=INTERNAL_SECRET_HEADER),
) -> None:
    """FastAPI 依赖：校验调用方是网关；未配置密钥时直接放行。"""
    if not settings.AI_INTERNAL_SECRET:
        return
    # compare_digest 逐字节比较，不因为提前返回而泄露前缀匹配情况。
    if not secrets.compare_digest(x_internal_secret, settings.AI_INTERNAL_SECRET):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="only the Go gateway may call this service",
        )
