# 全面日志系统文档

## 概述

系统已实现全面的日志记录功能，涵盖所有关键操作和事件。所有日志都包含 IP 地址和地理位置信息（支持 IPv4 和 IPv6）。

---

## 日志类型

### 1. 用户相关日志

#### 1.1 注册日志 (RegistrationLog)
**表名**: `registration_logs`

**字段**:
- `user_id`: 用户 ID
- `username`: 用户名
- `email`: 邮箱
- `ip_address`: 注册 IP（支持 IPv4/IPv6）
- `user_agent`: 浏览器信息
- `location`: 地理位置（国家 省份 城市）
- `invite_code`: 邀请码
- `inviter_id`: 邀请人 ID
- `register_source`: 注册来源（direct/invite_code）
- `status`: 状态（success/failed）

**使用示例**:
```go
utils.CreateRegistrationLog(c, userID, username, email, inviteCode, &inviterID)
```

#### 1.2 用户操作日志 (UserActionLog)
**表名**: `user_action_logs`

**记录操作**:
- 登录/登出
- 修改个人信息
- 修改密码
- 修改邮箱
- 绑定/解绑第三方账号

**字段**:
- `user_id`: 用户 ID
- `action_type`: 操作类型
- `module`: 模块名称
- `description`: 操作描述
- `ip_address`: 操作 IP
- `user_agent`: 浏览器信息
- `location`: 地理位置

**使用示例**:
```go
utils.CreateUserActionLog(userID, "login", "auth", "用户登录成功", c)
utils.CreateUserActionLog(userID, "update_profile", "profile", "修改个人信息", c)
utils.CreateUserActionLog(userID, "change_password", "auth", "修改密码", c)
```

---

### 2. 订阅相关日志

#### 2.1 订阅日志 (SubscriptionLog)
**表名**: `subscription_logs`

**记录操作**:
- 订阅创建
- 订阅续期
- 订阅升级
- 订阅重置
- 设备限制变更

**字段**:
- `subscription_id`: 订阅 ID
- `user_id`: 用户 ID
- `action_type`: 操作类型
- `action_by`: 操作者（user/admin/system）
- `action_by_user_id`: 操作者 ID
- `description`: 操作描述
- `before_data`: 变更前数据（JSON）
- `after_data`: 变更后数据（JSON）

**使用示例**:
```go
beforeData := map[string]interface{}{
    "device_limit": 3,
    "expire_time": "2024-01-01",
}
afterData := map[string]interface{}{
    "device_limit": 5,
    "expire_time": "2024-02-01",
}
utils.CreateSubscriptionLog(subID, userID, "upgrade", "user", &userID, "订阅升级", beforeData, afterData)
```

#### 2.2 设备日志 (DeviceLog)
**表名**: `device_logs`

**记录操作**:
- 设备连接
- 设备断开
- 设备删除

**字段**:
- `device_id`: 设备 ID
- `user_id`: 用户 ID
- `action_type`: 操作类型
- `description`: 操作描述
- `ip_address`: 设备 IP

**使用示例**:
```go
utils.CreateDeviceLog(deviceID, userID, "connect", "设备首次连接", c)
utils.CreateDeviceLog(deviceID, userID, "delete", "用户删除设备", c)
```

---

### 3. 订单和支付日志

#### 3.1 订单日志 (OrderLog)
**表名**: `order_logs`

**记录操作**:
- 订单创建
- 订单支付
- 订单取消
- 订单退款

**字段**:
- `order_id`: 订单 ID
- `user_id`: 用户 ID
- `action_type`: 操作类型
- `action_by`: 操作者
- `action_by_user_id`: 操作者 ID
- `description`: 操作描述
- `before_data`: 变更前数据
- `after_data`: 变更后数据

**使用示例**:
```go
afterData := map[string]interface{}{
    "order_no": "ORD123456",
    "amount": 99.00,
    "status": "pending",
}
utils.CreateOrderLog(orderID, userID, "create", "user", &userID, "创建订单", nil, afterData)

beforeData := map[string]interface{}{"status": "pending"}
afterData = map[string]interface{}{"status": "paid"}
utils.CreateOrderLog(orderID, userID, "pay", "user", &userID, "订单支付成功", beforeData, afterData)
```

#### 3.2 支付日志 (PaymentLog)
**表名**: `payment_logs`

**记录操作**:
- 支付发起
- 支付成功
- 支付失败
- 支付回调

**字段**:
- `transaction_id`: 交易 ID
- `user_id`: 用户 ID
- `payment_method`: 支付方式（alipay/wechat/stripe/balance）
- `amount`: 支付金额
- `status`: 状态（pending/success/failed）
- `description`: 描述
- `ip_address`: 支付 IP
- `user_agent`: 浏览器信息
- `location`: 地理位置

