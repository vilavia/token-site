#!/bin/bash
# =============================================================================
# Token Site 一键安装脚本
# =============================================================================
# 前置要求: 服务器上已安装 Docker、PostgreSQL、Redis
#
# 用法:
#   curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash
# =============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

REPO_URL="https://github.com/vilavia/token-site.git"
INSTALL_DIR="token-site"

print_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[OK]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

generate_secret() { openssl rand -hex 32; }
command_exists()  { command -v "$1" >/dev/null 2>&1; }

sed_replace() {
    if sed --version >/dev/null 2>&1; then
        sed -i "$1" "$2"
    else
        sed -i '' "$1" "$2"
    fi
}

# =============================================================================
check_deps() {
    local missing=()
    command_exists docker  || missing+=("docker")
    command_exists git     || missing+=("git")
    command_exists openssl || missing+=("openssl")

    if [ ${#missing[@]} -gt 0 ]; then
        print_error "缺少依赖: ${missing[*]}"
        echo "  Docker:  curl -fsSL https://get.docker.com | bash"
        echo "  Git:     apt install git / yum install git"
        exit 1
    fi

    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    elif command_exists docker-compose; then
        COMPOSE_CMD="docker-compose"
    else
        print_error "需要 Docker Compose"
        exit 1
    fi
    print_success "依赖检查通过"
}

# =============================================================================
main() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  Token Site 安装向导${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    check_deps

    # 检查已有安装
    if [ -d "$INSTALL_DIR" ] && [ -f "$INSTALL_DIR/deploy/.env" ]; then
        print_warning "检测到已有安装: $INSTALL_DIR/"
        read -p "重新安装会覆盖配置，继续? (y/N): " -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && exit 0
    fi

    # =========================================================================
    # 收集数据库和Redis连接信息
    # =========================================================================
    echo -e "${CYAN}PostgreSQL 连接信息${NC} (需要提前创建好数据库和用户)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    read -p "  数据库地址 [127.0.0.1]: " DB_HOST; DB_HOST=${DB_HOST:-127.0.0.1}
    read -p "  数据库端口 [5432]: " DB_PORT; DB_PORT=${DB_PORT:-5432}
    read -p "  数据库用户 [sub2api]: " DB_USER; DB_USER=${DB_USER:-sub2api}
    read -p "  数据库名称 [sub2api]: " DB_NAME; DB_NAME=${DB_NAME:-sub2api}
    read -sp "  数据库密码: " DB_PASS; echo ""
    echo ""

    echo -e "${CYAN}Redis 连接信息${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    read -p "  Redis 地址 [127.0.0.1]: " REDIS_HOST; REDIS_HOST=${REDIS_HOST:-127.0.0.1}
    read -p "  Redis 端口 [6379]: " REDIS_PORT; REDIS_PORT=${REDIS_PORT:-6379}
    read -p "  Redis 密码 (无密码直接回车): " REDIS_PASS
    echo ""

    # Docker 访问宿主机用 host.docker.internal
    DOCKER_DB_HOST="$DB_HOST"
    DOCKER_REDIS_HOST="$REDIS_HOST"
    if [ "$DB_HOST" = "127.0.0.1" ] || [ "$DB_HOST" = "localhost" ]; then
        DOCKER_DB_HOST="host.docker.internal"
    fi
    if [ "$REDIS_HOST" = "127.0.0.1" ] || [ "$REDIS_HOST" = "localhost" ]; then
        DOCKER_REDIS_HOST="host.docker.internal"
    fi

    # =========================================================================
    # 克隆代码 & 构建
    # =========================================================================
    print_info "克隆代码..."
    if [ -d "$INSTALL_DIR" ]; then
        cd "$INSTALL_DIR" && git pull --quiet && cd ..
    else
        git clone --quiet "$REPO_URL" "$INSTALL_DIR"
    fi
    print_success "代码准备完成"

    cd "$INSTALL_DIR/deploy"

    # =========================================================================
    # 生成 .env
    # =========================================================================
    print_info "生成配置..."
    JWT_SECRET=$(generate_secret)
    TOTP_KEY=$(generate_secret)

    cp .env.example .env

    sed_replace "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=unused_external_db/" .env
    sed_replace "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" .env
    sed_replace "s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=${TOTP_KEY}/" .env

    # 追加连接信息
    cat >> .env << EOF

# =============================================================================
# 外部数据库/Redis 连接 (安装脚本生成)
# =============================================================================
DATABASE_HOST=${DOCKER_DB_HOST}
DATABASE_PORT=${DB_PORT}
DATABASE_USER=${DB_USER}
DATABASE_PASSWORD=${DB_PASS}
DATABASE_DBNAME=${DB_NAME}
REDIS_HOST=${DOCKER_REDIS_HOST}
REDIS_PORT=${REDIS_PORT}
REDIS_PASSWORD=${REDIS_PASS}
EOF

    chmod 600 .env
    mkdir -p data
    print_success "配置文件生成完成"

    # =========================================================================
    # 数据库迁移
    # =========================================================================
    print_info "初始化数据库..."
    if command_exists psql; then
        PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -q << 'SQLEOF'
CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL,
    order_no VARCHAR(64) NOT NULL UNIQUE, trade_no VARCHAR(64),
    amount_usd DECIMAL(20,8) NOT NULL, amount_rmb DECIMAL(20,2) NOT NULL,
    exchange_rate DECIMAL(10,4) NOT NULL, status VARCHAR(32) NOT NULL DEFAULT 'pending',
    pay_type VARCHAR(32), paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payment_orders_user_id ON payment_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_status ON payment_orders(status);
CREATE TABLE IF NOT EXISTS chat_conversations (
    id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL DEFAULT 'New Chat', model VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_id ON chat_conversations(user_id);
CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL, content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);
SQLEOF
        print_success "数据库初始化完成"
    else
        print_warning "未检测到 psql，跳过数据库迁移"
        print_warning "请手动执行 backend/migrations/ 下的 SQL 文件"
    fi

    # =========================================================================
    # 构建并启动
    # =========================================================================
    print_info "构建并启动服务 (首次构建约需3-5分钟)..."
    $COMPOSE_CMD -f docker-compose.dev.yml up --build -d

    print_info "等待服务就绪..."
    for i in $(seq 1 30); do
        curl -sf http://localhost:8080/health >/dev/null 2>&1 && break
        sleep 2
    done

    if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
        print_success "服务启动成功"
    else
        print_warning "服务启动中，请查看日志: $COMPOSE_CMD -f docker-compose.dev.yml logs -f sub2api"
    fi

    # =========================================================================
    # 完成
    # =========================================================================
    SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo 'localhost')
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  安装完成!${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "  访问: http://${SERVER_IP}:8080"
    echo ""
    echo "  管理员密码 (首次启动自动生成):"
    echo "  cd $(pwd) && $COMPOSE_CMD -f docker-compose.dev.yml logs sub2api | grep -i password"
    echo ""
    echo "  常用命令 (在 $(pwd) 目录下):"
    echo "  $COMPOSE_CMD -f docker-compose.dev.yml logs -f sub2api  # 日志"
    echo "  $COMPOSE_CMD -f docker-compose.dev.yml restart sub2api  # 重启"
    echo "  $COMPOSE_CMD -f docker-compose.dev.yml down             # 停止"
    echo ""
    echo "  更新:"
    echo "  cd $(dirname $(pwd)) && git pull && cd deploy"
    echo "  $COMPOSE_CMD -f docker-compose.dev.yml up --build -d"
    echo ""
    print_warning "密钥已保存到 $(pwd)/.env，请妥善保管!"
    echo ""
}

main "$@"
