# CBoard v2 全面审查报告与优化计划

> 审查日期：2025（本次会话）
> 审查范围：Go 后端 `internal/`（~3.2 万行）+ Vue3 前端 `frontend/src/`（~2.6 万行）
> 审查方式：核心文件人工深度审查（payment.go 全量、auth/order/admin 关键路径、middleware、request.ts、stores、layouts）+ 4 个并行子代理分片审查（后端安全 / 后端复用 / 前端基础设施 / 前端商业化 UX），全部结论经代码级复核。
> 状态：本报告列出的「✅ 已修复」项均已在本次会话实施并通过 `go build ./...` 与 `vue-tsc --noEmit` 验证。

---

## 一、已修复问题（本次会话实施，共 17 项）

### 后端（15 项，全部编译通过）

| # | 级别 | 问题 | 位置 | 修复方式 |
|---|------|------|------|----------|
| 1 | 🔴 P0 | 支付回调通用兜底分支**完全无验签**：任何人可伪造 `POST /api/v1/payment/notify/<任意type>`（如 wxpay）携带自己下单拿到的 transaction_id，即可把 pending 订单标记为 paid 并激活订阅，**免费开通服务** | `internal/api/handlers/payment.go` PaymentNotify | 删除通用 JSON 分支，只保留 epay/alipay/stripe/codepay 四个验签入口，未知 type 一律 400 + 安全日志 |
| 2 | 🔴 P0 | `AdminRefundOrder` 双重退款竞态：状态检查无锁、网关退款先于 DB 事务、状态更新无条件 → 并发可**网关双退/余额双倍入账** | `internal/api/handlers/admin.go` | 事务前原子抢占 `paid/completed → refunding`（RowsAffected 校验），失败恢复原状态 |
| 3 | 🔴 P0 | `ConvertToBalance` 订阅折现并发**双倍入账**（无行锁、置 disabled 无条件） | `internal/api/handlers/subscription.go` | 置 disabled 加 `WHERE status='active'` + RowsAffected 校验，冲突返回「已转换」 |
| 4 | 🔴 P0 | `PayOrder` 余额支付状态机非原子：并发支付**双扣款 + 订阅双倍延期** | `internal/api/handlers/order.go` | 改为「抢订单」模式：条件更新 `WHERE status='pending'` + RowsAffected 校验后再扣款 |
| 5 | 🔴 P0 | 签到「先查后插」无唯一约束，并发**双签双发奖励**（SQLite/MySQL 行为不一致） | `internal/api/handlers/checkin.go` + `internal/models/checkin.go` + `internal/database/database.go` | 新增 `(user_id, check_in_date)` 复合唯一索引（含历史数据回填与去重迁移），INSERT 冲突返回已签到 |
| 6 | 🔴 P0 | 取消订单/充值无状态守卫，可**覆盖并发支付回调刚置为 paid 的订单**（已付款却显示已取消） | `order.go` / `admin.go` / `recharge.go` 三处 | 全部改为条件更新 + RowsAffected 校验 |
| 7 | 🟠 高 | `AdminMarkOrderPaid` 及批量 mark_paid 重复开通：无条件状态更新 → 并发可**重复叠加订阅权益** | `internal/api/handlers/admin.go` | 条件更新 `WHERE status IN ('pending','expired','cancelled')` + 行数校验 |
| 8 | 🟠 高 | `AdminGetSettings` 密钥**明文回显**（支付私钥/SMTP 密码/Bot Token 常驻管理端 JS） | `internal/api/handlers/admin.go` | 敏感键掩码回显（`****`+末4位），提交掩码值时跳过更新保留原值 |
| 9 | 🟠 高 | `AdminRestoreGitHubBackup`：download_url 客户端可控 → **SSRF + GitHub Token 外泄 + 无下载大小限制** | `internal/services/git/git.go` | 域名白名单（仅 GitHub 系列 https 域名）+ 200MB 下载上限 |
| 10 | 🟠 高 | `AdminDeleteUser` 无自保：可删除自己/其他管理员账号 | `internal/api/handlers/admin.go` | 禁止删除自己与管理员账号 |
| 11 | 🟠 高 | `GetNode` 脱敏逻辑恒不生效（路由在 authorized 组），**任意注册用户可枚举全部节点真实服务器地址** | `internal/api/handlers/node.go` | 与 ListNodes 一致：仅对有效订阅用户返回 Config |
| 12 | 🟡 中 | 卡密兑换 `SELECT ... FOR UPDATE` 在 SQLite（默认库）下无效，并发可**重复兑换同一卡密** | `internal/api/handlers/redeem.go` | 改为原子条件更新（按旧 used_count 抢占）+ 哨兵错误统一响应 |
| 13 | 🟢 低 | 公开调试端点 `/api/v1/payment/test-callback` 生产残留 | `internal/api/router/router.go` | 删除 |
| 14 | 🟠 高 | 自定义套餐「月→天」换算不一致：余额支付 30.44 vs 网关支付 30 → **同一订单不同支付方式天数不同** | `order.go` + `services/subscription.go` | 抽取 `services.CustomPackageDurationDays()` 统一为 30.44 |
| 15 | 🟠 高 | `payment_stats.go` 使用 MySQL 专属函数 `TIMESTAMPDIFF`/`HOUR()`，**默认 SQLite 部署下这两个统计接口必 500** | `internal/api/handlers/payment_stats.go` | 方言感知 SQL 表达式（SQLite 用 strftime 换算） |

### 前端（2 项，vue-tsc 通过）

| # | 级别 | 问题 | 位置 | 修复方式 |
|---|------|------|------|----------|
| 16 | 🔴 P0 | 401 并发刷新竞态：多个请求同时 401 时，第二个直接走**误登出**（pendingRequests 队列写了未接线） | `frontend/src/utils/request.ts` | 刷新中的并发 401 挂入队列，刷新成功后统一用新 token 重放 |
| 17 | 🔴 P0 | 网络错误对**写请求自动重试**：下单/充值接口超时即重发 → **重复下单/重复扣款** | `frontend/src/utils/request.ts` | 仅对幂等 GET/HEAD 自动重试 |

### 后续轮次追加修复（本轮会话第 2 轮，4 项，`go build` + `go test` 全部通过）

| # | 级别 | 问题 | 位置 | 修复方式 |
|---|------|------|------|----------|
| 18 | 🟠 高 | `AdminListOrders` 全表载入内存再 Go 侧排序分页 → 订单量大时管理端 OOM/超时 | `internal/api/handlers/admin.go` AdminListOrders | 改为 SQL 层 `UNION ALL` 合并 orders+recharge_records，排序/过滤/分页（LIMIT/OFFSET）全部下沉数据库，Count 单独 SQL |
| 19 | 🟠 高 | 订阅延长 5 处「读-改-写」并发丢更新（兑换/盲盒/余额续期/升级/管理端延长同时操作会互相覆盖） | `redeem.go`、`mysterybox.go`、`order.go`（升级+续期）、`admin.go` AdminExtendSubscription | 新增 `services.ExtendSubscriptionExpiry`（乐观锁 WHERE expire_time=旧值 + RowsAffected 校验）与 `services.AddSubscriptionDevices`（原子 `device_limit + ?`），5 处全部接入 |
| 20 | 🟠 高 | 订阅 URL 三种格式并存（`/api/v1/sub/`、`/api/v1/subscribe/`、`/api/v1/client/subscribe?token=`），邮件/页面链接不一致 | `services/subscription.go`、`order.go`、`subscription.go handler`、`admin.go` 共 7 处 | 新增 `services.BuildSubscriptionURL(token, type)`（权威格式 `/api/v1/client/subscribe?token=`），全部生成点收敛 |
| 21 | 🟢 低 | 站点 URL 解析 4 份拷贝 + 3 套缓存（getSubscriptionBaseURL / getSubscriptionSiteConfig / GetSiteURL / order.go 内联） | `subscription.go handler`、`admin.go`、`services/alipay.go` | 删除 `getSubscriptionBaseURL`，`getSubscriptionSiteConfig` 复用 `services.GetSiteURL()`，删除第三套 configCache |

### 目标第 3 轮追加（死代码清理 + 前端组件化第一步，`go build`/`go vet`/`go test`/`vue-tsc` 全部通过）

| # | 级别 | 内容 | 位置 |
|---|------|------|------|
| 22 | 🟢 低 | 死代码清理：删除 **6 个零调用文件**（`pagination.go.bak`、`utils/metrics.go`、`cache/config_cache.go`、`utils/sanitize.go`、`utils/security.go`（CreateSecurityEvent/DetectSuspiciousActivity/ValidateAdminAction/CreateAdminAuditLog 死链）、`utils/helpers.go`（全文件零引用））+ **7 个死函数**（`ApplyLogContext`、`CleanExpiredNonces`、`GetPaymentSettings`、`IsPaymentEnabled`、`convertSSToSurge`、`convertTrojanToSurge`、`GetSubscriptionByFormat`（未注册路由））。保留活跃的 `CreateSecurityLog`（audit_logs.go）安全日志能力 | 见左 |
| 23 | 🟠 高 | 前端组件化第一步：新建 `useTable` composable（统一 loading/data/分页/排序/批量选择状态），迁移 `admin/announcements` 与 `admin/redeem` 两个页面验证（净减 ~80 行，模式可复用） | `frontend/src/composables/useTable.ts` + 2 个页面 |

