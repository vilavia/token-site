#!/bin/bash
# =============================================================================
# Token Site 一键安装脚本
# =============================================================================
# 使用外部 PostgreSQL + Redis（不在 Docker 里跑数据库）
#
# 用法:
#   curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash
#
# 或者手动:
#   bash install.sh
# =============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

GITHUB_RAW_URL="https://raw.githubusercontent.com/vilavia/token-site/main/deploy"
INSTALL_DIR="token-site"

print_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[OK]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

generate_secret() { openssl rand -hex 32; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

download_file() {
    local url="$1" dest="$2"
    if command_exists curl; then
        curl -sSL "$url" -o "$dest"
    elif command_exists wget; then
        wget -q "$url" -O "$dest"
    else
        print_error "需要安装 curl 或 wget"
        exit 1
    fi
}

sed_replace() {
    local pattern="$1" file="$2"
    if sed --version >/dev/null 2>&1; then
        sed -i "$pattern" "$file"
    else
        sed -i '' "$pattern" "$file"
    fi
}

# =============================================================================
# 检查系统依赖
# =============================================================================
check_deps() {
    local missing=()
    command_exists docker || missing+=("docker")
    command_exists openssl || missing+=("openssl")

    if [ ${#missing[@]} -gt 0 ]; then
        print_error "缺少依赖: ${missing[*]}"
        echo ""
        echo "安装 Docker:"
        echo "  curl -fsSL https://get.docker.com | bash"
        exit 1
    fi

    # 检查 docker compose
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    elif command_exists docker-compose; then
        COMPOSE_CMD="docker-compose"
    else
        print_error "需要 Docker Compose (docker compose 或 docker-compose)"
        exit 1
    fi
    print_success "依赖检查通过 (Docker + Docker Compose)"
}

# =============================================================================
# 安装 PostgreSQL (如果服务器上没有)
# =============================================================================
install_postgres() {
    echo ""
    echo -e "${CYAN}数据库配置${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    if command_exists psql; then
        print_info "检测到本机已安装 PostgreSQL"
        read -p "使用本机 PostgreSQL? (Y/n): " -r
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            USE_LOCAL_PG=true
            DB_HOST="host.docker.internal"
            read -p "PostgreSQL 端口 [5432]: " -r DB_PORT
            DB_PORT=${DB_PORT:-5432}
            read -p "数据库用户名 [sub2api]: " -r DB_USER
            DB_USER=${DB_USER:-sub2api}
            read -p "数据库名 [sub2api]: " -r DB_NAME
            DB_NAME=${DB_NAME:-sub2api}
            read -sp "数据库密码: " DB_PASS
            echo ""
            return
        fi
    fi

    echo "本机未安装 PostgreSQL，可选择:"
    echo "  1) 自动安装 PostgreSQL 到本机 (推荐)"
    echo "  2) 使用远程数据库 (手动输入连接信息)"
    echo ""
    read -p "选择 [1]: " -r PG_CHOICE
    PG_CHOICE=${PG_CHOICE:-1}

    if [ "$PG_CHOICE" = "1" ]; then
        print_info "安装 PostgreSQL..."
        if command_exists apt-get; then
            sudo apt-get update -qq && sudo apt-get install -y -qq postgresql postgresql-client >/dev/null 2>&1
        elif command_exists yum; then
            sudo yum install -y -q postgresql-server postgresql >/dev/null 2>&1
            sudo postgresql-setup --initdb 2>/dev/null || true
            sudo systemctl start postgresql
            sudo systemctl enable postgresql
        elif command_exists dnf; then
            sudo dnf install -y -q postgresql-server postgresql >/dev/null 2>&1
            sudo postgresql-setup --initdb 2>/dev/null || true
            sudo systemctl start postgresql
            sudo systemctl enable postgresql
        else
            print_error "无法自动安装 PostgreSQL，请手动安装后重试"
            exit 1
        fi

        # 创建数据库和用户
        DB_HOST="host.docker.internal"
        DB_PORT=5432
        DB_USER="sub2api"
        DB_NAME="sub2api"
        DB_PASS=$(generate_secret | head -c 24)

        sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';" 2>/dev/null || true
        sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null || true
        sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>/dev/null || true

        # 允许密码登录
        PG_HBA=$(sudo -u postgres psql -t -c "SHOW hba_file;" | xargs)
        if [ -n "$PG_HBA" ]; then
            if ! grep -q "sub2api" "$PG_HBA" 2>/dev/null; then
                echo "host    sub2api    sub2api    172.16.0.0/12    md5" | sudo tee -a "$PG_HBA" >/dev/null
                echo "host    sub2api    sub2api    127.0.0.1/32     md5" | sudo tee -a "$PG_HBA" >/dev/null
                sudo systemctl reload postgresql 2>/dev/null || sudo pg_ctlcluster $(pg_lsclusters -h | head -1 | awk '{print $1, $2}') reload 2>/dev/null || true
            fi
        fi

        # 配置 listen_addresses
        PG_CONF=$(sudo -u postgres psql -t -c "SHOW config_file;" | xargs)
        if [ -n "$PG_CONF" ]; then
            if ! grep -q "^listen_addresses.*\*" "$PG_CONF" 2>/dev/null; then
                sudo sed -i "s/^#listen_addresses.*/listen_addresses = '*'/" "$PG_CONF" 2>/dev/null || true
                sudo systemctl restart postgresql 2>/dev/null || true
            fi
        fi

        print_success "PostgreSQL 安装完成"
        USE_LOCAL_PG=true
    else
        DB_HOST=""
        read -p "数据库地址: " -r DB_HOST
        read -p "数据库端口 [5432]: " -r DB_PORT
        DB_PORT=${DB_PORT:-5432}
        read -p "数据库用户名 [sub2api]: " -r DB_USER
        DB_USER=${DB_USER:-sub2api}
        read -p "数据库名 [sub2api]: " -r DB_NAME
        DB_NAME=${DB_NAME:-sub2api}
        read -sp "数据库密码: " DB_PASS
        echo ""
    fi
}

# =============================================================================
# 安装 Redis (如果服务器上没有)
# =============================================================================
install_redis() {
    echo ""
    echo -e "${CYAN}Redis 配置${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    if command_exists redis-cli; then
        print_info "检测到本机已安装 Redis"
        REDIS_HOST="host.docker.internal"
        REDIS_PORT=6379
        REDIS_PASS=""
        read -p "Redis 端口 [6379]: " -r REDIS_PORT
        REDIS_PORT=${REDIS_PORT:-6379}
        read -p "Redis 密码 (无密码直接回车): " -r REDIS_PASS
        return
    fi

    echo "本机未安装 Redis，可选择:"
    echo "  1) 自动安装 Redis 到本机 (推荐)"
    echo "  2) 使用远程 Redis"
    echo ""
    read -p "选择 [1]: " -r REDIS_CHOICE
    REDIS_CHOICE=${REDIS_CHOICE:-1}

    if [ "$REDIS_CHOICE" = "1" ]; then
        print_info "安装 Redis..."
        if command_exists apt-get; then
            sudo apt-get install -y -qq redis-server >/dev/null 2>&1
            sudo systemctl start redis-server
            sudo systemctl enable redis-server
        elif command_exists yum; then
            sudo yum install -y -q redis >/dev/null 2>&1
            sudo systemctl start redis
            sudo systemctl enable redis
        elif command_exists dnf; then
            sudo dnf install -y -q redis >/dev/null 2>&1
            sudo systemctl start redis
            sudo systemctl enable redis
        else
            print_error "无法自动安装 Redis，请手动安装后重试"
            exit 1
        fi
        REDIS_HOST="host.docker.internal"
        REDIS_PORT=6379
        REDIS_PASS=""
        print_success "Redis 安装完成"
    else
        read -p "Redis 地址: " -r REDIS_HOST
        read -p "Redis 端口 [6379]: " -r REDIS_PORT
        REDIS_PORT=${REDIS_PORT:-6379}
        read -p "Redis 密码 (无密码直接回车): " -r REDIS_PASS
    fi
}

# =============================================================================
# 主安装流程
# =============================================================================
main() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  Token Site 安装向导${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    check_deps

    # 如果已存在，提示
    if [ -d "$INSTALL_DIR" ] && [ -f "$INSTALL_DIR/docker-compose.yml" ]; then
        print_warning "检测到已有安装: $INSTALL_DIR/"
        read -p "重新安装? 会覆盖配置文件 (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "取消安装"
            exit 0
        fi
    fi

    # 数据库和Redis配置
    install_postgres
    install_redis

    # 创建安装目录
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"

    # 下载文件
    print_info "下载配置文件..."
    download_file "${GITHUB_RAW_URL}/docker-compose.standalone.yml" "docker-compose.yml"
    download_file "${GITHUB_RAW_URL}/.env.example" ".env.example"
    print_success "配置文件下载完成"

    # 生成密钥
    print_info "生成安全密钥..."
    JWT_SECRET=$(generate_secret)
    TOTP_KEY=$(generate_secret)

    # 生成 .env
    cp .env.example .env

    # 写入配置
    sed_replace "s/^POSTGRES_PASSWORD=.*/# (使用外部数据库，此项无效)/" .env
    sed_replace "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" .env
    sed_replace "s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=${TOTP_KEY}/" .env

    # 在 .env 末尾追加数据库和Redis连接信息
    cat >> .env << ENVEOF

# =============================================================================
# 外部数据库连接 (由安装脚本自动生成)
# =============================================================================
DATABASE_HOST=${DB_HOST}
DATABASE_PORT=${DB_PORT}
DATABASE_USER=${DB_USER}
DATABASE_PASSWORD=${DB_PASS}
DATABASE_DBNAME=${DB_NAME}
DATABASE_SSLMODE=disable

REDIS_HOST=${REDIS_HOST}
REDIS_PORT=${REDIS_PORT}
REDIS_PASSWORD=${REDIS_PASS}
ENVEOF

    # 修改 docker-compose.yml 中的镜像名
    # standalone 默认用 weishaw/sub2api:latest，改为从源码构建
    # 但用户可能没有源码，所以保持用官方镜像，通过 env 连接外部数据库

    # 创建数据目录
    mkdir -p data
    chmod 600 .env

    # 运行数据库迁移
    echo ""
    print_info "初始化数据库表..."
    PGPASSWORD="$DB_PASS" psql -h "${DB_HOST/host.docker.internal/127.0.0.1}" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
    CREATE TABLE IF NOT EXISTS payment_orders (
        id BIGSERIAL PRIMARY KEY,
        user_id BIGINT NOT NULL,
        order_no VARCHAR(64) NOT NULL UNIQUE,
        trade_no VARCHAR(64),
        amount_usd DECIMAL(20,8) NOT NULL,
        amount_rmb DECIMAL(20,2) NOT NULL,
        exchange_rate DECIMAL(10,4) NOT NULL,
        status VARCHAR(32) NOT NULL DEFAULT 'pending',
        pay_type VARCHAR(32),
        paid_at TIMESTAMPTZ,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    CREATE INDEX IF NOT EXISTS idx_payment_orders_user_id ON payment_orders(user_id);
    CREATE INDEX IF NOT EXISTS idx_payment_orders_status ON payment_orders(status);

    CREATE TABLE IF NOT EXISTS chat_conversations (
        id BIGSERIAL PRIMARY KEY,
        user_id BIGINT NOT NULL,
        title VARCHAR(200) NOT NULL DEFAULT 'New Chat',
        model VARCHAR(100) NOT NULL DEFAULT '',
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_id ON chat_conversations(user_id);

    CREATE TABLE IF NOT EXISTS chat_messages (
        id BIGSERIAL PRIMARY KEY,
        conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
        role VARCHAR(20) NOT NULL,
        content TEXT NOT NULL DEFAULT '',
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);
    " 2>/dev/null && print_success "数据库表初始化完成" || print_warning "数据库迁移跳过（可能已存在或连接失败，稍后可手动执行）"

    # 启动服务
    echo ""
    print_info "启动服务..."
    $COMPOSE_CMD up -d

    # 等待健康检查
    print_info "等待服务启动..."
    for i in $(seq 1 30); do
        if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
            break
        fi
        sleep 2
    done

    if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
        print_success "服务启动成功"
    else
        print_warning "服务可能还在启动中，请稍等后检查"
        echo "  查看日志: cd $INSTALL_DIR && $COMPOSE_CMD logs -f"
    fi

    # 完成
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  安装完成!${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "  访问地址:  http://$(hostname -I 2>/dev/null | awk '{print $1}' || echo 'localhost'):8080"
    echo ""
    echo "  生成的密钥 (已保存到 .env):"
    echo "  JWT_SECRET:          ${JWT_SECRET}"
    echo "  TOTP_ENCRYPTION_KEY: ${TOTP_KEY}"
    if [ -n "$DB_PASS" ] && [ "$USE_LOCAL_PG" = "true" ]; then
        echo "  DATABASE_PASSWORD:   ${DB_PASS}"
    fi
    echo ""
    echo "  管理员账号在首次启动时自动创建，查看日志获取密码:"
    echo "  cd $INSTALL_DIR && $COMPOSE_CMD logs sub2api | grep -i password"
    echo ""
    echo "  常用命令:"
    echo "  cd $INSTALL_DIR"
    echo "  $COMPOSE_CMD logs -f sub2api    # 查看日志"
    echo "  $COMPOSE_CMD restart sub2api    # 重启"
    echo "  $COMPOSE_CMD down               # 停止"
    echo ""
    print_warning "请妥善保管 .env 文件中的密钥!"
    echo ""
}

main "$@"
