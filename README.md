# CBoard v2

Go (Gin) + Vue 3 (Naive UI) 构建的代理订阅管理面板。支持 SQLite / MySQL / PostgreSQL，开箱即用。

---

## 目录

- [安装部署](#安装部署)
- [.env 配置说明](#env-配置说明)
- [管理后台使用说明](#管理后台使用说明)
- [用户端页面说明](#用户端页面说明)
- [后端 API 列表](#后端-api-列表)
- [项目结构](#项目结构)
- [常见问题](#常见问题)

---

## 安装部署

**本项目 GitHub：** <https://github.com/moneyfly1/newweb>

### 环境要求

- Linux：Ubuntu 20.04+、Debian 11+、CentOS 7+、AlmaLinux 8+、Rocky Linux 8+
- Go 1.22+（安装脚本自动安装）
- Node.js 18+（安装脚本自动安装）
- Nginx（反向代理，**安装时由脚本自动安装并配置**）
- 磁盘空间 ≥ 1GB

### 方式一：无宝塔面板一键安装

适用于纯净 Linux 服务器。安装目录：`/opt/cboard`。**安装过程中会自动安装并配置 Nginx**（前端静态 + API 反向代理）。

```bash
# 1. 克隆项目（使用本项目 GitHub 地址）
git clone https://github.com/moneyfly1/newweb.git /opt/cboard
cd /opt/cboard

# 2. 运行安装脚本（必须用 bash，需要 root 权限）
bash install.sh
```

> **注意**：请务必使用 `bash install.sh` 运行，不要使用 `sh install.sh`，否则可能因 shell 兼容导致安装中途静默退出。  
> **若 `/opt/cboard` 已存在**（已安装过或想更新代码）：不要删除目录、不要重新执行 `git clone`，否则会报「目录已存在」。请直接 `cd /opt/cboard && git pull && bash install.sh`，在菜单中选 **14 更新代码** 或 **12 重装网站(保留数据)**。

**全自动安装（无人值守）**：不进入菜单、不交互，直接完成安装（需预先设置管理员邮箱和密码）：
```bash
CBOARD_UNATTENDED=1 CBOARD_ADMIN_EMAIL=admin@example.com CBOARD_ADMIN_PASSWORD=你的密码 bash install.sh
```

安装脚本会自动完成以下步骤：
1. 检测系统环境（OS 类型、磁盘空间）
2. 安装系统依赖（Go、Node.js、**Nginx**、Git）
3. 交互式配置：域名、SSL、管理员邮箱和密码
4. 编译 Go 后端二进制
5. 构建 Vue 前端静态文件
6. 生成 `.env` 配置文件
7. **自动配置 Nginx 反向代理**（前端静态文件 + `/api/` 转发到后端）
8. 创建 systemd 服务并启动
9. 配置 Let's Encrypt SSL 证书（可选）
10. 配置防火墙放行端口

安装完成后脚本进入管理菜单，后续再次运行 `bash install.sh` 即可进入菜单。

管理菜单功能列表：

| 编号 | 功能 | 说明 |
|------|------|------|
| 1 | 安装系统 | 全新安装，包含所有依赖和配置 |
| 2 | 配置域名 & SSL | 修改域名、申请/续期 SSL 证书 |
| 3 | 修复常见错误 | 自动修复权限、Nginx 配置等常见问题 |
| 4 | 启动服务 | 启动 CBoard 后端服务 |
| 5 | 停止服务 | 停止 CBoard 后端服务 |
| 6 | 重启服务 | 重启 CBoard 后端服务 |
| 7 | 查看服务状态 | 显示服务运行状态、端口、PID |
| 8 | 查看服务日志 | 实时日志 / 最近 50/100/200 条 / 错误日志 |
| 9 | 重设管理员密码 | 交互式重置管理员密码 |
| 10 | 查看管理员账号 | 显示当前管理员邮箱 |
| 11 | 备份数据 | 备份数据库和配置文件 |
| 12 | 重装网站（保留数据） | 重新编译构建，不丢失数据库和配置 |
| 13 | 诊断 403 错误 | 检查文件权限、Nginx 配置、SELinux 等 |
| 14 | 更新代码（Git） | 从 GitHub 拉取最新代码并重新构建 |
| 15 | 修复 Nginx SSL 验证 | 修复 SSL 证书相关问题 |
| 16 | 诊断网站访问 | 检查 DNS、端口、Nginx、后端服务状态 |
| 17 | 卸载 CBoard | 完全卸载（可选保留数据） |

systemd 服务管理命令：
```bash
systemctl start cboard     # 启动
systemctl stop cboard      # 停止
systemctl restart cboard    # 重启
systemctl status cboard     # 查看状态
journalctl -u cboard -f    # 查看实时日志
```

### 方式二：宝塔面板一键安装

适用于已安装宝塔面板的服务器。安装目录：`/www/wwwroot/cboard`。**安装过程中会自动为站点配置 Nginx**（使用宝塔管理的 Nginx）。

前置要求：
- 已安装宝塔面板
- 已在宝塔面板「软件商店」中安装 Nginx

```bash
# 1. 克隆项目（使用本项目 GitHub 地址）
git clone https://github.com/moneyfly1/newweb.git /www/wwwroot/cboard
cd /www/wwwroot/cboard

# 2. 运行宝塔版安装脚本（需要 root 权限）
bash install_bt.sh
```

> **若 `/www/wwwroot/cboard` 已存在**：不要删除目录、不要重新 `git clone`。请直接 `cd /www/wwwroot/cboard && git pull && bash install_bt.sh`，在菜单中选 **14 更新代码** 或 **12 重装网站(保留数据)**。

安装流程与无宝塔版一致，区别在于：
- 自动检测宝塔面板和宝塔 Nginx 是否已安装
- **自动在宝塔 Nginx 中创建站点配置**（`/www/server/panel/vhost/nginx/`），无需在面板里手动添加反向代理
- 安装目录默认为 `/www/wwwroot/cboard`
- 同样提供 17 项管理菜单

> 注意：宝塔版安装后，不要在宝塔面板中手动修改该站点的 Nginx 配置，以免覆盖脚本生成的反向代理规则。

### 方式三：手动安装

```bash
# 1. 克隆代码（使用本项目 GitHub 地址）
git clone https://github.com/moneyfly1/newweb.git
cd newweb

# 2. 复制配置文件并编辑
cp .env.example .env
vim .env  # 修改配置，详见下方「.env 配置说明」

# 3. 编译后端
go build -o cboard ./cmd/server/

# 4. 构建前端
cd frontend && npm install && npm run build && cd ..

# 5. 启动服务
./cboard
```

也可使用 `start.sh` 管理脚本：
```bash
bash start.sh start      # 启动（自动编译构建）
bash start.sh stop       # 停止
bash start.sh restart    # 重启
bash start.sh rebuild    # 仅重新编译后端
bash start.sh status     # 查看运行状态
bash start.sh logs       # 查看日志
```

### 版本更新

**已安装过的服务器**：**不要删除安装目录、不要重新 `git clone`**。Git 会提示「destination path already exists」，且重复 clone 会覆盖或冲突。正确做法是进入安装目录拉取代码后，用脚本菜单更新或重装：

```bash
# 方法 1：通过安装脚本菜单（推荐）
cd /opt/cboard   # 无宝塔；宝塔则为 cd /www/wwwroot/cboard
git pull origin main
bash install.sh
# 选择 14「更新代码 (Git)」自动拉取、构建、重启；或选 12「重装网站(保留数据)」

# 方法 2：手动更新
cd /opt/cboard   # 无宝塔；或 /www/wwwroot/cboard（宝塔）
git pull origin main
go build -o cboard ./cmd/server/
cd frontend && npm install && npm run build && cd ..
systemctl restart cboard
```

### 默认管理员

安装脚本会提示设置管理员邮箱和密码。手动安装时，首次启动自动创建：
- 邮箱：`admin@example.com`
- 密码：`admin123`

> 首次登录后请立即修改密码。管理后台地址：`https://你的域名/admin/login`

### Nginx 反向代理参考

安装脚本会自动配置 Nginx。手动部署时参考：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态文件
    location / {
        root /opt/cboard/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## .env 配置说明

所有配置通过项目根目录的 `.env` 文件管理。安装脚本会自动生成，手动安装时复制 `.env.example` 为 `.env` 后修改。

### 基本配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PROJECT_NAME` | CBoard | 项目名称，显示在页面标题 |
| `VERSION` | 2.0.0 | 版本号 |
| `BASE_URL` | http://localhost:8000 | 站点访问地址，用于生成回调 URL 等 |
| `HOST` | 0.0.0.0 | 后端监听地址 |
| `PORT` | 8000 | 后端监听端口（安装脚本默认设为 9000） |
| `DEBUG` | false | 调试模式，开启后输出详细日志 |

### 数据库配置

默认使用 SQLite，无需额外配置。自动启用 WAL 模式优化并发。

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

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SECRET_KEY` | 自动生成 | JWT 签名密钥，留空则首次启动自动生成 32 字节随机密钥 |
| `JWT_ALGORITHM` | HS256 | JWT 签名算法 |
| `ACCESS_TOKEN_EXPIRE_MINUTES` | 1440 | access_token 有效期（分钟），默认 24 小时 |
| `REFRESH_TOKEN_EXPIRE_DAYS` | 7 | refresh_token 有效期（天） |

### 邮件 SMTP 配置

| 变量 | 说明 |
|------|------|
| `SMTP_HOST` | SMTP 服务器地址，如 `smtp.qq.com`、`smtp.gmail.com` |
| `SMTP_PORT` | SMTP 端口，587（STARTTLS）或 465（SSL） |
| `SMTP_USERNAME` | SMTP 登录用户名 |
| `SMTP_PASSWORD` | SMTP 密码或授权码（QQ 邮箱需使用授权码） |
| `SMTP_FROM_EMAIL` | 发件人邮箱地址 |
| `SMTP_FROM_NAME` | 发件人显示名称 |
| `SMTP_TLS` | 是否启用 TLS（true/false） |

### Telegram 机器人

| 变量 | 说明 |
|------|------|
| `TELEGRAM_BOT_TOKEN` | Bot Token，从 @BotFather 创建机器人后获取 |
| `TELEGRAM_WEBHOOK_URL` | Webhook 回调地址，格式：`https://你的域名/api/telegram/webhook` |

### 其他配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SUBSCRIPTION_URL_PREFIX` | 空 | 订阅链接自定义前缀域名 |
| `DEVICE_LIMIT_DEFAULT` | 3 | 新用户默认设备数限制 |
| `DEVICE_UPGRADE_PRICE_PER_MONTH` | 0 | 设备数升级每月价格，0 表示不可购买 |
| `UPLOAD_DIR` | uploads | 文件上传存储目录 |
| `MAX_FILE_SIZE` | 10485760 | 最大上传文件大小（字节），默认 10MB |
| `DISABLE_SCHEDULE_TASKS` | false | 禁用所有定时任务 |
| `OPTIMIZE_FOR_LOW_END` | false | 低配服务器优化模式 |
| `CORS_ORIGINS` | 空 | CORS 允许的来源，逗号分隔，开发时设为前端地址 |

---

## 管理后台使用说明

管理后台地址：`https://你的域名/admin/login`，使用管理员账号登录。

### 1. 仪表盘（`/admin`）

概览页面，展示核心运营数据：
- 统计卡片：总用户数、活跃订阅、今日收入、本月收入、待处理订单、待处理工单
- 收入趋势图：近 30 天每日收入柱状图
- 用户增长图：近 30 天每日新增用户折线图
- 最近订单表格：订单号、金额、状态、时间
- 待处理工单列表：工单标题、优先级、时间

### 2. 用户管理（`/admin/users`）

| 操作 | 说明 |
|------|------|
| 搜索 | 按邮箱、用户名搜索用户 |
| 创建用户 | 手动创建新用户，设置邮箱、密码、用户名 |
| 编辑用户 | 修改用户信息、余额、等级 |
| 启用/禁用 | 切换用户账户状态 |
| 重置密码 | 为用户设置新密码 |
| 登录为用户 | 以该用户身份登录前台（调试用） |
| 删除设备 | 删除用户的指定设备记录 |
| 完全删除 | 删除用户及其所有关联数据（订阅、订单等） |
| 批量操作 | 批量启用/禁用/删除选中的用户 |
| CSV 导出 | 导出用户列表为 CSV 文件 |
| CSV 导入 | 通过 CSV 文件批量导入用户 |

CSV 导入格式：`email,username,password`（每行一个用户）

### 3. 异常用户检测（`/admin/abnormal-users`）

自动检测异常行为的用户列表，如：短时间内大量请求、设备数异常等。

### 4. 订单管理（`/admin/orders`）

| 操作 | 说明 |
|------|------|
| 搜索 | 按订单号、用户邮箱搜索 |
| 状态筛选 | 全部 / 待支付 / 已支付 / 已取消 / 已退款 |
| 查看详情 | 订单号、金额、套餐、优惠券、支付方式、时间 |
| 退款 | 对已支付订单执行退款（余额退回用户账户） |

### 5. 套餐管理（`/admin/packages`）

| 字段 | 说明 |
|------|------|
| 名称 | 套餐显示名称 |
| 价格 | 套餐价格（元） |
| 时长 | 订阅天数 |
| 设备限制 | 该套餐允许的最大同时在线设备数 |
| 流量限制 | 流量上限（GB），0 表示不限 |
| 排序 | 数字越小越靠前 |
| 上架/下架 | 控制是否在用户端商店显示 |

操作：创建、编辑、删除套餐。

### 6. 节点管理（`/admin/nodes`）

| 字段 | 说明 |
|------|------|
| 名称 | 节点显示名称 |
| 地址 | 服务器地址 |
| 端口 | 服务端口 |
| 协议 | VMess / VLESS / Trojan / Shadowsocks / SSR / Hysteria2 |
| 国家 | 节点所在国家（用于前端筛选和国旗显示） |
| 速率 | 节点速率标注 |
| 排序 | 数字越小越靠前 |
| 健康状态 | 在线/离线 |

操作：创建、编辑、删除、批量导入（粘贴订阅链接）、单节点连通性测试。

### 7. 专线节点管理（`/admin/custom-nodes`）

专线节点可分配给指定用户，不在公共节点列表中显示。

| 操作 | 说明 |
|------|------|
| 创建 | 手动添加专线节点 |
| 批量导入 | 粘贴多条订阅链接批量导入 |
| 分配用户 | 将节点分配给指定用户 |
| 查看用户 | 查看节点已分配的用户列表 |
| 导出链接 | 导出节点的订阅链接 |
| 批量删除 | 选中多个节点批量删除 |

### 8. 订阅管理（`/admin/subscriptions`）

| 操作 | 说明 |
|------|------|
| 搜索 | 按用户邮箱搜索订阅 |
| 重置订阅 URL | 重新生成订阅链接（旧链接失效） |
| 延期 | 为订阅增加天数 |
| 设置到期时间 | 直接指定到期日期 |
| 修改设备限制 | 调整该订阅的设备数上限 |
| 发送邮件 | 向用户发送订阅信息邮件 |

### 9. 优惠券管理（`/admin/coupons`）

| 字段 | 说明 |
|------|------|
| 名称 | 优惠券名称 |
| 代码 | 用户输入的优惠码 |
| 类型 | `discount`（折扣百分比）/ `fixed`（固定金额）/ `free_days`（免费天数） |
| 值 | 折扣百分比(1-100) / 固定金额(元) / 免费天数 |
| 最低消费 | 订单金额达到此值才可使用 |
| 最大折扣 | 折扣类型的最大优惠金额上限 |
| 每人限用 | 每个用户最多使用次数 |
| 总数量 | 优惠券总可用次数 |
| 有效期 | 开始时间 ~ 结束时间 |
| 适用套餐 | 限定可使用的套餐（空表示全部适用） |

### 10. 工单管理（`/admin/tickets`）

| 操作 | 说明 |
|------|------|
| 状态筛选 | 待处理 / 处理中 / 已解决 / 已关闭 |
| 回复工单 | 以管理员身份回复（聊天气泡样式） |
| 修改状态 | 更新工单状态 |
| 修改优先级 | 低 / 普通 / 高 / 紧急 |

### 11. 用户等级管理（`/admin/levels`）

| 字段 | 说明 |
|------|------|
| 等级名称 | 如：普通、VIP、SVIP |
| 等级编号 | 数字标识 |
| 折扣率 | 该等级用户购买套餐的折扣（0.01-1.0，1.0 表示无折扣） |
| 设备限制 | 该等级的默认设备数限制 |

### 12. 卡密管理（`/admin/redeem`）

| 操作 | 说明 |
|------|------|
| 批量生成 | 设置类型（余额/套餐）、面值、数量，一键生成 |
| 查看列表 | 卡密代码、类型、面值、状态（未使用/已使用）、使用者 |
| 删除 | 删除未使用的卡密 |

### 13. 盲盒管理（`/admin/mystery-box`）

奖池管理：
| 字段 | 说明 |
|------|------|
| 名称 | 奖池名称 |
| 价格 | 开启一次的价格（元） |
| 描述 | 奖池描述文字 |
| 每日限次 | 每个用户每天最多开启次数 |
| 总限次 | 每个用户累计最多开启次数 |
| 最低等级 | 要求的最低用户等级 |
| 最低余额 | 要求的最低账户余额 |

奖品管理（每个奖池下配置多个奖品）：
| 字段 | 说明 |
|------|------|
| 名称 | 奖品名称 |
| 类型 | `balance`（余额）/ `coupon`（优惠券）/ `subscription_days`（订阅天数）/ `nothing`（谢谢参与） |
| 值 | 奖品数值（余额金额/天数等） |
| 概率权重 | 数字越大中奖概率越高 |

### 14. 公告管理（`/admin/announcements`）

| 字段 | 说明 |
|------|------|
| 标题 | 公告标题 |
| 内容 | 公告正文（支持富文本） |
| 类型 | info / warning / success / error |
| 状态 | 启用/禁用 |

### 15. 邮件队列（`/admin/email-queue`）

| 功能 | 说明 |
|------|------|
| 统计卡片 | 总邮件数、待发送、已发送、发送失败 |
| 状态筛选 | 全部 / 待发送 / 已发送 / 发送失败 |
| 查看详情 | 收件人、主题、邮件类型、内容类型、重试次数、错误信息、邮件正文预览 |
| 重试 | 对发送失败的邮件重新加入发送队列 |
| 删除 | 删除邮件记录 |

### 16. 数据统计（`/admin/stats`）

- 收入统计：总收入、今日/本月收入、支付方式分布饼图
- 用户统计：总用户、活跃用户、新增用户、付费用户
- 财务报表：按日期范围查询收入明细，支持导出 CSV

### 17. 系统日志（`/admin/logs`）

6 个分类 Tab：
| Tab | 说明 |
|-----|------|
| 审计日志 | 管理员操作记录（修改设置、删除用户等） |
| 登录日志 | 所有用户的登录记录（IP、设备、时间、成功/失败） |
| 注册日志 | 新用户注册记录 |
| 订阅日志 | 订阅创建、续期、重置、过期记录 |
| 余额日志 | 余额变动记录（充值、消费、退款、签到奖励等） |
| 佣金日志 | 邀请佣金记录 |

### 18. 配置更新（`/admin/config-update`）

远程配置更新功能，用于从远程源同步节点配置。

| 操作 | 说明 |
|------|------|
| 查看状态 | 当前更新任务的运行状态 |
| 配置 | 设置远程配置源地址和更新参数 |
| 启动/停止 | 启动或停止自动更新任务 |
| 查看日志 | 查看更新任务的执行日志 |
| 清除日志 | 清空历史日志 |

### 19. 系统设置（`/admin/settings`）

系统设置分为 8 个 Tab 页：

<!-- PLACEHOLDER_SECTION_3 -->