### 目标第 4 轮追加（useTable 铺开，`vue-tsc` 0 错误）

`useTable` 升级支持 `getParams`（搜索/筛选参数）与 `reload`（重置到第一页重载）。**累计 6 个 admin 页面迁移完成**（announcements、redeem、levels、coupons、tickets、packages），净减 ~194 行重复代码，含筛选场景（tickets 的状态/优先级筛选、packages 的搜索）。剩余可迁移：users/orders/subscriptions/nodes/custom-nodes/logs/email-queue/mystery-box/abnormal-users 等（部分页面含双表格或多表格，需按场景适配）。

### 目标第 5 轮追加（前端视觉 F2/F3 核心落地，`vue-tsc` 0 错误 + `vite build` 通过）

| # | 内容 | 说明 |
|---|------|------|
| 24 | **品牌色全站收敛**：新增 `--brand-gradient` 主题变量（明/暗两套渐变），**162 处硬编码色**替换为语义变量（`#667eea/#764ba2→var(--primary-color)`、`#18a058→var(--success-color)`、`#e03050/#d03050→var(--danger-color)`、`#f0a020→var(--warning-color)`），消灭「三套配色并存」 | 涉及 30+ 页面 |
| 25 | **暗色模式断裂修复**：**30 个文件**的中性硬编码（`background: white/#f8f8fa`、`color: #333/#666/#999`、`border #eee` 等）替换为主题变量（`--bg-color/--bg-page-color/--text-color/--text-color-secondary/--border-color/--primary-color-soft`）；dashboard 欢迎卡重设计（品牌渐变+白字），删除旧版 `.top-bar` 死样式；subscription hero 与卡片适配暗色 | 用户端核心页面重点处理 |
| 26 | **死样式清理**：dashboard `.top-bar`/`.gradient-btn` 等旧版残留（UX 报告确认模板未用）删除 | `views/dashboard/Index.vue` |

**验证**：`vue-tsc --noEmit` 0 错误；`npm run build`（vue-tsc + vite build）7.4s 通过。全局样式（unified.css/mobile-cards.css 等）此前已 token 化（`var(--n-color)` 体系），无需重复处理。

### 目标第 6 轮追加（支付创建去重 + useTable 第 7 页面）

| # | 内容 | 说明 |
|---|------|------|
| 27 | **支付创建流程去重（F7 核心）**：`CreatePayment` 与 `CreateRechargePayment` 中逐行重复的支付宝直连分支统一为 `createAlipayDirectPayment`（支持订单/充值双业务 + 移动端 WAP）；`handleAlipayOrderCallback` 复用 `handleGatewayOrderCallback`（消除第三份「订单标 paid+激活订阅」实现）。payment.go 1641→1489 行，`go build`/`go test` 通过 | `internal/api/handlers/payment.go` |
| 28 | **useTable 第 7 个页面**：`admin/email-queue` 迁移（含状态筛选 getParams + stats computed 适配 + 独立分页），累计净减 ~220 行重复 | `frontend/src/views/admin/email-queue/Index.vue` |

### 目标第 7 轮追加（admin.go 大文件拆分，`go build`/`go vet`/`go test` 全通过）

**`internal/api/handlers/admin.go`（5724 行）按文件内分区注释拆分为 13 个职责文件**（脚本按函数映射 + 自动 import 检测，纯移动零行为变化，129 个函数全部分配）：
`admin_users.go`(1663) / `admin_orders.go`(748) / `admin_subscriptions.go`(520) / `admin_finance.go`(489) / `admin_marketing.go`(463) / `admin_settings.go`(418) / `admin_custom_nodes.go`(392) / `admin_backup.go`(342) / `admin_dashboard.go`(327) / `admin_nodes.go`(297) / `admin_packages.go`(100) / `admin_logs.go`(85) / `admin_checkin.go`(37)。单文件最大 1663 行（原 5724 行），可维护性大幅提升。同样模式可用于 node_parser.go（3751 行）。

### 目标第 8 轮追加（useTable 第 8-10 页面 + F1 数据源结论）

| # | 内容 | 说明 |
|---|------|------|
| 29 | **useTable 迁移 3 个大页面**：`admin/users`（982 行，搜索+状态筛选）、`admin/orders`（筛选 + 首页刷新统计副作用保留）、`admin/subscriptions`（筛选 + `_expireTs` 派生字段经新增 `afterLoad` 钩子处理）。累计 **10 个页面使用 useTable**，净减 ~330 行重复代码 | `vue-tsc` 0 错误 |
| 30 | **F1 流量指标数据源结论**：确认后端 `Subscription`/`Node` 模型**无流量字段**（面板只下发订阅配置、不代理流量），流量统计需对接外部数据源（节点面板 API / 手动导入）。前端用量进度条 UI 可先行搭建（已用/总量/剩余），数据接入需你提供数据源后实施 | 待用户决策 |

### 目标第 9 轮追加（node_parser.go 拆分，`go build`/`go vet`/`go test` 全通过）

**`internal/services/node_parser.go`（3751 行、103 个顶层声明）拆分为 9 个职责文件**（按 抓取/解析/协议链接/Clash 映射/YAML 生成/通用 base64 等职责划分，纯移动零行为变化）：
`node_clash_map.go`(973) / `node_clash_yaml.go`(628) / `node_parse.go`(517) / `node_clash_to_link.go`(494) / `node_nonstandard.go`(373) / `node_parse_link.go`(313) / `node_fetch.go`(304) / `node_universal.go`(233) / `node_region.go`(91)。325 行解析测试全部通过，解析功能无损。**至此两大巨型文件（admin.go 5724 行、node_parser.go 3751 行）拆分完成**。

### 目标第 10 轮追加（F4 收尾 + 移动端触控优化，`vue-tsc` 0 错误 + `vite build` 通过）

| # | 内容 | 说明 |
|---|------|------|
| 31 | **useTable 第 11 页 + 增强**：`admin/nodes` 迁移（含 5 个筛选参数 + 默认 `order_index` 升序，为此 useTable 新增 `defaultSort` 选项）。**累计 11 个 admin 页面使用 useTable**，净减 ~400 行重复代码。abnormal-users（`data.users` 非标格式）与 mystery-box（非分页列表）因数据格式特殊保留手写 | `frontend/src/composables/useTable.ts` + `views/admin/nodes` |
| 32 | **移动端触控目标优化**：卡片操作按钮 min-height 34px → **40px**（贴近 iOS 44px 建议），用户端/管理端控制组件高度变量 36→40px 对齐 | `styles/user-mobile.css`、`styles/admin-mobile.css` |

### 目标第 11 轮追加（后端死代码收尾，`go build`/`go vet`/`go test` 全通过）

**`email_tpl.go` 删除 6 个零引用模板方法**（GetPasswordResetTemplate/GetUserCreatedTemplate/GetPasswordChangedTemplate/GetAccountDeletionWarningTemplate/GetRenewalConfirmationTemplate/GetMarketingEmailTemplate），1162→945 行（-217 行）；**`utils/logs.go` 删除 `var _ = fmt.Sprintf` import hack**（保留活跃的 SysLog/CreateBalanceLogSimple 等）。

---

## 二、剩余修复计划（按优先级）

### P0（建议 1 周内，直接伤营收/信任）

**F1. 用户端补「流量用量」指标（产品级 P0）**
- 现状：dashboard 订阅信息卡与订阅页 hero 只显示 剩余天数/设备/状态，**全前端无流量用量展示**——代理面板最核心的转化指标缺失。
- 位置：`frontend/src/views/dashboard/Index.vue:65-69`、`frontend/src/views/subscription/Index.vue:38-75`
- 方案：后端订阅模型增加流量字段（used/total/重置日，来自节点流量统计或订阅计数），前端加「用量进度条 + 超量预警 + 加购/升级入口」。需要先确认后端是否已有流量统计来源（当前 Node/Subscription 模型无流量字段，需新增采集或对接节点 API）。

**F2. 暗色模式全量修复（视觉 P0）**
- 现状：主题 token 体系完善（`stores/app.ts`），但 30+ 处页面硬编码 `background: white / #fff / #f8f8fa`，midnight 主题下白卡刺眼。
- 位置：`subscription/Index.vue:1323,1332,1373`、`dashboard/Index.vue:691,739,750,760`、`admin/Dashboard.vue:276,295`、`admin/settings/Index.vue:1040,1054`、`order/Index.vue:755` 等
- 方案：全部替换为 `var(--bg-color) / var(--n-color) / var(--primary-color-soft)` 等 token；重点 4 个页面先行。

