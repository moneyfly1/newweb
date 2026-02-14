#!/usr/bin/env bash
# ============================================================================
# CBoard v2 一键安装 & 管理脚本（宝塔面板版）
# 适用于已安装宝塔面板的 Linux 服务器
# 用法: bash install_bt.sh
# ============================================================================
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
INSTALL_DIR="/www/wwwroot/cboard"
SERVICE_NAME="cboard"
CBOARD_PORT=9000
BACKEND_PORT=9000
DOMAIN=""
ENABLE_SSL="n"
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
LOCK_FILE="/tmp/cboard_bt_install.lock"

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
    [[ $EUID -ne 0 ]] && fatal "请使用 root 用户运行此脚本"
}

# ---- 宝塔面板检测 ----
check_bt() {
    if [ ! -f /www/server/panel/BT-Panel ] && [ ! -d /www/server/panel ]; then
        fatal "未检测到宝塔面板，请先安装宝塔面板或使用 install.sh (无宝塔版)"
    fi
    ok "检测到宝塔面板"
}

# ---- 宝塔 Nginx 检测 ----
check_bt_nginx() {
    if [ ! -f /www/server/nginx/sbin/nginx ]; then
        warn "未检测到宝塔 Nginx"
        echo -e "${YELLOW}请在宝塔面板中安装 Nginx:${NC}"
        echo "  宝塔面板 -> 软件商店 -> Nginx -> 安装"
        fatal "请安装 Nginx 后重新运行此脚本"
    fi
    ok "检测到宝塔 Nginx"
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

# ---- 安装 Go ----
install_go() {
    if command -v go &>/dev/null; then
        ok "Go 已安装: $(go version | awk '{print $3}')"
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
    # 检查宝塔 PM2 管理器自带 Node
    if ls /www/server/nvm/versions/node/*/bin/node 2>/dev/null | head -1 | grep -q node; then
        local NODE_PATH
        NODE_PATH=$(ls -d /www/server/nvm/versions/node/*/bin 2>/dev/null | tail -1)
        export PATH=$NODE_PATH:$PATH
        ok "使用宝塔 Node.js: $(node -v)"
        return
    fi
    info "安装 Node.js 20.x..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - >/dev/null 2>&1 || {
        curl -fsSL https://rpm.nodesource.com/setup_20.x | bash - >/dev/null 2>&1
    }
    if command -v apt-get &>/dev/null; then
        apt-get install -y -qq nodejs >/dev/null 2>&1
    else
        yum install -y nodejs >/dev/null 2>&1
    fi
    ok "Node.js $(node -v) 安装完成"
}

# ---- 收集管理员信息 ----
get_admin_info() {
    echo ""
    echo -e "${CYAN}========== 管理员信息配置 ==========${NC}"
    echo ""

    while true; do
        read -rp "管理员邮箱: " ADMIN_EMAIL
        if [ -z "$ADMIN_EMAIL" ]; then
            err "邮箱不能为空"
        elif validate_email "$ADMIN_EMAIL"; then
            break
        else
            err "邮箱格式不正确"
        fi
    done

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
                err "两次密码不一致"
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
    echo -e "${CYAN}║     CBoard v2 安装配置 (宝塔面板版)     ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
    echo ""

    read -rp "安装目录 [$INSTALL_DIR]: " input
    INSTALL_DIR=${input:-$INSTALL_DIR}

    read -rp "后端端口 [$CBOARD_PORT]: " input
    CBOARD_PORT=${input:-$CBOARD_PORT}
    BACKEND_PORT=$CBOARD_PORT

    while true; do
        read -rp "绑定域名 (留空则使用 IP): " DOMAIN
        if [ -z "$DOMAIN" ]; then
            break
        fi
        DOMAIN=$(echo "$DOMAIN" | sed 's|^https\?://||' | sed 's|/$||')
        if validate_domain "$DOMAIN"; then
            break
        else
            err "域名格式不正确 (例如: example.com)"
        fi
    done

    if [ -n "$DOMAIN" ]; then
        read -rp "是否自动申请 SSL 证书? (y/n) [y]: " ENABLE_SSL
        ENABLE_SSL=${ENABLE_SSL:-y}
    fi

    get_admin_info

    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    info "安装目录: $INSTALL_DIR"
    info "后端端口: $CBOARD_PORT"
    info "绑定域名: ${DOMAIN:-无(IP 访问)}"
    info "SSL 证书: ${ENABLE_SSL}"
    info "管理员邮箱: $ADMIN_EMAIL"
    info "管理员密码: ${ADMIN_PASSWORD:0:2}******"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}提示: SSL 证书也可在宝塔面板中配置${NC}"
    echo ""
    read -rp "确认以上配置? (y/n) [y]: " confirm
    [[ "${confirm:-y}" != "y" ]] && fatal "安装已取消"
}

# ---- 生成 .env 配置文件 ----
create_env_file() {
    local target_dir="${1:-$INSTALL_DIR}"
    cd "$target_dir" || return 1

    if [ -f .env ]; then
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
# CBoard v2 配置文件 (宝塔版)
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

# ---- 基础配置 ----
DEBUG=false
PORT=$CBOARD_PORT
HOST=127.0.0.1
SECRET_KEY=$SECRET_KEY
BASE_URL=$BASE_URL

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
            cp "$PROJECT_PATH"/install_bt.sh "$INSTALL_DIR/" 2>/dev/null || true
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

    local success=false
    for i in 1 2; do
        if go build -o cboard cmd/server/main.go 2>&1; then
            success=true
            break
        else
            warn "后端构建失败 (尝试 $i/2)"
            [ $i -lt 2 ] && go clean -cache 2>/dev/null || true
        fi
    done
    [ "$success" = false ] && fatal "后端构建失败"

    chmod +x cboard
    ok "后端构建完成"
}

