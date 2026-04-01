#!/usr/bin/env bash
# ============================================================================
# CBoard v2 旗舰版一键安装 & 管理脚本 (纯净 Linux 环境)
# 支持系统: Ubuntu 20.04+, Debian 11+, CentOS 7+, AlmaLinux 8+, Rocky Linux 8+
# ============================================================================
# 强制使用 bash 运行
[ -n "$BASH_VERSION" ] || exec /usr/bin/env bash "$0" "$@"

# ---- 全局配置与版本 ----
SCRIPT_VERSION="2.1.0"
PROJECT_PATH="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="/opt/cboard"
SERVICE_NAME="cboard-v2"
CBOARD_PORT=9000
BACKEND_PORT=$CBOARD_PORT
LOCK_FILE="/tmp/cboard_install.lock"

# 软件版本定义
GO_VERSION="1.24.0"
NODE_VERSION="20.x"

# 运行时变量
DOMAIN=""
ENABLE_SSL="n"
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
OS=""
OS_VERSION=""
PKG_MGR=""

# ---- 颜色主题 ----
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BLUE='\033[0;34m'; MAGENTA='\033[0;35m'; NC='\033[0m'

# ---- 基础日志函数 ----
info()  { echo -e "${CYAN}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()   { echo -e "${RED}[ERROR]${NC} $*"; }
fatal() { echo -e "${RED}[FATAL]${NC} $*"; exit 1; }
pause() { echo ""; read -rp "按回车键继续..." </dev/tty; }

# ---- 基础工具函数 ----
check_cmd() { command -v "$1" &>/dev/null; }
is_service_running() { systemctl is-active --quiet "${1:-$SERVICE_NAME}" 2>/dev/null; }
is_port_listening() { check_cmd ss && ss -tlnp 2>/dev/null | grep -q ":${1:-$CBOARD_PORT} "; }

# ---- 并发运行检查 ----
check_concurrent() {
    if [ -f "$LOCK_FILE" ]; then
        local lock_pid=$(cat "$LOCK_FILE" 2>/dev/null)
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

# ---- 环境与系统检测 ----
check_root() {
    [[ $EUID -ne 0 ]] && fatal "请使用 root 用户运行此脚本"
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID; OS_VERSION=$VERSION_ID
    elif [ -f /etc/redhat-release ]; then
        OS="centos"
    else
        fatal "不支持的操作系统"
    fi

    # 确定包管理器
    case $OS in
        ubuntu|debian) PKG_MGR="apt-get" ;;
        centos|rhel|almalinux|rocky) PKG_MGR="yum" ;;
        *) fatal "不支持的系统或包管理器: $OS" ;;
    esac
    info "检测到系统: $OS $OS_VERSION ($PKG_MGR)"
}

check_disk_space() {
    local available=$(df -BM "$PROJECT_PATH" 2>/dev/null | awk 'NR==2{print $4}' | tr -d 'M')
    if [ -n "$available" ] && [ "$available" -lt 1024 ]; then
        err "磁盘可用空间不足: ${available}MB (需要至少 1GB)"
        return 1
    fi
    ok "磁盘空间充足: ${available}MB 可用"
    return 0
}

# 增加：检查并创建 Swap (防止编译 Go/Vite 时 OOM)
ensure_swap() {
    local mem_total=$(free -m | awk '/^Mem:/{print $2}')
    local swap_total=$(free -m | awk '/^Swap:/{print $2}')
    
    if [ "$mem_total" -lt 2048 ] && [ "$swap_total" -lt 1024 ]; then
        warn "系统物理内存不足 2GB 且未配置足够的 Swap，编译极易失败！"
        info "正在自动配置 2GB 临时虚拟内存 (Swap)..."
        fallocate -l 2G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=2048
        chmod 600 /swapfile
        mkswap /swapfile >/dev/null 2>&1
        swapon /swapfile >/dev/null 2>&1
        echo '/swapfile none swap sw 0 0' >> /etc/fstab
        ok "2GB Swap 虚拟内存配置完毕"
    fi
}