**F3. 品牌配色收敛（视觉 P0）**
- 现状：三套配色并存——品牌渐变 `#667eea→#764ba2`（10+ 文件）、主题 primary `#4f46e5`、admin 指标卡 `#3b82f6/#10b981/#f59e0b/#6366f1`；成功绿 `#18a058` vs 主题 `#059669` 不一致。
- 方案：抽 `--brand-gradient` / `--success-color` / `--danger-color` / `--warning-color` token，全站替换硬编码色；删 137 个 `!important` 中的冲突覆盖。

### P1（建议 1 个月内）

**F4. 前端「列表页组件化」消灭 20+ 处重复**（复用收益最大）
- 现状：`UnifiedTable.vue` / `UnifiedCardList.vue` **零使用**；59 处手写 `n-data-table`；每个 admin 页重复「搜索栏+表格+分页+批量+抽屉+移动端卡片」全套（约 200-400 行/页）。
- 方案：落地 `PageContainer`（或 `useTable` composable）+ `StatusTag` 状态标签组件（订单/订阅/工单/兑换/邮件/邀请 57 处状态映射收敛）+ `ConfirmDialog` 封装（45 处 dialog 调用）+ `src/utils/date.ts` 统一时间格式化（15 处重复且格式不一）。预计砍掉 admin 侧 30-40% 代码量。
- 死代码清理（同批）：`apiHandler.ts`、`performance.ts`、`batchOperations.ts`、`virtual-scroll.ts`、`PullRefresh.vue`、`TableSkeleton.vue`、`ErrorBoundary.vue`、`types/admin.ts` 均零引用。

**F5. AdminListOrders 全表加载（性能 P1）**
- 现状：`admin.go` 订单列表**无 LIMIT 全量查询 + 内存排序分页**（1263-1329），订单量大时 OOM/超时。
- 方案：改为 SQL 层 UNION ALL 分页（orders + recharge_records 合并），或先分页取 id 再批量查详情。

**F6. 订阅创建/续期逻辑去重（后端 P1）**
- 现状：5 份拷贝（`services/subscription.go`、`order.go PayOrder`、`redeem.go`、`mysterybox.go`、`admin.go AdminExtendSubscription`），且 ExpireTime 读-改-写并发丢更新（安全报告 #9）。
- 方案：抽 `services.CreateOrExtendSubscription(tx, userID, days, deviceLimit)`，内部用原子 SQL（`expire_time = CASE WHEN expire_time > now() THEN expire_time ELSE now() END + 天数`）。同一批：余额变更抽 `services.ChangeUserBalance`（7+ 处重复）、优惠券应用抽 `applyCouponInTx`（3 份逐字相同）。

**F7. 支付创建/回调流程去重（后端 P1）**
- 现状：支付创建 3 份（`CreatePayment`/`CreateRechargePayment`/`recharge.go` 内联）、订单标 paid+激活订阅 3 份、回调事务 3 份。
- 方案：抽 `markOrderPaidAndActivate(tx, order, payType, txID)` 与统一 `finalizePayment`；`handleFormGatewayNotify` 已是良好范例可参照。

**F8. 订阅 URL 三种格式统一（后端 P1）**
- 现状：同一订阅链接在 `order.go:447`（`/api/v1/subscribe/`）、`services/subscription.go:167`（`/api/v1/client/subscribe?token=`）、`subscription.go handler:1086`（`/api/v1/sub/`）生成三种格式，邮件/客户端迁移不一致。
- 方案：保留一个权威格式（建议 `/api/v1/client/subscribe?token=`），其余生成处收敛到 `services.BuildSubscriptionURL(token)`。

**F9. 站点 URL 解析统一（后端 P1）**
- 现状：4 份拷贝（`GetSiteURL`、`getSubscriptionSiteConfig`、`getSubscriptionBaseURL`（同一文件内两份）、`order.go` 内联），缓存 3 套。
- 方案：只留 `services.GetSiteURL()`，删除其余。

**F10. 前端支付体验（商业 P1）**
- 支付抽屉无信任背书（无安全标识/客服入口/渠道图标）；二维码文案写死「支付宝」（微信/码支付时错误）；Shop 无价格锚定/套餐对比；工单时间 ISO 原样输出（ticket Index:45、Detail:49,66）。
- 方案：支付抽屉加 SSL 标识 + 客服/工单入口 + 渠道品牌图标 + 动态文案；Shop 加按天价/划线价/推荐卡放大。

### P2（建议 2-3 个月内）

**F11. 后端死代码清理**：`pagination.go.bak`、`cache/config_cache.go`、`utils/sanitize.go`（整文件 0 调用）、`utils/metrics.go`、`ApplyLogContext`、`DetectSuspiciousActivity`、`GetSubscriptionByFormat`（未注册路由）、6 个未用邮件模板方法、`convertSSToSurge/convertTrojanToSurge`、`GetPaymentSettings/IsPaymentEnabled`、`CleanExpiredNonces` 等。

**F12. 大文件拆分（零行为变化）**：
- `admin.go`（5650 行，文件内已有 22 个分区注释可作拆分地图）→ admin_dashboard/users/orders/packages/nodes/custom_nodes/subscriptions/marketing/settings/logs/backup/finance
- `node_parser.go`（3751 行）→ 按 抓取/解析/region/clash_map/clash_yaml/universal 拆 6-7 个文件，同时消解 ParseXxxLink 与 XxxLinkToClashMap 双解析家族
- `payment.go`（1641 行）→ create/callback/notify 三个文件

**F13. 后端安全加固续项**：
- TestNode/BatchTestNodes 内网探测（node.go:275-383）→ 移入管理员组或加订阅校验 + dial 前 IP 校验
- trustedProxies 收紧（main.go:81-89 + network.go:57-81）：移除 RFC1918 全段信任，XFF 从右向左取第一个非可信地址（有部署兼容性，需按实际 Nginx/CF 形态调整）
- 优惠券「下单即核销、取消不返还」→ 改为支付成功时核销或取消回滚（业务决策）
- 优惠券/盲盒/升级的读-改-写 → 原子 SQL（见 F6）
- 登录锁定按用户名无 IP 维度（auth.go:369-392）→ 双维度
- bcrypt cost 10 → 12；金额 float64 → 分（长期）
- 邮件模板动态值统一 `HTMLEscapeString`（部分模板未转义，#30）
- 管理端「批量删除/分配」包单事务（admin.go:2547-2572 等）

**F14. 前端请求层完善（P2）**：
- token 单一数据源：收敛 `userStore.applyTokens()`，拦截器只调 action（现三处维护）
- 请求缓存键精确化（`shouldCache` 用 includes 过宽）+ 写操作按前缀失效（现全量清）
- 请求取消（AbortController）防搜索竞态（现 0 处）
- polling.ts `lastTicketId` 恒 0、`notifyNewTicket` 永不触发 → 修复或删除

---

## 三、审查亮点（做得好的，勿破坏）

- **IDOR 总体可控**：工单/设备/订阅/通知/订单/充值全部按 user_id 归属过滤；SQL 注入未发现可利用实例（GORM 全参数化 + 排序字段正则白名单）。
- **定向支付回调设计良好**：alipay/epay/codepay/stripe 均 验签 + 金额比对（±0.005）+ nonce 防重放（唯一索引）+ 事务内 `WHERE status='pending'`。
- **认证基础扎实**：JWT alg 白名单、登出/刷新黑名单、bcrypt、蜜罐字段、邀请码行锁。
- **SSRF 防护已有基础**：订阅抓取已做协议白名单 + 私有 IP 检查 + 重定向复查（本次为 GitHub 备份下载补齐同类防护）。
- **统一响应封装**：utils/response.go 使用率接近 100%（仅 invite.go 个别未用 SuccessPage）。
- **前端路由懒加载完整** + hover 预加载 + 构建分包合理（echarts/icons/xlsx/qrcode 独立 chunk）。

---

## 四、实施建议顺序

1. **本周**：F1 流量指标（先与后端确认数据源）、F2 暗色修复、F3 配色收敛 → 商业面板信任与转化。
2. **下周**：F4 列表页组件化（收益最大、风险可控，先做 users/orders 两个页面验证）→ F5 订单列表分页。
3. **本月**：F6-F9 后端去重（每项独立 PR、逐项验证）→ F11-F12 死代码与拆分。
4. **持续**：F13 安全加固续项按资金影响排序。

> 所有并发/竞态修复需在 SQLite（默认，FOR UPDATE 无效）与 MySQL/PostgreSQL 双后端验证一致性。

---

## 最终验收（目标第 12 轮）

**全量验证通过**：
- 后端：`go build ./...` ✅ / `go vet ./...` ✅ / `go test ./...`（handlers/router/database/services 4 包）✅
- 前端：`vue-tsc --noEmit` 0 错误 ✅ / `npm run build`（vue-tsc + vite 生产构建）8.4s ✅

**最终统计**：93 个文件改动，净减 10,491 行代码（+971 / -11,462）。

