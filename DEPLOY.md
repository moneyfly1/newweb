# CBoard 部署文档

## 环境要求

- Go 1.24+（`go.mod` 要求 go 1.24）
- Node.js 18+（安装脚本使用 20.x）
- SQLite 3 或 MySQL 5.7+ 或 PostgreSQL
- Redis（可选，用于缓存和限流）
- 快速部署推荐直接使用仓库内 `install.sh`（纯净 Linux）或 `install_bt.sh`（宝塔）一键安装；以下为手动部署步骤

## 快速部署

### 1. 后端部署

```bash
# 编译（产物为 ./cboard）
go build -o cboard ./cmd/server

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件：手动模式默认端口 8000；若按下方 Nginx 模板（proxy 9000）部署，
# 请把 .env 中 PORT 设为 9000（与安装脚本模式保持一致）
# PORT=9000

# 运行
./cboard
```

> 生产建议注册为 systemd 服务（单元名 `cboard-v2`，安装脚本自动完成）。
> 手动注册示例：`/etc/systemd/system/cboard-v2.service` 内 `ExecStart=/path/to/cboard`，然后 `systemctl enable --now cboard-v2`。

### 2. 前端部署

```bash
cd frontend
npm install
npm run build

# 将 dist 目录部署到 Nginx
```

### 3. Nginx 配置

以下为**最终标准模板**（与 `install.sh` / `install_bt.sh` 安装脚本生成的配置一致），完整包含：

- 前端静态资源由 Nginx 直接服务（`root .../frontend/dist`）
- `/assets/` 资源带 hash 文件名，长期缓存（`immutable`）
- `/api/`、`/nodes/` 反向代理到 Go 后端（`/nodes/` 用于 GitHub 节点文件同步公开外链）
- **`/index.html` 强制 no-cache**：SPA 外壳不缓存，保证每次发布后用户都能拿到引用新 hash JS 的最新页面（否则手机/浏览器可能长期停留在旧界面）

```nginx
# ---------- HTTP：跳转 HTTPS（配置 SSL 时） ----------
server {
    listen 80;
    server_name your-domain.com;
    location /.well-known/acme-challenge/ {
        root /path/to/frontend/dist;
        allow all;
    }
    location / {
        return 301 https://$host$request_uri;
    }
}

# ---------- HTTPS ----------
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    root /path/to/frontend/dist;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
    gzip_min_length 1024;

    # 后端 API
    location /api/ {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    # GitHub 节点文件同步公开外链（Go 后端从 uploads/nodes 提供）
    location /nodes/ {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 静态资源：Vite 产物文件名带 hash，可长期缓存
    location /assets/ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    # SPA 外壳不做缓存：保证发布后用户拿到最新 index.html（引用新 hash 的 JS）
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
    }

    # SPA 回退
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

> 只使用 HTTP（不配置 SSL）时，删除第一个 80 跳转 server，并把第二个 server 的 `listen 443 ssl http2` 改为 `listen 80`、去掉 `ssl_*` 行即可。

## 生产环境配置

### 安全配置

1. 修改默认密钥
2. 启用 HTTPS
3. 配置防火墙
4. 定期备份数据库

### 性能优化

1. 启用 Redis 缓存
2. 配置 CDN
3. 开启 Gzip/Brotli 预压缩静态资源（`gzip_static on;`，安装 ngx_brotli 后启用 `brotli_static on;`）

## 监控

建议使用：
- Prometheus + Grafana（性能监控）
- Sentry（错误追踪）