# ---- 安全释放端口 ----
safe_release_port() {
    local port=$1
    local pid=$(lsof -ti ":$port" 2>/dev/null || true)
    if [ -n "$pid" ]; then
        local pname=$(ps -p "$pid" -o comm= 2>/dev/null || true)
        if [[ "$pname" == "nginx" ]] || [[ "$pname" == "sshd" ]]; then
            return 0
        fi
        warn "端口 $port 被 PID $pid ($pname) 占用，正在释放..."
        kill "$pid" 2>/dev/null || true
        sleep 2
        kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
    fi
}

# ---- 依赖安装模块 ----
install_system_tools() {
    info "更新软件源并安装系统依赖..."
    if [ "$PKG_MGR" = "apt-get" ]; then
        $PKG_MGR update -qq
        $PKG_MGR install -y -qq curl wget git unzip lsof net-tools certbot >/dev/null 2>&1
    else
        $PKG_MGR install -y epel-release >/dev/null 2>&1 || true
        $PKG_MGR install -y curl wget git unzip lsof net-tools certbot >/dev/null 2>&1
    fi
    ok "系统依赖安装完成"
}

install_redis() {
    if check_cmd redis-server; then
        ok "Redis 已安装: $(redis-server --version | awk '{print $3}')"
    else
        info "安装 Redis 服务..."
        $PKG_MGR install -y redis-server >/dev/null 2>&1 || $PKG_MGR install -y redis >/dev/null 2>&1 || dnf install -y redis >/dev/null 2>&1 || true
        ok "Redis 安装完成"
    fi

    systemctl enable redis-server >/dev/null 2>&1 || systemctl enable redis >/dev/null 2>&1 || true
    systemctl start redis-server >/dev/null 2>&1 || systemctl start redis >/dev/null 2>&1 || true

    if redis-cli ping 2>/dev/null | grep -q PONG; then
        ok "Redis 服务已启动"
    else
        warn "Redis 启动失败，系统将回退到内存缓存模式"
    fi
}

install_nginx() {
    if check_cmd nginx; then
        ok "Nginx 已安装: $(nginx -v 2>&1 | awk -F/ '{print $2}')"
        return
    fi
    info "安装 Nginx..."
    $PKG_MGR install -y nginx python3-certbot-nginx >/dev/null 2>&1
    systemctl enable nginx >/dev/null 2>&1
    systemctl start nginx >/dev/null 2>&1
    ok "Nginx 安装完成"
}

install_go() {
    if check_cmd go; then
        ok "Go 已安装: $(go version | awk '{print $3}')"
        return
    fi
    info "安装 Go $GO_VERSION..."
    local arch=$(uname -m)
    case $arch in
        x86_64)  GO_ARCH="amd64" ;;
        aarch64) GO_ARCH="arm64" ;;
        *)       fatal "不支持的架构: $arch" ;;
    esac
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz || fatal "Go 下载失败"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    fi
    export GOPROXY=https://goproxy.cn,direct
    ok "Go $(go version | awk '{print $3}') 安装完成"
}

install_node() {
    if check_cmd node; then
        ok "Node.js 已安装: $(node -v)"
        return
    fi
    info "安装 Node.js $NODE_VERSION..."
    curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION} | bash - >/dev/null 2>&1 || {
        curl -fsSL https://rpm.nodesource.com/setup_${NODE_VERSION} | bash - >/dev/null 2>&1
    }
    $PKG_MGR install -y nodejs >/dev/null 2>&1
    ok "Node.js $(node -v) 安装完成"
}

# ---- 配置校验模块 ----
validate_domain() {
    [[ "$1" =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$ ]]
}

