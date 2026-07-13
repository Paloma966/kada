#!/bin/bash
set -e

echo "========================================="
echo "  Kada 服务器一键部署脚本"
echo "  IP: 47.122.124.48"
echo "========================================="

# ========== 1. 更新系统 + 装基础工具 ==========
echo "[1/6] 更新系统..."
apt update -y && apt upgrade -y
apt install -y curl wget git vim ufw htop ca-certificates gnupg lsb-release

# ========== 2. 安装 Docker ==========
echo "[2/6] 安装 Docker..."
curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt update -y
apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
systemctl enable docker --now

# ========== 3. 创建项目目录 + 克隆代码 ==========
echo "[3/6] 拉取项目代码..."
mkdir -p /opt/kada
cd /opt/kada

# 方式A: 从你的开发机 scp 过来（推荐，保留 .env 等配置）
# 先在你的开发机上跑: scp -r /home/chun/dev/projects/kada/* root@47.122.124.48:/opt/kada/

# 方式B: 或者用 git（如果有仓库的话）
# git clone https://github.com/yourusername/kada.git /opt/kada

echo "请用开发机 scp 上传代码:"
echo "  scp -r /home/chun/dev/projects/kada/backend root@47.122.124.48:/opt/kada/"
echo "  scp -r /home/chun/dev/projects/kada/docker-compose.yml root@47.122.124.48:/opt/kada/"
echo "  scp -r /home/chun/dev/projects/kada/nginx root@47.122.124.48:/opt/kada/"

# ========== 4. 配置防火墙 ==========
echo "[4/6] 配置防火墙..."
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# ========== 5. 启动服务 ==========
echo "[5/6] 启动 Docker 服务..."
# cd /opt/kada && docker compose up -d

# ========== 6. 检查状态 ==========
echo "[6/6] 检查服务状态..."
docker --version
docker compose version

echo ""
echo "========================================="
echo "  基础环境安装完成！"
echo "  接下来:"
echo "  1. scp 上传项目代码到 /opt/kada/"
echo "  2. cd /opt/kada && docker compose up -d"
echo "  3. curl http://localhost:8080/api/health"
echo "========================================="