# ---- 构建前端 ----
build_frontend() {
    info "构建前端..."
    cd "$INSTALL_DIR/frontend"

    local npm_success=false
    for i in 1 2 3; do
        if npm install --silent 2>/dev/null; then
            npm_success=true
            break
        else
            warn "前端依赖安装失败 (尝试 $i/3)"
            [ $i -lt 3 ] && rm -rf node_modules package-lock.json && sleep 3
        fi
    done
    [ "$npm_success" = false ] && fatal "前端依赖安装失败"

    export NODE_OPTIONS="--max-old-space-size=4096"
    npx vite build 2>&1 || fatal "前端构建失败"
    [ ! -f "dist/index.html" ] && fatal "前端构建失败: dist/index.html 不存在"

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
User=www
Group=www
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/cboard
Restart=always
RestartSec=5
LimitNOFILE=65536
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

    chown -R www:www "$INSTALL_DIR"
    systemctl daemon-reload
    systemctl enable ${SERVICE_NAME} >/dev/null 2>&1
    ok "systemd 服务已创建"
}

# ---- 配置宝塔 Nginx ----
setup_bt_nginx() {
    info "生成 Nginx 反向代理配置..."

    local FRONTEND_DIR="$INSTALL_DIR/frontend/dist"
    local NGINX_CONF_DIR="/www/server/panel/vhost/nginx"
    mkdir -p "$NGINX_CONF_DIR"

    local CONF_FILE="$NGINX_CONF_DIR/cboard.conf"

    cat > "$CONF_FILE" <<EOF
server {
    listen 80;
    server_name ${DOMAIN:-_};

    root $FRONTEND_DIR;
    index index.html;

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
    gzip_min_length 1024;

    # 日志 (宝塔标准路径)
    access_log /www/wwwlogs/cboard.log;
    error_log /www/wwwlogs/cboard.error.log;

    # Let's Encrypt 验证
    location /.well-known/acme-challenge/ {
        root $INSTALL_DIR;
        allow all;
        access_log off;
        log_not_found off;
    }

    location /.well-known/ {
        root $INSTALL_DIR;
        allow all;
        access_log off;
        log_not_found off;
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

    # 支付回调
    location /api/v1/payment/notify/ {
        proxy_pass http://127.0.0.1:$CBOARD_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_request_buffering off;
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

    # 复制到 Nginx vhost 目录
    mkdir -p /www/server/nginx/conf/vhost 2>/dev/null || true
    cp "$CONF_FILE" /www/server/nginx/conf/vhost/cboard.conf 2>/dev/null || true

    # 测试并重载
    if /www/server/nginx/sbin/nginx -t 2>/dev/null; then
        /www/server/nginx/sbin/nginx -s reload 2>/dev/null || true
        ok "Nginx 配置完成"
    else
        warn "Nginx 配置测试失败"
        echo ""
        echo -e "${YELLOW}手动配置步骤:${NC}"
        echo "  1. 宝塔面板 -> 网站 -> 添加站点"
        echo "  2. 域名: ${DOMAIN:-你的域名}"
        echo "  3. 根目录: $FRONTEND_DIR"
        echo "  4. 站点设置 -> 反向代理:"
        echo "     目标URL: http://127.0.0.1:$CBOARD_PORT"
        echo "     代理目录: /api"
        echo ""
    fi
}

# ---- 申请 SSL 证书 ----
setup_ssl() {
    if [[ ! "$ENABLE_SSL" =~ ^[Yy]$ ]] || [ -z "$DOMAIN" ]; then
        return 0
    fi

    # 检查已有证书
    if [ -f "/www/server/panel/vhost/cert/$DOMAIN/fullchain.pem" ] || \
       [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
        ok "检测到已有 SSL 证书"
        return 0
    fi

    info "申请 Let's Encrypt SSL 证书..."

    # 安装 certbot
    if ! command -v certbot &>/dev/null; then
        if command -v apt-get &>/dev/null; then
            apt-get install -y -qq certbot >/dev/null 2>&1
        else
            yum install -y certbot >/dev/null 2>&1
        fi
    fi

    # 检查 80 端口
    local NGINX_WAS_RUNNING=false
    local PORT_FREE=false

    if ! lsof -ti :80 &>/dev/null; then
        PORT_FREE=true
    else
        info "80 端口被占用，尝试临时停止 Nginx..."
        if systemctl is-active --quiet nginx 2>/dev/null; then
            NGINX_WAS_RUNNING=true
            systemctl stop nginx 2>/dev/null || /www/server/nginx/sbin/nginx -s stop 2>/dev/null || true
            sleep 2
            if ! lsof -ti :80 &>/dev/null; then
                PORT_FREE=true
                ok "80 端口已释放"
            fi
        fi
    fi

    if [ "$PORT_FREE" = false ]; then
        warn "无法释放 80 端口"
        echo -e "${YELLOW}请手动检查: lsof -i :80${NC}"
        if [ "$NGINX_WAS_RUNNING" = true ]; then
            systemctl start nginx 2>/dev/null || /www/server/nginx/sbin/nginx 2>/dev/null || true
        fi
        echo ""
        echo -e "${YELLOW}建议在宝塔面板中申请 SSL 证书:${NC}"
        echo "  网站 -> $DOMAIN -> SSL -> Let's Encrypt"
        return 0
    fi

    # 申请证书
    local CERT_SUCCESS=false
    if certbot certonly --standalone -d "$DOMAIN" --register-unsafely-without-email --agree-tos --non-interactive 2>&1 | tee /tmp/certbot_install.log; then
        if [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
            CERT_SUCCESS=true
            ok "SSL 证书申请成功"

            # 复制到宝塔标准路径
            mkdir -p "/www/server/panel/vhost/cert/$DOMAIN"
            cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "/www/server/panel/vhost/cert/$DOMAIN/fullchain.pem" 2>/dev/null || true
            cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" "/www/server/panel/vhost/cert/$DOMAIN/privkey.pem" 2>/dev/null || true
            ok "证书已复制到宝塔标准路径"

            # 更新 Nginx 配置加入 SSL
            update_nginx_ssl
        fi
    fi

    # 恢复 Nginx
    if [ "$NGINX_WAS_RUNNING" = true ]; then
        systemctl start nginx 2>/dev/null || /www/server/nginx/sbin/nginx 2>/dev/null || true
    fi

    if [ "$CERT_SUCCESS" = false ]; then
        warn "SSL 证书申请失败"
        echo ""
        echo -e "${YELLOW}可能的原因:${NC}"
        echo "  1. 域名 DNS 未正确解析到此服务器"
        echo "  2. 80 端口未开放"
        echo "  3. 防火墙阻止了访问"
        echo "  4. Let's Encrypt 频率限制"
        echo ""
        echo -e "${YELLOW}建议在宝塔面板中申请:${NC}"
        echo "  网站 -> $DOMAIN -> SSL -> Let's Encrypt"
        echo ""
        read -rp "是否继续安装? (y/n) [y]: " cont
        [[ "${cont:-y}" != "y" ]] && fatal "安装已取消"
    fi
}

# ---- 更新 Nginx SSL 配置 ----
update_nginx_ssl() {
    local FRONTEND_DIR="$INSTALL_DIR/frontend/dist"
    local CERT_PATH="/www/server/panel/vhost/cert/$DOMAIN"
    local CONF_FILE="/www/server/panel/vhost/nginx/cboard.conf"

    [ ! -f "$CERT_PATH/fullchain.pem" ] && return

    cat > "$CONF_FILE" <<EOF
server {
    listen 80;
    server_name $DOMAIN;
    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;

    ssl_certificate $CERT_PATH/fullchain.pem;
    ssl_certificate_key $CERT_PATH/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE:ECDH:AES:HIGH:!NULL:!aNULL:!MD5:!ADH:!RC4;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    root $FRONTEND_DIR;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
    gzip_min_length 1024;

    access_log /www/wwwlogs/cboard.log;
    error_log /www/wwwlogs/cboard.error.log;

    location /.well-known/acme-challenge/ {
        root $INSTALL_DIR;
        allow all;
    }

    location /.well-known/ {
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

    location /api/v1/payment/notify/ {
        proxy_pass http://127.0.0.1:$CBOARD_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_request_buffering off;
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

    cp "$CONF_FILE" /www/server/nginx/conf/vhost/cboard.conf 2>/dev/null || true

    if /www/server/nginx/sbin/nginx -t 2>/dev/null; then
        /www/server/nginx/sbin/nginx -s reload 2>/dev/null || true
        ok "Nginx SSL 配置已更新"
    fi
}

# ============================================================================
# 安装主流程
# ============================================================================
install_system() {
    echo -e "${BLUE}========== 开始安装 CBoard v2 (宝塔版) ==========${NC}"
    echo ""

    check_root
    check_bt
    check_bt_nginx

    if ! check_disk_space; then
        fatal "磁盘空间不足"
    fi

    interactive_config

    echo ""
    info "开始安装..."
    echo ""

    install_go
    install_node
    deploy_project
    build_backend
    build_frontend
    create_service
    setup_bt_nginx
    setup_ssl

    # 启动服务
    info "启动服务..."
    safe_release_port "$CBOARD_PORT"
    systemctl start ${SERVICE_NAME}
    sleep 3

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        ok "服务启动成功"
    else
        err "服务启动失败"
        journalctl -u ${SERVICE_NAME} -n 20 --no-pager 2>/dev/null | tail -10
    fi

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
    echo -e "${GREEN}║   CBoard v2 安装完成! (宝塔面板版)      ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  访问地址:    ${CYAN}$BASE_URL${NC}"
    echo -e "  API 端口:    ${CYAN}$CBOARD_PORT${NC}"
    echo -e "  安装目录:    ${CYAN}$INSTALL_DIR${NC}"
    echo ""
    echo -e "  管理员邮箱:  ${YELLOW}$ADMIN_EMAIL${NC}"
    echo -e "  管理员密码:  ${YELLOW}${ADMIN_PASSWORD:0:2}******${NC}"
    echo -e "  ${RED}请登录后立即修改默认密码!${NC}"
    echo ""
    echo -e "  ${YELLOW}宝塔面板后续操作:${NC}"
    echo "    1. 在宝塔面板 -> 网站 中查看站点"
    echo "    2. 如需 SSL，在站点设置 -> SSL 中申请证书"
    echo "    3. 如需修改配置，编辑 $INSTALL_DIR/.env 后重启"
    echo ""

    # 服务状态
    echo -e "${BLUE}服务状态:${NC}"
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        echo -e "  后端服务: ${GREEN}运行中${NC}"
    else
        echo -e "  后端服务: ${RED}已停止${NC}"
    fi
    if /www/server/nginx/sbin/nginx -t 2>/dev/null; then
        echo -e "  Nginx:    ${GREEN}运行中${NC}"
    else
        echo -e "  Nginx:    ${RED}配置异常${NC}"
    fi
    echo ""
}

# ============================================================================
# 菜单选项 2: 配置域名
# ============================================================================
configure_domain() {
    echo -e "${BLUE}========== 配置域名 (宝塔版) ==========${NC}"
    echo ""

    set +e
    check_root

    cd "$INSTALL_DIR" 2>/dev/null || cd "$PROJECT_PATH" || {
        err "无法进入项目目录"
        read -rp "按回车键继续..."
        set -e
        return 1
    }

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
        for key in DOMAIN BASE_URL SSL_ENABLED SUBSCRIPTION_URL_PREFIX ALIPAY_NOTIFY_URL ALIPAY_RETURN_URL; do
            case $key in
                DOMAIN) val="$DOMAIN" ;;
                BASE_URL) val="$BASE_URL" ;;
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
    fi

    # 更新 Nginx
    setup_bt_nginx

    # SSL
    if [[ "$USE_HTTPS" =~ ^[Yy]$ ]]; then
        read -rp "是否现在申请 SSL 证书? (y/n) [y]: " apply_ssl
        if [[ "${apply_ssl:-y}" =~ ^[Yy]$ ]]; then
            setup_ssl
        else
            echo ""
            echo -e "${YELLOW}请在宝塔面板中申请 SSL 证书:${NC}"
            echo "  网站 -> $DOMAIN -> SSL -> Let's Encrypt"
        fi
    fi

    # 重启
    /www/server/nginx/sbin/nginx -s reload 2>/dev/null || true
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
    echo -e "${BLUE}========== 修复常见错误 (宝塔版) ==========${NC}"
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
        install_go
    fi
    echo ""

    echo -e "${YELLOW}3. 检查 Node.js 环境...${NC}"
    if command -v node &>/dev/null; then
        ok "Node.js: $(node -v)"
    else
        warn "Node.js 未安装，正在安装..."
        install_node
    fi
    echo ""

    echo -e "${YELLOW}4. 检查 .env 配置...${NC}"
    if [ -f .env ]; then
        ok ".env 文件存在"
    else
        warn ".env 文件不存在"
    fi
    echo ""

    echo -e "${YELLOW}5. 检查数据库文件...${NC}"
    if [ -f cboard.db ]; then
        ok "数据库文件存在 ($(du -h cboard.db | awk '{print $1}'))"
        # 修复权限
        chown www:www cboard.db 2>/dev/null || true
        chmod 664 cboard.db 2>/dev/null || true
    else
        warn "数据库文件不存在"
    fi
    echo ""

    echo -e "${YELLOW}6. 检查前端构建...${NC}"
    if [ -f "frontend/dist/index.html" ]; then
        ok "前端构建产物存在"
    else
        warn "前端构建产物不存在"
        read -rp "是否重新构建? (y/n) [y]: " rebuild
        if [[ "${rebuild:-y}" =~ ^[Yy]$ ]]; then
            build_frontend
        fi
    fi
    echo ""

    echo -e "${YELLOW}7. 修复文件权限...${NC}"
    chown -R www:www "$INSTALL_DIR" 2>/dev/null || chown -R www:www "$PROJECT_PATH" 2>/dev/null || true
    ok "文件权限已修复"
    echo ""

    echo -e "${YELLOW}8. 检查 systemd 服务...${NC}"
    if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
        systemctl daemon-reload
        ok "服务配置已重新加载"
    else
        warn "systemd 服务不存在"
    fi
    echo ""

    echo -e "${YELLOW}9. 检查宝塔 Nginx 配置...${NC}"
    if /www/server/nginx/sbin/nginx -t 2>/dev/null; then
        ok "Nginx 配置正确"
    else
        err "Nginx 配置有错误:"
        /www/server/nginx/sbin/nginx -t 2>&1 | tail -5
    fi
    echo ""

    echo -e "${YELLOW}10. 重启服务...${NC}"
    safe_release_port "$CBOARD_PORT"
    if systemctl is-enabled ${SERVICE_NAME} &>/dev/null; then
        systemctl restart ${SERVICE_NAME} 2>/dev/null || true
        sleep 3
        if systemctl is-active --quiet ${SERVICE_NAME}; then
            ok "服务重启成功"
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
# 服务管理
# ============================================================================
start_service() {
    echo -e "${BLUE}========== 启动服务 ==========${NC}"
    check_root
    if systemctl is-active --quiet ${SERVICE_NAME}; then
        ok "服务已在运行中"
    else
        systemctl start ${SERVICE_NAME}
        local count=0
        while ! systemctl is-active --quiet ${SERVICE_NAME} && [ $count -lt 10 ]; do
            sleep 1; count=$((count + 1))
        done
        if systemctl is-active --quiet ${SERVICE_NAME}; then
            ok "服务启动成功"
        else
            err "服务启动失败"
            journalctl -u ${SERVICE_NAME} -n 20 --no-pager 2>/dev/null | tail -10
        fi
    fi
    echo ""; read -rp "按回车键继续..."
}

stop_service() {
    echo -e "${BLUE}========== 停止服务 ==========${NC}"
    check_root
    if ! systemctl is-active --quiet ${SERVICE_NAME}; then
        warn "服务未运行"
    else
        systemctl stop ${SERVICE_NAME}
        local count=0
        while systemctl is-active --quiet ${SERVICE_NAME} && [ $count -lt 10 ]; do
            sleep 1; count=$((count + 1))
        done
        if ! systemctl is-active --quiet ${SERVICE_NAME}; then
            ok "服务已停止"
        else
            systemctl kill --signal=SIGKILL ${SERVICE_NAME} 2>/dev/null || true
        fi
    fi
    echo ""; read -rp "按回车键继续..."
}

restart_service() {
    echo -e "${BLUE}========== 重启服务 ==========${NC}"
    check_root
    systemctl stop ${SERVICE_NAME} 2>/dev/null || true
    sleep 2
    safe_release_port "$CBOARD_PORT"
    systemctl start ${SERVICE_NAME}
    local count=0
    while ! systemctl is-active --quiet ${SERVICE_NAME} && [ $count -lt 15 ]; do
        sleep 1; count=$((count + 1))
    done
    if systemctl is-active --quiet ${SERVICE_NAME}; then
        ok "服务重启成功"
    else
        err "服务重启失败"
        journalctl -u ${SERVICE_NAME} -n 20 --no-pager 2>/dev/null | tail -10
    fi
    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 查看服务状态
# ============================================================================
check_service_status() {
    echo -e "${BLUE}========== 服务状态 (宝塔版) ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"

    local domain="" base_url=""
    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        base_url=$(grep "^BASE_URL=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi

    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}【后端服务】${NC}"
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        echo -e "   状态: ${GREEN}运行中${NC}"
    else
        echo -e "   状态: ${RED}已停止${NC}"
    fi
    if command -v ss &>/dev/null && ss -tlnp 2>/dev/null | grep -q ":$CBOARD_PORT "; then
        echo -e "   端口 $CBOARD_PORT: ${GREEN}已监听${NC}"
    else
        echo -e "   端口 $CBOARD_PORT: ${RED}未监听${NC}"
    fi

    echo ""
    echo -e "${GREEN}【宝塔 Nginx】${NC}"
    if /www/server/nginx/sbin/nginx -t 2>/dev/null; then
        echo -e "   状态: ${GREEN}配置正确${NC}"
    else
        echo -e "   状态: ${RED}配置异常${NC}"
    fi

    # SSL 证书
    echo ""
    echo -e "${GREEN}【SSL 证书】${NC}"
    if [ -n "$domain" ]; then
        if [ -f "/www/server/panel/vhost/cert/$domain/fullchain.pem" ]; then
            echo -e "   状态: ${GREEN}已配置 (宝塔路径)${NC}"
        elif [ -f "/etc/letsencrypt/live/$domain/fullchain.pem" ]; then
            echo -e "   状态: ${GREEN}已配置 (Let's Encrypt)${NC}"
        else
            echo -e "   状态: ${YELLOW}未配置${NC}"
        fi
    else
        echo -e "   状态: ${YELLOW}未配置域名${NC}"
    fi

    echo ""
    echo -e "${GREEN}【配置信息】${NC}"
    echo -e "   域名: ${CYAN}${domain:-未配置}${NC}"
    echo -e "   地址: ${CYAN}${base_url:-未配置}${NC}"
    echo -e "   目录: ${CYAN}$work_dir${NC}"

    echo ""
    echo -e "${GREEN}【最近日志】${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    journalctl -u ${SERVICE_NAME} -n 5 --no-pager --no-hostname 2>/dev/null || echo "   无法获取日志"

    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 查看日志
# ============================================================================
view_service_logs() {
    echo -e "${BLUE}========== 查看服务日志 ==========${NC}"
    echo ""
    echo -e "   ${GREEN}1.${NC} 实时日志 (tail -f)"
    echo -e "   ${GREEN}2.${NC} 最近 50 行"
    echo -e "   ${GREEN}3.${NC} 最近 100 行"
    echo -e "   ${GREEN}4.${NC} 仅错误日志"
    echo -e "   ${GREEN}5.${NC} Nginx 访问日志"
    echo -e "   ${GREEN}6.${NC} Nginx 错误日志"
    echo -e "   ${GREEN}0.${NC} 返回"
    echo ""
    read -rp "请选择 [0-6]: " log_choice
    case $log_choice in
        1) echo -e "${YELLOW}按 Ctrl+C 退出${NC}"; journalctl -u ${SERVICE_NAME} -f ;;
        2) journalctl -u ${SERVICE_NAME} -n 50 --no-pager ;;
        3) journalctl -u ${SERVICE_NAME} -n 100 --no-pager ;;
        4) journalctl -u ${SERVICE_NAME} -p err -n 50 --no-pager ;;
        5) tail -100 /www/wwwlogs/cboard.log 2>/dev/null || echo "日志不存在" ;;
        6) tail -100 /www/wwwlogs/cboard.error.log 2>/dev/null || echo "日志不存在" ;;
        0) return 0 ;;
        *) err "无效的选择" ;;
    esac
    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 重设管理员密码
# ============================================================================
reset_admin_password() {
    echo -e "${BLUE}========== 重设管理员密码 ==========${NC}"
    echo ""

    cd "$INSTALL_DIR" 2>/dev/null || cd "$PROJECT_PATH" || {
        err "无法进入项目目录"; read -rp "按回车键继续..."; return 1
    }

    local new_password
    while true; do
        read -rsp "新密码 (至少8位): " new_password; echo
        if [ ${#new_password} -lt 8 ]; then
            err "密码长度至少8位"
        else
            break
        fi
    done

    if [ -f cboard ]; then
        ./cboard reset-password --password "$new_password" 2>/dev/null && ok "密码重置成功" || warn "重置失败，请手动操作"
    else
        warn "cboard 可执行文件不存在"
    fi
    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 查看管理员账号
# ============================================================================
view_admin_account() {
    echo -e "${BLUE}========== 查看管理员账号 ==========${NC}"
    echo ""
    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"
    if [ -f "$work_dir/.env" ]; then
        local admin_email
        admin_email=$(grep "^ADMIN_EMAIL=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        echo -e "  管理员邮箱: ${GREEN}${admin_email:-未配置}${NC}"
    else
        warn ".env 文件不存在"
    fi
    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 备份数据
# ============================================================================
backup_data() {
    echo -e "${BLUE}========== 备份数据 ==========${NC}"
    echo ""
    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"
    cd "$work_dir" || { err "无法进入项目目录"; read -rp "按回车键继续..."; return 1; }

    local BACKUP_DIR="$work_dir/backups"
    mkdir -p "$BACKUP_DIR"
    local TIMESTAMP; TIMESTAMP=$(date +%Y%m%d_%H%M%S)

    [ -f "cboard.db" ] && cp "cboard.db" "$BACKUP_DIR/cboard_${TIMESTAMP}.db" && ok "数据库已备份"
    [ -f ".env" ] && cp ".env" "$BACKUP_DIR/env_${TIMESTAMP}.bak" && ok ".env 已备份"

    echo ""
    echo -e "${YELLOW}现有备份:${NC}"
    ls -lh "$BACKUP_DIR"/*.db 2>/dev/null || echo "  无备份文件"
    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 重装网站 (保留数据)
# ============================================================================
reinstall_website() {
    echo -e "${BLUE}========== 重装网站 (保留数据) ==========${NC}"
    echo ""
    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"
    cd "$work_dir" || { err "无法进入项目目录"; read -rp "按回车键继续..."; return 1; }

    echo -e "${YELLOW}此操作将重新构建，但保留数据库和配置${NC}"
    read -rp "确认继续? (y/n) [n]: " confirm
    [[ ! "$confirm" =~ ^[Yy]$ ]] && { echo "已取消"; read -rp "按回车键继续..."; return 0; }

    # 备份
    local BACKUP_DIR="$work_dir/backup_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    [ -f "cboard.db" ] && cp "cboard.db" "$BACKUP_DIR/"
    [ -f ".env" ] && cp ".env" "$BACKUP_DIR/"
    ok "备份完成: $BACKUP_DIR"

    systemctl stop ${SERVICE_NAME} 2>/dev/null || true

    # 重新构建
    export PATH=$PATH:/usr/local/go/bin
    info "构建后端..."
    go build -o cboard cmd/server/main.go 2>&1 || { err "后端构建失败"; read -rp "按回车键继续..."; return 1; }
    chmod +x cboard

    info "构建前端..."
    cd frontend
    rm -rf node_modules dist 2>/dev/null || true
    npm install --silent 2>/dev/null || { err "前端依赖安装失败"; cd ..; read -rp "按回车键继续..."; return 1; }
    export NODE_OPTIONS="--max-old-space-size=4096"
    npx vite build 2>/dev/null || { err "前端构建失败"; cd ..; read -rp "按回车键继续..."; return 1; }
    cd ..

    chown -R www:www "$work_dir" 2>/dev/null || true
    systemctl daemon-reload
    systemctl start ${SERVICE_NAME}
    sleep 3

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        ok "服务启动成功"
    else
        err "服务启动失败"
    fi

    ok "重装完成!"
    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 诊断 403 错误
# ============================================================================
diagnose_403_error() {
    echo -e "${BLUE}========== 诊断 403 错误 (宝塔版) ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"

    local domain=""
    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi
    [ -z "$domain" ] && read -rp "请输入域名: " domain

    local FRONTEND_ROOT="$work_dir/frontend/dist"

    echo -e "${YELLOW}1. 检查前端目录...${NC}"
    if [ -d "$FRONTEND_ROOT" ] && [ -f "$FRONTEND_ROOT/index.html" ]; then
        ok "前端文件存在"
    else
        err "前端文件不存在，请先安装系统"
    fi

    echo -e "${YELLOW}2. 检查文件权限...${NC}"
    if [ -d "$FRONTEND_ROOT" ]; then
        local owner
        owner=$(stat -c%U "$FRONTEND_ROOT" 2>/dev/null || echo "unknown")
        echo "  当前所有者: $owner (推荐: www)"
        read -rp "是否修复权限? (y/n) [y]: " fix
        if [[ "${fix:-y}" =~ ^[Yy]$ ]]; then
            chown -R www:www "$FRONTEND_ROOT" 2>/dev/null || true
            chmod -R 755 "$FRONTEND_ROOT" 2>/dev/null || true
            find "$FRONTEND_ROOT" -type f -exec chmod 644 {} \; 2>/dev/null || true
            ok "权限已修复"
        fi
    fi

    echo -e "${YELLOW}3. 检查宝塔 Nginx 配置...${NC}"
    local bt_conf="/www/server/panel/vhost/nginx/cboard.conf"
    if [ -f "$bt_conf" ]; then
        ok "宝塔 Nginx 配置存在"
        if grep -q "root.*$FRONTEND_ROOT" "$bt_conf" 2>/dev/null; then
            ok "root 路径正确"
        else
            err "root 路径可能不正确"
            echo "  应为: root $FRONTEND_ROOT;"
        fi
        if grep -q "try_files.*index.html" "$bt_conf" 2>/dev/null; then
            ok "SPA 路由配置正确"
        else
            warn "缺少 try_files 配置"
        fi
    else
        err "宝塔 Nginx 配置不存在"
        echo "  请在宝塔面板中添加站点"
    fi

    echo -e "${YELLOW}4. 检查 .user.ini 文件...${NC}"
    if [ -f "$FRONTEND_ROOT/.user.ini" ]; then
        warn "检测到 .user.ini 文件 (宝塔面板创建)"
        echo "  这可能导致 403 错误"
        read -rp "是否删除? (y/n) [y]: " del
        if [[ "${del:-y}" =~ ^[Yy]$ ]]; then
            chattr -i "$FRONTEND_ROOT/.user.ini" 2>/dev/null || true
            rm -f "$FRONTEND_ROOT/.user.ini" 2>/dev/null || true
            ok "已删除 .user.ini"
        fi
    else
        ok "无 .user.ini 文件"
    fi

    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 更新代码
# ============================================================================
update_code() {
    echo -e "${BLUE}========== 更新代码 ==========${NC}"
    echo ""
    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"
    cd "$work_dir" || { err "无法进入项目目录"; read -rp "按回车键继续..."; return 1; }

    if [ ! -d ".git" ]; then
        err "当前目录不是 Git 仓库"
        read -rp "按回车键继续..."; return 1
    fi

    local branch; branch=$(git branch --show-current 2>/dev/null || echo "main")
    echo -e "  分支: ${GREEN}$branch${NC}"

    if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
        warn "检测到未提交的更改"
        echo "  1. 暂存 (stash)  2. 放弃 (reset)  3. 取消"
        read -rp "选择 [1-3, 默认: 2]: " handle
        case ${handle:-2} in
            1) git stash push -m "Auto stash $(date +%Y%m%d_%H%M%S)" 2>/dev/null ;;
            2) git reset --hard HEAD 2>/dev/null ;;
            3) read -rp "按回车键继续..."; return 0 ;;
        esac
    fi

    git config --global --add safe.directory "$work_dir" 2>/dev/null || true

    info "拉取更新..."
    if git pull origin "$branch" 2>&1; then
        ok "代码更新成功"
    else
        err "更新失败"; read -rp "按回车键继续..."; return 1
    fi

    read -rp "是否重新构建? (y/n) [y]: " rebuild
    if [[ "${rebuild:-y}" =~ ^[Yy]$ ]]; then
        systemctl stop ${SERVICE_NAME} 2>/dev/null || true
        export PATH=$PATH:/usr/local/go/bin
        go build -o cboard cmd/server/main.go 2>&1 || err "后端构建失败"
        chmod +x cboard 2>/dev/null || true
        cd frontend && npm install --silent 2>/dev/null && npx vite build 2>/dev/null; cd ..
        chown -R www:www "$work_dir" 2>/dev/null || true
        systemctl start ${SERVICE_NAME}
        sleep 3
        systemctl is-active --quiet ${SERVICE_NAME} && ok "服务启动成功" || err "服务启动失败"
    fi

    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 修复 Nginx SSL 验证
# ============================================================================
fix_nginx_for_ssl() {
    echo -e "${BLUE}========== 修复 Nginx SSL 验证配置 ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"

    local domain=""
    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi
    [ -z "$domain" ] && read -rp "请输入域名: " domain
    [ -z "$domain" ] && { err "域名不能为空"; read -rp "按回车键继续..."; return 1; }

    # 检查宝塔 Nginx 配置
    local conf="/www/server/panel/vhost/nginx/cboard.conf"
    [ ! -f "$conf" ] && conf="/www/server/nginx/conf/vhost/cboard.conf"
    [ ! -f "$conf" ] && { err "Nginx 配置不存在"; read -rp "按回车键继续..."; return 1; }

    # 备份
    cp "$conf" "${conf}.bak.$(date +%Y%m%d_%H%M%S)"

    if grep -q "\.well-known" "$conf"; then
        ok ".well-known 配置已存在"
    else
        info "添加 .well-known 配置..."
        sed -i '/location \/api\//i\    location \/.well-known\/acme-challenge\/ {\n        root '"$work_dir"';\n        allow all;\n        access_log off;\n        log_not_found off;\n    }\n\n    location \/.well-known\/ {\n        root '"$work_dir"';\n        allow all;\n        access_log off;\n        log_not_found off;\n    }\n' "$conf"
        ok "配置已添加"
    fi

    if /www/server/nginx/sbin/nginx -t 2>/dev/null; then
        /www/server/nginx/sbin/nginx -s reload 2>/dev/null || true
        ok "Nginx 已重载"
        echo ""
        echo -e "${YELLOW}现在可以在宝塔面板中申请 SSL 证书:${NC}"
        echo "  网站 -> $domain -> SSL -> Let's Encrypt"
    else
        err "Nginx 配置测试失败"
    fi

    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 诊断网站访问
# ============================================================================
diagnose_website_access() {
    echo -e "${BLUE}========== 诊断网站访问 (宝塔版) ==========${NC}"
    echo ""

    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"

    local domain="" base_url=""
    if [ -f "$work_dir/.env" ]; then
        domain=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        base_url=$(grep "^BASE_URL=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi
    [ -z "$domain" ] && read -rp "请输入域名: " domain
    [ -z "$base_url" ] && base_url="http://$domain"

    echo -e "  域名: ${CYAN}$domain${NC}"
    echo ""

    echo -e "${YELLOW}1. 后端服务...${NC}"
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        ok "后端服务运行中"
        local code; code=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$CBOARD_PORT/api/health" 2>/dev/null || echo "000")
        [ "$code" = "200" ] && ok "API 可访问" || warn "API 返回 HTTP $code"
    else
        err "后端服务未运行"
    fi

    echo -e "${YELLOW}2. 宝塔 Nginx...${NC}"
    if /www/server/nginx/sbin/nginx -t 2>/dev/null; then
        ok "Nginx 配置正确"
    else
        err "Nginx 配置有错误"
    fi

    echo -e "${YELLOW}3. 前端文件...${NC}"
    [ -f "$work_dir/frontend/dist/index.html" ] && ok "前端文件存在" || err "前端文件不存在"

    echo -e "${YELLOW}4. DNS 解析...${NC}"
    if [ -n "$domain" ] && command -v dig &>/dev/null; then
        local dns_ip; dns_ip=$(dig +short "$domain" 2>/dev/null | head -1)
        if [ -n "$dns_ip" ]; then
            ok "域名解析到: $dns_ip"
            local server_ip; server_ip=$(curl -s4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
            [ "$dns_ip" = "$server_ip" ] && ok "指向当前服务器" || warn "未指向当前服务器 ($server_ip)"
        else
            err "域名无法解析"
        fi
    fi

    echo -e "${YELLOW}5. 网站访问测试...${NC}"
    if [ -n "$base_url" ]; then
        local code; code=$(curl -s -o /dev/null -w "%{http_code}" "$base_url" 2>/dev/null || echo "000")
        case "$code" in
            200) ok "网站可访问" ;;
            403) err "403 禁止访问 - 运行选项 13 诊断" ;;
            502|503|504) err "HTTP $code - 后端异常" ;;
            000) err "无法连接" ;;
            *) warn "HTTP $code" ;;
        esac
    fi

    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 卸载
# ============================================================================
uninstall_cboard() {
    echo -e "${BLUE}========== 卸载 CBoard v2 ==========${NC}"
    echo ""
    echo -e "${RED}此操作将停止服务并删除配置 (数据目录保留)${NC}"
    read -rp "确定要卸载吗? (输入 YES 确认): " confirm
    [ "$confirm" != "YES" ] && { echo "已取消"; read -rp "按回车键继续..."; return 0; }

    systemctl stop ${SERVICE_NAME} 2>/dev/null || true
    systemctl disable ${SERVICE_NAME} 2>/dev/null || true
    rm -f /etc/systemd/system/${SERVICE_NAME}.service
    rm -f /www/server/panel/vhost/nginx/cboard.conf
    rm -f /www/server/nginx/conf/vhost/cboard.conf
    rm -f /usr/local/bin/cboard-ctl
    systemctl daemon-reload
    /www/server/nginx/sbin/nginx -s reload 2>/dev/null || true

    ok "CBoard 服务已卸载 (数据目录保留)"
    echo ""; read -rp "按回车键继续..."
}

# ============================================================================
# 创建管理脚本
# ============================================================================
create_management_script() {
    local work_dir="$INSTALL_DIR"
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"

    cat > "$work_dir/cboard-ctl" <<'MGMT'
#!/usr/bin/env bash
SERVICE="cboard"
case "$1" in
    start)   systemctl start $SERVICE && echo "CBoard 已启动" ;;
    stop)    systemctl stop $SERVICE && echo "CBoard 已停止" ;;
    restart) systemctl restart $SERVICE && echo "CBoard 已重启" ;;
    status)  systemctl status $SERVICE ;;
    log)     journalctl -u $SERVICE -f --no-pager ;;
    update)
        echo "正在更新 CBoard..."
        cd "$(dirname "$0")"
        systemctl stop $SERVICE
        export PATH=$PATH:/usr/local/go/bin
        go build -o cboard cmd/server/main.go
        cd frontend && npm install --silent && npx vite build 2>/dev/null
        cd ..
        chown -R www:www .
        systemctl start $SERVICE
        echo "更新完成"
        ;;
    backup)
        BACKUP_DIR="$(dirname "$0")/backups"
        mkdir -p "$BACKUP_DIR"
        TIMESTAMP=$(date +%Y%m%d_%H%M%S)
        cp "$(dirname "$0")/cboard.db" "$BACKUP_DIR/cboard_${TIMESTAMP}.db"
        echo "备份完成: $BACKUP_DIR/cboard_${TIMESTAMP}.db"
        ;;
    uninstall)
        echo "确定要卸载 CBoard 吗? (y/n)"
        read -r confirm
        if [ "$confirm" = "y" ]; then
            systemctl stop $SERVICE
            systemctl disable $SERVICE
            rm -f /etc/systemd/system/${SERVICE}.service
            rm -f /www/server/panel/vhost/nginx/cboard.conf
            rm -f /www/server/nginx/conf/vhost/cboard.conf
            systemctl daemon-reload
            /www/server/nginx/sbin/nginx -s reload 2>/dev/null
            echo "CBoard 服务已卸载 (数据目录保留)"
        fi
        ;;
    *)
        echo "用法: cboard-ctl {start|stop|restart|status|log|update|backup|uninstall}"
        ;;
esac
MGMT
    chmod +x "$work_dir/cboard-ctl"
    ln -sf "$work_dir/cboard-ctl" /usr/local/bin/cboard-ctl 2>/dev/null || true
}

# ============================================================================
# 显示菜单
# ============================================================================
show_menu() {
    clear
    echo -e "${CYAN}"
    echo "  ╔══════════════════════════════════════════════╗"
    echo "  ║   CBoard v2 管理面板 (宝塔版) v$SCRIPT_VERSION      ║"
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
    [ ! -d "$work_dir" ] && work_dir="$PROJECT_PATH"
    if [ -f "$work_dir/.env" ]; then
        CBOARD_PORT=$(grep "^PORT=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
        CBOARD_PORT=${CBOARD_PORT:-9000}
        BACKEND_PORT=$CBOARD_PORT
        DOMAIN=$(grep "^DOMAIN=" "$work_dir/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | xargs)
    fi

    check_concurrent

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
