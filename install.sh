#!/bin/bash
#
# Token Site 安装脚本
# 基于 sub2api 安装脚本修改，指向 vilavia/token-site
#
# 用法: curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash
#
# 命令:
#   install.sh                  # 安装最新版
#   install.sh upgrade          # 升级到最新版
#   install.sh uninstall        # 卸载
#   install.sh list-versions    # 列出可用版本
#

set -e

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'

GITHUB_REPO="vilavia/token-site"
INSTALL_DIR="/opt/sub2api"
SERVICE_NAME="sub2api"
SERVICE_USER="sub2api"
SERVER_HOST="0.0.0.0"
SERVER_PORT="8080"

info()  { echo -e "${BLUE}[INFO]${NC} $1"; }
ok()    { echo -e "${GREEN}[OK]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
fail()  { echo -e "${RED}[ERROR]${NC} $1"; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

# =============================================================================
# 检测平台
# =============================================================================
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) fail "不支持的架构: $ARCH"; exit 1 ;;
    esac
    case "$OS" in
        linux) ;;
        darwin) ;;
        *) fail "不支持的系统: $OS"; exit 1 ;;
    esac
    info "平台: ${OS}/${ARCH}"
}

# =============================================================================
# 检查依赖
# =============================================================================
check_deps() {
    local missing=()
    command_exists curl || missing+=("curl")
    command_exists tar  || missing+=("tar")
    if [ ${#missing[@]} -gt 0 ]; then
        fail "缺少依赖: ${missing[*]}"
        exit 1
    fi
}

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        fail "请使用 root 权限运行 (sudo)"
        exit 1
    fi
}

# =============================================================================
# GitHub API
# =============================================================================
get_latest_version() {
    info "获取最新版本..."
    LATEST_VERSION=$(curl -sSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$LATEST_VERSION" ]; then
        fail "获取版本失败"; exit 1
    fi
    ok "最新版本: $LATEST_VERSION"
}

list_versions() {
    info "可用版本:"
    curl -sSL "https://api.github.com/repos/${GITHUB_REPO}/releases" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/  \1/'
}

# =============================================================================
# 下载并安装
# =============================================================================
download_and_extract() {
    local version="${1:-$LATEST_VERSION}"
    local filename="sub2api_${OS}_${ARCH}.tar.gz"
    local url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${filename}"
    local checksum_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/checksums.txt"

    local temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT

    info "下载 ${filename}..."
    curl -sSL "$url" -o "$temp_dir/$filename"
    ok "下载完成"

    # 校验
    info "校验文件..."
    if curl -sSL "$checksum_url" -o "$temp_dir/checksums.txt" 2>/dev/null; then
        cd "$temp_dir"
        if command_exists sha256sum; then
            sha256sum -c checksums.txt --ignore-missing 2>/dev/null && ok "校验通过" || warn "校验跳过"
        fi
        cd - >/dev/null
    fi

    # 解压
    info "解压..."
    tar xzf "$temp_dir/$filename" -C "$temp_dir"

    # 安装二进制
    mkdir -p "$INSTALL_DIR"
    cp "$temp_dir/sub2api/sub2api" "$INSTALL_DIR/sub2api"
    chmod +x "$INSTALL_DIR/sub2api"

    # 复制 deploy 文件（如有）
    if [ -d "$temp_dir/sub2api/deploy" ]; then
        cp -r "$temp_dir/sub2api/deploy/"* "$INSTALL_DIR/" 2>/dev/null || true
    fi

    ok "已安装到 $INSTALL_DIR/sub2api"
}

# =============================================================================
# 系统用户和目录
# =============================================================================
create_user() {
    if id "$SERVICE_USER" &>/dev/null; then
        info "用户 $SERVICE_USER 已存在"
    else
        info "创建系统用户 $SERVICE_USER..."
        useradd -r -s /bin/sh -d "$INSTALL_DIR" "$SERVICE_USER"
        ok "用户已创建"
    fi
}

setup_dirs() {
    mkdir -p "$INSTALL_DIR" "$INSTALL_DIR/data" "/etc/sub2api"
    chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR" "/etc/sub2api"
    ok "目录配置完成"
}

# =============================================================================
# systemd 服务
# =============================================================================
install_service() {
    info "安装 systemd 服务..."
    cat > /etc/systemd/system/sub2api.service << EOF
[Unit]
Description=Token Site (sub2api)
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=sub2api
Group=sub2api
WorkingDirectory=/opt/sub2api
ExecStart=/opt/sub2api/sub2api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sub2api
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/sub2api /etc/sub2api
Environment=GIN_MODE=release
Environment=SERVER_HOST=${SERVER_HOST}
Environment=SERVER_PORT=${SERVER_PORT}

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    ok "服务已安装"
}

# =============================================================================
# 升级
# =============================================================================
upgrade() {
    if [ ! -f "$INSTALL_DIR/sub2api" ]; then
        fail "未检测到安装，请先安装"; exit 1
    fi
    info "停止服务..."
    systemctl stop sub2api 2>/dev/null || true

    get_latest_version

    # 备份
    cp "$INSTALL_DIR/sub2api" "$INSTALL_DIR/sub2api.backup"
    ok "备份已创建"

    download_and_extract "$LATEST_VERSION"
    chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"

    info "启动服务..."
    systemctl start sub2api
    ok "升级完成 → $LATEST_VERSION"
}

# =============================================================================
# 卸载
# =============================================================================
uninstall() {
    warn "将卸载 Token Site"
    read -p "确认卸载? (y/N): " -r < /dev/tty; echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 0

    systemctl stop sub2api 2>/dev/null || true
    systemctl disable sub2api 2>/dev/null || true
    rm -f /etc/systemd/system/sub2api.service
    systemctl daemon-reload

    read -p "删除数据目录 $INSTALL_DIR? (y/N): " -r < /dev/tty; echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$INSTALL_DIR"
        ok "数据已删除"
    else
        rm -f "$INSTALL_DIR/sub2api" "$INSTALL_DIR/sub2api.backup"
        ok "二进制已删除，数据保留在 $INSTALL_DIR"
    fi

    ok "卸载完成"
}

# =============================================================================
# 配置服务器
# =============================================================================
configure_server() {
    # curl | bash 时 stdin 是管道，需要从 /dev/tty 读取用户输入
    if [ -t 0 ] || [ -e /dev/tty ]; then
        echo ""
        read -p "监听地址 [$SERVER_HOST]: " -r input < /dev/tty
        SERVER_HOST="${input:-$SERVER_HOST}"
        read -p "监听端口 [$SERVER_PORT]: " -r input < /dev/tty
        SERVER_PORT="${input:-$SERVER_PORT}"
    fi
}

# =============================================================================
# 主流程
# =============================================================================
main() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  Token Site 安装脚本${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    case "${1:-}" in
        upgrade|update)
            check_root; detect_platform; check_deps
            get_latest_version; upgrade; exit 0 ;;
        uninstall|remove)
            check_root; uninstall; exit 0 ;;
        list-versions|versions)
            list_versions; exit 0 ;;
        --help|-h)
            echo "用法: $0 [命令]"
            echo ""
            echo "  (无参数)       安装最新版"
            echo "  upgrade        升级到最新版"
            echo "  uninstall      卸载"
            echo "  list-versions  列出可用版本"
            exit 0 ;;
    esac

    # 默认: 全新安装 / 重新安装
    check_root
    detect_platform
    check_deps

    # 如果已有旧安装在运行，先停掉
    if systemctl is-active sub2api >/dev/null 2>&1; then
        info "检测到服务正在运行，先停止..."
        systemctl stop sub2api
    fi

    configure_server
    get_latest_version
    download_and_extract
    create_user
    setup_dirs
    install_service

    # 获取 IP
    PUBLIC_IP=$(curl -s --max-time 5 https://ipinfo.io/ip 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "YOUR_IP")

    # 启动
    systemctl start sub2api
    systemctl enable sub2api 2>/dev/null || true

    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  安装完成!${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "  确保 PostgreSQL 和 Redis 已运行，然后打开浏览器:"
    echo ""
    echo "    http://${PUBLIC_IP}:${SERVER_PORT}"
    echo ""
    echo "  首次访问进入引导页面，配置数据库和管理员账号"
    echo ""
    echo "  常用命令:"
    echo "    systemctl status sub2api    # 状态"
    echo "    journalctl -u sub2api -f    # 日志"
    echo "    systemctl restart sub2api   # 重启"
    echo "    systemctl stop sub2api      # 停止"
    echo ""
    echo "  升级: curl -sSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash -s upgrade"
    echo ""
}

# 当通过 curl | bash -s upgrade 调用时，参数在 $0 而非 $1
# 检测并修正
if [ "${0:-}" = "upgrade" ] || [ "${0:-}" = "uninstall" ] || [ "${0:-}" = "list-versions" ]; then
    main "$0" "$@"
else
    main "$@"
fi
