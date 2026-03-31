#!/usr/bin/env bash
# ============================================================================
# CBoard v2 一键安装 & 管理脚本（宝塔面板版）- 优化版
# 适用于已安装宝塔面板的 Linux 服务器
# ============================================================================
[ -n "$BASH_VERSION" ] || exec /usr/bin/env bash "$0" "$@"
set -e

# ---- 版本 & 变量 ----
SCRIPT_VERSION="2.1.0"
PROJECT_PATH="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="/www/wwwroot/cboard"
SERVICE_NAME="cboard-v2"
CBOARD_PORT=9000
BACKEND_PORT=9000
DOMAIN=""
ENABLE_SSL="n"
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
LOCK_FILE="/tmp/cboard_bt_install.lock"

# 宝塔相关固定路径
BT_NGINX_BIN="/www/server/nginx/sbin/nginx"
BT_NGINX_VHOST_DIR="/www/server/panel/vhost/nginx"
BT_NGINX_CERT_DIR="/www/server/panel/vhost/cert"

# ---- 颜色 & 基础输出 ----
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BLUE='\033[0;34m'; NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()   { echo -e "${RED}[ERROR]${NC} $*"; }
fatal() { echo -e "${RED}[FATAL]${NC} $*"; exit 1; }
pause() { echo ""; read -rp "按回车键继续..." _; }

# ============================================================================
# 助手函数 (Helpers)
# ============================================================================

# 获取实际工作目录
get_work_dir() {
    [ -d "$INSTALL_DIR" ] && echo "$INSTALL_DIR" || echo "$PROJECT_PATH"
}

# 进入工作目录
enter_work_dir() {
    local wd; wd=$(get_work_dir)
    cd "$wd" 2>/dev/null || { err "无法进入目录 $wd"; return 1; }
}

# 安全获取 .env 配置项
get_env_val() {
    local key="$1"
    local wd; wd=$(get_work_dir)
    if [ -f "$wd/.env" ]; then
        grep "^${key}=" "$wd/.env" 2>/dev/null | cut -d'=' -f2- | tr -d '"' | xargs
    fi
}

# 统一重载 Nginx
reload_nginx() {
    if $BT_NGINX_BIN -t 2>/dev/null; then
        $BT_NGINX_BIN -s reload 2>/dev/null || true
        return 0
    else
        warn "Nginx 配置测试失败"
        return 1
    fi
}

# 等待服务状态
wait_for_service() {
    local target="$1" max=10 count=0
    while [ $count -lt $max ]; do
        if [ "$target" = "active" ] && systemctl is-active --quiet ${SERVICE_NAME}; then return 0; fi
        if [ "$target" = "inactive" ] && ! systemctl is-active --quiet ${SERVICE_NAME}; then return 0; fi
        sleep 1; ((count++))
    done
    return 1
}

# 安全释放端口
safe_release_port() {
    local pid; pid=$(lsof -ti ":$1" 2>/dev/null || true)
    if [ -n "$pid" ]; then
        local pname; pname=$(ps -p "$pid" -o comm= 2>/dev/null || true)
        [[ "$pname" == "nginx" || "$pname" == "sshd" ]] && return 0
        warn "端口 $1 被 PID $pid ($pname) 占用，正在释放..."
        kill "$pid" 2>/dev/null || true
        sleep 2
        kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
    fi
}