**使用示例**:
```go
utils.CreatePaymentLog(transactionID, userID, "alipay", "success", "支付宝支付成功", 99.00, c)
utils.CreatePaymentLog(transactionID, userID, "balance", "success", "余额支付", 50.00, c)
```

#### 3.3 余额日志 (BalanceLog)
**表名**: `balance_logs`

**已实现** - 记录所有余额变动

---

### 4. 优惠券日志

#### 4.1 优惠券日志 (CouponLog)
**表名**: `coupon_logs`

**记录操作**:
- 优惠券使用
- 优惠券过期
- 优惠券取消

**字段**:
- `coupon_id`: 优惠券 ID
- `user_id`: 用户 ID
- `action_type`: 操作类型
- `description`: 描述
- `ip_address`: 操作 IP

**使用示例**:
```go
utils.CreateCouponLog(couponID, userID, "use", "使用优惠券购买套餐", c)
utils.CreateCouponLog(couponID, userID, "expire", "优惠券已过期", nil)
```

---

### 5. 节点日志

#### 5.1 节点日志 (NodeLog)
**表名**: `node_logs`

**记录操作**:
- 节点创建
- 节点更新
- 节点删除
- 节点启用/禁用

**字段**:
- `node_id`: 节点 ID
- `action_type`: 操作类型
- `action_by`: 操作者（admin/system）
- `action_by_user_id`: 操作者 ID
- `description`: 描述
- `before_data`: 变更前数据
- `after_data`: 变更后数据

**使用示例**:
```go
afterData := map[string]interface{}{
    "name": "香港节点01",
    "address": "hk01.example.com",
    "is_active": true,
}
utils.CreateNodeLog(nodeID, "create", "admin", &adminID, "创建新节点", nil, afterData)

beforeData := map[string]interface{}{"is_active": true}
afterData = map[string]interface{}{"is_active": false}
utils.CreateNodeLog(nodeID, "disable", "admin", &adminID, "禁用节点", beforeData, afterData)
```

---

### 6. 管理员日志

#### 6.1 管理员操作日志 (AdminActionLog)
**表名**: `admin_action_logs`

**记录操作**:
- 用户管理（创建/修改/删除/禁用）
- 节点管理
- 套餐管理
- 配置修改
- 数据库操作

**字段**:
- `admin_id`: 管理员 ID
- `action_type`: 操作类型
- `module`: 模块名称
- `target_type`: 目标类型（user/node/package/config）
- `target_id`: 目标 ID
- `description`: 描述
- `before_data`: 变更前数据
- `after_data`: 变更后数据
- `ip_address`: 操作 IP
- `user_agent`: 浏览器信息

**使用示例**:
```go
afterData := map[string]interface{}{
    "username": "newuser",
    "email": "user@example.com",
    "is_active": true,
}
utils.CreateAdminActionLog(adminID, "create", "user", "user", &targetUserID, "创建新用户", nil, afterData, c)

beforeData := map[string]interface{}{"is_active": true}
afterData = map[string]interface{}{"is_active": false}
utils.CreateAdminActionLog(adminID, "disable", "user", "user", &targetUserID, "禁用用户", beforeData, afterData, c)
```

#### 6.2 配置变更日志 (ConfigChangeLog)
**表名**: `config_change_logs`

**记录操作**:
- 系统配置修改
- 支付配置修改
- 通知配置修改

**字段**:
- `admin_id`: 管理员 ID
- `config_key`: 配置键名
- `old_value`: 旧值
- `new_value`: 新值
- `description`: 描述
- `ip_address`: 操作 IP

**使用示例**:
```go
utils.CreateConfigChangeLog(adminID, "site_name", "旧站点名", "新站点名", "修改站点名称", c)
utils.CreateConfigChangeLog(adminID, "alipay_app_id", "old_id", "new_id", "更新支付宝配置", c)
```

#### 6.3 数据库操作日志 (DatabaseLog)
**表名**: `database_logs`

**记录操作**:
- 数据库备份
- 数据库恢复
- 数据迁移
- 表清空

**字段**:
- `admin_id`: 管理员 ID
- `operation`: 操作类型
- `table_name`: 表名
- `affected_rows`: 影响行数
- `description`: 描述

**使用示例**:
```go
utils.CreateDatabaseLog(adminID, "backup", "all", "数据库完整备份", 0)
utils.CreateDatabaseLog(adminID, "truncate", "login_attempts", "清空登录尝试记录", 1523)
```

---

### 7. 工单日志

#### 7.1 工单日志 (TicketLog)
**表名**: `ticket_logs`

