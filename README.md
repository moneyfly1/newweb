# CBoard v2 - 代理订阅管理面板

Go (Gin) + Vue 3 (Naive UI) 构建的现代化代理订阅管理面板。支持 SQLite / MySQL / PostgreSQL，开箱即用。

## ✨ 特性

- 🚀 **开箱即用** - 一键安装脚本，自动配置 Nginx 反向代理
- 💎 **现代化界面** - 基于 Vue 3 + Naive UI，响应式设计，完美支持移动端
- 🔐 **企业级安全** - 经过 5 轮全面安全审计，修复 20+ 个安全漏洞
- 📊 **功能完善** - 用户管理、订单系统、优惠券、盲盒、邀请返佣等
- 🎨 **主题切换** - 支持亮色/暗色主题
- 🌍 **多数据库** - 支持 SQLite（默认）、MySQL、PostgreSQL
- 📱 **移动优先** - 完美适配手机、平板、桌面端

## 📦 快速开始

### 环境要求

- Linux：Ubuntu 20.04+、Debian 11+、CentOS 7+、AlmaLinux 8+、Rocky Linux 8+
- Go 1.22+（安装脚本自动安装）
- Node.js 18+（安装脚本自动安装）
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
```

## 🔧 管理菜单

安装完成后，运行 `bash install.sh` 进入管理菜单：

| 功能 | 说明 |
|------|------|
| 1. 安装系统 | 全新安装，包含所有依赖和配置 |
| 2. 配置域名 & SSL | 修改域名、申请/续期 SSL 证书 |
| 3. 修复常见错误 | 自动修复权限、Nginx 配置等 |
| 4-6. 启动/停止/重启 | 服务管理 |
| 7. 查看服务状态 | 显示运行状态、端口、PID |
| 8. 查看服务日志 | 实时日志/历史日志/错误日志 |
| 9. 重设管理员密码 | 交互式重置密码 |
| 10. 查看管理员账号 | 显示当前管理员邮箱 |
| 11. 备份数据 | 备份数据库和配置文件 |
| 12. 重装网站（保留数据） | 重新编译构建，不丢失数据 |
| 13. 诊断 403 错误 | 检查权限、Nginx、SELinux |
| 14. 更新代码（Git） | 从 GitHub 拉取最新代码 |
| 15. 修复 Nginx SSL | 修复 SSL 证书问题 |
| 16. 诊断网站访问 | 检查 DNS、端口、服务状态 |
| 17. 卸载 CBoard | 停止服务、删除配置 |

## 🎯 核心功能

### 用户端功能

- ✅ 用户注册/登录（支持邮箱、Telegram）
- ✅ 订阅管理（Clash、通用订阅）
- ✅ 套餐购买（标准套餐、自定义套餐、升级套餐）
- ✅ 优惠券系统
- ✅ 余额充值
- ✅ 卡密兑换
- ✅ 每日签到
- ✅ 盲盒抽奖
- ✅ 邀请返佣
- ✅ 工单系统
- ✅ 设备管理
- ✅ 登录历史
- ✅ 通知设置

### 管理端功能

- ✅ 用户管理（创建、编辑、禁用、删除、批量操作）
- ✅ 订单管理（查看、退款）
- ✅ 套餐管理（标准套餐、自定义套餐）
- ✅ 节点管理（公共节点、专线节点、批量导入）
- ✅ 订阅管理（重置、延期、设备限制）
- ✅ 优惠券管理
- ✅ 工单管理
- ✅ 用户等级管理
- ✅ 卡密管理（批量生成）
- ✅ 盲盒管理（奖池、奖品配置）
- ✅ 公告管理
- ✅ 邮件队列管理
- ✅ 数据统计（收入、用户、财务报表）
- ✅ 系统日志（审计、登录、注册、订阅、余额、佣金）
- ✅ 系统设置（基础、邮件、支付、Telegram、备份）

## 🔐 安全特性

经过 **5 轮全面安全审计**，修复 20+ 个安全漏洞：

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

详细安全审计报告：[docs/security-audits/](./docs/security-audits/)

## ⚙️ 配置说明

所有配置通过 `.env` 文件管理：

### 基本配置

```bash
PROJECT_NAME=CBoard
VERSION=2.0.0
BASE_URL=http://localhost:8000
HOST=0.0.0.0
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
SECRET_KEY=自动生成
JWT_ALGORITHM=HS256
ACCESS_TOKEN_EXPIRE_MINUTES=1440
REFRESH_TOKEN_EXPIRE_DAYS=7
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

## 📚 文档

- [安全审计报告](./docs/security-audits/) - 5 轮安全审计详细报告
- [优惠券修复补丁](./docs/security-audits/COUPON_FIX_PATCH.md) - 优惠券竞态条件修复指南
- [API 文档](#) - 后端 API 接口文档（待完善）

## 🔄 版本更新

**已安装过的服务器**，不要删除安装目录、不要重新 `git clone`：

```bash
# 方法 1：通过安装脚本菜单（推荐）
cd /opt/cboard   # 无宝塔；宝塔则为 cd /www/wwwroot/cboard
git pull origin main
bash install.sh
# 选择 14「更新代码 (Git)」

# 方法 2：手动更新
cd /opt/cboard
git pull origin main
go build -o cboard ./cmd/server/
cd frontend && npm install && npm run build && cd ..
systemctl restart cboard
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
- **支付**: 易支付、支付宝、Stripe

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

本项目已经过 5 轮全面安全审计，但仍有 4 个需要手动修复的问题（优惠券竞态条件等），详见 [COUPON_FIX_PATCH.md](./docs/security-audits/COUPON_FIX_PATCH.md)。

---

Made with ❤️ by CBoard Team
