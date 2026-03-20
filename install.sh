#!/bin/bash
# =============================================================================
# Token Site 一键安装脚本
# =============================================================================
# 前置要求: 服务器已安装 Docker, PostgreSQL, Redis
# 数据库连接信息在首次打开网页时通过引导页面配置
#
# 用法:
#   curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash
# =============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

IMAGE="ghcr.io/vilavia/token-site:latest"
INSTALL_DIR="token-site"
GITHUB_RAW="https://raw.githubusercontent.com/vilavia/token-site/main"

print_info()    { echo -e "  ${CYAN}▸${NC} $1"; }
print_success() { echo -e "  ${GREEN}✓${NC} $1"; }
print_warning() { echo -e "  ${YELLOW}!${NC} $1"; }
print_error()   { echo -e "  ${RED}✗${NC} $1"; }

generate_secret() { openssl rand -hex 32; }
command_exists()  { command -v "$1" >/dev/null 2>&1; }

download() {
    if command_exists curl; then
        curl -sSL "$1" -o "$2"
    elif command_exists wget; then
        wget -q "$1" -O "$2"
    else
        print_error "需要 curl 或 wget"; exit 1
    fi
}

sed_replace() {
    if sed --version >/dev/null 2>&1; then
        sed -i "$1" "$2"
    else
        sed -i '' "$1" "$2"
    fi
}

# =============================================================================
main() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  Token Site 安装${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # 检查依赖
    local missing=()
    command_exists docker  || missing+=("docker")
    command_exists openssl || missing+=("openssl")
    if [ ${#missing[@]} -gt 0 ]; then
        print_error "缺少: ${missing[*]}"
        echo "  安装 Docker: curl -fsSL https://get.docker.com | bash"
        exit 1
    fi

    if docker compose version >/dev/null 2>&1; then
        COMPOSE="docker compose"
    elif command_exists docker-compose; then
        COMPOSE="docker-compose"
    else
        print_error "需要 Docker Compose"; exit 1
    fi
    print_success "Docker 就绪"

    # 检查已有安装
    if [ -d "$INSTALL_DIR" ] && [ -f "$INSTALL_DIR/.env" ]; then
        print_warning "检测到已有安装"
        read -p "  覆盖配置重新安装? (y/N): " -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && exit 0
    fi

    # 创建目录
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"

    # 下载 compose 和 env 模板
    print_info "下载配置文件..."
    download "${GITHUB_RAW}/deploy/docker-compose.standalone.yml" "docker-compose.yml"
    download "${GITHUB_RAW}/deploy/.env.example" ".env.example"

    # 替换镜像为我们的
    sed_replace "s|image: weishaw/sub2api:latest|image: ${IMAGE}|" docker-compose.yml

    # 添加 TOTP 环境变量（原版 standalone 里没有）
    sed_replace '/JWT_EXPIRE_HOUR/a\      - TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY:-}' docker-compose.yml

    # 添加易支付环境变量
    sed_replace '/TZ=.*/a\      - EPAY_ENABLED=${EPAY_ENABLED:-false}\n      - EPAY_API_URL=${EPAY_API_URL:-}\n      - EPAY_PID=${EPAY_PID:-0}\n      - EPAY_KEY=${EPAY_KEY:-}\n      - EPAY_NOTIFY_URL=${EPAY_NOTIFY_URL:-}\n      - EPAY_RETURN_URL=${EPAY_RETURN_URL:-}\n      - EPAY_USD_TO_RMB=${EPAY_USD_TO_RMB:-7.2}' docker-compose.yml

    print_success "配置文件就绪"

    # 生成 .env
    print_info "生成密钥..."
    cp .env.example .env
    JWT_SECRET=$(generate_secret)
    TOTP_KEY=$(generate_secret)
    sed_replace "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" .env
    sed_replace "s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=${TOTP_KEY}/" .env

    chmod 600 .env
    mkdir -p data
    print_success "密钥已生成"

    # 拉取镜像并启动
    print_info "拉取镜像..."
    docker pull "$IMAGE" --quiet 2>/dev/null || docker pull "$IMAGE"
    print_success "镜像就绪"

    print_info "启动服务..."
    $COMPOSE up -d

    # 等待
    print_info "等待服务启动..."
    for i in $(seq 1 30); do
        curl -sf http://localhost:8080/health >/dev/null 2>&1 && break
        sleep 2
    done

    SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo 'localhost')

    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  安装完成!${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "  打开浏览器访问: http://${SERVER_IP}:8080"
    echo "  首次访问会进入引导页面，按提示配置数据库连接"
    echo ""
    echo "  安装目录: $(pwd)"
    echo ""
    echo "  常用命令:"
    echo "    $COMPOSE logs -f sub2api     # 查看日志"
    echo "    $COMPOSE restart sub2api     # 重启"
    echo "    $COMPOSE down                # 停止"
    echo "    $COMPOSE pull && $COMPOSE up -d  # 更新"
    echo ""

    # 数据库迁移提示
    echo -e "${YELLOW}  重要: 配置好数据库后，需要执行一次建表:${NC}"
    echo ""
    echo '    psql -h 数据库地址 -U 用户名 -d 数据库名 << EOF'
    echo '    CREATE TABLE IF NOT EXISTS payment_orders ('
    echo '      id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL,'
    echo '      order_no VARCHAR(64) NOT NULL UNIQUE, trade_no VARCHAR(64),'
    echo '      amount_usd DECIMAL(20,8) NOT NULL, amount_rmb DECIMAL(20,2) NOT NULL,'
    echo '      exchange_rate DECIMAL(10,4) NOT NULL, status VARCHAR(32) DEFAULT '"'"'pending'"'"','
    echo '      pay_type VARCHAR(32), paid_at TIMESTAMPTZ,'
    echo '      created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW());'
    echo '    CREATE INDEX IF NOT EXISTS idx_payment_orders_user_id ON payment_orders(user_id);'
    echo '    CREATE TABLE IF NOT EXISTS chat_conversations ('
    echo '      id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL,'
    echo '      title VARCHAR(200) DEFAULT '"'"'New Chat'"'"', model VARCHAR(100) DEFAULT '"'"''"'"','
    echo '      created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW());'
    echo '    CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_id ON chat_conversations(user_id);'
    echo '    CREATE TABLE IF NOT EXISTS chat_messages ('
    echo '      id BIGSERIAL PRIMARY KEY,'
    echo '      conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,'
    echo '      role VARCHAR(20) NOT NULL, content TEXT DEFAULT '"'"''"'"','
    echo '      created_at TIMESTAMPTZ DEFAULT NOW());'
    echo '    CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);'
    echo '    EOF'
    echo ""
}

main "$@"