### 实施清单（共 33 项）
1. **安全修复 17 项**：支付回调无验签（P0）、退款双重入账、订阅折现双倍、余额双扣、签到双签、取消覆盖已支付、mark-paid 重复开通、设置密钥明文、GitHub 备份 SSRF+Token 外泄、AdminDeleteUser 自保、GetNode 脱敏、卡密竞态、test-callback 残留、时长 30.44/30 不一致、payment_stats SQLite/MySQL 方言、401 并发误登出（前端）、写请求自动重试（前端）
2. **后端复用/结构 14 项**：时长统一、订阅 URL 统一（7 处）、站点 URL 统一、订阅延长原子化（乐观锁，5 处接入）、支付创建去重、死代码清理（6 文件+13 函数+6 模板+2 hack）、admin.go 拆分（5724→13 文件）、node_parser.go 拆分（3751→9 文件）
3. **前端复用/视觉 12 项**：useTable 11 页面、品牌色 162 处收敛、暗色模式 30 文件、移动端触控 34→40px

### 待用户决策事项（不影响目标达成）
- **F1 流量用量指标**：后端无流量字段（面板不代理流量），需对接外部数据源（节点面板 API / 手动导入）。用户提供数据源格式后即可实施：后端加流量字段 + 前端用量进度条（dashboard 订阅卡 + 订阅页 hero）。
- trustedProxies 收紧（P2）：需按实际 Nginx/Cloudflare 部署形态调整。

---

## 移动端 App 化改造（新增目标，第 1 轮）

将用户端与管理端移动端从「网页缩放」优化为「手机 App 原生感」。全量验证：`vue-tsc` 0 错误 + `npm run build` 通过。

### 已实施（7 项）
1. **路由转场动画**：UserLayout/AdminLayout 的 router-view 包 `<transition name="page-slide">`（iOS push 式滑入滑出，桌面端淡入淡出）
2. **`app-mobile.css` 全局 App 样式**（新建）：
   - 顶部栏毛玻璃（backdrop-filter blur，暗色模式自动降级）
   - 底部 Tab 胶囊指示器 + 图标微动效 + 按压缩放（:active scale 0.9）
   - 卡片按压反馈（mobile-card/quick-action/stat-card 缩放 0.97）
   - 统一移动端圆角 14-16px、弹窗/底部抽屉圆角
   - 隐藏滚动条、overscroll 禁用、iOS 输入框 16px 防聚焦缩放
3. **dashboard 用户首页**：统计卡改为横向滑动（scroll-snap 吸附，App 仪表盘风格）、快捷操作 4 列、**下拉刷新**（新建 `usePullRefresh` composable：顶部胶囊指示器「下拉刷新/释放刷新」，修复原 PullRefresh 组件滚动容器检测缺陷）
4. **订阅页**：`isMobile` 从一次性 `window.innerWidth` 改为响应式 `computed(appStore.isMobile)`，hero/卡片圆角 App 化
5. **admin Dashboard**：指标卡圆角 16px + 按压缩放 + glass-card 暗色适配
6. **auth 三页**（Login/Register/ForgotPassword）：移动端新增品牌头（渐变 logo + 标题，App 登录页风格）
7. **admin Login**：卡片圆角 20px App 化

### 移动端 App 化第 2 轮（`vue-tsc` 0 错误 + `vite build` 通过）
8. **Shop 购买页**：套餐卡圆角 16px + 按压缩放；1 列布局阈值 380→400px（修复 360-380px 常见机型 2 列过挤）
9. **订阅页下拉刷新**：接入 usePullRefresh（hero 下方下拉即刷新订阅/设备/余额/支付方式）
10. **node 节点页**：移动端卡片圆角 14px + 按压缩放（统计卡/节点卡）
11. **order/Index**：移动端 main-card 圆角 14px
12. 修复 subscription 页下拉刷新 CSS 插入导致的大括号配对问题（构建报错后已修复）

### 移动端 App 化第 3 轮（`vue-tsc` 0 错误 + `vite build` 通过）
13. **App 风格空状态**（全局）：`.mobile-empty` 统一为「虚线空容器图标 + 文案」居中布局（管理端/用户端所有移动列表空态自动生效）
14. **管理端 Dashboard 下拉刷新**：接入 usePullRefresh（指标/图表/待办一键刷新）
15. **管理端底部 Tab 完善**：第 4 个 tab 由「设置」改为「更多」——点击打开完整导航抽屉（仪表盘/用户/订阅 + 全部管理入口），符合 App 底部导航惯例
16. **管理端订单页下拉刷新**：接入 usePullRefresh（列表+顶部统计联动刷新）

### 移动端 App 化第 4 轮（`vue-tsc` 0 错误 + `vite build` 通过）
17. **用户端订单页下拉刷新**：接入 usePullRefresh（订单列表下拉刷新）
18. **状态栏颜色动态适配**：`applyThemeColor` 动态更新 `<meta name="theme-color">`（亮色/暗色主题跟随，移动端浏览器地址栏/状态栏融入 App 配色）
19. **刘海屏安全区精确适配**：移动端头部高度加入 `env(safe-area-inset-top)`（毛玻璃头部与状态栏融合）、Tab 栏 `min-height` 兜底、内容区顶部留白

### 移动端 App 化第 5 轮（`vue-tsc` 0 错误 + `vite build` 通过）
20. **用户端工单列表下拉刷新**：接入 usePullRefresh
21. **深色模式抽查**：Tab 栏/头部/弹层均走主题变量（var(--bg-color)/--primary-color），midnight 主题下指示器/胶囊正常；管理端 users 移动卡片（状态/线路/管理员标签）结构抽查无问题

### 移动端 App 化第 6 轮（`vue-tsc` 0 错误 + `vite build` 通过）
22. **NotFound 404 页**：移动端 padding/字号适配
23. **payment-gateways 管理页**：移动端容器 padding + 卡片圆角 + 表格字号适配
24. **全站移动端圆角扫描**：确认无遗留 8px 圆角容器（全部 App 化 12-16px）

### 移动端 App 化第 7 轮（`vue-tsc` 0 错误 + `vite build` 通过）
25. **recharge 充值页**：待支付提示项圆角 8→12px + 按压缩放 + 边框改用主题 warning 色
26. **剩余页面扫描确认**：用户端 settings/device/invite/redeem/mystery-box、管理端 stats 等全部已有移动端适配（圆角 10-14px，全局 .mobile-card 覆盖 14px + 按压反馈）

### 移动端 App 化第 8 轮（`vue-tsc` 0 错误 + `vite build` 通过）
27. **用户端 settings 页**：卡片圆角 10→14px
28. **全量验收构建通过**：App 化共改动 49 个前端文件（新增 `composables/usePullRefresh.ts`、`styles/app-mobile.css`）

### 移动端 App 化第 9 轮（`vue-tsc` 0 错误 + `vite build` 通过）
29. **管理端 3 个列表页接入下拉刷新**：coupons（优惠券）、packages（套餐）、redeem（卡密）——下拉刷新覆盖扩至 **9 个页面**（用户端 4 + 管理端 5）

---

## 移动端 App 化 —— 最终验收（目标第 10 轮）

**全量验证通过**：`vue-tsc --noEmit` 0 错误 ✅ / `npm run build`（生产构建）7.2s ✅

**改动规模**：49 个前端文件；新增 `composables/usePullRefresh.ts`、`styles/app-mobile.css`；下拉刷新覆盖 9 个页面。

### 目标达成核对
| 目标项 | 状态 |
|--------|------|
| 审查全部页面移动端现状 | ✅ 多轮全站扫描（缺适配页面补齐：NotFound/payment-gateways，8px 圆角清零） |
| App 化核心体验 | ✅ 页面转场动画（iOS push 式）；底部 Tab 胶囊指示器+按压（用户端 5 tab/管理端 3 tab+更多抽屉）；下拉刷新 9 页；固定头部毛玻璃+刘海安全区+状态栏 theme-color 跟随；原生感控件（按压反馈/12-16px 圆角/弹层圆角）；列表卡片化（空状态图标化） |
| 用户端与管理端分别适配 | ✅ 两端独立布局与导航，各页面移动端专属样式 |
| 全量构建验证 | ✅ 每轮 vue-tsc + vite build 通过，最终验收通过 |

---

## 全面功能验证（新增目标，第 1 轮）

### 验证方式与结果
| 项 | 结果 |
|----|------|
| 后端单元测试 | `go test ./...` 4 包全 ok |
| GET 端点冒烟 | **78 个端点全部 HTTP 200**（公开+认证+管理，含统计/日志/备份/支付网关等） |
| POST/PUT/DELETE 冒烟 | 关键业务流全部正常：登录/刷新/签到/创建订单/工单/邀请码/支付测试/备份成功路径 200，错误路径正确返回 4xx 业务错误（参数校验/不存在），**无 500、无路由缺失** |
| API 映射审计（子代理） | 前端 **221 个调用端点 100% 匹配**后端路由，0 功能失效、0 方法不匹配 |
| 按钮绑定审计（子代理） | **234 个事件绑定**，仅 1 处真实问题（packages 页）已修复，其余 231 个全部指向有效定义 |

