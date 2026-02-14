# CBoard v2.0 项目说明文档

## 一、项目简介

CBoard 是一个代理订阅聚合管理平台。核心理念是**不对接任何代理服务器**，而是通过采集/导入节点信息，为用户提供聚合订阅地址，并通过设备指纹识别来控制同时在线设备数量。

适用场景：将机场资源分享给朋友，通过设备限制和订阅重置机制防止滥用。

## 二、设计理念

### 2.1 架构设计

采用前后端分离架构，最终通过 `go:embed` 打包为单一二进制文件部署。

```
v2/
├── cmd/server/main.go      # 程序入口
├── internal/                # 后端全部代码
│   ├── api/
│   │   ├── handlers/        # 请求处理器（15 个文件）
│   │   ├── middleware/       # 中间件（认证、限流、安全头）
│   │   └── router/          # 路由注册
│   ├── config/              # 配置加载（Viper）
│   ├── database/            # 数据库初始化与迁移
│   ├── models/              # GORM 数据模型（19 个文件）
│   └── utils/               # 工具函数（响应、分页、加密）
├── frontend/                # 前端全部代码
│   ├── src/
│   │   ├── api/             # API 请求封装（8 个文件）
│   │   ├── layouts/         # 布局组件（用户端 + 管理端）
│   │   ├── router/          # 路由配置
│   │   ├── stores/          # Pinia 状态管理
│   │   ├── utils/           # Axios 封装
│   │   └── views/           # 页面组件（25 个视图）
│   ├── package.json
│   └── vite.config.ts
├── go.mod
└── .env.example
```

### 2.2 技术选型

| 层级 | 技术 | 选型理由 |
|------|------|----------|
| 后端框架 | Go 1.24 + Gin | 高性能、单二进制部署、内存占用低 |
| ORM | GORM | Go 生态最成熟的 ORM，支持多数据库 |
| 数据库 | SQLite / MySQL / PostgreSQL | SQLite 零配置适合小规模，MySQL/PG 适合生产 |
| 前端框架 | Vue 3 + TypeScript | 组合式 API、类型安全、生态丰富 |
| UI 组件库 | Naive UI | 原生 TypeScript、Tree-shaking 友好、主题定制强 |
| 状态管理 | Pinia | Vue 3 官方推荐，轻量直观 |
| 构建工具 | Vite 5 | 极速 HMR、原生 ESM |
| 认证 | JWT 双令牌 | access_token + refresh_token，支持令牌黑名单 |
| 配置 | Viper | 支持 .env / 环境变量 / 配置文件 |

### 2.3 前端设计理念

v2 前端完全重新设计，不复用 v1 的 Element Plus 代码：

- **卡片化布局**：所有数据展示使用圆角卡片（border-radius: 12px），层次分明
- **渐变色彩系统**：统计卡片使用渐变色调背景，视觉区分度高
- **响应式网格**：n-grid 自适应列数（手机 1 列、平板 2 列、桌面 3-6 列）
- **深色模式**：全局一键切换，Naive UI 原生支持
- **分屏登录页**：左侧品牌展示区（渐变背景 + 特性列表），右侧表单区
- **聊天式工单**：工单回复采用气泡对话样式，用户蓝色靠右、管理员灰色靠左
- **悬浮交互**：卡片 hover 上浮 + 阴影加深，操作反馈明确

## 三、功能模块

### 3.1 用户系统

| 功能 | 说明 |
|------|------|
| 注册 | 邮箱 + 密码 + 用户名，支持邀请码 |
| 登录 | 邮箱 + 密码，返回 JWT 双令牌 |
| 令牌刷新 | refresh_token 换取新 access_token |
| 登出 | 令牌加入黑名单 |
| 忘记密码 | 邮箱验证码重置 |
| 个人资料 | 修改用户名、密码、偏好设置 |
| 通知设置 | 邮件通知、登录提醒等开关 |
| 隐私设置 | 隐藏在线状态等 |
| 用户等级 | 不同等级享受不同折扣率和设备限制 |
| 登录历史 | 记录 IP、设备、地理位置 |
| 活动记录 | 用户操作行为追踪 |

