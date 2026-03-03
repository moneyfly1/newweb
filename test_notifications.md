# 通知功能测试指南

## 通知系统架构

### 1. 用户通知（NotifyUser）
**检查流程**：
1. 系统级开关（system_configs 表）
2. 用户级开关（users 表）
3. 发送邮件

**系统级开关**（在后台设置 → 通知设置）：
- `user_notify_welcome` - 欢迎邮件
- `user_notify_payment` - 支付成功通知
- `user_notify_expiry` - 到期提醒
- `user_notify_expired` - 已过期通知
- `user_notify_reset` - 订阅重置通知
- `user_notify_account_status` - 账户状态变更
- `user_notify_unpaid_order` - 未支付订单提醒

**用户级开关**（用户个人设置）：
- `email_notifications` - 总开关
- `notify_order` - 订单通知
- `notify_expiry` - 到期通知
- `notify_subscription` - 订阅通知
- `abnormal_login_alert_enabled` - 异常登录提醒

### 2. 管理员通知（NotifyAdmin）
**支持三种渠道**：
1. 邮件（Email）
2. Telegram
3. Bark（iOS 推送）

**系统级开关**（在后台设置 → 通知设置）：
- `notify_new_order` - 新订单通知
- `notify_payment_success` - 支付成功通知
- `notify_recharge_success` - 充值成功通知
- `notify_new_ticket` - 新工单通知
- `notify_new_user` - 新用户注册通知
- `notify_subscription_reset` - 订阅重置通知
- `notify_abnormal_login` - 异常登录通知
- `notify_unpaid_order` - 未支付订单通知
- `notify_expiry_reminder` - 到期提醒通知

**渠道开关**：
- `notify_email_enabled` - 邮件通知开关
- `notify_telegram_enabled` - Telegram 通知开关
- `notify_bark_enabled` - Bark 通知开关

## 测试场景

### 场景 1：新订单通知
**触发位置**：`handlers/order.go:155-161`
```go
go services.NotifyUser(userID, "new_order", ...)
go services.NotifyAdmin("new_order", ...)
```

**测试步骤**：
1. 后台开启 `notify_new_order`
2. 用户购买套餐
3. 检查：
   - 用户收到订单邮件（如果用户开启了 `notify_order`）
   - 管理员收到邮件/Telegram/Bark 通知

### 场景 2：支付成功通知
**触发位置**：`handlers/order.go:356`
```go
go services.NotifyAdmin("payment_success", ...)
```

**测试步骤**：
1. 后台开启 `notify_payment_success`
2. 用户完成支付
3. 检查管理员收到通知

### 场景 3：充值成功通知
**触发位置**：`handlers/payment.go:894`
```go
go services.NotifyAdmin("recharge_success", ...)
```

**测试步骤**：
1. 后台开启 `notify_recharge_success`
2. 用户充值余额
3. 检查管理员收到通知

### 场景 4：新工单通知
**触发位置**：`handlers/ticket.go:79`
```go
go services.NotifyAdmin("new_ticket", ...)
```

**测试步骤**：
1. 后台开启 `notify_new_ticket`
2. 用户创建工单
3. 检查管理员收到通知

### 场景 5：新用户注册通知
**触发位置**：`handlers/auth.go:303`
```go
go services.NotifyAdmin("new_user", ...)
```

**测试步骤**：
1. 后台开启 `notify_new_user`
2. 新用户注册
3. 检查管理员收到通知

### 场景 6：订阅重置通知
**触发位置**：
- 用户自己重置：`handlers/subscription.go:647`
- 管理员重置：`handlers/admin.go:1540`

**测试步骤**：
1. 后台开启 `notify_subscription_reset`
2. 重置订阅
3. 检查：
   - 用户收到邮件（如果开启 `notify_subscription`）
   - 管理员收到通知

### 场景 7：异常登录通知
**触发位置**：`handlers/auth.go:400`
```go
go services.NotifyUser(user.ID, "abnormal_login", ...)
```

**测试步骤**：
1. 用户开启 `abnormal_login_alert_enabled`
2. 从新 IP 登录
3. 检查用户收到邮件

### 场景 8：账户状态变更通知
**触发位置**：
- 启用/禁用：`handlers/admin.go:481-483`
- 删除：`handlers/admin.go:373, 2704`

**测试步骤**：
1. 后台开启 `user_notify_account_status`
2. 管理员启用/禁用/删除用户
3. 检查用户收到邮件

## 验证方法

### 1. 检查系统配置
```sql
SELECT key, value FROM system_configs
WHERE key LIKE 'notify_%' OR key LIKE 'user_notify_%';
```

### 2. 检查用户配置
```sql
SELECT id, username, email_notifications, notify_order, notify_expiry,
       notify_subscription, abnormal_login_alert_enabled
FROM users WHERE id = ?;
```

### 3. 检查邮件队列
```sql
SELECT * FROM email_queue
WHERE status = 'pending'
ORDER BY created_at DESC LIMIT 10;
```

### 4. 查看系统日志
```sql
SELECT * FROM system_logs
WHERE module = 'notify'
ORDER BY created_at DESC LIMIT 20;
```

## 通知逻辑流程图

```
用户通知流程：
┌─────────────────┐
│  触发事件       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 检查系统级开关  │ ← system_configs 表
└────────┬────────┘
         │ 关闭 → 不发送
         ▼ 开启
┌─────────────────┐
│ 检查用户级开关  │ ← users 表
└────────┬────────┘
         │ 关闭 → 不发送
         ▼ 开启
┌─────────────────┐
│  发送邮件       │
└─────────────────┘

管理员通知流程：
┌─────────────────┐
│  触发事件       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 检查事件开关    │ ← notify_new_order 等
└────────┬────────┘
         │ 关闭 → 不发送
         ▼ 开启
┌─────────────────┬─────────────────┬─────────────────┐
│  邮件渠道       │  Telegram 渠道  │  Bark 渠道      │
│ (检查开关)      │  (检查开关)     │  (检查开关)     │
└─────────────────┴─────────────────┴─────────────────┘
```

## 常见问题

### Q1: 用户没收到邮件？
检查：
1. 系统级开关是否开启
2. 用户级开关是否开启
3. 邮件服务是否配置正确
4. 查看 email_queue 表状态

### Q2: 管理员没收到通知？
检查：
1. 事件开关是否开启（如 `notify_new_order`）
2. 渠道开关是否开启（如 `notify_email_enabled`）
3. 渠道配置是否正确（邮箱、Bot Token、Chat ID 等）
4. 查看 system_logs 表错误信息

### Q3: 如何测试 Telegram 通知？
后台设置 → 通知设置 → Telegram 配置 → 点击"测试 Telegram"按钮

### Q4: Bark 通知格式？
URL 格式：`{server}/{device_key}/{title}/{body}`
