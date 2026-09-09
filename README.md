# CBoard v2 - 代理订阅管理面板

Go (Gin) + Vue 3 (Naive UI) 构建的现代化代理订阅管理面板。支持 SQLite / MySQL / PostgreSQL，开箱即用。

## ✨ 特性

- 🚀 **开箱即用** - 一键安装脚本，自动配置 Nginx 反向代理
- 💎 **现代化界面** - 基于 Vue 3 + Naive UI，响应式设计，完美支持移动端
- 🔐 **企业级安全** - 经过多轮全面安全审计，修复 20+ 个安全漏洞
- 📊 **功能完善** - 用户管理、订单系统、优惠券、盲盒、邀请返佣等
- 🎨 **主题切换** - 支持亮色/暗色主题
- 🌍 **多数据库** - 支持 SQLite（默认）、MySQL、PostgreSQL
- 📱 **移动优先** - 完美适配手机、平板、桌面端

## 📦 快速开始

### 环境要求

- Linux：Ubuntu 20.04+、Debian 11+、CentOS 7+、AlmaLinux 8+、Rocky Linux 8+
- Go 1.24+（安装脚本自动安装；`go.mod` 要求 go 1.24）
- Node.js 20+（安装脚本自动安装；前端需 Node 18+ 即可构建）
- Nginx（安装脚本自动安装并配置）
- 磁盘空间 ≥ 1GB

### 方式一：无宝塔面板一键安装（推荐）

适用于纯净 Linux 服务器，安装目录：`/opt/cboard`

```bash
# 一键安装
git clone https://github.com/moneyfly1/newweb.git /opt/cboard
cd /opt/cboard
bash install.sh
```

**首次运行**：脚本会自动进入安装流程，按提示输入域名、SSL、管理员邮箱和密码。

**全自动安装（无人值守）**：
```bash
CBOARD_UNATTENDED=1 \
CBOARD_DOMAIN=your-domain.com \
CBOARD_ADMIN_EMAIL=admin@example.com \
CBOARD_ADMIN_PASSWORD=你的密码 \
bash install.sh
```

### 方式二：宝塔面板一键安装

适用于已安装宝塔面板的服务器，安装目录：`/www/wwwroot/cboard`

```bash
# 一键安装
git clone https://github.com/moneyfly1/newweb.git /www/wwwroot/cboard
cd /www/wwwroot/cboard
bash install_bt.sh
```

### 方式三：手动安装

```bash
# 1. 克隆代码
git clone https://github.com/moneyfly1/newweb.git
cd newweb

# 2. 配置环境变量
cp .env.example .env
vim .env  # 修改配置

# 3. 编译后端
go build -o cboard ./cmd/server/

# 4. 构建前端
cd frontend && npm install && npm run build && cd ..

# 5. 启动服务
./cboard

# 6. 配置 Nginx（反向代理前端 + API）
#    使用仓库内最终标准模板，见下方「🔧 Nginx 配置模板」
```

> 手动部署时请务必使用下方最终 Nginx 模板：除反向代理外，它包含
> **`/index.html` no-cache 规则**（保证发布后用户立即拿到新版页面，
> 不会被旧缓存卡住）与 `/nodes/` 转发（节点文件同步公开外链）。
> 完整部署文档见 [DEPLOY.md](./DEPLOY.md)。

## 🔧 Nginx 配置模板（最终标准版）

与 `install.sh` / `install_bt.sh` 自动生成配置保持一致：

```nginx
# ---------- HTTP：跳转 HTTPS（配置 SSL 时启用） ----------
server {
    listen 80;
    server_name your-domain.com;
    location /.well-known/acme-challenge/ {
        root /path/to/frontend/dist;
        allow all;
    }
    location / { return 301 https://$host$request_uri; }
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
    location / { try_files $uri $uri/ /index.html; }
}
```

> 仅使用 HTTP 时：删除第一个 80 跳转 server，第二个 server 改为 `listen 80` 并移除 `ssl_*` 行。

## 🔧 管理菜单

安装完成后，运行 `bash install.sh`（宝塔版为 `bash install_bt.sh`）进入管理菜单：

**方式一 install.sh（纯净 Linux）菜单：**

| 功能 | 说明 |
|------|------|
| 1. 全新安装系统 | 全新安装，包含所有依赖和配置 |
| 2. 配置域名与 SSL | 修改域名、申请/续期 SSL 证书 |
| 3. 环境健康诊断 | 检查依赖、目录、端口等环境问题 |
| 4/5/6. 启动/停止/重启服务 | 服务管理 |
| 7. 查看运行状态 | 显示服务/反代/SSL 状态 |
| 8. 查阅实时日志 | 实时日志/最近 100 行/错误日志 |
| 9. 重置管理员密码 | 交互式重置密码 |
| 10. 查看当前管理员 | 显示管理员邮箱 |
| 11. 备份本地数据库 | 备份数据库与配置 |
| 12. 强力重装（保留数据） | 重新编译构建，不丢失数据 |
| 14. 热更新代码（Git Pull） | 从 GitHub 拉取最新代码并重编译 |
| 15. Redis 缓存管家 | Redis 安装/状态/清理 |
| 18. 卸载系统 | 停止服务、删除配置 |
| 0. 退出 | 退出管理面板 |

