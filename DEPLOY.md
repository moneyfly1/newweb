# CBoard 部署文档

## 环境要求

- Go 1.21+
- Node.js 18+
- SQLite 3 或 MySQL 5.7+
- Redis（可选，用于缓存和限流）

## 快速部署

### 1. 后端部署

```bash
# 编译
go build -o cboard ./cmd/server

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件

# 运行
./cboard
```

### 2. 前端部署

```bash
cd frontend
npm install
npm run build

# 将 dist 目录部署到 Nginx
```

### 3. Nginx 配置

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态资源（Vite 文件名带 hash，可长期缓存）
    location /assets/ {
        root /path/to/frontend/dist;
        access_log off;
        add_header Cache-Control "public, max-age=31536000, immutable";
        gzip_static on;
        # 如已安装 ngx_brotli 模块，开启下一行以直接返回 npm run build 生成的 .br 文件
        # brotli_static on;
        try_files $uri =404;
    }

    # 后端 API
    location /api/ {
        proxy_pass http://localhost:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SPA 入口不做长期缓存，避免发布后用户拿到旧 index.html
    location / {
        root /path/to/frontend/dist;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        try_files $uri $uri/ /index.html;
    }
}
```

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