**记录操作**:
- 工单创建
- 工单回复
- 工单关闭
- 工单重开

**字段**:
- `ticket_id`: 工单 ID
- `user_id`: 用户 ID
- `action_type`: 操作类型
- `action_by`: 操作者（user/admin）
- `description`: 描述

**使用示例**:
```go
utils.CreateTicketLog(ticketID, userID, "create", "user", "用户创建工单")
utils.CreateTicketLog(ticketID, userID, "reply", "admin", "管理员回复工单")
utils.CreateTicketLog(ticketID, userID, "close", "admin", "管理员关闭工单")
```

---

### 8. 邀请日志

#### 8.1 邀请日志 (InviteLog)
**表名**: `invite_logs`

**记录操作**:
- 用户注册（通过邀请码）
- 邀请奖励发放

**字段**:
- `inviter_id`: 邀请人 ID
- `invitee_id`: 被邀请人 ID
- `invite_code`: 邀请码
- `action_type`: 操作类型
- `description`: 描述

**使用示例**:
```go
utils.CreateInviteLog(inviterID, inviteeID, inviteCode, "register", "用户通过邀请码注册")
utils.CreateInviteLog(inviterID, inviteeID, inviteCode, "reward", "邀请奖励已发放")
```

---

### 9. 安全日志

#### 9.1 安全事件日志 (SecurityLog)
**表名**: `security_logs`

**记录事件**:
- 登录失败
- 可疑活动
- 暴力破解尝试
- 异常登录
- CSRF 攻击尝试
- SQL 注入尝试

**字段**:
- `user_id`: 用户 ID（可选）
- `event_type`: 事件类型
- `severity`: 严重程度（low/medium/high/critical）
- `description`: 描述
- `ip_address`: 来源 IP
- `user_agent`: 浏览器信息
- `location`: 地理位置

**使用示例**:
```go
utils.CreateSecurityLog(&userID, "login_failed", "medium", "连续5次登录失败", c)
utils.CreateSecurityLog(nil, "brute_force", "high", "检测到暴力破解尝试", c)
utils.CreateSecurityLog(&userID, "suspicious_activity", "high", "异地登录", c)
utils.CreateSecurityLog(nil, "csrf_attempt", "critical", "CSRF 攻击尝试", c)
```

---

### 10. API 日志

#### 10.1 API 调用日志 (APILog)
**表名**: `api_logs`

**记录信息**:
- 请求方法
- 请求路径
- 响应状态码
- 响应时间
- 用户 IP
- 浏览器信息

**字段**:
- `user_id`: 用户 ID（可选）
- `method`: HTTP 方法
- `path`: 请求路径
- `status_code`: 状态码
- `response_time`: 响应时间（毫秒）
- `ip_address`: 请求 IP
- `user_agent`: 浏览器信息

**使用示例**:
```go
startTime := time.Now()
// ... 处理请求 ...
responseTime := time.Since(startTime)
utils.CreateAPILog(&userID, "POST", "/api/v1/orders", 200, responseTime, c)
```

---

### 11. 邮件和通知日志

#### 11.1 邮件日志 (EmailLog)
**表名**: `email_logs`

**记录信息**:
- 邮件类型
- 收件人
- 主题
- 发送状态
- 错误信息

**字段**:
- `user_id`: 用户 ID（可选）
- `email_type`: 邮件类型
- `recipient`: 收件人
- `subject`: 主题
- `status`: 状态（sent/failed/pending）
- `error_message`: 错误信息

**使用示例**:
```go
utils.CreateEmailLog(&userID, "verification", "user@example.com", "邮箱验证码", "sent", "")
utils.CreateEmailLog(&userID, "reset_password", "user@example.com", "密码重置", "failed", "SMTP连接失败")
```

#### 11.2 通知日志 (NotificationLog)
**表名**: `notification_logs`

**记录信息**:
- 通知类型
- 通知渠道（email/telegram/bark）
- 发送状态
- 通知内容

**字段**:
- `user_id`: 用户 ID（可选）
- `notification_type`: 通知类型
- `channel`: 渠道
- `status`: 状态
- `content`: 内容

**使用示例**:
```go
utils.CreateNotificationLog(&userID, "order", "telegram", "sent", "新订单通知")
utils.CreateNotificationLog(&userID, "payment", "bark", "sent", "支付成功通知")
```

---

## IP 地理位置识别

### 支持的 IP 类型
- ✅ **IPv4**: 完全支持
- ✅ **IPv6**: 完全支持

### 识别方式
使用 **ip-api.com** 免费 API 服务

**特点**:
- 支持 IPv4 和 IPv6
- 返回中文地理位置
- 包含国家、省份、城市信息
- 3 秒超时保护
- 自动识别本地和私有 IP