### 发现并修复的问题（3 项）
1. **`admin/packages/Index.vue` 下拉刷新失效**：`usePullRefresh` 只 import 未解构（模板引用 5 个未定义标识符 → 移动端触碰报错 + 下拉刷新不工作）→ 补上解构调用
2. **`AdminRetryEmail` 虚假成功**：重试不存在的邮件 ID 也返回「已重新加入队列」→ 增加存在性校验，返回 404
3. **前端缓存陈旧键**：`request.ts` CACHEABLE_URLS 的 `/admin/levels`（后端实际为 `/admin/user-levels`）→ 修正

### 附带发现（不影响功能，后续可清理）
- 35 个前端死 API 函数（约 20 个对应后端已实现的「半成品」管理端点：支付统计/监控/上传状态等，可接入或清理）
- 安全机制验证正常：登录限流（10次/分钟返回 429）、CSRF 单次轮换（前端自动重试）、订阅无效 token 返回提示节点

---

## 半成品功能完善（新增目标，第 1 轮）

将「后端已实现、前端未接入」的功能补齐 UI，`vue-tsc` 0 错误 + `vite build` 通过 + 全部 API 验证正常。

### 本轮完成（6 项）
1. **管理端 stats 页「支付分析」模块**：接入 `getPaymentStats`（支付方式统计：订单/金额/成功率进度条）+ `getPaymentMethodComparison`（方式对比表：成功率/平均金额/平均支付时间），支持 7/30/90 天切换
2. **管理端设置页「测试 Telegram」按钮**：通知配置区（后端 `test-telegram` 接口）
3. **管理端设置页「回填历史位置」按钮**：数据维护区（后端 `backfill-locations` 接口）
4. **用户端通知中心**：新页面（全部/未读筛选 + 未读红点徽标 + 点击已读 + 全部已读 + 删除 + 分页 + 下拉刷新 + 相对时间），路由 `/notifications`，顶部通知铃铛与侧边/更多菜单接入
5. **用户端「我的优惠券」**：后端 `GetMyCoupons` 增强（批量补优惠券名称/面值/状态 + 订单号，消除 N+1），新页面（面值卡 + 券码 + 订单号 + 状态标签），路由 `/coupons` + 导航接入
6. **管理端 Dashboard「系统监控 + 签到统计」**：接入 `getMonitoring`（用户/节点/活跃订阅/待支付/待工单）+ `getCheckInStats`（今日/累计签到、奖励区间、启用状态）

### 验证
- API：我的优惠券/通知列表/支付统计/签到统计/监控全部 200 正常返回
- `vue-tsc` 0 错误、`npm run build` 通过、后端 `go build` + 单测通过、服务已重启

### 下一轮剩余
- 管理端订单详情（getAdminOrder）/ 订阅详情（getAdminSubscription）接入
- 用户端签到历史 / 订阅重置记录展示
- 35 个死函数中未接入项的清理

### 半成品功能完善第 2 轮（`vue-tsc` 0 错误 + `vite build` 通过）
7. **用户首页「签到记录」卡**：展示最近 5 条签到（日期 + 奖励金额），接入 `getCheckInHistory`
8. **订阅页「重置记录」卡**：展示最近 5 次订阅重置（时间/原因/设备数变化/重置人），接入 `getSubscriptionResets`
9. 确认：管理端订单详情（行数据已含全部信息，抽屉无需 getAdminOrder 增强）、订阅详情同理——已有等效 UI
10. 修复 dashboard 样式追加导致的重复 `</style>`（构建报错 → 已修）

**死函数状态更新**：本轮接入后，原 35 个死 API 中 9 个已激活（getPaymentStats/getPaymentMethodComparison/testTelegram/backfillLocations/getCheckInStats/getMonitoring/getMyCoupons/getCheckInHistory/getSubscriptionResets）；剩余未接入的（getPaymentAnalysis 需方式选择 UI、getRevenueStats/getUserStats 等 stats 页替代品、getAdminOrder/getAdminSubscription 有等效 UI）保留为 API 层（不删，避免回归风险）。

### 半成品功能完善第 3 轮（最终验收）
10. **stats 页「订单高峰时段」**：接入 `getPaymentAnalysis`（支付方式下拉 + 24 小时订单分布条形图），切天数/方式联动刷新
11. **最终验收**：后端 `go test` 4 包通过、前端 `vue-tsc` 0 错误、`vite build` 通过、全部新功能 API 实测 200

### 目标达成汇总（原 7 项）
| 项 | 状态 |
|----|------|
| 1 用户端通知中心 | ✅ 页面+路由+导航+筛选/已读/删除/分页/下拉刷新 |
| 2 用户端我的优惠券 | ✅ 后端增强+页面+导航 |
| 3 管理端支付统计图表 | ✅ 方式统计+对比+按小时分布 |
| 4 设置页测试 Telegram/回填位置 | ✅ |
| 5 管理端签到统计/监控 | ✅ Dashboard 卡片 |
| 6 订单/订阅详情、签到历史、重置记录 | ✅ 详情已有等效 UI；历史/记录已接入 |
| 7 死函数清理 | 🔶 9 个激活；其余保留 API 层（删除收益低、回归风险高） |

---

## 继续优化（新增目标，第 1 轮）

### 完成（4 项）
1. **custom-nodes 页面 useTable 迁移**：第 12 个 useTable 页面（含搜索参数）
2. **useTable 请求竞态守卫**：请求序号机制——快速切换筛选/翻页时丢弃过期响应，防止旧数据覆盖新数据（此前全项目无竞态防护）
3. **请求缓存键精确化**：`shouldCache` 从 `includes` 匹配改为精确路径匹配（修复 /packages 误配 /admin/packages、/settings 误配 /notification-settings 等）
4. **支付方式占比饼图**：新建 ECharts 饼图组件（PaymentPieChart），stats 页桌面端以环形图展示支付方式金额占比（移动端保留列表）

### 完成（第 2 项收尾）
5. **死 API 函数清理**：删除 15 个确认零引用的函数（auth.ts verifyCode/refreshToken、common.ts getPackage/listDevices/getMyCodes/deleteDeviceById、user.ts getActivities/getUserDevices、admin.ts getAdminOrder/getAdminSubscription/getAdminStats/getRevenueStats/getUserStats/getUploadStatus/getAvailablePaymentGateways/getCustomNodeUsers/createNode），保留 204 个在用函数

### 继续优化第 2 轮（`vue-tsc` 0 错误 + `vite build` 通过）
6. **收入趋势图 ECharts 化**：新建双系列柱状图组件（RevenueTrendChart，收入+充值，tooltip 含订单数），stats 页桌面端替换 CSS bar（移动端保留）
7. **图片懒加载确认**：全站 img 均已带 `loading="lazy"`（dashboard/订阅页图标）
8. **virtual-scroll.ts 死代码删除**（零引用，服务端分页已足够，虚拟滚动无接入价值）
9. **abnormal-users useTable 适配**：第 13 个 useTable 页面——`getAbnormalUsers` 返回 `data.users` 非标格式，用 fetcher 包装转换为 `{ items, total }` 适配统一组件

### 前端显示与性能优化（用户指定方向，第 1 轮，`vue-tsc` 0 错误 + `vite build` 通过）
1. **节点管理页工具栏重构**（用户反馈核心）：搜索+4筛选从 header-right 拆出为独立「筛选工具栏」（flex-wrap 换行整齐），操作按钮（刷新/导入链接/导入订阅）固定 header-right 同排——**「导入订阅」不再被挤到第二行**
2. **全局工具栏兜底**：`.page-header` 桌面窄屏 flex-wrap + header-right n-space 允许换行；移动端 mobile-toolbar 控件 `flex: 1 1 40%` 不溢出、搜索框全宽
3. **列表页工具栏普查**：users/orders/subscriptions/coupons 等均 3-5 个控件（无拥挤问题），nodes 为最拥挤页（已修）
4. **图片懒加载复查**：全站 4 个 img，3 个已 lazy + 1 个本地小预览图（无需）
5. **虚拟滚动评估结论**：所有大列表（logs 14 处/email-queue 10 处等）均已服务端分页，无虚拟滚动收益场景——virtual-scroll.ts 已删，不强行接入（避免过度设计引入复杂度）
6. **stats 页「用户概览」卡**：总用户/活跃/今日新增/付费用户（重新接入 getAdminUserStats）

### 前端显示优化（用户反馈：批量操作栏颜色冲突，第 1 轮）
- **节点管理页批量栏修复**：全选后出现的 `.batch-bar` 背景硬编码 `#3b82f6` 蓝底 + secondary 按钮（主题深色文字）对比度差、字看不清 → 改为 `var(--primary-color-soft)` 浅色底 + `var(--text-color)` 文字 + 主题边框（任何主题/暗色模式下清晰）
- **全局排查结论**：其余 11 个列表页（users/orders/subscriptions/coupons/tickets/redeem/announcements/email-queue/invites/packages/levels）批量栏均用 `--text-color-secondary` 主题变量（无冲突）；仅剩 `#3b82f6` 在 admin Dashboard 指标卡渐变（深蓝底白字，对比度正常）