**方式二 install_bt.sh（宝塔面板）菜单：** 1 安装系统、2 配置域名/SSL、3 修复环境、4/5/6 启动/停止/重启、7 服务状态、8 查看日志、9 重设密码、10 备份数据、11 更新代码、12 卸载系统、0 退出。

> 代码热更新：install.sh 选 **14**，install_bt.sh 选 **11**。

## 🎯 核心功能

### 用户端功能

- ✅ 用户注册/登录（支持邮箱、Telegram 登录）
- ✅ 仪表盘（订阅概览、每日签到、数据统计）
- ✅ 订阅管理（Clash / 通用订阅链接、到期/流量/设备查看）
- ✅ 购买套餐（标准套餐、自定义套餐、升级套餐）
- ✅ 我的订单（订单列表、在线支付、取消订单）
- ✅ 通知中心（站内信、未读提醒、通知偏好设置）
- ✅ 工单系统（新建/回复工单，**支持图片/视频/附件上传**，手机端可从相册或文件选择）
- ✅ 节点状态（公共节点、专线节点列表与测速）
- ✅ 我的设备（设备列表、新增/删除设备、备注管理）
- ✅ 邀请返利（邀请链接/返佣记录）
- ✅ 我的优惠券
- ✅ 卡密兑换
- ✅ 盲盒抽奖
- ✅ 余额充值
- ✅ 登录历史
- ✅ 帮助/下载（常见问题、客户端软件下载，自动匹配系统版本）
- ✅ 个人设置（资料修改、修改密码、主题切换）
- ✅ 服务条款 / 隐私政策

### 管理端功能

- ✅ 用户管理（创建、编辑、启用/禁用切换、删除、批量操作、备注）
- ✅ 异常用户（异常/风险用户监控）
- ✅ 订阅管理（重置、延期、设备限制、到期时间设置、邮件通知）
- ✅ 订单管理（查看、退款、取消、批量操作）
- ✅ 套餐管理（标准套餐、自定义套餐）
- ✅ 节点管理（公共节点、专线节点、批量导入、分配用户）
- ✅ 节点自动更新（定时拉取订阅节点配置）
- ✅ 优惠券管理
- ✅ 工单管理（查看/回复用户工单，可查看附件、回复附图片/视频/文件）
- ✅ 用户等级管理
- ✅ 卡密管理（批量生成、回收）
- ✅ 邀请码管理
- ✅ 盲盒管理（奖池、奖品配置）
- ✅ 公告管理
- ✅ 邮件队列管理（重试、删除）
- ✅ 数据统计（收入、用户、区域、财务报表与导出）
- ✅ 系统日志（审计、登录、注册、订阅、余额、佣金）
- ✅ 系统设置（基础、邮件 SMTP、支付网关、Telegram/Bark 通知、备份、GitHub 节点同步、软件下载配置等）

## 🔐 安全特性

经过**多轮全面安全审计**，已修复支付回调重放、CSRF、竞态条件、SQL 注入等 20+ 个安全漏洞（详见下方安全特性列表）：

- ✅ 支付回调重放防护（nonce 机制）
- ✅ 支付金额验证（所有支付方式）
- ✅ Token 刷新安全（黑名单机制）
- ✅ CSRF 防护（中间件 + token）
- ✅ 签到重放防护（事务双重检查）
- ✅ 订阅枚举防护（频率限制 + 日志）
- ✅ 余额转换竞态（事务 + 原子更新）
- ✅ 页码上限（MaxPageNumber = 10000）
- ✅ 优惠券验证频率限制（10次/分钟）
- ✅ 卡密兑换频率限制（5次/分钟）
- ✅ 卡密兑换竞态（行锁 + 事务）
- ✅ 盲盒开启竞态（事务 + 条件更新）
- ✅ 优惠券过期检查（所有位置已修复）
- ✅ SQL 注入防护（参数化查询）
- ✅ 前端敏感信息检查（无泄露）

> 各修复的具体实现与历史评审记录见 [docs/REVIEW_AND_OPTIMIZATION_PLAN.md](./docs/REVIEW_AND_OPTIMIZATION_PLAN.md)。

## ⚙️ 配置说明

所有配置通过 `.env` 文件管理：

### 基本配置

```bash
PROJECT_NAME=CBoard
VERSION=2.0.0
BASE_URL=http://localhost:8000
HOST=0.0.0.0
# 手动运行默认 8000；使用 install.sh / install_bt.sh 安装时脚本固定为 9000
# （与 Nginx 模板 proxy_pass 一致），两种模式端口不同属正常
PORT=8000
DEBUG=false
```

### 数据库配置

```bash
# SQLite（默认，零配置）
DATABASE_URL=sqlite:///./cboard.db

# MySQL
DATABASE_URL=mysql
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=your_password
MYSQL_DATABASE=cboard

# PostgreSQL
DATABASE_URL=postgres
POSTGRES_SERVER=127.0.0.1
POSTGRES_USER=postgres
POSTGRES_PASS=your_password
POSTGRES_DB=cboard
```

