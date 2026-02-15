#!/usr/bin/env bash
# ============================================================================
# CBoard v2 一键安装 & 管理脚本（无宝塔 / 纯净 Linux 环境）
# 支持: Ubuntu 20.04+, Debian 11+, CentOS 7+, AlmaLinux 8+, Rocky Linux 8+
# 用法: bash install.sh   （必须用 bash 运行，不要用 sh install.sh）
# 全自动安装: CBOARD_UNATTENDED=1 CBOARD_ADMIN_EMAIL=admin@example.com CBOARD_ADMIN_PASSWORD=你的密码 bash install.sh
# ============================================================================
# 必须使用 bash，否则 read -rp 等会失败导致脚本静默退出
[ -n "$BASH_VERSION" ] || exec /usr/bin/env bash "$0" "$@"
set -e

# ---- 版本 ----
SCRIPT_VERSION="2.0.0"

# ---- 颜色 ----
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BLUE='\033[0;34m'; NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()   { echo -e "${RED}[ERROR]${NC} $*"; }
fatal() { echo -e "${RED}[FATAL]${NC} $*"; exit 1; }

# ---- 配置变量 ----
PROJECT_PATH="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="/opt/cboard"
SERVICE_NAME="cboard"
CBOARD_PORT=9000
BACKEND_PORT=9000
DOMAIN=""
ENABLE_SSL="n"
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
LOCK_FILE="/tmp/cboard_install.lock"

# ---- 并发运行检查 ----
check_concurrent() {
    if [ -f "$LOCK_FILE" ]; then
        local lock_pid
        lock_pid=$(cat "$LOCK_FILE" 2>/dev/null)
        if [ -n "$lock_pid" ] && kill -0 "$lock_pid" 2>/dev/null; then
            fatal "另一个安装进程正在运行 (PID: $lock_pid)，请等待其完成或删除 $LOCK_FILE"
        else
            warn "发现残留锁文件，已清理"
            rm -f "$LOCK_FILE"
        fi
    fi
    echo $$ > "$LOCK_FILE"
    trap 'rm -f "$LOCK_FILE"' EXIT
}

# ---- Root 检查 ----
check_root() {
    if [[ $EUID -ne 0 ]]; then
        fatal "请使用 root 用户运行此脚本"
    fi
}

# ---- 系统检测 ----
OS=""
OS_VERSION=""
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    elif [ -f /etc/redhat-release ]; then
        OS="centos"
    else
        fatal "不支持的操作系统"
    fi
    info "检测到系统: $OS $OS_VERSION"
}

# ---- 磁盘空间检查 ----
check_disk_space() {
    local available
    available=$(df -BM "$PROJECT_PATH" 2>/dev/null | awk 'NR==2{print $4}' | tr -d 'M')
    if [ -n "$available" ] && [ "$available" -lt 1024 ]; then
        err "磁盘可用空间不足: ${available}MB (需要至少 1GB)"
        return 1
    fi
    ok "磁盘空间充足: ${available}MB 可用"
    return 0
}

# ---- 安全释放端口 ----
safe_release_port() {
    local port=$1
    local pid
    pid=$(lsof -ti ":$port" 2>/dev/null || true)
    if [ -n "$pid" ]; then
        # 不杀 nginx 和 sshd
        local pname
        pname=$(ps -p "$pid" -o comm= 2>/dev/null || true)
        if [[ "$pname" == "nginx" ]] || [[ "$pname" == "sshd" ]]; then
            return 0
        fi
        warn "端口 $port 被 PID $pid ($pname) 占用，正在释放..."
        kill "$pid" 2>/dev/null || true
        sleep 2
        if kill -0 "$pid" 2>/dev/null; then
            kill -9 "$pid" 2>/dev/null || true
        fi
    fi
}

# ---- 安装系统工具 ----
install_system_tools() {
    info "安装系统依赖..."
    case $OS in
        ubuntu|debian)
            apt-get update -qq
            apt-get install -y -qq curl wget git unzip lsof net-tools certbot >/dev/null 2>&1
            ;;
        centos|rhel|almalinux|rocky)
            yum install -y epel-release >/dev/null 2>&1 || true
            yum install -y curl wget git unzip lsof net-tools certbot >/dev/null 2>&1
            ;;
        *)
            fatal "不支持的系统: $OS"
            ;;
    esac
    ok "系统依赖安装完成"
}

# ---- 安装 Nginx ----
install_nginx() {
    if command -v nginx &>/dev/null; then
        ok "Nginx 已安装: $(nginx -v 2>&1 | awk -F/ '{print $2}')"
        return
    fi
    info "安装 Nginx..."
    case $OS in
        ubuntu|debian)
            apt-get install -y -qq nginx python3-certbot-nginx >/dev/null 2>&1
            ;;
        centos|rhel|almalinux|rocky)
            yum install -y nginx python3-certbot-nginx >/dev/null 2>&1
            ;;
    esac
    systemctl enable nginx >/dev/null 2>&1
    systemctl start nginx >/dev/null 2>&1
    ok "Nginx 安装完成"
}

# ---- 安装 Go ----
install_go() {
    if command -v go &>/dev/null; then
        local go_ver
        go_ver=$(go version | awk '{print $3}')
        ok "Go 已安装: $go_ver"
        return
    fi
    info "安装 Go 1.24..."
    local arch
    arch=$(uname -m)
    case $arch in
        x86_64)  GO_ARCH="amd64" ;;
        aarch64) GO_ARCH="arm64" ;;
        *)       fatal "不支持的架构: $arch" ;;
    esac
    wget -q "https://go.dev/dl/go1.24.0.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz || fatal "Go 下载失败"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    fi
    ok "Go $(go version | awk '{print $3}') 安装完成"
}

# ---- 安装 Node.js ----
install_node() {
    if command -v node &>/dev/null; then
        ok "Node.js 已安装: $(node -v)"
        return
    fi
    info "安装 Node.js 20.x..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - >/dev/null 2>&1 || {
        curl -fsSL https://rpm.nodesource.com/setup_20.x | bash - >/dev/null 2>&1
    }
    case $OS in
        ubuntu|debian) apt-get install -y -qq nodejs >/dev/null 2>&1 ;;
        *)             yum install -y nodejs >/dev/null 2>&1 ;;
    esac
    ok "Node.js $(node -v) 安装完成"
}

# ---- 域名格式验证 ----
validate_domain() {
    local domain="$1"
    if [[ "$domain" =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$ ]]; then
        return 0
    fi
    return 1
}