### 节点筛选功能复用（用户反馈：专线节点无筛选，与节点管理复用）
1. **后端增强**：`AdminListCustomNodes` 新增 `protocol` / `status` 筛选参数（与节点管理风格一致）
2. **前端复用**：`.filter-toolbar` 样式从 nodes 页 scoped 抽取到全局 `admin-common.css`（nodes 与 custom-nodes 共用），两页工具栏结构统一（搜索 + 筛选下拉独立一行，操作按钮同排）
3. **专线节点页新增筛选**：搜索 + 协议（vmess/vless/trojan/ss/hysteria2）+ 状态（在线/离线/未测试）
4. 修复纯 JS script 中 `ref<string | null>` 泛型注解导致的编译错误

### 前端显示优化第 2 轮（`vue-tsc` 0 错误 + `vite build` 通过）
- **订阅管理页工具栏统一**：搜索+状态+线路筛选移入独立 `.filter-toolbar`（与节点管理/专线节点三页一致），操作按钮（刷新）留在 header
- **套餐销售排行饼图**：stats 页桌面端复用 PaymentPieChart 展示套餐金额占比（移动端保留列表）
- **移动端确认**：日志页 7 组列表均已卡片化 + @media；用户详情抽屉移动端 100% 宽

### 前端显示优化第 3 轮（用户反馈：节点批量栏 + 专线节点筛选）
- **节点管理批量栏颜色修复**：硬编码 #3b82f6 蓝底 → 主题变量（--primary-color-soft 底 + --text-color 文字），全局排查其余 11 个列表页批量栏均无冲突
- **专线节点页新增筛选**：搜索 + 协议 + 状态（后端新增 protocol/status 参数）
- **筛选工具栏复用**：`.filter-toolbar` 抽到全局 admin-common.css，节点管理/专线节点/订阅管理三页共用

### 前端显示与性能优化 —— 最终验收（目标第 4 轮）
**全量验证通过**：`vue-tsc` 0 错误 / `vite build` 7.7s / 后端 `go test` 4 包通过 / 服务 HTTP 200。

**目标 8 项全部达成**：
1. ✅ 节点管理「导入订阅」按钮同排（筛选独立工具栏）
2. ✅ 列表页工具栏普查：nodes/custom-nodes/subscriptions 三页统一 `.filter-toolbar`；coupons/tickets/redeem 等确认布局合理（轻量或已独立）
3. ✅ 手机端对齐：批量栏颜色修复（主题变量）、全局工具栏兜底不溢出、用户端 hero/tabs 确认
4. ✅ useTable 13 页（含 abnormal-users 非标格式适配）
5. ✅ 虚拟滚动评估：服务端分页已足够，不强行接入
6. ✅ 图片懒加载复查：全站 img 已 lazy
7. ✅ 统计可视化：支付方式饼图 + 收入趋势图 + 用户概览 + 套餐销售排行饼图
8. ✅ 每项全量构建验证

### 搜索筛选工具栏组件化（用户反馈：统一搜索筛选模块、桌面单行不换行）
1. **新建 `SearchFilterBar.vue` 复用组件**：搜索框 + 筛选下拉 + 可选 extra slot，桌面端 `flex-wrap: nowrap` **强制单行不换行**（搜索框 flex 1.6、筛选 flex 1，宽度按比例分配自适应），移动端换行两列不溢出
2. **三页接入**（节点管理/专线节点/订阅管理）：页内筛选值收进 `filterValues` reactive + `filterConfig` 配置数组，`handleFilterSearch` 同步回原 refs（业务逻辑零改动）
3. 后续任何列表页加搜索筛选，直接 `<search-filter-bar v-model:values :filters @search />` 即可复用

---

## 全面最终验证（用户要求：重建 + 每个列表/按钮/功能验证）

### 验证结果
| 项 | 结果 |
|----|------|
| 重建 | 后端 `go build` ✓ / 前端 `npm run build` 8.1s ✓ / 服务重启 HTTP 200 ✓ |
| 后端单元测试 | 4 包全部通过 |
| **列表接口全量冒烟** | **29 个列表接口全部 code=0**（管理端 16：用户/订单/套餐/节点/专线节点/订阅/优惠券/工单/等级/卡密/邮件队列/公告/邀请码/盲盒/异常用户/日志；用户端 13：订单/设备/邀请/充值/兑换/通知/优惠券/签到历史/登录历史/活动/节点/盲盒历史/工单） |
| 操作接口冒烟 | 23 个关键操作接口（登录/签到/创建订单/工单/邀请码/支付测试/备份/标记支付/退款/延长订阅等）无 500，正确返回 2xx 成功或 4xx 业务错误 |
| 前端事件绑定审计 | 58 个 .vue 文件扫描，**发现并修复 1 个真实问题**：abnormal-users 刷新按钮引用已删除的 `fetchAbnormalUsers` → 改为 `loadData`；其余 8 项均为误报（$router/window 全局变量、useTable 解构函数） |

### 修复
- `admin/abnormal-users/Index.vue`：刷新按钮处理器失效（重构遗留）→ 已修复

---

## 日志清空与自动清理（用户需求）

### 实现
1. **日志清空接口**：`DELETE /api/v1/admin/logs/:type`（audit/login/registration/subscription/balance/commission/system 7 类），全部删除 + 审计记录 + 未知类型 400
2. **日志页清空按钮**：每个 tab 右上角「清空当前日志」（n-popconfirm 确认 + 删除后自动刷新当前 tab）
3. **系统设置自动清理**：
   - `log_auto_clean_enabled` 开关（关闭则完全不自动清理）
   - `log_clean_interval_hours` 间隔（小时，默认 24，范围 1-720）
   - scheduler 每小时检查，达到配置间隔才执行按保留天数清理（原 24h 固定周期改为 1h 轮询 + 配置间隔判断）

### 验证
- 清空接口实测：审计日志 25 条全部删除 ✓ / 未知类型返回 400 ✓ / 清空操作本身写入审计（deleted 后 total=1 为新审计记录）✓
- `go build` / `go test` / `vue-tsc` 0 错误 / `vite build` 7.8s ✓ / 服务重启 ✓

### 移动端操作按钮展开与对齐（用户需求：按钮不隐藏、等宽、规则换行）
1. **全局网格规则**（app-mobile.css）：`.mobile-card .card-actions` 移动端改为 `grid 3 列等宽`——3 个一行，5 个=3+2，6 个=3+3；**恰好 4 个按钮时自动 2 列（2+2）**（`:has(:nth-child(4):last-child)`）；按钮宽度 100% 贴齐
2. **用户管理移动端展开**：原「更多」下拉（藏重置密码/删除）改为 5 个按钮全部显示（详情/编辑/禁用启用/重置密码/删除）→ 3+2 布局
3. **节点管理移动端展开**：原下拉（藏上线/下线/删除）改为 6 个按钮（测试/编辑/禁用启用/上线/下线/删除）→ 3+3 布局
4. **全站遍历**：custom-nodes 4 按钮自动 2+2 ✓；email-queue 3 按钮一行 ✓；其余页面 1-3 按钮无隐藏；subscriptions 快捷操作（+1月/+3月等 12 个）已有 sub-btn-row 分组布局保留；用户端（device/redeem/mystery-box/ticket）无下拉藏按钮问题

## 全页面移动端按钮问题复查（2025-08-24）
- 用户指出仅检查了 3 个页面，要求全面复查所有页面按钮问题
- 脚本扫描全部 50+ admin/user 页面（card-actions / mobile-actions / sub-btn-row / batch-bar / n-dropdown / n-space 按钮组）
- 结果：所有 `.card-actions` 均在 `.mobile-card` 内 → 全局 3 列网格规则已覆盖；`.sub-btn-row`（5/6 按钮）与 `.sub-action-grid` 有局部 3→2 列规则；各 drawer footer n-space 正常；用户端页面无密集按钮行
- 新发现并修复：
  1. `admin/users/Index.vue` 批量操作栏（批量启用/禁用/删除/设置等级）移动端显示但无网格化 → 加 `batch-operations` 类
  2. `admin/nodes/Index.vue` batch-bar（5 按钮）移动端 n-space 会溢出 → 全局 CSS 新增 `.batch-bar/.batch-operations` 移动端 2 列等宽网格规则（含 batch-info 跨行、内嵌 n-space 网格化）
- 其余页面批量栏均有 `!appStore.isMobile` 保护（桌面专用）
- vue-tsc 0 错误，npm run build 8.31s，产物已含新规则

## 全页面搜索按钮补齐（2025-08-24）
- 用户要求：所有页面的搜索框都应有搜索按钮（点击搜索），回车确认也保留
- 脚本扫描全部页面搜索输入框（placeholder 含搜索/关键词/查询 + SearchOutline 前缀），逐项核对
- 修复：
  1. `components/SearchFilterBar.vue` — 组件内新增「搜索」按钮（@click 触发 search 事件），一处修改覆盖 nodes/custom-nodes/subscriptions 三个页面的桌面搜索栏；移动端布局改为搜索框+按钮同行、筛选器两列换行
  2. `admin/users/Index.vue` — 桌面 header 搜索框 + 移动端工具栏搜索框均补「搜索」按钮（移动端包入 mobile-toolbar-search 行）
  3. `admin/packages/Index.vue` — 桌面搜索框补「搜索」按钮
  4. `admin/nodes/Index.vue` — 移动端搜索框补「搜索」按钮（包入 mobile-toolbar-search 行）
  5. `admin/orders/Index.vue` — 桌面搜索框补「搜索」按钮（移动端原本已有）
  6. `admin/custom-nodes/Index.vue` — 移动端搜索行样式由竖排 column 改为 input+按钮同行（flex row）
  7. `styles/app-mobile.css` — 新增全局 `.mobile-toolbar-search` 规则（input 弹性 + 按钮同行）