validate_email() {
    [[ "$1" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]
}

get_admin_info() {
    echo -e "\n${CYAN}========== 管理员信息配置 ==========${NC}\n"
    while true; do
        read -rp "管理员邮箱: " ADMIN_EMAIL
        if validate_email "$ADMIN_EMAIL"; then break; else err "邮箱格式不正确，请重新输入"; fi
    done

    while true; do
        read -rsp "管理员密码 (至少8位): " ADMIN_PASSWORD; echo
        if [ ${#ADMIN_PASSWORD} -lt 8 ]; then err "密码长度至少8位"; continue; fi
        read -rsp "确认密码: " ADMIN_PASSWORD_CONFIRM; echo
        if [ "$ADMIN_PASSWORD" == "$ADMIN_PASSWORD_CONFIRM" ]; then break; else err "两次密码不一致"; fi
    done
    ok "管理员信息配置完毕"
}

interactive_config() {
    echo -e "\n${CYAN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║       CBoard v2 安装配置 (无宝塔版)      ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}\n"

    read -rp "安装目录 [$INSTALL_DIR]: " input
    INSTALL_DIR=${input:-$INSTALL_DIR}
    CBOARD_PORT=9000; BACKEND_PORT=$CBOARD_PORT

    while true; do
        read -rp "绑定域名 (留空使用 IP 访问): " DOMAIN
        [ -z "$DOMAIN" ] && break
        DOMAIN=$(echo "$DOMAIN" | sed -e 's|^https\?://||' -e 's|/$||')
        if validate_domain "$DOMAIN"; then break; else err "域名格式不正确"; fi
    done

    [ -n "$DOMAIN" ] && { read -rp "是否自动申请 SSL 证书? (y/n) [y]: " ENABLE_SSL; ENABLE_SSL=${ENABLE_SSL:-y}; }

    get_admin_info

    echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    info "安装目录: $INSTALL_DIR"
    info "绑定域名: ${DOMAIN:-无(IP 访问)}"
    info "SSL 证书: ${ENABLE_SSL}"
    info "管理员邮箱: $ADMIN_EMAIL"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    read -rp "确认以上配置? (y/n) [y]: " confirm
    [[ "${confirm:-y}" != "y" ]] && fatal "安装已取消"
}

# ---- 项目构建模块 ----
create_env_file() {
    local target_dir="${1:-$INSTALL_DIR}"
    cd "$target_dir" || return 1

    if [ -f .env ]; then
        if ! grep -q '^CORS_ORIGINS=' .env 2>/dev/null; then
            local base_url=$(grep '^BASE_URL=' .env 2>/dev/null | cut -d'=' -f2- | tr -d '"' | xargs)
            [ -n "$base_url" ] && { echo "CORS_ORIGINS=$base_url" >> .env; ok "已补充 CORS_ORIGINS 到 .env"; }
        fi
        warn ".env 已存在，跳过生成"; return 0
    fi

    local SECRET_KEY=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n')
    local BASE_URL="http://$(curl -s4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}'):$CBOARD_PORT"
    if [ -n "$DOMAIN" ]; then
        [[ "$ENABLE_SSL" =~ ^[Yy]$ ]] && BASE_URL="https://$DOMAIN" || BASE_URL="http://$DOMAIN"
    fi

    cat > .env <<EOF
# CBoard v2 配置文件
DEBUG=false
PORT=$CBOARD_PORT
HOST=127.0.0.1
SECRET_KEY=$SECRET_KEY
BASE_URL=$BASE_URL
CORS_ORIGINS=$BASE_URL
DATABASE_URL=sqlite:///./cboard.db
DOMAIN=${DOMAIN}
SSL_ENABLED=$([ "${ENABLE_SSL,,}" = "y" ] && echo "true" || echo "false")
ADMIN_EMAIL=$ADMIN_EMAIL
SUBSCRIPTION_URL_PREFIX=$BASE_URL/sub
ALIPAY_NOTIFY_URL=$BASE_URL/api/v1/payment/notify/alipay
ALIPAY_RETURN_URL=$BASE_URL/payment/return
REDIS_ADDR=127.0.0.1:6379
EOF
    ok ".env 配置文件已生成"
}

deploy_project() {
    info "部署 CBoard v2 到 $INSTALL_DIR ..."
    mkdir -p "$INSTALL_DIR"
    if [ -f "$PROJECT_PATH/go.mod" ] && grep -q "cboard/v2" "$PROJECT_PATH/go.mod" 2>/dev/null; then
        if [ "$PROJECT_PATH" != "$INSTALL_DIR" ]; then
            cp -r "$PROJECT_PATH"/* "$INSTALL_DIR/" 2>/dev/null || true
            cp -r "$PROJECT_PATH"/.env.example "$PROJECT_PATH"/install.sh "$INSTALL_DIR/" 2>/dev/null || true
        fi
    else
        fatal "请将此脚本放在 CBoard v2 源码根目录下运行"
    fi
    create_env_file "$INSTALL_DIR"
}

build_backend() {
    info "构建后端 (自动代理加速)..."
    cd "$INSTALL_DIR"
    export PATH=$PATH:/usr/local/go/bin
    export GOPROXY=https://goproxy.cn,direct

    info "下载 Go 依赖包..."
    go mod download 2>&1 | tee /tmp/go-download.log || warn "依赖下载告警，尝试继续构建..."

    local success=false
    for i in 1 2; do
        if go build -v -o cboard ./cmd/server/main.go 2>&1 | tee /tmp/go-build.log; then
            success=true; break
        else
            warn "后端构建失败 (尝试 $i/2)"
            [ $i -lt 2 ] && { go clean -cache; sleep 2; }
        fi
    done

    if [ "$success" = false ]; then
        tail -30 /tmp/go-build.log 2>/dev/null
        fatal "后端构建失败，请检查 Go 环境和源码"
    fi
    chmod +x cboard
    ok "后端构建完成"
}

build_frontend() {
    info "构建前端..."
    cd "$INSTALL_DIR/frontend"

    local npm_success=false
    for i in 1 2 3; do
        if npm install --silent --legacy-peer-deps 2>/dev/null; then npm_success=true; break; fi
        warn "前端依赖安装失败 (尝试 $i/3)"
        [ $i -lt 3 ] && { rm -rf node_modules package-lock.json; sleep 3; }
    done
    [ "$npm_success" = false ] && fatal "前端依赖安装失败"

    export NODE_OPTIONS="--max-old-space-size=4096"
    npx vite build 2>&1 || fatal "前端构建失败"
    [ ! -f "dist/index.html" ] && fatal "前端构建失败: dist/index.html 不存在"
    ok "前端构建完成"
}

# ---- Nginx 与系统服务 ----
create_service() {
    info "创建 systemd 服务..."
    cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=CBoard v2 Server
After=network.target redis-server.service redis.service

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

# 整合并优化了 Nginx 配置写入逻辑 (消除冗余)
generate_nginx_conf() {
    local conf_file="/etc/nginx/conf.d/cboard.conf"
    local dist_dir="$INSTALL_DIR/frontend/dist"
    local use_ssl=$1

    cat > "$conf_file" <<EOF
# CBoard Nginx Configuration
EOF

    if [ "$use_ssl" = true ]; then
        cat >> "$conf_file" <<EOF
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
EOF
    else
        cat >> "$conf_file" <<EOF
server {
    listen 80;
    server_name ${DOMAIN:-_};
EOF
    fi

    # 公共 Location 及 Gzip 配置
    cat >> "$conf_file" <<EOF
    root $dist_dir;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
    gzip_min_length 1024;

    location /.well-known/acme-challenge/ {
        root $INSTALL_DIR;
        allow all;
    }

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

    # 清理默认配置并测试
    rm -f /etc/nginx/sites-enabled/default /etc/nginx/conf.d/default.conf 2>/dev/null || true
    if nginx -t >/dev/null 2>&1; then
        systemctl reload nginx 2>/dev/null || systemctl restart nginx
        return 0
    else
        err "Nginx 配置异常:"
        nginx -t 2>&1 | tail -5
        return 1
    fi
}

setup_nginx() {
    info "配置 Nginx 反向代理..."
    generate_nginx_conf false && ok "Nginx 基础配置完成"
}

setup_ssl() {
    [[ ! "$ENABLE_SSL" =~ ^[Yy]$ ]] || [ -z "$DOMAIN" ] && return 0

    if [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
        ok "检测到已有 SSL 证书，正在应用..."
        generate_nginx_conf true && ok "HTTPS 配置完成"
        return 0
    fi

    info "申请 Let's Encrypt SSL 证书..."
    local NGINX_WAS_RUNNING=false
    is_service_running nginx && NGINX_WAS_RUNNING=true

    if certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" --redirect 2>/dev/null; then
        ok "SSL 证书申请成功 (Nginx Plugin)"
        return 0
    fi

    warn "Nginx 方式失败，尝试 Standalone 方式..."
    [ "$NGINX_WAS_RUNNING" = true ] && systemctl stop nginx 2>/dev/null

    if certbot certonly --standalone -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" 2>/dev/null; then
        ok "SSL 证书申请成功 (Standalone)"
        generate_nginx_conf true && ok "HTTPS 配置完成"
    else
        warn "SSL 证书申请失败，请检查 80 端口和 DNS 解析。稍后可从菜单重新配置。"
    fi

    [ "$NGINX_WAS_RUNNING" = true ] && systemctl start nginx 2>/dev/null
}

setup_firewall() {
    if check_cmd ufw; then
        ufw allow 80/tcp >/dev/null 2>&1
        ufw allow 443/tcp >/dev/null 2>&1
        ok "UFW 防火墙已放行 80/443"
    elif check_cmd firewall-cmd; then
        firewall-cmd --permanent --add-service=http >/dev/null 2>&1
        firewall-cmd --permanent --add-service=https >/dev/null 2>&1
        firewall-cmd --reload >/dev/null 2>&1
        ok "Firewalld 防火墙已放行 80/443"
    fi
}

# ============================================================================
# 核心安装流程
# ============================================================================
install_system() {
    clear
    echo -e "${BLUE}========== 开始安装 CBoard v2 ==========${NC}\n"
    
    check_root
    detect_os
    check_disk_space || fatal "磁盘空间不足，请清理后重试"
    ensure_swap

    # 交互或无人值守
    if [ -n "$CBOARD_UNATTENDED" ] && [ "$CBOARD_UNATTENDED" != "0" ]; then
        INSTALL_DIR=${CBOARD_INSTALL_DIR:-/opt/cboard}
        CBOARD_PORT=${CBOARD_PORT:-9000}; BACKEND_PORT=$CBOARD_PORT
        DOMAIN=${CBOARD_DOMAIN:-}
        ENABLE_SSL=${CBOARD_SSL:-n}
        ADMIN_EMAIL=${CBOARD_ADMIN_EMAIL:-admin@example.com}
        ADMIN_PASSWORD=${CBOARD_ADMIN_PASSWORD:-$(openssl rand -base64 12 | tr -dc 'a-zA-Z0-9' | head -c 12)}
        ok "无人值守模式：目录=$INSTALL_DIR 端口=$CBOARD_PORT"
    else
        interactive_config
    fi

    echo -e "\n${CYAN}>>> 正在执行环境初始化与依赖安装...${NC}"
    if is_service_running; then
        info "停止已有服务以避免冲突..."
        systemctl stop ${SERVICE_NAME} 2>/dev/null
    fi

    # 阶段执行
    install_system_tools
    install_redis
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

    info "启动 CBoard 服务..."
    safe_release_port "$CBOARD_PORT"
    systemctl start ${SERVICE_NAME}
    sleep 3

    if is_service_running; then
        ok "服务启动成功"
        if [ -n "$ADMIN_EMAIL" ] && [ -n "$ADMIN_PASSWORD" ] && [ -f "$INSTALL_DIR/cboard" ]; then
            (cd "$INSTALL_DIR" && ./cboard reset-password --email "$ADMIN_EMAIL" --password "$ADMIN_PASSWORD" >/dev/null 2>&1) && ok "初始管理员账号配置成功" || warn "密码同步失败，请于面板菜单手动重置"
        fi
    else
        err "服务启动异常，可通过 journalctl -u ${SERVICE_NAME} 查看日志"
    fi

    print_install_result
    pause
}

print_install_result() {
    local BASE_URL="http://$(curl -s4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')"
    [ -n "$DOMAIN" ] && { [[ "$ENABLE_SSL" =~ ^[Yy]$ ]] && BASE_URL="https://$DOMAIN" || BASE_URL="http://$DOMAIN"; }

    echo -e "\n${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           CBoard v2 安装完成!            ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}\n"
    echo -e "  访问地址:    ${CYAN}$BASE_URL${NC} (直接打开，无需加端口)"
    echo -e "  安装目录:    ${CYAN}$INSTALL_DIR${NC}"
    echo -e "  管理员邮箱:  ${YELLOW}$ADMIN_EMAIL${NC}"
    echo -e "  管理员密码:  ${YELLOW}${ADMIN_PASSWORD:0:2}******${NC} ${RED}(请登录后立即修改)${NC}\n"
}

# ============================================================================
# 公共辅助集
# ============================================================================
get_work_dir() { [ -d "${1:-$INSTALL_DIR}" ] && echo "${1:-$INSTALL_DIR}" || echo "$PROJECT_PATH"; }
ensure_in_project_dir() { cd "$(get_work_dir "$1")" 2>/dev/null || { err "无法进入项目目录"; return 1; }; }
read_env() { grep "^${1}=" "$(get_work_dir "$2")/.env" 2>/dev/null | cut -d'=' -f2- | tr -d '"' | xargs; }
wait_for_service() {
    local count=0; while ! is_service_running && [ $count -lt 15 ]; do sleep 1; count=$((count + 1)); done
    is_service_running
}
ensure_redis_in_env() {
    local w_dir="${1:-$(get_work_dir)}"
    if [ -f "$w_dir/.env" ] && ! grep -q '^REDIS_ADDR=' "$w_dir/.env"; then
        echo -e "\nREDIS_ADDR=127.0.0.1:6379" >> "$w_dir/.env"
    fi
}

# ============================================================================
# 菜单功能模块
# ============================================================================
configure_domain() {
    echo -e "\n${BLUE}========== 域名与 SSL 配置 ==========${NC}\n"
    ensure_in_project_dir || return 1
    
    while true; do
        read -rp "请输入新域名 (例: example.com): " DOMAIN
        DOMAIN=$(echo "$DOMAIN" | sed -e 's|^https\?://||' -e 's|/$||')
        validate_domain "$DOMAIN" && break || err "域名格式错误"
    done

    read -rp "是否使用 HTTPS/SSL? (y/n) [y]: " USE_HTTPS
    local BASE_URL SSL_ENABLED
    if [[ "${USE_HTTPS:-y}" =~ ^[Yy]$ ]]; then
        BASE_URL="https://$DOMAIN"; SSL_ENABLED="true"; ENABLE_SSL="y"
    else
        BASE_URL="http://$DOMAIN"; SSL_ENABLED="false"; ENABLE_SSL="n"
    fi

    info "更新环境配置..."
    sed -i -e "s|^DOMAIN=.*|DOMAIN=$DOMAIN|" \
           -e "s|^BASE_URL=.*|BASE_URL=$BASE_URL|" \
           -e "s|^CORS_ORIGINS=.*|CORS_ORIGINS=$BASE_URL|" \
           -e "s|^SSL_ENABLED=.*|SSL_ENABLED=$SSL_ENABLED|" \
           -e "s|^SUBSCRIPTION_URL_PREFIX=.*|SUBSCRIPTION_URL_PREFIX=$BASE_URL/sub|" .env 2>/dev/null || warn ".env 更新异常"
    
    [[ "$ENABLE_SSL" == "y" ]] && { read -rp "请输入证书关联邮箱: " ADMIN_EMAIL; setup_ssl; } || setup_nginx
    
    systemctl restart ${SERVICE_NAME} 2>/dev/null
    ok "域名重配置完成，最新地址: $BASE_URL"
}

manage_service() {
    local action=$1
    echo -e "\n${BLUE}========== 服务管理: $action ==========${NC}\n"
    case $action in
        "启动") systemctl start ${SERVICE_NAME}; wait_for_service && ok "服务已启动" || err "启动失败" ;;
        "停止") systemctl stop ${SERVICE_NAME}; ! is_service_running && ok "服务已停止" || err "停止失败" ;;
        "重启") systemctl restart ${SERVICE_NAME}; wait_for_service && ok "服务已重启" || err "重启失败" ;;
    esac
}

check_service_status() {
    echo -e "\n${BLUE}========== 系统运行状态 ==========${NC}\n"
    local w_dir=$(get_work_dir)
    echo -e "   后端核心: $(is_service_running && echo -e "${GREEN}运行中${NC}" || echo -e "${RED}已停止${NC}")"
    echo -e "   网络反代: $(is_service_running nginx && echo -e "${GREEN}运行中${NC}" || echo -e "${RED}已停止${NC}")"
    echo -e "   配置域名: ${CYAN}$(read_env DOMAIN "$w_dir" || echo "未配置")${NC}"
    echo -e "   安全策略: SSL=$(read_env SSL_ENABLED "$w_dir")"
    echo -e "\n${YELLOW}▶ 最近 5 条服务日志:${NC}"
    journalctl -u ${SERVICE_NAME} -n 5 --no-pager --no-hostname 2>/dev/null || echo "无日志"
}

view_service_logs() {
    echo -e "\n${BLUE}========== 日志查阅面板 ==========${NC}\n"
    echo -e "  1. 实时跟随日志 (Ctrl+C 退出)\n  2. 最近 100 行记录\n  3. 过滤系统错误日志\n  0. 返回"
    read -rp "请选择: " lc
    case $lc in
        1) journalctl -u ${SERVICE_NAME} -f ;;
        2) journalctl -u ${SERVICE_NAME} -n 100 --no-pager ;;
        3) journalctl -u ${SERVICE_NAME} -p err -n 50 --no-pager ;;
    esac
}

reset_admin_password() {
    echo -e "\n${BLUE}========== 重置管理员密码 ==========${NC}\n"
    ensure_in_project_dir || return 1
    [ ! -f cboard.db ] && { err "数据库文件不存在，请确保服务已正常初始化"; return 1; }
    
    read -rp "指定管理员邮箱 (留空使用环境默认): " new_email
    new_email=${new_email:-$(read_env ADMIN_EMAIL)}
    
    while true; do
        read -rsp "输入新密码 (至少8位): " new_pwd; echo
        [ ${#new_pwd} -ge 8 ] && break || err "密码过短"
    done

    if ./cboard reset-password --email "$new_email" --password "$new_pwd" 2>/dev/null; then
        ok "重置成功"
    else
        err "重置失败，检查邮箱是否合法或查阅日志"
    fi
}

reinstall_website() {
    echo -e "\n${BLUE}========== 保留数据重装 ==========${NC}\n"
    ensure_in_project_dir || return 1
    warn "此操作将重新编译代码，但保留数据库与 .env 配置！"
    read -rp "确认执行? (y/n) [n]: " cfm
    [[ ! "$cfm" =~ ^[Yy]$ ]] && return 0

    local b_dir="backup_$(date +%Y%m%d%H%M)"
    mkdir -p "$b_dir"; cp cboard.db .env "$b_dir/" 2>/dev/null
    
    systemctl stop ${SERVICE_NAME} 2>/dev/null
    build_backend && build_frontend
    systemctl start ${SERVICE_NAME}
    wait_for_service && ok "无损重装完毕！" || err "重装后启动失败"
}

update_code() {
    echo -e "\n${BLUE}========== Git 代码更新 ==========${NC}\n"
    ensure_in_project_dir || return 1
    [ ! -d ".git" ] && { err "非 Git 仓库模式，无法热更新"; return 1; }
    
    git config --global --add safe.directory "$(pwd)" 2>/dev/null
    git stash push -m "Auto stash before update" 2>/dev/null
    if git pull origin "$(git branch --show-current 2>/dev/null)"; then
        ok "代码拉取成功，开始重构建..."
        systemctl stop ${SERVICE_NAME} 2>/dev/null
        build_backend && build_frontend
        systemctl start ${SERVICE_NAME}
        wait_for_service && ok "热更新部署成功！" || err "部署出现异常"
    else
        err "网络或冲突导致拉取失败"
    fi
}

manage_redis_menu() {
    echo -e "\n${BLUE}========== Redis 缓存管理 ==========${NC}\n"
    is_service_running redis-server && ok "状态: 运行中" || warn "状态: 未运行/未安装"
    echo -e "  1. 一键安装并启动\n  2. 清理全部缓存 (FlushDB)\n  0. 返回"
    read -rp "选择: " rc
    case $rc in
        1) detect_os; install_redis; ensure_redis_in_env; systemctl restart ${SERVICE_NAME} ;;
        2) redis-cli FLUSHDB 2>/dev/null && ok "缓存已释放" || err "清理失败，Redis 可能离线" ;;
    esac
}

uninstall_cboard() {
    echo -e "\n${RED}========== 灾难性操作: 卸载系统 ==========${NC}\n"
    read -rp "输入大写 YES 确认卸载: " cfm
    [ "$cfm" != "YES" ] && return 0

    info "清理系统服务..."
    systemctl stop ${SERVICE_NAME} nginx 2>/dev/null
    systemctl disable ${SERVICE_NAME} 2>/dev/null
    rm -f /etc/systemd/system/${SERVICE_NAME}.service /etc/nginx/conf.d/cboard.conf
    systemctl daemon-reload
    
    read -rp "是否彻底删除程序数据目录 $(get_work_dir) ? (y/n): " ddir
    [[ "$ddir" =~ ^[Yy]$ ]] && rm -rf "$(get_work_dir)" && ok "已删库跑路" || ok "配置与数据已保留"
}

# ============================================================================
# 界面与主循环
# ============================================================================
show_menu() {
    clear
    echo -e "${MAGENTA}================================================================${NC}"
    echo -e "${CYAN}   ██████╗██████╗  ██████╗  █████╗ ██████╗ ██████╗   ${NC}"
    echo -e "${CYAN}  ██╔════╝██╔══██╗██╔═══██╗██╔══██╗██╔══██╗██╔══██╗  ${NC}"
    echo -e "${CYAN}  ██║     ██████╔╝██║   ██║███████║██████╔╝██║  ██║  ${NC}"
    echo -e "${CYAN}  ██║     ██╔══██╗██║   ██║██╔══██║██╔══██╗██║  ██║  ${NC}"
    echo -e "${CYAN}  ╚██████╗██████╔╝╚██████╔╝██║  ██║██║  ██║██████╔╝  ${NC}"
    echo -e "${MAGENTA}================================================================${NC}"
    echo -e "  ${YELLOW}CBoard v2 管理面板 (纯净版)${NC} | 脚本版本: ${GREEN}v$SCRIPT_VERSION${NC}\n"
    
    echo -e "  ${GREEN}[ 系统核心 ]${NC}"
    echo -e "   1. 🚀 全新安装系统        12. 🔄 强力重装 (保留数据)"
    echo -e "   2. 🌐 配置域名与 SSL      14. ⬇️  热更新代码 (Git Pull)"
    echo -e "   3. 🩺 环境健康诊断"
    
    echo -e "\n  ${GREEN}[ 进程管控 ]${NC}"
    echo -e "   4. 🟢 启动服务            7.  📊 查看运行状态"
    echo -e "   5. 🔴 停止服务            8.  📄 查阅实时日志"
    echo -e "   6. 🔄 重启服务            15. 🗄️  Redis 缓存管家"

    echo -e "\n  ${GREEN}[ 安全数据 ]${NC}"
    echo -e "   9. 🔑 重置管理员密码      11. 📦 备份本地数据库"
    echo -e "  10. 👤 查看当前管理员      18. 💣 卸载系统"

    echo -e "\n  ${GREEN} 0. 退出管理面板${NC}"
    echo -e "${MAGENTA}================================================================${NC}"
}

main() {
    check_concurrent

    if [ -n "$CBOARD_UNATTENDED" ] && [ "$CBOARD_UNATTENDED" != "0" ]; then
        install_system; exit 0
    fi

    # 新环境检测：直接进入安装
    local wdir=$(get_work_dir)
    if [ ! -f "$wdir/.env" ] && [ ! -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
        install_system
    fi

    while true; do
        show_menu
        read -rp "请输入指令序号 [0-18]: " choice
        
        # 捕获菜单内部错误避免退出循环
        set +e
        case $choice in
            1)  install_system ;;
            2)  configure_domain ;;
            3)  bash "$0" --health-check || echo "此项整合在各项排错中，暂留位" && pause ;;
            4)  manage_service "启动" ; pause ;;
            5)  manage_service "停止" ; pause ;;
            6)  manage_service "重启" ; pause ;;
            7)  check_service_status ; pause ;;
            8)  view_service_logs ; pause ;;
            9)  reset_admin_password ; pause ;;
            10) echo -e "\n${CYAN}管理员邮箱:${NC} $(read_env ADMIN_EMAIL)\n"; pause ;;
            11) cp -v "$wdir/cboard.db" "/tmp/cboard_bak_$(date +%s).db" && ok "已备份到 /tmp 目录" ; pause ;;
            12) reinstall_website ; pause ;;
            14) update_code ; pause ;;
            15) manage_redis_menu ; pause ;;
            18) uninstall_cboard ; pause ;;
            0)  echo -e "\n${GREEN}感谢使用，再见！${NC}\n"; exit 0 ;;
            *)  err "无效指令，请重新输入" ; sleep 1 ;;
        esac
        set -e
    done
}

main "$@"