### 3.2 订阅系统

| 功能 | 说明 |
|------|------|
| 聚合订阅 | 生成唯一订阅 URL，聚合所有可用节点 |
| 多格式输出 | Clash YAML、V2Ray Base64、Surge、Quantumult X、Shadowrocket、Stash |
| 设备指纹 | SHA256(UserAgent + IP) 识别设备 |
| 设备限制 | 根据套餐/等级限制同时在线设备数 |
| 设备管理 | 查看已连接设备，可删除指定设备 |
| 订阅重置 | 重新生成订阅 URL（旧地址失效） |
| 转余额 | 将剩余订阅天数折算为账户余额 |

### 3.3 节点管理

| 功能 | 说明 |
|------|------|
| 节点列表 | 展示所有可用节点，按国家/协议筛选 |
| 协议支持 | VMess、VLESS、Trojan、Shadowsocks、SSR、Hysteria2 |
| 节点属性 | 名称、地址、端口、国家、速率、排序、健康状态 |
| 专线节点 | CustomNode 支持分配给指定用户 |
| 节点导入 | 批量导入节点（TODO） |
| 节点测速 | 健康检查与测速（TODO） |

### 3.4 套餐与订单

| 功能 | 说明 |
|------|------|
| 套餐管理 | 名称、价格、时长、设备限制、流量限制、排序 |
| 创建订单 | 选择套餐 + 可选优惠券，生成待支付订单 |
| 余额支付 | 扣减账户余额，自动创建/续期订阅 |
| 订单过期 | 30 分钟未支付自动过期 |
| 取消订单 | 用户可取消待支付订单 |
| 订单退款 | 管理员可退款已支付订单 |

### 3.5 支付系统

| 功能 | 说明 |
|------|------|
| 余额支付 | 已实现，直接扣减余额 |
| 支付宝 | 支付配置模型已建，对接待实现 |
| 微信支付 | 支付配置模型已建，对接待实现 |
| 易支付 | 支付配置模型已建，对接待实现 |
| 支付回调 | PaymentCallback 模型记录回调数据 |

### 3.6 优惠券系统

| 功能 | 说明 |
|------|------|
| 优惠券类型 | 折扣（百分比）、固定金额、免费天数 |
| 使用限制 | 最低消费、最大折扣、每人使用次数、总数量 |
| 有效期 | valid_from ~ valid_until 时间范围 |
| 适用套餐 | 可限定适用的套餐范围 |
| 验证接口 | 下单前验证优惠券有效性 |

### 3.7 卡密兑换（v2 新增）

| 功能 | 说明 |
|------|------|
| 卡密类型 | 余额充值、套餐兑换 |
| 批量生成 | 管理员可一次生成多个卡密 |
| 兑换记录 | 记录兑换用户、时间、金额 |

### 3.8 邀请返利

| 功能 | 说明 |
|------|------|
| 生成邀请码 | 设置最大使用次数、过期天数、双方奖励金额 |
| 邀请统计 | 总邀请人数、总奖励金额 |
| 邀请关系 | 记录邀请人与被邀请人的关系 |

### 3.9 工单系统

| 功能 | 说明 |
|------|------|
| 工单类型 | 技术问题、账单问题、账户问题、其他 |
| 优先级 | 低、普通、高、紧急 |
| 状态流转 | 待处理 → 处理中 → 已解决 → 已关闭 |
| 聊天式回复 | 用户和管理员的对话式交互 |
| 工单评价 | 用户可对已解决工单评分 |
| 附件支持 | 模型已建，上传待实现 |

### 3.10 通知系统

| 功能 | 说明 |
|------|------|
| 站内通知 | 列表、未读计数、标记已读、全部已读 |
| 邮件模板 | 可配置的邮件模板 |
| 邮件队列 | 异步发送，失败可重试 |

### 3.11 充值系统

| 功能 | 说明 |
|------|------|
| 充值记录 | 用户发起充值请求 |
| 充值取消 | 用户可取消待处理的充值 |

### 3.12 管理后台