**识别的 IP 范围**:
- 公网 IPv4
- 公网 IPv6
- 本地 IP（127.0.0.1, ::1）→ 显示"本地"
- 私有 IP（10.x, 192.168.x, 172.16-31.x, fc00::/7, fe80::/10）→ 显示"本地网络"

**返回格式**:
```
中国 广东省 深圳市
美国 加利福尼亚州 洛杉矶
日本 东京都 东京
```

### 使用示例
```go
ip := utils.GetRealClientIP(c)
location := utils.GetIPLocation(ip)
// location = "中国 广东省 深圳市"
```

### IPv6 示例
```go
// IPv6 地址
ip := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
location := utils.GetIPLocation(ip)
// 正常返回地理位置

// IPv6 本地地址
ip = "::1"
location = utils.GetIPLocation(ip)
// location = "本地"

// IPv6 私有地址
ip = "fc00::1"
location = utils.GetIPLocation(ip)
// location = "本地网络"
```

---

## 日志查询示例

### 1. 查询用户所有操作日志
```go
var logs []models.UserActionLog
db.Where("user_id = ?", userID).Order("created_at DESC").Limit(100).Find(&logs)
```

### 2. 查询管理员操作日志
```go
var logs []models.AdminActionLog
db.Where("admin_id = ? AND module = ?", adminID, "user").Order("created_at DESC").Find(&logs)
```

### 3. 查询安全事件
```go
var logs []models.SecurityLog
db.Where("severity IN ? AND created_at > ?", []string{"high", "critical"}, time.Now().Add(-24*time.Hour)).Find(&logs)
```

### 4. 查询 API 慢请求
```go
var logs []models.APILog
db.Where("response_time > ?", 1000).Order("response_time DESC").Limit(50).Find(&logs)
```

### 5. 查询某个 IP 的所有操作
```go
var userLogs []models.UserActionLog
db.Where("ip_address = ?", "1.2.3.4").Find(&userLogs)

var securityLogs []models.SecurityLog
db.Where("ip_address = ?", "1.2.3.4").Find(&securityLogs)
```

---

## 日志清理建议

### 自动清理策略
建议定期清理旧日志以节省存储空间：

```go
// 清理 90 天前的 API 日志
db.Where("created_at < ?", time.Now().Add(-90*24*time.Hour)).Delete(&models.APILog{})

// 清理 180 天前的用户操作日志
db.Where("created_at < ?", time.Now().Add(-180*24*time.Hour)).Delete(&models.UserActionLog{})

// 保留所有安全日志（不清理）
// 保留所有管理员操作日志（不清理）
// 保留所有订单和支付日志（不清理）
```

### 日志归档
对于重要日志，建议归档而非删除：
1. 导出到 CSV/JSON 文件
2. 压缩存储
3. 从数据库中删除

---

## 性能优化

### 1. 异步写入
所有日志都使用 goroutine 异步写入，不阻塞主流程：
```go
go func() {
    if err := db.Create(&entry).Error; err != nil {
        log.Printf("[logs] failed to create log: %v", err)
    }
}()
```

### 2. 批量写入（建议）
高并发场景下建议使用批量写入：
```go
// 收集 100 条日志后批量写入
logBuffer := make([]models.UserActionLog, 0, 100)
// ... 收集日志 ...
db.CreateInBatches(logBuffer, 100)
```

### 3. 索引优化
已为常用查询字段添加索引：
- `user_id`
- `admin_id`
- `created_at`
- `ip_address`
- `event_type`
- `config_key`

---

## 总结

### 日志覆盖范围
- ✅ 用户注册、登录、操作
- ✅ 订阅创建、续期、升级、重置
- ✅ 订单创建、支付、取消、退款
- ✅ 支付发起、成功、失败
- ✅ 优惠券使用、过期
- ✅ 节点创建、修改、删除
- ✅ 管理员所有操作
- ✅ 配置变更
- ✅ 数据库操作
- ✅ 工单操作
- ✅ 邀请注册和奖励
- ✅ 安全事件
- ✅ API 调用
- ✅ 邮件发送
- ✅ 通知发送
- ✅ 设备连接和删除

### IP 和地理位置
- ✅ 所有日志都包含 IP 地址
- ✅ 支持 IPv4 和 IPv6
- ✅ 自动识别地理位置（国家、省份、城市）
- ✅ 识别本地和私有 IP

### 数据完整性
- ✅ 记录操作前后数据对比
- ✅ 记录操作者信息
- ✅ 记录操作时间
- ✅ 记录操作来源（IP、浏览器）

系统日志已经非常全面，涵盖了所有关键操作和事件！