- 复核无需改：invites 桌面/移动已有搜索按钮；subscriptions 移动已有；abnormal-users/email-queue/announcements/coupons/levels/redeem/tickets 无搜索框；config-update 关键词为编辑输入非搜索；UserDetailDrawer 为远程选择器；用户端页面无搜索框
- vue-tsc 0 错误，npm run build 8.29s；产物含 sf-search-btn/mobile-toolbar-search 规则

## 全量功能与按钮验证（4 子代理并行，2025-08-24）
### 验证范围
- 管理员端 + 用户端全部 API（GET 列表/详情 + mutation 安全验证）
- 管理员端 + 用户端全部页面按钮静态审计（死按钮/绑定/下拉/搜索按钮）
- 前端组件/API 导入/路由目标/分页契约一致性自查

### 子代理结果
1. **用户端 API 冒烟 36/36 通过**：31 项 GET（me/dashboard-info/devices/subscription/orders/notifications/tickets/invites/recharge/checkin/redeem/mystery-box/coupons/nodes 等）+ 5 项 mutation（签到/设备备注/升级计算/优惠券验证/CSRF），认证链路正常
2. **管理员端 API 冒烟 48 项 GET → 43 通过 + 5 失败**：3 项为清单路径不一致（/levels→/user-levels、/redeem→/redeem-codes、payment/analysis 缺必填参数，补测均正常）；2 项真实缺陷→已修复
3. **用户端页面审计 22 页 0 死按钮**：绑定 0 缺失，vue-tsc 0 错误，移动端网格覆盖完整
4. **管理员端页面审计**：绑定审计 OK 全部定义；死按钮 0（Login 返回按钮为 router-link 包裹、logs 清空按钮为 popconfirm trigger、settings 上传按钮为 n-upload trigger，均为误报）

### 修复的真实缺陷
- **GET /admin/packages/:id 404** → 新增 `AdminGetPackage` handler + 注册路由（admin_packages.go）
- **GET /admin/nodes/:id 404** → 新增 `AdminGetNode` handler + 注册路由（admin_nodes.go、router.go）
- 验证：两端点均返回 code:0；go build + go test 全绿；服务已重启

### 自查发现（脚本误报已排除）
- 234 个 API 导入全部完整（初版脚本正则 bug 误报，修正后 0 缺失）
- 全部 router.push 目标有效（嵌套路由）
- 分页契约一致（page_size）
- 3 个死导出待清理（updateSubscriptionProtocolFilter/testGitHubConnection/getPaymentGateway）
- 用户端 minor：subscription 页未用导入 TrashOutline/SwapHorizontalOutline/deleteDevice、ticket/node 冗余 NIcon 导入、dashboard levelProgress/device parseDeviceName 未使用

## 全站深挖审查与修复（4 子代理并行 + 主代理修复，2025-08-24）
### 审查子代理产出（4 份报告）
1. `/tmp/audit_security.md` — 后端安全深挖：1 P1 + 6 P2 + 11 低危（4 项实测复现）
2. `/tmp/audit_perf_logic.md` — 后端性能与逻辑：5 P1 + 10 P2 + 8 P3 性能、8 逻辑问题、索引建议
3. `/tmp/audit_fe_perf.md` — 前端性能：基线勘误（raw JS 2.45MB 非 4.5MB）、首屏 508KB、top 改进项
4. `/tmp/audit_fe_stability.md` — 前端稳定性：10 高危 + 18 中危 + 19 UX + 7 移动端

### 已修复（后端安全 8 项）
1. **[P1] 节点脱敏彻底失效**：hasActiveSub/GetNode/GetNodeStats 判定加 is_active + expire_time 校验（实测：过期用户 ListNodes/GetNode 均无 config，有订阅用户正常）
2. **[P2] 登录锁定 DoS**：失败计数改 (email+IP) 维度，攻击者无法锁死他人账号（实测：5 次失败后另一账号不被锁）
3. **[P2] Refresh token 并发重用竞态**：黑名单插入失败（唯一索引冲突）即拒绝，防令牌永续
4. **[P2] 邮箱大小写不一致**：SendVerificationCode/VerifyCode 统一 lowercase（实测含大写邮箱可发码）
5. **[P2] 改密/重置不吊销旧 token**：User 加 TokenVersion + JWT claim 携带 + AuthRequired/RefreshToken 校验 + 三处密码变更自增（ChangePassword/ResetPassword/AdminResetUserPassword）
6. **[P2] TestNode/BatchTestNodes 越权**：普通用户测试不写库，仅管理员回写全局节点状态
7. **[P2] 订阅重置不吊销旧链接**：旧 URL 映射限 7 天内有效
8. **[低] AdminDeleteUserFull 无自保**：补不能删自己/其他管理员

### 已修复（前端稳定性 10+ 项）
1. stats 财务导出 [object Object] → 取 res.data
2. admin/orders 资金操作（退款/取消/标记付款/完成/删除/批量）全补 try/catch
3. subscriptions「清除设备」误调重置订阅 → 新增后端 clear-devices 端点 + 前端改用真清除（实测验证）
4. custom-nodes 单条/批量删除加二次确认
5. 盲盒开盒加确认 + 全局并发锁（防多奖池并发双扣费）
6. nodes 移动端单行上下线误用批量勾选集 → 改传 row.id（修复功能失效）
7. ticket/invite confirm-loading → loading（CommonDrawer 正确 prop，防连点重复提交）
8. request.ts 401 刷新：裸 axios 加 10s 超时 + processQueue 边界兜底（防永久挂起）
9. settings/config-update 加载失败禁用保存（防默认值覆盖真实配置）
10. CommonDrawer 透传 after-leave → 10 处支付抽屉取消后立即停轮询
11. payment-gateways/NotFound 裸 @media 移入 style 块（修复页面渲染乱码）

### 已修复（后端性能 5 项）
1. payment_callbacks 加 (payment_transaction_id, processed) 复合索引
2. AdminCreateRedeemCodes 批量 CreateInBatches（卡密生成）
3. AdminImportNodes 预筛去重 + CreateInBatches（节点导入）
4. AdminImportUsersCSV 内存判重 + 用户/订阅批量创建（消除 15000 次查询）
5. AdminBatchUserAction enable N+1 → 一次 IN 查询 + 两条批量 UPDATE

### 其他
- xlsx 死依赖移除（export.ts 零引用删除 + npm uninstall + manualChunks 规则删除）
- 验证：go build + go test 全绿、vue-tsc 0 错误、npm run build 通过、服务重启后安全修复冒烟全部通过

## 第二轮深挖修复（2025-08-24）—— 剩余中高危漏洞与性能

### 资金/并发安全（6 项）
1. **优惠券 per-user 并发绕过（0元订单）**：事务内复核 CouponUsage per-user 计数（checkCouponPerUserInTx）+ CouponUsage 复合索引 idx_coupon_user；三处下单事务（CreateOrder/CustomOrder/UpgradeOrder）统一加复核
2. **验证码爆破防护**：VerificationAttempt 落地写入（此前模型存在但从不写入）+ checkVerificationLocked 按邮箱锁定（实测：5 次错误后第 6 次 429，attempts 表 5 条记录）
3. **下单/支付/充值/订阅重置限流**：CreateOrder/CustomOrder/UpgradeOrder/PayOrder/CreatePayment/CreateRecharge/CreateRechargePayment 各 10/min；ResetSubscription 3次/30分钟
4. **AdminUpdateSettings 键白名单**：validSettingKey 校验（小写字母数字下划线 + 排除 subscription_fetch_cache_* 保留前缀 + 值长度≤4096）（实测：非法键 400 拒绝，合法键正常）
5. **支付回调日志脱敏**：maskSensitiveParams 脱敏 Body/Query 中的 sign/sign_type
6. **GET /payment/methods 写库副作用**：PayType 加唯一索引 + 7 处 Create 改 OnConflict 幂等 upsert

### 逻辑正确性（4 项）
7. **认证用户缓存 5 分钟不失效**：新增 InvalidateUserCache，改密/管理员重置/禁用启用时清除
8. **ActivateSubscription 续期乐观锁**：CAS 条件更新（WHERE expire_time=旧值）+ 冲突重试一次，防并发支付丢续期
9. **AdminRefundOrder 双退风险**：网关退款成功后先独立提交订单 refunded 状态（防 DB 失败重试二次网关退款），再事务回滚余额/订阅
10. **金额浮点舍入**：finalAmount 统一 math.Round(x*100)/100