| 模块 | 功能 |
|------|------|
| 仪表盘 | 用户数、订阅数、收入、待处理订单/工单、最近订单、待处理工单 |
| 用户管理 | 搜索、编辑、禁用/启用、删除、异常用户检测 |
| 订单管理 | 搜索、查看详情、退款 |
| 套餐管理 | CRUD、上下架、排序 |
| 节点管理 | CRUD、协议配置、国家设置 |
| 专线节点 | CRUD、分配给指定用户 |
| 订阅管理 | 搜索、重置 URL、延期 |
| 优惠券管理 | CRUD、类型/金额/有效期配置 |
| 工单管理 | 筛选、回复、状态更新 |
| 用户等级 | CRUD、折扣率/设备限制配置 |
| 卡密管理 | 批量生成、删除 |
| 邮件队列 | 查看状态、失败重试 |
| 系统设置 | 键值对配置管理 |
| 公告管理 | CRUD、类型/状态控制 |
| 数据统计 | 收入统计（总/日/月）、用户统计（总/活跃/新增/付费） |
| 系统日志 | 审计日志、登录日志、注册日志、订阅日志、余额日志、佣金日志 |
| 系统监控 | 用户数、节点数、活跃订阅、待处理工单/订单 |
| 数据备份 | 创建/列出备份（TODO） |

## 四、数据模型

共 39 个数据表，分布在 19 个模型文件中：

```
models/
├── user.go           # User, UserLevel
├── subscription.go   # Subscription, SubscriptionReset
├── device.go         # Device
├── node.go           # Node, CustomNode, UserCustomNode
├── order.go          # Order
├── package.go        # Package
├── payment.go        # PaymentTransaction, PaymentCallback, PaymentConfig
├── coupon.go         # Coupon, CouponUsage
├── invite.go         # InviteCode, InviteRelation
├── notification.go   # Notification, EmailTemplate, EmailQueue
├── ticket.go         # Ticket, TicketReply, TicketAttachment, TicketRead
├── recharge.go       # RechargeRecord
├── config.go         # SystemConfig, Announcement, ThemeConfig
├── security.go       # LoginAttempt, VerificationAttempt, VerificationCode, TokenBlacklist
├── activity.go       # UserActivity, LoginHistory
├── logs.go           # RegistrationLog, SubscriptionLog, BalanceLog, CommissionLog
├── audit_log.go      # AuditLog
└── redeem.go         # RedeemCode, RedeemRecord
```

## 五、API 设计

### 5.1 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

分页响应：
```json
{
  "code": 0,
  "message": "success",
  "data": [],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

### 5.2 API 路由总览

| 分组 | 前缀 | 认证 | 端点数 |
|------|------|------|--------|
| 认证 | /api/auth | 否 | 8 |
| 公共 | /api/config, /api/packages | 否 | 3 |
| 用户 | /api/user | JWT | 14 |
| 订阅 | /api/subscription | JWT | 8 |
| 订单 | /api/orders | JWT | 5 |
| 支付 | /api/payment | JWT | 3 |
| 卡密 | /api/redeem | JWT | 2 |
| 节点 | /api/nodes | JWT | 3 |
| 设备 | /api/devices | JWT | 2 |
| 优惠券 | /api/coupons | JWT | 2 |
| 通知 | /api/notifications | JWT | 5 |
| 工单 | /api/tickets | JWT | 5 |
| 邀请 | /api/invite | JWT | 4 |
| 充值 | /api/recharge | JWT | 3 |
| 订阅获取 | /api/subscribe/:url | 否 | 1 |
| 管理后台 | /api/admin/* | JWT+Admin | 50+ |

## 六、安全设计

- **密码存储**：bcrypt 哈希，cost=10
- **JWT 认证**：access_token（短期）+ refresh_token（长期），支持令牌黑名单
- **请求限流**：基于 IP 的内存限流器，可配置速率和窗口
- **安全头**：X-Content-Type-Options、X-Frame-Options、X-XSS-Protection、Referrer-Policy
- **CORS**：可配置允许的域名
- **登录保护**：记录登录尝试，支持异常检测
- **验证码**：邮箱验证码，有过期时间和使用次数限制
- **设备指纹**：SHA256 哈希，防止订阅链接被随意分享

## 七、部署方式

### 7.1 环境变量配置

复制 `.env.example` 为 `.env`，修改关键配置：

```bash
# 数据库（默认 SQLite）
DB_TYPE=sqlite
DB_PATH=./data/cboard.db

