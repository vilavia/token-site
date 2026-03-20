#!/bin/bash
# =============================================================================
# Token Site 一键安装脚本
# =============================================================================
# 前置要求: Docker, PostgreSQL, Redis
# 数据库/Redis 连接信息在首次打开网页时配置
#
# curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash
# =============================================================================

set -e

CYAN='\033[0;36m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'

IMAGE="ghcr.io/vilavia/token-site:latest"
INSTALL_DIR="token-site"
GITHUB_RAW="https://raw.githubusercontent.com/vilavia/token-site/main"

info()    { echo -e "  ${CYAN}▸${NC} $1"; }
ok()      { echo -e "  ${GREEN}✓${NC} $1"; }
warn()    { echo -e "  ${YELLOW}!${NC} $1"; }
fail()    { echo -e "  ${RED}✗${NC} $1"; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

download() {
    if command_exists curl; then curl -sSL "$1" -o "$2"
    elif command_exists wget; then wget -q "$1" -O "$2"
    else fail "需要 curl 或 wget"; exit 1; fi
}

sed_i() {
    if sed --version >/dev/null 2>&1; then sed -i "$1" "$2"
    else sed -i '' "$1" "$2"; fi
}

main() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  Token Site 安装${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # 1. 检查 Docker
    if ! command_exists docker; then
        fail "未安装 Docker"
        echo "  安装: curl -fsSL https://get.docker.com | bash"
        exit 1
    fi
    if docker compose version >/dev/null 2>&1; then
        COMPOSE="docker compose"
    elif command_exists docker-compose; then
        COMPOSE="docker-compose"
    else
        fail "需要 Docker Compose"; exit 1
    fi
    ok "Docker 就绪"

    # 2. 检查已有安装
    if [ -d "$INSTALL_DIR" ] && [ -f "$INSTALL_DIR/.env" ]; then
        warn "检测到已有安装: $(pwd)/$INSTALL_DIR"
        read -p "  覆盖重新安装? (y/N): " -r; echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && exit 0
    fi

    mkdir -p "$INSTALL_DIR" && cd "$INSTALL_DIR"

    # 3. 下载 compose 文件
    info "下载配置文件..."
    download "${GITHUB_RAW}/deploy/docker-compose.standalone.yml" "docker-compose.yml"
    download "${GITHUB_RAW}/deploy/.env.example" ".env.example"

    # 替换镜像为我们的
    sed_i "s|image: weishaw/sub2api:latest|image: ${IMAGE}|" docker-compose.yml

    # 补充原版 standalone 里缺少的环境变量
    sed_i '/JWT_EXPIRE_HOUR/a\      - TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY:-}' docker-compose.yml
    sed_i '/TZ=.*/a\      - EPAY_ENABLED=${EPAY_ENABLED:-false}\n      - EPAY_API_URL=${EPAY_API_URL:-}\n      - EPAY_PID=${EPAY_PID:-0}\n      - EPAY_KEY=${EPAY_KEY:-}\n      - EPAY_NOTIFY_URL=${EPAY_NOTIFY_URL:-}\n      - EPAY_RETURN_URL=${EPAY_RETURN_URL:-}\n      - EPAY_USD_TO_RMB=${EPAY_USD_TO_RMB:-7.2}' docker-compose.yml

    ok "配置文件就绪"

    # 4. 生成 .env + 密钥
    info "生成密钥..."
    cp .env.example .env
    JWT=$(openssl rand -hex 32)
    TOTP=$(openssl rand -hex 32)
    sed_i "s/^JWT_SECRET=.*/JWT_SECRET=${JWT}/" .env
    sed_i "s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=${TOTP}/" .env
    chmod 600 .env
    mkdir -p data
    ok "密钥已生成"

    # 5. 拉取镜像
    info "拉取镜像 (自动识别 amd64/arm64)..."
    docker pull "$IMAGE" 2>/dev/null || docker pull "$IMAGE"
    ok "镜像就绪"

    # 6. 启动
    info "启动服务..."
    $COMPOSE up -d
    info "等待启动..."
    for i in $(seq 1 20); do
        curl -sf http://localhost:8080/ >/dev/null 2>&1 && break
        sleep 2
    done

    IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo 'localhost')

    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  安装完成${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "  打开浏览器: http://${IP}:8080"
    echo "  首次访问进入引导页面，配置数据库和管理员账号"
    echo ""
    echo "  目录: $(pwd)"
    echo ""
    echo "  常用命令:"
    echo "    $COMPOSE logs -f sub2api      # 日志"
    echo "    $COMPOSE restart sub2api      # 重启"
    echo "    $COMPOSE down                 # 停止"
    echo "    $COMPOSE pull && $COMPOSE up -d  # 更新"
    echo ""
}

main "$@"