# ---- 邮箱格式验证 ----
validate_email() {
    local email="$1"
    if [[ "$email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        return 0
    fi
    return 1
}

# ---- 收集管理员信息 ----
get_admin_info() {
    echo ""
    echo -e "${CYAN}========== 管理员信息配置 ==========${NC}"
    echo ""

    # 管理员邮箱
    while true; do
        read -rp "管理员邮箱: " ADMIN_EMAIL
        if [ -z "$ADMIN_EMAIL" ]; then
            err "邮箱不能为空"
        elif validate_email "$ADMIN_EMAIL"; then
            break
        else
            err "邮箱格式不正确，请重新输入"
        fi
    done

    # 管理员密码
    while true; do
        read -rsp "管理员密码 (至少8位): " ADMIN_PASSWORD
        echo
        if [ -z "$ADMIN_PASSWORD" ]; then
            err "密码不能为空"
        elif [ ${#ADMIN_PASSWORD} -lt 8 ]; then
            err "密码长度至少8位"
        else
            read -rsp "确认密码: " ADMIN_PASSWORD_CONFIRM
            echo
            if [ "$ADMIN_PASSWORD" != "$ADMIN_PASSWORD_CONFIRM" ]; then
                err "两次密码不一致，请重新输入"
            else
                break
            fi
        fi
    done

    ok "管理员信息已收集"
}

# ---- 交互式配置 ----
interactive_config() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║       CBoard v2 安装配置 (无宝塔版)     ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
    echo ""

    # 安装目录
    read -rp "安装目录 [$INSTALL_DIR]: " input
    INSTALL_DIR=${input:-$INSTALL_DIR}

    # 后端端口固定为 9000（仅内部使用，Nginx 转发 80/443 到该端口，用户直接访问域名即可，无需加端口）
    CBOARD_PORT=9000
    BACKEND_PORT=$CBOARD_PORT

    # 域名
    while true; do
        read -rp "绑定域名 (留空则使用 IP 访问): " DOMAIN
        if [ -z "$DOMAIN" ]; then
            break
        fi
        # 移除协议前缀
        DOMAIN=$(echo "$DOMAIN" | sed 's|^https\?://||' | sed 's|/$||')
        if validate_domain "$DOMAIN"; then
            break
        else
            err "域名格式不正确，请重新输入 (例如: example.com)"
        fi
    done

    # SSL
    if [ -n "$DOMAIN" ]; then
        read -rp "是否自动申请 SSL 证书? (y/n) [y]: " ENABLE_SSL
        ENABLE_SSL=${ENABLE_SSL:-y}
    fi

    # 管理员信息
    get_admin_info

    # 确认
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    info "安装目录: $INSTALL_DIR"
    info "绑定域名: ${DOMAIN:-无(IP 访问)}（用户直接打开域名即可，无需加端口）"
    info "SSL 证书: ${ENABLE_SSL}"
    info "管理员邮箱: $ADMIN_EMAIL"
    info "管理员密码: ${ADMIN_PASSWORD:0:2}******"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    read -rp "确认以上配置? (y/n) [y]: " confirm
    if [[ "${confirm:-y}" != "y" ]]; then
        fatal "安装已取消"
    fi
}

# ---- 生成 .env 配置文件 ----
create_env_file() {
    local target_dir="${1:-$INSTALL_DIR}"
    cd "$target_dir" || return 1

    if [ -f .env ]; then
        # 旧 .env 可能缺少 CORS_ORIGINS，补上以免生产环境启动失败
        if ! grep -q '^CORS_ORIGINS=' .env 2>/dev/null; then
            local base_url
            base_url=$(grep '^BASE_URL=' .env 2>/dev/null | cut -d'=' -f2- | tr -d '"' | xargs)
            if [ -n "$base_url" ]; then echo "CORS_ORIGINS=$base_url" >> .env; ok "已补充 CORS_ORIGINS 到 .env"; fi
        fi
        warn ".env 已存在，跳过生成"
        return 0
    fi

    local SECRET_KEY
    SECRET_KEY=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n')

    local BASE_URL
    if [ -n "$DOMAIN" ]; then
        if [[ "$ENABLE_SSL" =~ ^[Yy]$ ]]; then
            BASE_URL="https://$DOMAIN"
        else
            BASE_URL="http://$DOMAIN"
        fi
    else
        BASE_URL="http://$(curl -s4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}'):$CBOARD_PORT"
    fi

    cat > .env <<EOF
# CBoard v2 配置文件
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

# ---- 基础配置 ----
DEBUG=false
PORT=$CBOARD_PORT
HOST=127.0.0.1
SECRET_KEY=$SECRET_KEY
BASE_URL=$BASE_URL
CORS_ORIGINS=$BASE_URL

# ---- 数据库 ----
DATABASE_URL=sqlite:///./cboard.db

# ---- 域名 ----
DOMAIN=${DOMAIN}

# ---- SSL ----
SSL_ENABLED=$([ "$ENABLE_SSL" = "y" ] && echo "true" || echo "false")

# ---- 管理员 ----
ADMIN_EMAIL=$ADMIN_EMAIL

# ---- 订阅 ----
SUBSCRIPTION_URL_PREFIX=$BASE_URL/sub

# ---- 支付回调 ----
ALIPAY_NOTIFY_URL=$BASE_URL/api/v1/payment/notify/alipay
ALIPAY_RETURN_URL=$BASE_URL/payment/return
EOF

    ok ".env 配置文件已生成"
}

# ---- 部署项目 ----
deploy_project() {
    info "部署 CBoard v2 到 $INSTALL_DIR ..."
    mkdir -p "$INSTALL_DIR"

    if [ -f "$PROJECT_PATH/go.mod" ] && grep -q "cboard/v2" "$PROJECT_PATH/go.mod" 2>/dev/null; then
        if [ "$PROJECT_PATH" != "$INSTALL_DIR" ]; then
            cp -r "$PROJECT_PATH"/* "$INSTALL_DIR/" 2>/dev/null || true
            cp "$PROJECT_PATH"/.env.example "$INSTALL_DIR/" 2>/dev/null || true
            cp "$PROJECT_PATH"/install.sh "$INSTALL_DIR/" 2>/dev/null || true
        fi
    else
        fatal "请将此脚本放在 CBoard v2 源码根目录下运行"
    fi

    create_env_file "$INSTALL_DIR"
}

# ---- 构建后端 ----
build_backend() {
    info "构建后端..."
    cd "$INSTALL_DIR"
    export PATH=$PATH:/usr/local/go/bin

    # 最多重试2次
    local success=false
    for i in 1 2; do
        if go build -o cboard cmd/server/main.go 2>&1; then
            success=true
            break
        else
            warn "后端构建失败 (尝试 $i/2)"
            if [ $i -lt 2 ]; then
                info "清理缓存后重试..."
                go clean -cache 2>/dev/null || true
            fi
        fi
    done

    if [ "$success" = false ]; then
        fatal "后端构建失败，请检查 Go 环境和源码"
    fi

    chmod +x cboard
    ok "后端构建完成"
}

# ---- 构建前端 ----
build_frontend() {
    info "构建前端..."
    cd "$INSTALL_DIR/frontend"

    # 安装依赖
    local npm_success=false
    for i in 1 2 3; do
        if npm install --silent 2>/dev/null; then
            npm_success=true
            break
        else
            warn "前端依赖安装失败 (尝试 $i/3)"
            if [ $i -lt 3 ]; then
                rm -rf node_modules package-lock.json
                sleep 3
            fi
        fi
    done
    if [ "$npm_success" = false ]; then
        fatal "前端依赖安装失败"
    fi

    # 构建
    export NODE_OPTIONS="--max-old-space-size=4096"
    if ! npx vite build 2>&1; then
        fatal "前端构建失败"
    fi

    # 验证
    if [ ! -f "dist/index.html" ]; then
        fatal "前端构建失败: dist/index.html 不存在"
    fi

    ok "前端构建完成"
}

# ---- 创建 systemd 服务 ----
create_service() {
    info "创建 systemd 服务..."
    cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=CBoard v2 Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/cboard
Restart=always
RestartSec=5
LimitNOFILE=65536
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable ${SERVICE_NAME} >/dev/null 2>&1
    ok "systemd 服务已创建"
}

# ---- 配置 Nginx ----
setup_nginx() {
    info "配置 Nginx 反向代理..."

    local NGINX_CONF="/etc/nginx/conf.d/cboard.conf"
    local FRONTEND_DIR="$INSTALL_DIR/frontend/dist"

    cat > "$NGINX_CONF" <<EOF
server {
    listen 80;
    server_name ${DOMAIN:-_};

    root $FRONTEND_DIR;
    index index.html;

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
    gzip_min_length 1024;

    # Let's Encrypt 验证
    location /.well-known/acme-challenge/ {
        root $INSTALL_DIR;
        allow all;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:$CBOARD_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    # 静态资源缓存
    location /assets/ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    # SPA 路由回退
    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF

    # 删除默认配置
    rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true
    rm -f /etc/nginx/conf.d/default.conf 2>/dev/null || true

    if nginx -t 2>/dev/null; then
        systemctl enable nginx >/dev/null 2>&1
        systemctl restart nginx
        ok "Nginx 配置完成"
    else
        err "Nginx 配置测试失败"
        nginx -t 2>&1 | tail -5
    fi
}

# ---- 写入 Nginx SSL 配置（含 80 跳转 + 443） ----
write_nginx_ssl_conf() {
    local NGINX_CONF="/etc/nginx/conf.d/cboard.conf"
    local FRONTEND_DIR="$INSTALL_DIR/frontend/dist"
    # 80: 保留 .well-known 供 certbot 续期，其余跳转 https
    # 443: 证书 + 站点
    cat > "$NGINX_CONF" <<EOF
server {
    listen 80;
    server_name $DOMAIN;
    location /.well-known/acme-challenge/ {
        root $INSTALL_DIR;
        allow all;
    }
    location / {
        return 301 https://\$host\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;

    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    root $FRONTEND_DIR;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
    gzip_min_length 1024;

    location /api/ {
        proxy_pass http://127.0.0.1:$CBOARD_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location /assets/ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF
    if nginx -t 2>/dev/null; then
        systemctl reload nginx 2>/dev/null || true
        return 0
    fi
    return 1
}

# ---- 申请 SSL 证书 ----
setup_ssl() {
    if [[ ! "$ENABLE_SSL" =~ ^[Yy]$ ]] || [ -z "$DOMAIN" ]; then
        return 0
    fi

    # 检查是否已有证书：有则确保 Nginx 已配置 443 并 reload
    if [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ] && [ -f "/etc/letsencrypt/live/$DOMAIN/privkey.pem" ]; then
        ok "检测到已有 SSL 证书"
        if write_nginx_ssl_conf; then
            ok "Nginx 已配置 HTTPS 并重载"
        else
            warn "Nginx SSL 配置写入失败，请检查 nginx -t"
        fi
        return 0
    fi

    info "申请 Let's Encrypt SSL 证书..."

    # 检查 80 端口
    local NGINX_WAS_RUNNING=false
    if systemctl is-active --quiet nginx 2>/dev/null; then
        NGINX_WAS_RUNNING=true
    fi

    # 尝试 nginx 方式
    if certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" --redirect 2>&1; then
        ok "SSL 证书申请成功 (nginx 方式)"
        return 0
    fi

    warn "nginx 方式失败，尝试 standalone 方式..."

    # 停止 nginx 释放 80 端口
    if [ "$NGINX_WAS_RUNNING" = true ]; then
        systemctl stop nginx 2>/dev/null || true
        sleep 2
    fi

    local CERT_SUCCESS=false
    if certbot certonly --standalone -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" 2>&1; then
        if [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
            CERT_SUCCESS=true
            ok "SSL 证书申请成功 (standalone 方式)"
            write_nginx_ssl_conf && ok "Nginx 已配置 HTTPS" || true
        fi
    fi

    # 恢复 Nginx
    if [ "$NGINX_WAS_RUNNING" = true ]; then
        systemctl start nginx 2>/dev/null || true
    fi

    if [ "$CERT_SUCCESS" = false ]; then
        warn "SSL 证书申请失败"
        echo ""
        echo -e "${YELLOW}可能的原因:${NC}"
        echo "  1. 域名 DNS 未正确解析到此服务器"
        echo "  2. 80 端口未开放或被占用"
        echo "  3. 防火墙阻止了访问"
        echo "  4. Let's Encrypt 频率限制"
        echo ""
        echo -e "${YELLOW}建议: 稍后使用菜单选项 2 重新配置域名和 SSL${NC}"
        echo ""
        read -rp "是否继续安装? (y/n) [y]: " cont
        if [[ "${cont:-y}" != "y" ]]; then
            fatal "安装已取消"
        fi
    fi
}

# ---- 防火墙配置 ----
setup_firewall() {
    if command -v ufw &>/dev/null; then
        ufw allow 80/tcp >/dev/null 2>&1 || true
        ufw allow 443/tcp >/dev/null 2>&1 || true
        ok "UFW 防火墙已放行 80/443"
    elif command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --add-service=http >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-service=https >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
        ok "Firewalld 防火墙已放行 80/443"
    fi
}

# ============================================================================
# 安装主流程
# ============================================================================
install_system() {
    echo -e "${BLUE}========== 开始安装 CBoard v2 ==========${NC}"
    echo ""

    info "正在检查 root 权限..."
    check_root
    ok "root 检查通过"

    info "正在检查磁盘空间..."
    if ! check_disk_space; then
        fatal "磁盘空间不足，请至少保留 1GB 可用空间"
    fi

    info "正在检测操作系统..."
    detect_os
    ok "系统: $OS $OS_VERSION"

    # 无人值守：使用环境变量或默认值，跳过交互
    if [ -n "$CBOARD_UNATTENDED" ] && [ "$CBOARD_UNATTENDED" != "0" ]; then
        INSTALL_DIR=${CBOARD_INSTALL_DIR:-/opt/cboard}
        CBOARD_PORT=${CBOARD_PORT:-9000}
        BACKEND_PORT=$CBOARD_PORT
        DOMAIN=${CBOARD_DOMAIN:-}
        ENABLE_SSL=${CBOARD_SSL:-n}
        ADMIN_EMAIL=${CBOARD_ADMIN_EMAIL:-admin@example.com}
        ADMIN_PASSWORD=${CBOARD_ADMIN_PASSWORD:-}
        if [ -z "$ADMIN_PASSWORD" ]; then
            ADMIN_PASSWORD=$(openssl rand -base64 12 2>/dev/null | tr -dc 'a-zA-Z0-9' | head -c 12)
            if [ -z "$ADMIN_PASSWORD" ]; then
                ADMIN_PASSWORD="Cboard$(date +%s | tail -c 6)"
            fi
            warn "未设置 CBOARD_ADMIN_PASSWORD，已生成随机密码，安装完成后请查看下方输出"
        fi
        ok "无人值守模式：安装目录=$INSTALL_DIR 端口=$CBOARD_PORT 管理员=$ADMIN_EMAIL"
    else
        interactive_config
    fi

    echo ""
    info "开始安装依赖与构建..."
    echo ""

    # 若已安装过，先停止服务，避免占用二进制/端口，确保新构建生效
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        info "停止已有 cboard 服务..."
        systemctl stop ${SERVICE_NAME} 2>/dev/null || true
        sleep 2
        ok "已停止"
    fi

    install_system_tools
    install_nginx
    install_go
    install_node
    deploy_project
    build_backend
    build_frontend
    create_service
    setup_nginx
    setup_ssl
    setup_firewall

    # 启动服务
    info "启动服务..."
    safe_release_port "$CBOARD_PORT"
    systemctl start ${SERVICE_NAME}
    sleep 3

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        ok "服务启动成功"
        # 本次输入的管理员邮箱/密码同步到数据库（.env 已存在时不会重新生成；若该邮箱无管理员则自动创建）
        if [ -n "$ADMIN_EMAIL" ] && [ -n "$ADMIN_PASSWORD" ] && [ -f "$INSTALL_DIR/cboard" ]; then
            if (cd "$INSTALL_DIR" && ./cboard reset-password --email "$ADMIN_EMAIL" --password "$ADMIN_PASSWORD"); then
                ok "管理员账号/密码已同步到数据库"
            else
                warn "管理员密码同步失败，请用菜单 9 重设密码"
            fi
        fi
    else
        err "服务启动失败，请查看日志: journalctl -u ${SERVICE_NAME} -n 50"
    fi

    # 打印结果
    print_install_result

    read -rp "按回车键继续..."
}

# ---- 打印安装结果 ----
print_install_result() {
    local BASE_URL
    if [ -n "$DOMAIN" ]; then
        if [[ "$ENABLE_SSL" =~ ^[Yy]$ ]]; then
            BASE_URL="https://$DOMAIN"
        else
            BASE_URL="http://$DOMAIN"
        fi
    else
        local SERVER_IP
        SERVER_IP=$(curl -s4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
        BASE_URL="http://${SERVER_IP}"
    fi

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║       CBoard v2 安装完成!                ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  访问地址:    ${CYAN}$BASE_URL${NC}  （直接打开即可，无需加端口）"
    echo -e "  安装目录:    ${CYAN}$INSTALL_DIR${NC}"
    echo ""
    echo -e "  管理员邮箱:  ${YELLOW}$ADMIN_EMAIL${NC}"
    echo -e "  管理员密码:  ${YELLOW}${ADMIN_PASSWORD:0:2}******${NC}"
    echo -e "  ${RED}请登录后立即修改默认密码!${NC}"
    echo ""

    # 服务状态
    echo -e "${BLUE}服务状态:${NC}"
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        echo -e "  后端服务: ${GREEN}运行中${NC}"
    else
        echo -e "  后端服务: ${RED}已停止${NC}"
    fi
    if systemctl is-active --quiet nginx 2>/dev/null; then
        echo -e "  Nginx:    ${GREEN}运行中${NC}"
    else
        echo -e "  Nginx:    ${RED}已停止${NC}"
    fi
    echo ""
}

# ============================================================================
# 菜单选项 2: 配置域名
# ============================================================================
configure_domain() {
    echo -e "${BLUE}========== 配置域名 ==========${NC}"
    echo ""

    set +e
    check_root

    cd "$INSTALL_DIR" 2>/dev/null || cd "$PROJECT_PATH" || {
        err "无法进入项目目录"
        read -rp "按回车键继续..."
        set -e
        return 1
    }

    # 获取域名
    while true; do
        read -rp "请输入域名 (例如: example.com): " DOMAIN
        if [ -z "$DOMAIN" ]; then
            err "域名不能为空"
            continue
        fi
        DOMAIN=$(echo "$DOMAIN" | sed 's|^https\?://||' | sed 's|/$||')
        if validate_domain "$DOMAIN"; then
            break
        else
            err "域名格式不正确"
        fi
    done

    # HTTPS
    read -rp "是否使用 HTTPS? (y/n) [y]: " USE_HTTPS
    USE_HTTPS=${USE_HTTPS:-y}

    local BASE_URL SSL_ENABLED
    if [[ "$USE_HTTPS" =~ ^[Yy]$ ]]; then
        BASE_URL="https://$DOMAIN"
        SSL_ENABLED="true"
        ENABLE_SSL="y"
    else
        BASE_URL="http://$DOMAIN"
        SSL_ENABLED="false"
        ENABLE_SSL="n"
    fi

    # 更新 .env
    info "更新 .env 文件..."
    if [ -f .env ]; then
        # 更新已有字段
        for key in DOMAIN BASE_URL CORS_ORIGINS SSL_ENABLED SUBSCRIPTION_URL_PREFIX ALIPAY_NOTIFY_URL ALIPAY_RETURN_URL; do
            case $key in
                DOMAIN) val="$DOMAIN" ;;
                BASE_URL) val="$BASE_URL" ;;
                CORS_ORIGINS) val="$BASE_URL" ;;
                SSL_ENABLED) val="$SSL_ENABLED" ;;
                SUBSCRIPTION_URL_PREFIX) val="$BASE_URL/sub" ;;
                ALIPAY_NOTIFY_URL) val="$BASE_URL/api/v1/payment/notify/alipay" ;;
                ALIPAY_RETURN_URL) val="$BASE_URL/payment/return" ;;
            esac
            if grep -q "^${key}=" .env; then
                sed -i "s|^${key}=.*|${key}=$val|" .env
            else
                echo "${key}=$val" >> .env
            fi
        done
        ok ".env 已更新"
    else
        warn ".env 不存在，请先安装系统"
    fi

    # 更新 Nginx
    setup_nginx

    # SSL
    if [[ "$USE_HTTPS" =~ ^[Yy]$ ]]; then
        read -rp "是否现在申请 SSL 证书? (y/n) [y]: " apply_ssl
        if [[ "${apply_ssl:-y}" =~ ^[Yy]$ ]]; then
            if [ -z "$ADMIN_EMAIL" ]; then
                read -rp "请输入邮箱 (用于 SSL 证书): " ADMIN_EMAIL
            fi
            setup_ssl
        fi
    fi

    # 重启服务
    systemctl restart nginx 2>/dev/null || true
    systemctl restart ${SERVICE_NAME} 2>/dev/null || true

    echo ""
    ok "域名配置完成: $BASE_URL"
    echo ""
    set -e
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 3: 修复常见错误
# ============================================================================
fix_common_errors() {
    echo -e "${BLUE}========== 修复常见错误 ==========${NC}"
    echo ""

    cd "$INSTALL_DIR" 2>/dev/null || cd "$PROJECT_PATH" || {
        err "无法进入项目目录"
        read -rp "按回车键继续..."
        return 1
    }

    echo -e "${YELLOW}1. 检查磁盘空间...${NC}"
    check_disk_space || true
    echo ""

    echo -e "${YELLOW}2. 检查 Go 环境...${NC}"
    export PATH=$PATH:/usr/local/go/bin
    if command -v go &>/dev/null; then
        ok "Go: $(go version | awk '{print $3}')"
    else
        warn "Go 未安装，正在安装..."
        detect_os
        install_go
    fi
    echo ""

    echo -e "${YELLOW}3. 检查 Node.js 环境...${NC}"
    if command -v node &>/dev/null; then
        ok "Node.js: $(node -v)"
    else
        warn "Node.js 未安装，正在安装..."
        detect_os
        install_node
    fi
    echo ""

    echo -e "${YELLOW}4. 检查 .env 配置...${NC}"
    if [ -f .env ]; then
        ok ".env 文件存在"
    else
        warn ".env 文件不存在，请先安装系统"
    fi
    echo ""

    echo -e "${YELLOW}5. 检查数据库文件...${NC}"
    if [ -f cboard.db ]; then
        local db_size
        db_size=$(du -h cboard.db | awk '{print $1}')
        ok "数据库文件存在 ($db_size)"
        # 检查权限
        if [ -w cboard.db ]; then
            ok "数据库文件可写"
        else
            warn "数据库文件不可写，修复权限..."
            chmod 664 cboard.db
        fi
    else
        warn "数据库文件不存在 (首次启动时会自动创建)"
    fi
    echo ""

    echo -e "${YELLOW}6. 检查前端构建...${NC}"
    if [ -f "frontend/dist/index.html" ]; then
        ok "前端构建产物存在"
    else
        warn "前端构建产物不存在"
        read -rp "是否重新构建前端? (y/n) [y]: " rebuild
        if [[ "${rebuild:-y}" =~ ^[Yy]$ ]]; then
            build_frontend
        fi
    fi
    echo ""

    echo -e "${YELLOW}7. 检查 systemd 服务...${NC}"
    if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
        systemctl daemon-reload
        ok "服务配置已重新加载"
    else
        warn "systemd 服务不存在，请先安装系统"
    fi
    echo ""

    echo -e "${YELLOW}8. 检查 Nginx 配置...${NC}"
    if command -v nginx &>/dev/null; then
        if nginx -t 2>/dev/null; then
            ok "Nginx 配置正确"
        else
            err "Nginx 配置有错误:"
            nginx -t 2>&1 | tail -5
        fi
    else
        warn "Nginx 未安装"
    fi
    echo ""

    echo -e "${YELLOW}9. 检查端口占用...${NC}"
    safe_release_port "$CBOARD_PORT"
    echo ""

    echo -e "${YELLOW}10. 重启服务...${NC}"
    if systemctl is-enabled ${SERVICE_NAME} &>/dev/null; then
        systemctl restart ${SERVICE_NAME} 2>/dev/null || true
        sleep 3
        if systemctl is-active --quiet ${SERVICE_NAME}; then
            ok "服务重启成功"
            # 检查端口
            if command -v ss &>/dev/null; then
                if ss -tlnp 2>/dev/null | grep -q ":$CBOARD_PORT "; then
                    ok "端口 $CBOARD_PORT 正在监听"
                else
                    warn "端口 $CBOARD_PORT 未监听，请查看日志"
                fi
            fi
        else
            err "服务重启失败"
            journalctl -u ${SERVICE_NAME} -n 20 --no-pager 2>/dev/null | tail -10
        fi
    fi

    echo ""
    ok "修复检查完成"
    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 4/5/6: 服务管理
# ============================================================================
start_service() {
    echo -e "${BLUE}========== 启动服务 ==========${NC}"
    check_root

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        ok "服务已在运行中"
        # 验证端口
        if command -v ss &>/dev/null && ss -tlnp 2>/dev/null | grep -q ":$CBOARD_PORT "; then
            ok "端口 $CBOARD_PORT 正在监听"
        fi
    else
        info "正在启动服务..."
        systemctl start ${SERVICE_NAME}
        local count=0
        while ! systemctl is-active --quiet ${SERVICE_NAME} && [ $count -lt 10 ]; do
            sleep 1
            count=$((count + 1))
        done
        if systemctl is-active --quiet ${SERVICE_NAME}; then
            sleep 2
            ok "服务启动成功"
        else
            err "服务启动失败"
            journalctl -u ${SERVICE_NAME} -n 20 --no-pager 2>/dev/null | tail -10
        fi
    fi
    echo ""
    read -rp "按回车键继续..."
}

stop_service() {
    echo -e "${BLUE}========== 停止服务 ==========${NC}"
    check_root

    if ! systemctl is-active --quiet ${SERVICE_NAME}; then
        warn "服务未运行"
    else
        info "正在停止服务..."
        systemctl stop ${SERVICE_NAME}
        local count=0
        while systemctl is-active --quiet ${SERVICE_NAME} && [ $count -lt 10 ]; do
            sleep 1
            count=$((count + 1))
        done
        if ! systemctl is-active --quiet ${SERVICE_NAME}; then
            ok "服务已停止"
        else
            warn "服务停止超时，强制终止..."
            systemctl kill --signal=SIGKILL ${SERVICE_NAME} 2>/dev/null || true
            sleep 2
        fi
    fi
    echo ""
    read -rp "按回车键继续..."
}

restart_service() {
    echo -e "${BLUE}========== 重启服务 ==========${NC}"
    check_root

    cd "$INSTALL_DIR" 2>/dev/null || cd "$PROJECT_PATH" || true

    info "正在重启服务..."
    systemctl stop ${SERVICE_NAME} 2>/dev/null || true
    sleep 2
    safe_release_port "$CBOARD_PORT"
    systemctl start ${SERVICE_NAME}

    local count=0
    while ! systemctl is-active --quiet ${SERVICE_NAME} && [ $count -lt 15 ]; do
        sleep 1
        count=$((count + 1))
    done

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        sleep 3
        ok "服务重启成功"
        if command -v ss &>/dev/null && ss -tlnp 2>/dev/null | grep -q ":$CBOARD_PORT "; then
            ok "端口 $CBOARD_PORT 正在监听"
        else
            warn "端口 $CBOARD_PORT 未监听，请查看日志"
        fi
    else
        err "服务重启失败"
        journalctl -u ${SERVICE_NAME} -n 30 --no-pager 2>/dev/null | tail -15
    fi
    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 7: 查看服务状态
# ============================================================================
check_service_status() {
    echo -e "${BLUE}========== 服务状态 ==========${NC}"
    echo ""

    # 读取配置
    local domain="" base_url="" ssl_enabled=""
    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        base_url=$(grep "^BASE_URL=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        ssl_enabled=$(grep "^SSL_ENABLED=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi

    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}【后端服务】${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        echo -e "   状态: ${GREEN}运行中${NC}"
    else
        echo -e "   状态: ${RED}已停止${NC}"
    fi

    if command -v ss &>/dev/null; then
        if ss -tlnp 2>/dev/null | grep -q ":$CBOARD_PORT "; then
            echo -e "   端口 $CBOARD_PORT: ${GREEN}已监听${NC}"
        else
            echo -e "   端口 $CBOARD_PORT: ${RED}未监听${NC}"
        fi
    fi

    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}【Nginx】${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if command -v nginx &>/dev/null && systemctl is-active --quiet nginx 2>/dev/null; then
        echo -e "   状态: ${GREEN}运行中${NC}"
    else
        echo -e "   状态: ${RED}已停止${NC}"
    fi

    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}【配置信息】${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "   域名: ${CYAN}${domain:-未配置}${NC}"
    echo -e "   访问地址: ${CYAN}${base_url:-未配置}${NC}"
    echo -e "   SSL: ${CYAN}${ssl_enabled:-未配置}${NC}"
    echo -e "   安装目录: ${CYAN}$work_dir${NC}"

    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}【最近日志】${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    journalctl -u ${SERVICE_NAME} -n 5 --no-pager --no-hostname 2>/dev/null || echo "   无法获取日志"

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 8: 查看日志
# ============================================================================
view_service_logs() {
    echo -e "${BLUE}========== 查看服务日志 ==========${NC}"
    echo ""
    echo -e "   ${GREEN}1.${NC} 实时日志 (tail -f)"
    echo -e "   ${GREEN}2.${NC} 最近 50 行"
    echo -e "   ${GREEN}3.${NC} 最近 100 行"
    echo -e "   ${GREEN}4.${NC} 最近 200 行"
    echo -e "   ${GREEN}5.${NC} 仅错误日志"
    echo -e "   ${GREEN}6.${NC} Nginx 错误日志"
    echo -e "   ${GREEN}0.${NC} 返回"
    echo ""

    read -rp "请选择 [0-6]: " log_choice

    case $log_choice in
        1) echo -e "${YELLOW}按 Ctrl+C 退出实时日志${NC}"; journalctl -u ${SERVICE_NAME} -f ;;
        2) journalctl -u ${SERVICE_NAME} -n 50 --no-pager ;;
        3) journalctl -u ${SERVICE_NAME} -n 100 --no-pager ;;
        4) journalctl -u ${SERVICE_NAME} -n 200 --no-pager ;;
        5) journalctl -u ${SERVICE_NAME} -p err -n 50 --no-pager ;;
        6) tail -100 /var/log/nginx/error.log 2>/dev/null || echo "Nginx 错误日志不存在" ;;
        0) return 0 ;;
        *) err "无效的选择" ;;
    esac

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 9: 重设管理员密码
# ============================================================================
reset_admin_password() {
    echo -e "${BLUE}========== 重设管理员密码 ==========${NC}"
    echo ""

    cd "$INSTALL_DIR" 2>/dev/null || cd "$PROJECT_PATH" || {
        err "无法进入项目目录"
        read -rp "按回车键继续..."
        return 1
    }

    local new_email new_password

    read -rp "管理员邮箱 (留空使用当前): " new_email

    while true; do
        read -rsp "新密码 (至少8位): " new_password
        echo
        if [ ${#new_password} -lt 8 ]; then
            err "密码长度至少8位"
        else
            break
        fi
    done

    # 通过 API 或直接操作数据库
    if [ -f cboard.db ] && command -v sqlite3 &>/dev/null; then
        # 使用 Go 程序重置密码
        if [ -f cboard ]; then
            ./cboard reset-password --email "${new_email:-admin}" --password "$new_password" 2>/dev/null && {
                ok "密码重置成功"
            } || {
                warn "通过程序重置失败，请手动操作数据库"
            }
        else
            warn "cboard 可执行文件不存在，请先构建"
        fi
    else
        warn "数据库文件或 sqlite3 工具不存在"
        echo "请手动重置密码或重新安装"
    fi

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 10: 查看管理员账号
# ============================================================================
view_admin_account() {
    echo -e "${BLUE}========== 查看管理员账号 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    if [ -f "$work_dir/.env" ]; then
        local admin_email
        admin_email=$(grep "^ADMIN_EMAIL=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        echo -e "  管理员邮箱: ${GREEN}${admin_email:-未配置}${NC}"
    else
        warn ".env 文件不存在"
    fi

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 11: 备份数据
# ============================================================================
backup_data() {
    echo -e "${BLUE}========== 备份数据 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    cd "$work_dir" || {
        err "无法进入项目目录"
        read -rp "按回车键继续..."
        return 1
    }

    local BACKUP_DIR="$work_dir/backups"
    mkdir -p "$BACKUP_DIR"
    local TIMESTAMP
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)

    # 备份数据库
    if [ -f "cboard.db" ]; then
        cp "cboard.db" "$BACKUP_DIR/cboard_${TIMESTAMP}.db"
        ok "数据库已备份: $BACKUP_DIR/cboard_${TIMESTAMP}.db"
    else
        warn "数据库文件不存在"
    fi

    # 备份 .env
    if [ -f ".env" ]; then
        cp ".env" "$BACKUP_DIR/env_${TIMESTAMP}.bak"
        ok ".env 已备份"
    fi

    # 显示备份列表
    echo ""
    echo -e "${YELLOW}现有备份:${NC}"
    ls -lh "$BACKUP_DIR"/*.db 2>/dev/null || echo "  无备份文件"

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 12: 重装网站 (保留数据)
# ============================================================================
reinstall_website() {
    echo -e "${BLUE}========== 重装网站 (保留数据) ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    cd "$work_dir" || {
        err "无法进入项目目录"
        read -rp "按回车键继续..."
        return 1
    }

    echo -e "${YELLOW}此操作将重新构建，但保留以下数据:${NC}"
    echo "  - 数据库文件 (cboard.db)"
    echo "  - 配置文件 (.env)"
    echo "  - 备份文件 (backups/)"
    echo ""

    read -rp "确认继续? (y/n) [n]: " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then echo "已取消"; read -rp "按回车键继续..."; return 0; fi

    # 备份
    info "1. 备份重要文件..."
    local BACKUP_DIR="$work_dir/backup_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    if [ -f "cboard.db" ]; then cp "cboard.db" "$BACKUP_DIR/"; fi
    if [ -f ".env" ]; then cp ".env" "$BACKUP_DIR/"; fi
    ok "备份完成: $BACKUP_DIR"

    # 停止服务
    info "2. 停止服务..."
    systemctl stop ${SERVICE_NAME} 2>/dev/null || true

    # 重新构建后端
    info "3. 重新构建后端..."
    export PATH=$PATH:/usr/local/go/bin
    go build -o cboard cmd/server/main.go 2>&1 || {
        err "后端构建失败"
        read -rp "按回车键继续..."
        return 1
    }
    chmod +x cboard
    ok "后端构建完成"

    # 重新构建前端
    info "4. 重新构建前端..."
    cd frontend
    rm -rf node_modules dist 2>/dev/null || true
    npm install --silent 2>/dev/null || { err "前端依赖安装失败"; cd ..; read -rp "按回车键继续..."; return 1; }
    export NODE_OPTIONS="--max-old-space-size=4096"
    npx vite build 2>/dev/null || { err "前端构建失败"; cd ..; read -rp "按回车键继续..."; return 1; }
    cd ..
    ok "前端构建完成"

    # 重新加载服务
    info "5. 重启服务..."
    systemctl daemon-reload
    systemctl start ${SERVICE_NAME}
    sleep 3

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        ok "服务启动成功"
    else
        err "服务启动失败"
    fi

    echo ""
    ok "重装完成! 数据已保留"
    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 13: 诊断 403 错误
# ============================================================================
diagnose_403_error() {
    echo -e "${BLUE}========== 诊断 403 错误 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    local domain=""
    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi
    if [ -z "$domain" ]; then read -rp "请输入域名: " domain; fi

    local FRONTEND_ROOT="$work_dir/frontend/dist"

    echo -e "${YELLOW}1. 检查前端目录...${NC}"
    if [ -d "$FRONTEND_ROOT" ]; then
        ok "前端目录存在: $FRONTEND_ROOT"
    else
        err "前端目录不存在"
        echo "  请先运行选项 1 安装系统"
    fi

    echo -e "${YELLOW}2. 检查关键文件...${NC}"
    if [ -f "$FRONTEND_ROOT/index.html" ]; then
        local fsize
        fsize=$(stat -c%s "$FRONTEND_ROOT/index.html" 2>/dev/null || echo "0")
        ok "index.html 存在 (${fsize} 字节)"
    else
        err "index.html 不存在"
    fi

    echo -e "${YELLOW}3. 检查文件权限...${NC}"
    if [ -d "$FRONTEND_ROOT" ]; then
        local dir_perm
        dir_perm=$(stat -c%a "$FRONTEND_ROOT" 2>/dev/null || echo "unknown")
        echo "  目录权限: $dir_perm (推荐 755)"

        read -rp "是否自动修复权限? (y/n) [y]: " fix_perm
        if [[ "${fix_perm:-y}" =~ ^[Yy]$ ]]; then
            chmod -R 755 "$FRONTEND_ROOT" 2>/dev/null || true
            find "$FRONTEND_ROOT" -type f -exec chmod 644 {} \; 2>/dev/null || true
            ok "权限已修复"
        fi
    fi

    echo -e "${YELLOW}4. 检查 Nginx 配置...${NC}"
    local nginx_conf="/etc/nginx/conf.d/cboard.conf"
    if [ -f "$nginx_conf" ]; then
        ok "Nginx 配置文件存在"
        if grep -q "root.*$FRONTEND_ROOT" "$nginx_conf" 2>/dev/null; then
            ok "root 路径配置正确"
        else
            err "root 路径可能不正确"
            echo "  当前: $(grep '^\s*root' "$nginx_conf" | head -1)"
            echo "  应为: root $FRONTEND_ROOT;"
        fi
    else
        err "Nginx 配置文件不存在"
    fi

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 14: 更新代码
# ============================================================================
update_code() {
    echo -e "${BLUE}========== 更新代码 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    cd "$work_dir" || {
        err "无法进入项目目录"
        read -rp "按回车键继续..."
        return 1
    }

    if [ ! -d ".git" ]; then
        err "当前目录不是 Git 仓库"
        read -rp "按回车键继续..."
        return 1
    fi

    # 显示当前状态
    local branch commit
    branch=$(git branch --show-current 2>/dev/null || echo "unknown")
    commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    echo -e "  分支: ${GREEN}$branch${NC}"
    echo -e "  提交: ${GREEN}$commit${NC}"
    echo ""

    # 处理未提交更改
    if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
        warn "检测到未提交的更改"
        echo "  1. 暂存更改 (stash)"
        echo "  2. 放弃更改 (reset)"
        echo "  3. 取消更新"
        read -rp "请选择 [1-3, 默认: 2]: " handle
        case ${handle:-2} in
            1) git stash push -m "Auto stash $(date +%Y%m%d_%H%M%S)" 2>/dev/null; ok "已暂存" ;;
            2) git reset --hard HEAD 2>/dev/null; ok "已重置" ;;
            3) read -rp "按回车键继续..."; return 0 ;;
        esac
    fi

    # 修复 Git 所有权问题
    git config --global --add safe.directory "$work_dir" 2>/dev/null || true

    # 拉取更新
    info "拉取远程更新..."
    if git pull origin "$branch" 2>&1; then
        ok "代码更新成功"
    else
        err "代码更新失败"
        read -rp "按回车键继续..."
        return 1
    fi

    # 重新构建
    read -rp "是否重新构建? (y/n) [y]: " rebuild
    if [[ "${rebuild:-y}" =~ ^[Yy]$ ]]; then
        info "停止服务..."
        systemctl stop ${SERVICE_NAME} 2>/dev/null || true

        info "构建后端..."
        export PATH=$PATH:/usr/local/go/bin
        go build -o cboard cmd/server/main.go 2>&1 || { err "后端构建失败"; }
        chmod +x cboard 2>/dev/null || true

        info "构建前端..."
        cd frontend
        npm install --silent 2>/dev/null || true
        export NODE_OPTIONS="--max-old-space-size=4096"
        npx vite build 2>/dev/null || { err "前端构建失败"; }
        cd ..

        info "启动服务..."
        systemctl start ${SERVICE_NAME}
        sleep 3
        if systemctl is-active --quiet ${SERVICE_NAME}; then
            ok "服务启动成功"
        else
            err "服务启动失败"
        fi
    fi

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 15: 修复 Nginx (SSL 验证)
# ============================================================================
fix_nginx_for_ssl() {
    echo -e "${BLUE}========== 修复 Nginx SSL 验证配置 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    local domain=""
    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi
    if [ -z "$domain" ]; then read -rp "请输入域名: " domain; fi
    if [ -z "$domain" ]; then err "域名不能为空"; read -rp "按回车键继续..."; return 1; fi

    local nginx_conf="/etc/nginx/conf.d/cboard.conf"
    if [ ! -f "$nginx_conf" ]; then
        err "Nginx 配置文件不存在: $nginx_conf"
        read -rp "按回车键继续..."
        return 1
    fi

    # 备份
    cp "$nginx_conf" "${nginx_conf}.bak.$(date +%Y%m%d_%H%M%S)"
    ok "已备份原配置"

    # 检查是否已有 .well-known 配置
    if grep -q "\.well-known" "$nginx_conf"; then
        ok ".well-known 配置已存在"
    else
        info "添加 .well-known 配置..."
        # 在 location /api/ 之前插入
        sed -i '/location \/api\//i\    # Let'\''s Encrypt 验证\n    location \/.well-known\/acme-challenge\/ {\n        root '"$work_dir"';\n        allow all;\n    }\n' "$nginx_conf"
        ok ".well-known 配置已添加"
    fi

    # 测试并重载
    if nginx -t 2>/dev/null; then
        systemctl reload nginx 2>/dev/null || true
        ok "Nginx 配置已重载"
        echo ""
        echo -e "${YELLOW}现在可以在申请 SSL 证书了:${NC}"
        echo "  certbot --nginx -d $domain"
    else
        err "Nginx 配置测试失败，已恢复备份"
        cp "${nginx_conf}.bak."* "$nginx_conf" 2>/dev/null || true
    fi

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 16: 诊断网站访问
# ============================================================================
diagnose_website_access() {
    echo -e "${BLUE}========== 诊断网站访问 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    local domain="" base_url=""
    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        base_url=$(grep "^BASE_URL=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi
    if [ -z "$domain" ]; then read -rp "请输入域名: " domain; fi
    if [ -z "$base_url" ]; then base_url="http://$domain"; fi

    echo -e "  域名: ${CYAN}$domain${NC}"
    echo -e "  地址: ${CYAN}$base_url${NC}"
    echo ""

    # 1. 后端服务
    echo -e "${YELLOW}1. 检查后端服务...${NC}"
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        ok "后端服务运行中"
        if command -v ss &>/dev/null && ss -tlnp 2>/dev/null | grep -q ":$CBOARD_PORT "; then
            ok "端口 $CBOARD_PORT 正在监听"
        else
            err "端口 $CBOARD_PORT 未监听"
        fi
        # 测试 API
        local http_code
        http_code=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$CBOARD_PORT/api/health" 2>/dev/null || echo "000")
        if [ "$http_code" = "200" ]; then
            ok "后端 API 可访问"
        else
            warn "后端 API 返回 HTTP $http_code"
        fi
    else
        err "后端服务未运行"
    fi
    echo ""

    # 2. Nginx
    echo -e "${YELLOW}2. 检查 Nginx...${NC}"
    if command -v nginx &>/dev/null; then
        if systemctl is-active --quiet nginx 2>/dev/null; then
            ok "Nginx 运行中"
        else
            err "Nginx 未运行"
        fi
        if nginx -t 2>&1 | grep -q "successful"; then
            ok "Nginx 配置正确"
        else
            err "Nginx 配置有错误"
            nginx -t 2>&1 | tail -3
        fi
    else
        err "Nginx 未安装"
    fi
    echo ""

    # 3. 前端文件
    echo -e "${YELLOW}3. 检查前端文件...${NC}"
    if [ -f "$work_dir/frontend/dist/index.html" ]; then
        ok "前端文件存在"
    else
        err "前端文件不存在"
    fi
    echo ""

    # 4. DNS
    echo -e "${YELLOW}4. 检查 DNS 解析...${NC}"
    if [ -n "$domain" ] && command -v dig &>/dev/null; then
        local dns_ip
        dns_ip=$(dig +short "$domain" 2>/dev/null | head -1)
        if [ -n "$dns_ip" ]; then
            ok "域名解析到: $dns_ip"
            local server_ip
            server_ip=$(curl -s4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
            if [ "$dns_ip" = "$server_ip" ]; then
                ok "DNS 指向当前服务器"
            else
                warn "DNS 未指向当前服务器 (服务器 IP: $server_ip)"
            fi
        else
            err "域名无法解析"
        fi
    fi
    echo ""

    # 5. 网站访问测试
    echo -e "${YELLOW}5. 测试网站访问...${NC}"
    if [ -n "$base_url" ]; then
        local code
        code=$(curl -s -o /dev/null -w "%{http_code}" "$base_url" 2>/dev/null || echo "000")
        case "$code" in
            200) ok "网站可访问 (HTTP $code)" ;;
            301|302) warn "网站重定向 (HTTP $code)" ;;
            403) err "403 禁止访问 - 请运行选项 13 诊断" ;;
            502|503|504) err "HTTP $code - 后端服务可能异常" ;;
            000) err "无法连接 - 检查 DNS/防火墙/Nginx" ;;
            *) warn "HTTP $code" ;;
        esac
    fi

    echo ""
    read -rp "按回车键继续..."
}

# ============================================================================
# 菜单选项 17: 卸载
# ============================================================================
uninstall_cboard() {
    echo -e "${BLUE}========== 卸载 CBoard v2 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi

    echo -e "${RED}此操作将:${NC}"
    echo "  - 停止并删除 systemd 服务 (cboard)"
    echo "  - 删除 Nginx 站点配置"
    echo "  - 可选：删除安装目录及全部数据"
    echo ""

    read -rp "确定要卸载吗? (输入 yes 确认): " confirm
    if [ "$(echo "$confirm" | tr '[:lower:]' '[:upper:]')" != "YES" ]; then
        echo "已取消"
        read -rp "按回车键继续..."
        return 0
    fi

    # 停止并移除服务
    info "停止并移除 systemd 服务..."
    systemctl stop ${SERVICE_NAME} 2>/dev/null || true
    systemctl disable ${SERVICE_NAME} 2>/dev/null || true
    rm -f /etc/systemd/system/${SERVICE_NAME}.service
    systemctl daemon-reload
    ok "systemd 服务已删除"

    # 删除 Nginx 配置
    info "删除 Nginx 配置..."
    rm -f /etc/nginx/conf.d/cboard.conf
    if nginx -t 2>/dev/null; then
        systemctl reload nginx 2>/dev/null || true
        ok "Nginx 配置已删除并重载"
    else
        warn "已删除 cboard 站点配置，请检查 Nginx 其他配置后手动 reload"
    fi

    rm -f /usr/local/bin/cboard-ctl 2>/dev/null || true

    # 是否删除安装目录
    echo ""
    read -rp "是否同时删除安装目录及全部数据? $work_dir (y/n) [n]: " del_dir
    if [[ "${del_dir}" =~ ^[Yy]$ ]]; then
        if [ -d "$work_dir" ]; then
            info "正在删除 $work_dir ..."
            cd / 2>/dev/null || true
            rm -rf "$work_dir"
            ok "安装目录已删除"
        else
            warn "目录不存在: $work_dir"
        fi
    else
        echo -e "${YELLOW}安装目录已保留: $work_dir${NC}"
    fi

    echo ""
    ok "CBoard 已完全卸载"
    read -rp "按回车键继续..."
}

# ============================================================================
# 显示菜单
# ============================================================================
show_menu() {
    clear
    echo -e "${CYAN}"
    echo "  ╔══════════════════════════════════════════════╗"
    echo "  ║     CBoard v2 管理面板 (无宝塔版) v$SCRIPT_VERSION    ║"
    echo "  ╚══════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo -e "  ${GREEN} 1.${NC} 安装系统"
    echo -e "  ${GREEN} 2.${NC} 配置域名 & SSL"
    echo -e "  ${GREEN} 3.${NC} 修复常见错误"
    echo ""
    echo -e "  ${GREEN} 4.${NC} 启动服务"
    echo -e "  ${GREEN} 5.${NC} 停止服务"
    echo -e "  ${GREEN} 6.${NC} 重启服务"
    echo -e "  ${GREEN} 7.${NC} 查看服务状态"
    echo -e "  ${GREEN} 8.${NC} 查看服务日志"
    echo ""
    echo -e "  ${GREEN} 9.${NC} 重设管理员密码"
    echo -e "  ${GREEN}10.${NC} 查看管理员账号"
    echo -e "  ${GREEN}11.${NC} 备份数据"
    echo -e "  ${GREEN}12.${NC} 重装网站 (保留数据)"
    echo ""
    echo -e "  ${GREEN}13.${NC} 诊断 403 错误"
    echo -e "  ${GREEN}14.${NC} 更新代码 (Git)"
    echo -e "  ${GREEN}15.${NC} 修复 Nginx SSL 验证"
    echo -e "  ${GREEN}16.${NC} 诊断网站访问"
    echo -e "  ${GREEN}17.${NC} 卸载 CBoard"
    echo ""
    echo -e "  ${GREEN} 0.${NC} 退出"
    echo ""
}

# ============================================================================
# 主流程
# ============================================================================
main() {
    # 读取已有配置
    local work_dir="$INSTALL_DIR"
    if [ ! -d "$work_dir" ]; then work_dir="$PROJECT_PATH"; fi
    if [ -f "$work_dir/.env" ]; then
        CBOARD_PORT=$(grep "^PORT=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        CBOARD_PORT=${CBOARD_PORT:-9000}
        BACKEND_PORT=$CBOARD_PORT
        DOMAIN=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi

    check_concurrent

    # 无人值守：直接执行安装后退出，不进入菜单
    if [ -n "$CBOARD_UNATTENDED" ] && [ "$CBOARD_UNATTENDED" != "0" ]; then
        install_system
        exit 0
    fi

    # 一键安装：若尚未安装，直接进入安装流程，完成后进入菜单（无需先选 1）
    if [ ! -f "$work_dir/.env" ] && [ ! -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
        install_system
    fi

    while true; do
        show_menu
        read -rp "请选择操作 [0-17]: " choice

        case $choice in
            1)  install_system ;;
            2)  configure_domain ;;
            3)  fix_common_errors ;;
            4)  start_service ;;
            5)  stop_service ;;
            6)  restart_service ;;
            7)  check_service_status ;;
            8)  view_service_logs ;;
            9)  reset_admin_password ;;
            10) view_admin_account ;;
            11) backup_data ;;
            12) reinstall_website ;;
            13) diagnose_403_error ;;
            14) update_code ;;
            15) fix_nginx_for_ssl ;;
            16) diagnose_website_access ;;
            17) uninstall_cboard ;;
            0)  echo -e "${GREEN}再见!${NC}"; exit 0 ;;
            *)  err "无效的选择"; sleep 2 ;;
        esac
    done
}

main "$@"