# JWT 密钥（留空自动生成）
SECRET_KEY=

# 服务端口
PORT=8080

# SMTP 邮件（可选）
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=
```

### 7.2 编译运行

```bash
# 后端编译
cd v2
go build -o cboard ./cmd/server

# 前端编译
cd frontend
npm install
npm run build

# 运行
./cboard
```

默认管理员账户：`admin@example.com` / `admin123`

### 7.3 多数据库支持

```bash
# SQLite（默认）
DB_TYPE=sqlite
DB_PATH=./data/cboard.db

# MySQL
DB_TYPE=mysql
DB_DSN=user:pass@tcp(127.0.0.1:3306)/cboard?charset=utf8mb4&parseTime=True

# PostgreSQL
DB_TYPE=postgres
DB_DSN=host=127.0.0.1 user=postgres password=pass dbname=cboard port=5432 sslmode=disable
```

## 八、前端页面清单

### 用户端（11 个页面）

| 页面 | 路由 | 说明 |
|------|------|------|
| 登录 | /login | 分屏设计，左侧品牌展示 |
| 注册 | /register | 支持邀请码 |
| 仪表盘 | / | 统计卡片 + 快捷操作 + 公告 |
| 我的订阅 | /subscription | 订阅状态 + 格式选择 + 设备管理 |
| 购买套餐 | /shop | 定价卡片网格 + 优惠券 |
| 我的订单 | /orders | 订单列表 + 支付/取消操作 |
| 工单列表 | /tickets | 工单列表 + 新建工单 |
| 工单详情 | /tickets/:id | 聊天气泡式回复 |
| 节点状态 | /nodes | 节点卡片网格 + 筛选 |
| 我的设备 | /devices | 设备表格 + 删除 |
| 邀请返利 | /invite | 统计 + 生成邀请码 + 列表 |

### 管理端（14 个页面）

| 页面 | 路由 | 说明 |
|------|------|------|
| 仪表盘 | /admin | 6 项统计 + 最近订单 + 待处理工单 |
| 用户管理 | /admin/users | 搜索 + 编辑 + 禁用 + 删除 |
| 订单管理 | /admin/orders | 搜索 + 退款 |
| 套餐管理 | /admin/packages | CRUD |
| 节点管理 | /admin/nodes | CRUD + 协议配置 |
| 订阅管理 | /admin/subscriptions | 搜索 + 重置 + 延期 |
| 优惠券 | /admin/coupons | CRUD + 类型配置 |
| 工单管理 | /admin/tickets | 筛选 + 回复 + 状态更新 |
| 用户等级 | /admin/levels | CRUD |
| 卡密管理 | /admin/redeem | 批量生成 + 删除 |
| 系统设置 | /admin/settings | 键值对配置 |
| 公告管理 | /admin/announcements | CRUD |
| 数据统计 | /admin/stats | 收入 + 用户统计 |
| 系统日志 | /admin/logs | 6 类日志分 Tab 展示 |

## 九、待实现功能

以下功能模型和接口已建好，业务逻辑待完善：

| 功能 | 状态 | 说明 |
|------|------|------|
| 邮件发送 | TODO | SMTP 配置已有，发送逻辑待实现 |
| 支付宝/微信/易支付 | TODO | 支付配置模型已建，网关对接待实现 |
| USDT/加密货币支付 | TODO | v2 新增计划 |
| 节点测速/健康检查 | TODO | 接口已建，检测逻辑待实现 |
| 节点批量导入 | TODO | 接口已建，解析逻辑待实现 |
| 订阅格式转换 | TODO | Surge/QuanX/Shadowrocket 格式待实现 |
| Telegram Bot | TODO | v2 新增计划 |
| 数据备份 | TODO | 接口已建，备份逻辑待实现 |
| 工单附件上传 | TODO | 模型已建，文件处理待实现 |
| go:embed 前端 | TODO | 生产部署时内嵌前端静态文件 |