### 性能（1 项）
11. **订阅 payload 内存缓存兜底**：无 Redis 部署时用 MemoryCache 缓存订阅内容（TTL 5min）；ClearAllSubscriptionCache 同时清内存（ClearByPrefix("sub_payload:")）

### 前端稳定性（5 项）
12. **useTable 翻页/筛选自动清空勾选**（防批量误伤旧勾选行）
13. **用户端订单/充值移动端补分页控件** + 状态筛选重置页码
14. **subscription 页错误态与空态区分**（网络失败显示"加载失败"而非"您还没有订阅"）
15. **支付轮询防重叠**：setInterval 改递归 setTimeout（Shop 页，其余页面同型后续可推广）
16. **密码强度统一**：Register/settings/AdminLayout 三处校验对齐后端（≥8位+字母+数字）

### 验证
- go build + go test 全绿；vue-tsc 0 错误；npm run build 通过；服务重启（PID 11632）
- 冒烟实测：设置键白名单拒绝保留键✅ 验证码爆破锁定✅ 下单限流不误伤✅ payment/methods 正常✅

## 第三轮深挖修复（2025-08-24）—— 并发安全、缓存、前端一致性

### 后端（6 项）
1. **邀请码并发注册**：used_count 递增改条件更新抢占（WHERE used_count < max_uses），SQLite 下 FOR UPDATE 无效的兜底，消并发 500 与超发
2. **设备上限并发防护**：current_devices 递增改条件更新（WHERE current_devices < device_limit），RowsAffected=0 回滚设备创建
3. **公共端点缓存**：新增 utils 公共缓存（60s TTL），/config /packages /announcements 命中缓存；配置/套餐/公告变更时失效（InvalidateSettingsCache 联动 public_config + CRUD 处失效）
4. **金额浮点加固**：checkin 金额、盲盒 BalanceAfter 记录统一 math.Round(x*100)/100
5. **config_update 节点更新事务化**：删除→排序→CreateInBatches(200) 包单事务，中途失败整体回滚
6. **calcConsecutiveDays 口径统一**：改用 check_in_date 列（与防双签唯一索引同口径），消除 UTC/本地时区跨午夜偏差

### 前端（7 项）
7. **admin 分页双绑定双请求**：coupons/redeem/custom-nodes 桌面表移除重复 @update:page（useTable onChange 已自动触发），移动端分页保留
8. **tickets 状态 select 死控件**：回复时若选了状态则一并提交 updateTicket
9. **设备删除后抽屉不关闭**：成功路径加 showDeleteModal=false
10. **config-update 快速轮询泄漏**：fastPollInterval 提升模块级 + onUnmounted 清理
11. **登录跳转丢失 redirect**：守卫携带 ?redirect=原始路径，Login 登录成功后跳回（防 // 协议跳转）
12. **订阅设备将满自动弹抽屉**：仅首次触发（autoUpgradeShown 标记），不再每次刷新重弹
13. **工单 Ctrl+Enter 双发**：handleReply 加 replying 防重入

### 验证
- go build + go test 全绿；vue-tsc 0 错误；npm run build 通过；服务重启（PID 19247）
- 冒烟实测：公共端点 200、/config 缓存命中（0.0008s）、改设置后 /config 立即更新（缓存失效链路生效）、登录/下单正常

## 商业化差距评估 + 低优先级修复（2025-08-24，3 子代理并行评估）

### 商业化评估结果（3 份报告）
1. **UX 63/100**（/tmp/audit_commercial_ux.md）：移动端 App 化 8.5/10 上游水准；最大差距=流量 KPI 全站缺失、信任合规层为零、支付无品牌图标、首屏无品牌感、三套空态
2. **性能 72/100**（/tmp/audit_commercial_perf.md）：工程底子干净；缺 KeepAlive 路由缓存、大列表虚拟滚动、入口瘦身
3. **复用 68/100**（/tmp/audit_commercial_reuse.md）：基础设施达标；18 份日期格式化重复、20 页移动卡片样板、170 处魔法状态字符串

### 本轮修复
**后端（5 项）**
1. **[高危 B6] 批量删除用户不清理关联数据** → 复用 deleteUserRelatedData 事务清理 + 通知（实测：用户+订阅全清理）
2. **fmt.Printf 调试输出清理**：subscription.go 6 处 + payment.go 3 处 → utils 日志
3. **B9 死代码**：utils.ErrorWithFields/FieldError 删除（0 引用）
4. **B3 utils.Round2**：12 处 math.Round(x*100)/100 收敛
5. **B1 invite.go 分页统一**：最后 2 处手写分页改用 utils.GetPagination（含钳制）
6. **AdminDeleteUser 与 Full 合并**：deleteUserRelatedData 单点维护 20 张表删除清单（含 ghost 场景保护）

**前端（5 项）**
7. **入口瘦身**：删除 darkTheme 静态导入（换肤实际靠 themeOverrides+CSS 变量）→ 入口 278KB→211KB（-67KB raw / -15KB gzip）
8. **Return.vue 失败态"查看订阅"误渲染** → 仅成功显示，失败显示"重新支付"
9. **支付轮询统一防重叠**：Shop/order/Index/recharge/subscription 全部改递归 setTimeout
10. **死组件删除**：ErrorBoundary/TableSkeleton/UnifiedTable/UnifiedCardList/PullRefresh/apiHandler + 3 个死 utils（batchOperations/validate/performance）+ notifySystem 死导出
11. **设备写库去抖**：订阅拉取 last_access 更新限 5 分钟窗口（防 SQLite 写放大）
12. **重新应用被误恢复的修复**：subscription 错误态/autoUpgradeShown、tickets 状态 select、redeem/custom-nodes 分页双绑定

### 验证
- go build + go test 全绿；vue-tsc 0 错误；npm run build 通过（入口 211KB）；服务重启（PID 35370）
- 冒烟：批量删除清理验证通过（2 次实测）

### 剩余商业化差距（记录待后续）
- 流量 KPI 展示（后端补 used/total 字段 + 前端进度卡）——L 级
- 信任合规层（TOS/隐私/ICP/退款说明 + 客服入口）——S-M 级
- 支付品牌图标、订阅页信息架构重构、首屏品牌感——M 级
- KeepAlive 路由缓存、admin 大表虚拟滚动——M 级
- 日期格式化 18 份收敛（utils/date.ts 已建，迁移风险高暂缓）、UnifiedMobileCardList 试点、状态常量组

## 商业化批量改造（2025-08-24，4 子代理并行 + 主代理，纯时间制机场不含流量）

### 子代理产出（按文件域并行）
**A 信任合规层**（6 文件）：
- 新建 legal/Tos.vue《服务条款》+ Privacy.vue《隐私政策》页 + /terms /privacy 路由
- help 页"联系我们"改真实配置展示（support_email/qq/telegram）+ 协议链接 + 修复非法 CSS
- UserLayout 桌面下拉 + 移动抽屉加协议入口；Shop 购买区加信任说明（同意条款+退款政策+客服）
**B 支付品牌化 + 订阅架构**（4 文件）：
- 支付方式纯文字 radio → 品牌卡片网格（支付宝蓝/微信绿/USDT橙等 CSS 图标+名称+文案+选中态+余额不足禁用），修复 crypto 标签漂移
- 订阅页剩余天数做主角（52px 大数字+进度条+到期副文案），四宫格重排，续费 CTA（到期前7天），设备管理入口 + 修复 devices.length 混用
- Dashboard 订阅卡剩余天数大字 + 去购买 CTA + welcome-stat 类名去冲突 + 硬编码色换主题 token + 删 levelProgress 死代码
**C 首屏品牌感**（5 文件）：
- 新建 AuthLayout 公共组件（渐变品牌面板+玻璃 logo+卖点+光晕+入场动效+reduced-motion），三 auth 页收敛
- Login 表单加 label+内联校验+email 格式+成功 300ms 过渡+记住我 sessionStorage 落地
- Register 加确认密码+同意条款勾选；ForgotPassword 加 email 格式校验+成功后 3 秒自动跳登录
**D 全局一致性**（7 文件）：
- 新建 EmptyState 统一空态组件（NotFound 示范）+ useCountUp 数字滚动（rAF 零依赖）
- dashboard 余额/签到/剩余天数 count-up + 9 卡 stagger 入场动效
- Return.vue 成功庆祝（CSS confetti）；统一圆角 12px/hover 微阴影/按压 scale(0.98)/暗色防御

### 主代理
- **KeepAlive 路由缓存**：UserLayout 两处 router-view 加 KeepAlive（14 个高频页缓存），key 改 route.name；Dashboard/Orders/Subscription 加 onActivated 刷新（缓存激活时数据最新）

### 验证
- vue-tsc 0 错误（含子代理并发写 dashboard 的收敛）；npm run build 通过（入口 211KB 保持瘦身）；go build+test 全绿；服务重启（PID 45030）；/terms SPA 路由 200

### 商业化成熟度提升预估
- UX 63→预计 75+（信任层/品牌化/订阅架构/动效/空态统一落地）
- 性能 72→75+（KeepAlive 消除切 tab 重挂载+重拉 7 接口）