### JWT / 安全配置

```bash
# 生产环境必须设置 ≥32 字符的随机密钥；未设置时开发模式自动生成随机密钥
SECRET_KEY=change-me-to-a-strong-random-key-32-bytes
JWT_ALGORITHM=HS256
ACCESS_TOKEN_EXPIRE_MINUTES=1440   # 24 小时
REFRESH_TOKEN_EXPIRE_DAYS=30       # 30 天（代码默认值，.env.example 同）
```

### 邮件 SMTP 配置

```bash
SMTP_HOST=smtp.qq.com
SMTP_PORT=587
SMTP_USERNAME=your_email@qq.com
SMTP_PASSWORD=your_auth_code
SMTP_FROM_EMAIL=your_email@qq.com
SMTP_FROM_NAME=CBoard
SMTP_TLS=true
```

### Telegram 机器人

```bash
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_WEBHOOK_URL=https://your-domain.com/api/telegram/webhook
```

### Redis（可选，用于缓存与限流）

```bash
# 留空则不使用 Redis（走内存实现）；配置后用于缓存与限流
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### 订阅与设备限制

```bash
# 订阅链接前缀（自定义域名/端口时使用）
SUBSCRIPTION_URL_PREFIX=
# 默认设备数限制
DEVICE_LIMIT_DEFAULT=3
# 设备数升级单价（元/月/台）
DEVICE_UPGRADE_PRICE_PER_MONTH=0
```

### 文件上传

```bash
# 上传根目录（相对运行目录）
UPLOAD_DIR=uploads
# 通用上传大小上限（字节，默认 10MB）
MAX_FILE_SIZE=10485760
# 注：工单附件（图片/视频/文档）上限为独立的 20MB 常量，不受 MAX_FILE_SIZE 控制
```

### 调度与性能

```bash
# 是否禁用定时任务（订阅重置/到期提醒等）
DISABLE_SCHEDULE_TASKS=false
# 低配机器优化（减少后台任务开销）
OPTIMIZE_FOR_LOW_END=false
```

### CORS 允许来源（可选，逗号分隔）

```bash
CORS_ORIGINS=http://localhost:5173,http://localhost:3000
```

## 📚 文档

- [部署文档（DEPLOY.md）](./DEPLOY.md) - 手动部署与 Nginx 最终配置模板
- [API 文档（API.md）](./API.md) - 后端接口速查
- [Review & 优化计划](./docs/REVIEW_AND_OPTIMIZATION_PLAN.md) - 历史评审与优化记录

## 🔄 版本更新

**已安装过的服务器**，不要删除安装目录、不要重新 `git clone`：

```bash
# 方法 1：通过安装脚本菜单（推荐）
cd /opt/cboard        # 无宝塔（install.sh）；宝塔版为 cd /www/wwwroot/cboard（install_bt.sh）
git pull origin main
bash install.sh       # 或 bash install_bt.sh
# 菜单热更新代码：install.sh 选 14「热更新代码 (Git Pull)」；install_bt.sh 选 11「更新代码」

# 方法 2：手动更新
cd /opt/cboard
git pull origin main
go build -o cboard ./cmd/server/
cd frontend && npm install && npm run build && cd ..
systemctl restart cboard-v2
```

## 🛠️ 开发

### 后端开发

```bash
# 安装依赖
go mod download

# 运行开发服务器
go run cmd/server/main.go

# 编译
go build -o cboard cmd/server/main.go
```

### 前端开发

```bash
cd frontend

# 安装依赖
npm install

# 运行开发服务器
npm run dev

# 构建生产版本
npm run build
```

## 📊 技术栈

### 后端

- **框架**: Gin (Go Web Framework)
- **ORM**: GORM
- **数据库**: SQLite / MySQL / PostgreSQL
- **认证**: JWT
- **邮件**: SMTP
- **支付**: 易支付（微信/QQ/支付宝）、码支付、支付宝当面付、Stripe
- **通知**: SMTP 邮件、Telegram Bot、Bark
- **工单附件**: 图片/视频/文档上传（本地磁盘存储，鉴权访问）

### 前端

- **框架**: Vue 3 (Composition API)
- **UI 库**: Naive UI
- **状态管理**: Pinia
- **路由**: Vue Router
- **HTTP 客户端**: Axios
- **构建工具**: Vite
- **语言**: TypeScript

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 🙏 致谢

感谢所有贡献者和使用者的支持！

## 📞 联系方式

- GitHub: https://github.com/moneyfly1/newweb
- Issues: https://github.com/moneyfly1/newweb/issues

---

**⚠️ 重要提醒**

1. 首次安装后请立即修改管理员密码
2. 生产环境请配置 SSL 证书
3. 定期备份数据库
4. 关注 GitHub 仓库获取最新更新

**🔒 安全提示**

本项目经过多轮全面安全审计与修复（支付回调重放防护、CSRF、竞态条件、SQL 注入等均已内置），生产使用请遵循上方「重要提醒」：更换默认 SECRET_KEY、启用 SSL、定期备份数据库。

---

Made with ❤️ by CBoard Team