# 验证格式
validate_domain() { [[ "$1" =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$ ]]; }
validate_email() { [[ "$1" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; }

# 并发检查 & 环境检查
check_concurrent() {
    if [ -f "$LOCK_FILE" ]; then
        local lock_pid; lock_pid=$(cat "$LOCK_FILE" 2>/dev/null)
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

check_root() { [[ $EUID -ne 0 ]] && fatal "请使用 root 用户运行此脚本"; }
check_bt() { [ ! -f /www/server/panel/BT-Panel ] && [ ! -d /www/server/panel ] && fatal "未检测到宝塔面板，请先安装宝塔面板"; ok "检测到宝塔面板"; }
check_bt_nginx() { [ ! -f $BT_NGINX_BIN ] && fatal "未检测到宝塔 Nginx，请在宝塔软件商店安装后重试"; ok "检测到宝塔 Nginx"; }
check_disk_space() {
    local available; available=$(df -BM "$PROJECT_PATH" 2>/dev/null | awk 'NR==2{print $4}' | tr -d 'M')
    if [ -n "$available" ] && [ "$available" -lt 1024 ]; then
        err "磁盘可用空间不足: ${available}MB (需要至少 1GB)"; return 1
    fi
    ok "磁盘空间充足: ${available}MB 可用"; return 0
}

# ============================================================================
# 依赖安装模块
# ============================================================================
install_go() {
    command -v go &>/dev/null && { ok "Go 已安装: $(go version | awk '{print $3}')"; return; }
    info "安装 Go 1.24..."
    local arch; arch=$(uname -m)
    [ "$arch" = "x86_64" ] && GO_ARCH="amd64" || [ "$arch" = "aarch64" ] && GO_ARCH="arm64" || fatal "不支持的架构: $arch"
    wget -q "https://go.dev/dl/go1.24.0.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz || fatal "Go 下载失败"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz && rm -f /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null || echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    ok "Go $(go version | awk '{print $3}') 安装完成"
}

install_node() {
    command -v node &>/dev/null && { ok "Node.js 已安装: $(node -v)"; return; }
    if ls /www/server/nvm/versions/node/*/bin/node 2>/dev/null | head -1 | grep -q node; then
        export PATH=$(ls -d /www/server/nvm/versions/node/*/bin 2>/dev/null | tail -1):$PATH
        ok "使用宝塔 Node.js: $(node -v)"; return
    fi
    info "安装 Node.js 20.x..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - >/dev/null 2>&1 || curl -fsSL https://rpm.nodesource.com/setup_20.x | bash - >/dev/null 2>&1
    command -v apt-get &>/dev/null && apt-get install -y -qq nodejs >/dev/null 2>&1 || yum install -y nodejs >/dev/null 2>&1
    ok "Node.js $(node -v) 安装完成"
}

install_redis() {
    if command -v redis-server &>/dev/null; then
        ok "Redis 已安装: $(redis-server --version | awk '{print $3}')"
    else
        info "安装 Redis..."
        if command -v apt-get &>/dev/null; then apt-get install -y -qq redis-server >/dev/null 2>&1
        elif command -v yum &>/dev/null; then yum install -y redis >/dev/null 2>&1 || dnf install -y redis >/dev/null 2>&1
        else warn "无法自动安装 Redis，请手动安装"; return 0; fi
        ok "Redis 安装完成"
    fi
    systemctl enable redis-server >/dev/null 2>&1 || systemctl enable redis >/dev/null 2>&1 || true
    systemctl start redis-server >/dev/null 2>&1 || systemctl start redis >/dev/null 2>&1 || true
    redis-cli ping 2>/dev/null | grep -q PONG && ok "Redis 服务已启动" || warn "Redis 启动失败，系统将回退到内存模式"
}

# ============================================================================
# 核心安装逻辑
# ============================================================================
interactive_config() {
    echo -e "\n${CYAN}========== CBoard v2 安装配置 ==========${NC}\n"
    read -rp "安装目录 [$INSTALL_DIR]: " input; INSTALL_DIR=${input:-$INSTALL_DIR}
    CBOARD_PORT=9000; BACKEND_PORT=$CBOARD_PORT

    while true; do
        read -rp "绑定域名 (留空则使用 IP): " DOMAIN
        [ -z "$DOMAIN" ] && break
        DOMAIN=$(echo "$DOMAIN" | sed 's|^https\?://||' | sed 's|/$||')
        validate_domain "$DOMAIN" && break || err "域名格式不正确"
    done

    [ -n "$DOMAIN" ] && read -rp "是否自动申请 SSL 证书? (y/n) [y]: " ENABLE_SSL && ENABLE_SSL=${ENABLE_SSL:-y}

    while true; do
        read -rp "管理员邮箱: " ADMIN_EMAIL
        validate_email "$ADMIN_EMAIL" && break || err "邮箱格式不正确"
    done

    while true; do
        read -rsp "管理员密码 (至少8位): " ADMIN_PASSWORD; echo
        [ ${#ADMIN_PASSWORD} -lt 8 ] && { err "密码长度至少8位"; continue; }
        read -rsp "确认密码: " ADMIN_PASSWORD_CONFIRM; echo
        [ "$ADMIN_PASSWORD" == "$ADMIN_PASSWORD_CONFIRM" ] && break || err "两次密码不一致"
    done

    echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    info "安装目录: $INSTALL_DIR | 域名: ${DOMAIN:-IP访问} | SSL: ${ENABLE_SSL:-n} | 邮箱: $ADMIN_EMAIL"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    read -rp "确认以上配置? (y/n) [y]: " confirm
    [[ "${confirm:-y}" != "y" ]] && fatal "安装已取消"
}

create_env_file() {
    cd "$INSTALL_DIR" || return 1
    if [ -f .env ]; then
        warn ".env 已存在，跳过生成"
        return 0
    fi
    local SECRET_KEY; SECRET_KEY=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n')
    local BASE_URL="http://$(curl -s4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}'):$CBOARD_PORT"
    [ -n "$DOMAIN" ] && BASE_URL=$([[ "$ENABLE_SSL" =~ ^[Yy]$ ]] && echo "https://$DOMAIN" || echo "http://$DOMAIN")

    cat > .env <<EOF
DEBUG=false
PORT=$CBOARD_PORT
HOST=127.0.0.1
SECRET_KEY=$SECRET_KEY
BASE_URL=$BASE_URL
CORS_ORIGINS=$BASE_URL
DATABASE_URL=sqlite:///./cboard.db
DOMAIN=${DOMAIN}
SSL_ENABLED=$([ "$ENABLE_SSL" = "y" ] && echo "true" || echo "false")
ADMIN_EMAIL=$ADMIN_EMAIL
SUBSCRIPTION_URL_PREFIX=$BASE_URL/sub
ALIPAY_NOTIFY_URL=$BASE_URL/api/v1/payment/notify/alipay
ALIPAY_RETURN_URL=$BASE_URL/payment/return
REDIS_ADDR=127.0.0.1:6379
EOF
    ok ".env 配置文件已生成"
}

deploy_and_build() {
    mkdir -p "$INSTALL_DIR"
    if grep -q "cboard/v2" "$PROJECT_PATH/go.mod" 2>/dev/null; then
        [ "$PROJECT_PATH" != "$INSTALL_DIR" ] && cp -r "$PROJECT_PATH"/* "$PROJECT_PATH"/.[a-zA-Z0-9]* "$INSTALL_DIR/" 2>/dev/null || true
    else
        fatal "请将此脚本放在源码根目录下运行"
    fi
    create_env_file

    info "构建后端..."; cd "$INSTALL_DIR"
    export PATH=$PATH:/usr/local/go/bin
    go build -o cboard cmd/server/main.go 2>&1 || { go clean -cache; go build -o cboard cmd/server/main.go || fatal "后端构建失败"; }
    chmod +x cboard && ok "后端构建完成"

    info "构建前端..."; cd "$INSTALL_DIR/frontend"
    npm install --silent 2>/dev/null || { rm -rf node_modules; sleep 2; npm install || fatal "前端依赖失败"; }
    export NODE_OPTIONS="--max-old-space-size=4096"
    npx vite build 2>&1 || fatal "前端构建失败"
    [ ! -f "dist/index.html" ] && fatal "前端 dist/index.html 不存在"
    ok "前端构建完成"
}

setup_systemd() {
    cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=CBoard v2 Server
After=network.target redis-server.service redis.service

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
    systemctl daemon-reload && systemctl enable ${SERVICE_NAME} >/dev/null 2>&1
    ok "systemd 服务已创建"
}

setup_bt_nginx() {
    info "生成 Nginx 配置..."
    local CONF_FILE="$BT_NGINX_VHOST_DIR/cboard.conf"
    mkdir -p "$BT_NGINX_VHOST_DIR" /www/server/nginx/conf/vhost 2>/dev/null || true

    cat > "$CONF_FILE" <<EOF
server {
    listen 80;
    server_name ${DOMAIN:-_};
    root $INSTALL_DIR/frontend/dist;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;

    access_log /www/wwwlogs/cboard.log;
    error_log /www/wwwlogs/cboard.error.log;

    location /.well-known/ { root $INSTALL_DIR; allow all; access_log off; }
    
    location /api/ {
        proxy_pass http://127.0.0.1:$CBOARD_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    
    location /assets/ { expires 30d; add_header Cache-Control "public, immutable"; }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
    cp "$CONF_FILE" /www/server/nginx/conf/vhost/cboard.conf 2>/dev/null || true
    reload_nginx && ok "Nginx 配置完成" || warn "Nginx 重载失败，请前往宝塔手动检查"
}

setup_ssl() {
    [[ ! "$ENABLE_SSL" =~ ^[Yy]$ ]] || [ -z "$DOMAIN" ] && return 0
    local CERT_PATH="$BT_NGINX_CERT_DIR/$DOMAIN"

    if [ -f "$CERT_PATH/fullchain.pem" ] || [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
        ok "检测到已有证书，跳过申请，直接更新配置"
        [ ! -f "$CERT_PATH/fullchain.pem" ] && mkdir -p "$CERT_PATH" && cp /etc/letsencrypt/live/$DOMAIN/*.pem "$CERT_PATH/" 2>/dev/null
    else
        info "申请 Let's Encrypt SSL..."
        local nginx_was_running=false
        pgrep -x nginx >/dev/null 2>&1 && nginx_was_running=true
        
        if certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" --redirect 2>/dev/null; then
            mkdir -p "$CERT_PATH" && cp /etc/letsencrypt/live/$DOMAIN/*.pem "$CERT_PATH/" 2>/dev/null
            ok "SSL 申请成功 (nginx)"
        else
            [ "$nginx_was_running" = true ] && systemctl stop nginx 2>/dev/null || $BT_NGINX_BIN -s stop 2>/dev/null || true
            if certbot certonly --standalone -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" 2>/dev/null; then
                mkdir -p "$CERT_PATH" && cp /etc/letsencrypt/live/$DOMAIN/*.pem "$CERT_PATH/" 2>/dev/null
                ok "SSL 申请成功 (standalone)"
            else
                warn "SSL 申请失败，请前往宝塔面板手动申请！"
            fi
            [ "$nginx_was_running" = true ] && systemctl start nginx 2>/dev/null || $BT_NGINX_BIN 2>/dev/null || true
        fi
    fi

    # 更新为 HTTPS 配置
    if [ -f "$CERT_PATH/fullchain.pem" ]; then
        local CONF_FILE="$BT_NGINX_VHOST_DIR/cboard.conf"
        sed -i 's/listen 80;/listen 80;\n    listen 443 ssl http2;\n    ssl_certificate '"${CERT_PATH//\//\\/}"'\/fullchain.pem;\n    ssl_certificate_key '"${CERT_PATH//\//\\/}"'\/privkey.pem;/g' "$CONF_FILE"
        cp "$CONF_FILE" /www/server/nginx/conf/vhost/cboard.conf 2>/dev/null || true
        reload_nginx && ok "HTTPS 配置已更新" || warn "HTTPS 验证失败"
    fi
}

install_system() {
    echo -e "${BLUE}========== 开始安装 CBoard v2 ==========${NC}\n"
    check_root; check_bt; check_bt_nginx; check_disk_space || exit 1
    interactive_config

    systemctl is-active --quiet ${SERVICE_NAME} && { info "停止旧服务..."; systemctl stop ${SERVICE_NAME}; }
    install_go; install_node; install_redis
    deploy_and_build; setup_systemd; setup_bt_nginx; setup_ssl

    safe_release_port "$CBOARD_PORT"
    systemctl start ${SERVICE_NAME}
    if wait_for_service active; then
        ok "服务启动成功"
        [ -n "$ADMIN_EMAIL" ] && (cd "$INSTALL_DIR" && ./cboard reset-password --email "$ADMIN_EMAIL" --password "$ADMIN_PASSWORD" >/dev/null 2>&1)
    else
        err "服务启动失败"; journalctl -u ${SERVICE_NAME} -n 10 --no-pager
    fi
    create_management_script
    print_install_result; pause
}

print_install_result() {
    local BASE_URL=$(get_env_val "BASE_URL")
    echo -e "\n${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   CBoard v2 安装完成! (宝塔面板版)      ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}\n"
    echo -e "  访问地址:    ${CYAN}${BASE_URL:-IP}${NC}"
    echo -e "  安装目录:    ${CYAN}$INSTALL_DIR${NC}"
    echo -e "  管理员邮箱:  ${YELLOW}$ADMIN_EMAIL${NC}"
    echo -e "  管理员密码:  ${YELLOW}${ADMIN_PASSWORD:0:2}******${NC}"
    echo -e "  ${RED}请登录后立即修改默认密码!${NC}\n"
}

create_management_script() {
    cat > "$INSTALL_DIR/cboard-ctl" <<'MGMT'
#!/usr/bin/env bash
SERVICE="cboard-v2"
case "$1" in
    start|stop|restart|status) systemctl $1 $SERVICE ;;
    log) journalctl -u $SERVICE -f --no-pager ;;
    *) echo "用法: cboard-ctl {start|stop|restart|status|log}" ;;
esac
MGMT
    chmod +x "$INSTALL_DIR/cboard-ctl"
    ln -sf "$INSTALL_DIR/cboard-ctl" /usr/local/bin/cboard-ctl 2>/dev/null || true
}

# ============================================================================
# 菜单功能模块
# ============================================================================
configure_domain() {
    enter_work_dir || return 1
    read -rp "请输入域名 (例如: example.com): " DOMAIN
    validate_domain "$DOMAIN" || { err "格式错误"; pause; return 1; }
    read -rp "是否使用 HTTPS? (y/n) [y]: " USE_HTTPS
    
    local BASE_URL="http://$DOMAIN" SSL_ENABLED="false"
    [[ "${USE_HTTPS:-y}" =~ ^[Yy]$ ]] && { BASE_URL="https://$DOMAIN"; SSL_ENABLED="true"; ENABLE_SSL="y"; }

    if [ -f .env ]; then
        for key in DOMAIN BASE_URL CORS_ORIGINS SSL_ENABLED SUBSCRIPTION_URL_PREFIX ALIPAY_NOTIFY_URL ALIPAY_RETURN_URL; do
            local val; case $key in
                DOMAIN) val="$DOMAIN" ;; BASE_URL|CORS_ORIGINS) val="$BASE_URL" ;; SSL_ENABLED) val="$SSL_ENABLED" ;;
                SUBSCRIPTION_URL_PREFIX) val="$BASE_URL/sub" ;; ALIPAY_NOTIFY_URL) val="$BASE_URL/api/v1/payment/notify/alipay" ;;
                ALIPAY_RETURN_URL) val="$BASE_URL/payment/return" ;;
            esac
            grep -q "^${key}=" .env && sed -i "s|^${key}=.*|${key}=$val|" .env || echo "${key}=$val" >> .env
        done
        ok ".env 已更新"
    fi
    setup_bt_nginx; [[ "${USE_HTTPS:-y}" =~ ^[Yy]$ ]] && setup_ssl
    systemctl restart ${SERVICE_NAME} 2>/dev/null || true
    ok "配置完成: $BASE_URL"; pause
}

fix_common_errors() {
    enter_work_dir || return 1
    echo -e "\n${YELLOW}执行自动修复检查...${NC}"
    check_disk_space || true
    command -v go &>/dev/null || install_go
    command -v node &>/dev/null || install_node
    [ -f cboard.db ] && chown www:www cboard.db && chmod 664 cboard.db
    [ ! -f "frontend/dist/index.html" ] && build_frontend
    chown -R www:www "$PWD" 2>/dev/null || true
    reload_nginx
    systemctl restart ${SERVICE_NAME}; ok "修复完毕"; pause
}

service_manager() {
    local action="$1" target_state="$2"
    systemctl "$action" ${SERVICE_NAME} 2>/dev/null || true
    if wait_for_service "$target_state"; then ok "操作成功 ($action)"; else err "操作失败"; fi
    pause
}

check_service_status() {
    local wd; wd=$(get_work_dir)
    local domain; domain=$(get_env_val "DOMAIN")
    local base_url; base_url=$(get_env_val "BASE_URL")
    
    echo -e "\n${GREEN}【后端服务】${NC}"
    systemctl is-active --quiet ${SERVICE_NAME} && echo -e "   状态: ${GREEN}运行中${NC}" || echo -e "   状态: ${RED}已停止${NC}"
    echo -e "\n${GREEN}【配置信息】${NC}\n   域名: ${domain:-未配置}\n   地址: ${base_url:-未配置}\n   目录: $wd"
    pause
}

view_service_logs() {
    echo -e "\n1.实时日志 2.最近50行 3.错误日志 0.返回"
    read -rp "请选择: " lc
    case $lc in
        1) journalctl -u ${SERVICE_NAME} -f ;;
        2) journalctl -u ${SERVICE_NAME} -n 50 --no-pager ;;
        3) journalctl -u ${SERVICE_NAME} -p err -n 50 --no-pager ;;
    esac
    pause
}

reset_admin_password() {
    enter_work_dir || return 1
    [ ! -f cboard.db ] && { err "数据库不存在"; pause; return 1; }
    read -rp "邮箱: " email
    read -rsp "新密码: " pwd; echo
    [ ${#pwd} -lt 8 ] && { err "密码太短"; pause; return 1; }
    ./cboard reset-password --email "${email:-admin}" --password "$pwd" && ok "重置成功" || err "失败"; pause
}

# 备份、更新、诊断等函数简化逻辑，复用 enter_work_dir
backup_data() {
    enter_work_dir || return 1
    local dir="backups/$(date +%Y%m%d_%H%M%S)"; mkdir -p "$dir"
    [ -f cboard.db ] && cp cboard.db "$dir/"
    [ -f .env ] && cp .env "$dir/"
    ok "已备份至 $dir"; pause
}

update_code() {
    enter_work_dir || return 1
    git config --global --add safe.directory "$PWD"
    git fetch origin main && git reset --hard origin/main || { err "Git更新失败"; pause; return 1; }
    read -rp "是否重新构建? (y/n) [y]: " rb
    if [[ "${rb:-y}" =~ ^[Yy]$ ]]; then
        systemctl stop ${SERVICE_NAME}
        export PATH=$PATH:/usr/local/go/bin
        go build -o cboard cmd/server/main.go
        cd frontend && npm install && npx vite build && cd ..
        chown -R www:www .
        systemctl start ${SERVICE_NAME}
    fi
    ok "更新完成"; pause
}

uninstall_cboard() {
    read -rp "输入 yes 确认完全卸载: " confirm
    [[ "$confirm" != "yes" ]] && return 0
    systemctl stop ${SERVICE_NAME} 2>/dev/null; systemctl disable ${SERVICE_NAME} 2>/dev/null
    rm -f /etc/systemd/system/${SERVICE_NAME}.service
    rm -f $BT_NGINX_VHOST_DIR/cboard.conf /www/server/nginx/conf/vhost/cboard.conf
    systemctl daemon-reload; reload_nginx
    read -rp "是否删除文件数据(y/n)[n]: " del
    [[ "$del" =~ ^[Yy]$ ]] && rm -rf "$(get_work_dir)"
    ok "卸载完毕"; pause
}

# ============================================================================
# 主菜单
# ============================================================================
show_menu() {
    clear; echo -e "${CYAN}  ╔══════════════════════════════════════════════╗"
    echo "  ║   CBoard v2 管理面板 (优化版) v$SCRIPT_VERSION        ║"
    echo -e "  ╚══════════════════════════════════════════════╝${NC}"
    echo -e "  ${GREEN}1.${NC} 安装系统        ${GREEN}2.${NC} 配置域名/SSL   ${GREEN}3.${NC} 修复环境"
    echo -e "  ${GREEN}4.${NC} 启动服务        ${GREEN}5.${NC} 停止服务       ${GREEN}6.${NC} 重启服务"
    echo -e "  ${GREEN}7.${NC} 服务状态        ${GREEN}8.${NC} 查看日志       ${GREEN}9.${NC} 重设密码"
    echo -e "  ${GREEN}10.${NC}备份数据        ${GREEN}11.${NC}更新代码       ${GREEN}12.${NC}卸载系统"
    echo -e "  ${GREEN}0.${NC} 退出\n"
}

main() {
    check_concurrent
    local wd; wd=$(get_work_dir)
    # 一键安装拦截
    if [ ! -f "$wd/.env" ] && ! systemctl is-enabled ${SERVICE_NAME} >/dev/null 2>&1; then
        install_system
    fi

    while true; do
        show_menu; read -rp "选择 [0-12]: " choice
        case $choice in
            1) install_system ;; 2) configure_domain ;; 3) fix_common_errors ;;
            4) service_manager start active ;; 5) service_manager stop inactive ;; 6) service_manager restart active ;;
            7) check_service_status ;; 8) view_service_logs ;; 9) reset_admin_password ;;
            10) backup_data ;; 11) update_code ;; 12) uninstall_cboard ;;
            0) echo -e "${GREEN}Bye!${NC}"; exit 0 ;;
            *) err "无效选项"; sleep 1 ;;
        esac
    done
}

main "$@"
