# 安全加固报告

## 修复日期
2026-03-02

## 已修复的安全漏洞

### 1. 支付回调重放攻击防护 ✅

**问题描述：**
- 支付回调（易支付/支付宝/Stripe）缺少 nonce 验证
- 攻击者可以重放合法的支付回调，导致重复充值或订单激活

**修复方案：**
- 新增 `payment_nonces` 表记录已处理的回调
- 每个支付回调在处理前检查 nonce 是否已存在
- nonce 24小时后自动过期清理
- 在事务内原子性记录 nonce，防止并发重放

**影响文件：**
- `/internal/models/payment_nonce.go` (新增)
- `/internal/api/handlers/payment.go` (修改)
- `/migrations/add_payment_nonces.sql` (新增)

**代码示例：**
```go
// 检查 nonce 防止重放
if models.IsNonceProcessed(db, outTradeNo, "epay") {
    utils.SysError("payment", fmt.Sprintf("易支付回调重放攻击检测: %s", outTradeNo))
    c.String(200, "success")
    return
}

// 在事务内记录 nonce
if err := models.RecordNonce(tx, outTradeNo, "epay", tradeNo); err != nil {
    return fmt.Errorf("记录 nonce 失败: %w", err)
}
```

---

### 2. 支付金额校验增强 ✅

**问题描述：**
- 部分支付回调缺少金额验证
- 金额验证不严格，可能被篡改

**修复方案：**
- 所有支付回调强制验证金额字段
- 金额不匹配时记录安全日志并拒绝处理
- Stripe 金额校验考虑汇率精度，允许 1 分误差
- 缺少金额字段时拒绝处理

**代码示例：**
```go
// 金额校验（必须严格匹配）
if callbackMoney != "" {
    expectedAmount := fmt.Sprintf("%.2f", txn.Amount)
    if callbackMoney != expectedAmount {
        utils.SysError("payment", fmt.Sprintf("易支付金额不匹配: 订单 %s, 期望 %s, 实际 %s", outTradeNo, expectedAmount, callbackMoney))
        return fmt.Errorf("金额不匹配: 期望 %s, 实际 %s", expectedAmount, callbackMoney)
    }
} else {
    utils.SysError("payment", fmt.Sprintf("易支付回调缺少金额字段: %s", outTradeNo))
    return fmt.Errorf("回调数据缺少金额字段")
}
```

---

### 3. 订阅地址枚举防护 ✅

**问题描述：**
- 订阅 URL 仅 32 位随机字符串
- 无访问频率限制，可能被暴力枚举

**修复方案：**
- 记录失败的订阅访问尝试（含 IP）
- 添加安全日志用于检测枚举攻击
- 建议：在路由层添加订阅访问频率限制（见下方建议）

**代码示例：**
```go
if err := db.Where("subscription_url = ?", url).First(&sub).Error; err != nil {
    // 记录失败的订阅访问（用于检测枚举攻击）
    utils.SysError("subscription", fmt.Sprintf("订阅地址不存在访问尝试: %s from IP: %s", url, clientIP))
    ctx.Status = subStatusNotFound
    return ctx
}
```

---

### 4. CSRF 保护机制 ✅

**问题描述：**
- 关键操作（充值、订单、订阅重置）缺少 CSRF token
- 可能被跨站请求伪造攻击

**修复方案：**
- 新增 CSRF 中间件
- 为每个用户生成唯一的 CSRF token
- Token 24小时有效期，自动清理过期 token
- 使用常量时间比较防止时序攻击

**影响文件：**
- `/internal/api/middleware/csrf.go` (新增)

**使用方法：**
```go
// 在路由中添加 CSRF 保护
protected := router.Group("/api/v1")
protected.Use(middleware.AuthRequired())
protected.Use(middleware.CSRFProtection()) // 添加 CSRF 保护

// 获取 CSRF token
protected.GET("/csrf-token", middleware.GetCSRFToken)

// 前端请求时携带 token
headers: {
    'X-CSRF-Token': csrfToken
}
```

---

### 5. Token 刷新安全增强 ✅

**问题描述：**
- Refresh token 刷新后旧 token 仍可使用
- 存在会话固定攻击风险

**修复方案：**
- 刷新 token 时将旧 refresh token 加入黑名单
- 防止旧 token 被重用
- 检查 refresh token 是否在黑名单中

**代码示例：**
```go
// 将旧的 refresh token 加入黑名单（防止重用）
db := database.GetDB()
expiresAt := time.Now().Add(time.Duration(config.AppConfig.RefreshTokenExpireDays) * 24 * time.Hour)
models.AddToBlacklist(db, tokenHash, user.ID, expiresAt)
```

---

## 仍需改进的安全问题

### 1. 订阅访问频率限制 ⚠️

**建议：**
在订阅路由添加更严格的频率限制：

```go
// 在 router.go 中
subGroup := router.Group("/api/v1/sub")
subGroup.Use(middleware.RateLimit(60, 1*time.Minute)) // 每分钟最多 60 次
subGroup.GET("/:url", handlers.GetSubscription)
subGroup.GET("/clash/:url", handlers.GetSubscription)
```

### 2. 前端敏感信息脱敏 ⚠️

**建议：**
- 订阅地址仅在用户主动点击"显示"时才显示完整内容
- 默认显示为 `****...****` 格式
- 用户余额可选择性隐藏

### 3. 支付回调 IP 白名单 ⚠️

**建议：**
为支付回调添加 IP 白名单验证：

```go
// 在 payment.go 中添加
func verifyCallbackIP(c *gin.Context, payType string) bool {
    clientIP := utils.GetRealClientIP(c)
    whitelist := utils.GetSetting(fmt.Sprintf("pay_%s_callback_ips", payType))
    if whitelist == "" {
        return true // 未配置白名单则允许
    }
    // 检查 IP 是否在白名单中
    return strings.Contains(whitelist, clientIP)
}
```

### 4. 订单过期自动取消 ⚠️

**建议：**
添加定时任务自动取消过期订单：

```go
// 在 scheduler.go 中添加
func CancelExpiredOrders() {
    db := database.GetDB()
    db.Model(&models.Order{}).
        Where("status = ? AND expire_time < ?", "pending", time.Now()).
        Update("status", "expired")
}
```

### 5. 数据库查询参数化 ✅

**当前状态：**
代码已使用 GORM 参数化查询，无 SQL 注入风险。

### 6. XSS 防护 ✅

**当前状态：**
- 已设置 `X-Content-Type-Options: nosniff`
- 已设置 CSP 策略
- 前端使用 Vue.js 自动转义

---

## 安全配置建议

### 1. 环境变量保护

确保 `.env` 文件不被提交到版本控制：
```bash
# .gitignore
.env
*.db
*.log
```

### 2. 数据库备份

定期备份数据库，防止数据丢失：
```bash
# 每天凌晨 2 点备份
0 2 * * * sqlite3 /path/to/cboard.db ".backup '/path/to/backups/cboard_$(date +\%Y\%m\%d).db'"
```

### 3. 日志监控

监控安全日志，及时发现异常：
```bash
# 监控支付回调异常
grep "金额不匹配\|重放攻击" backend.log

# 监控订阅枚举尝试
grep "订阅地址不存在访问尝试" backend.log
```

### 4. HTTPS 强制

生产环境必须使用 HTTPS：
```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

---

## 测试建议

### 1. 支付回调重放测试

```bash
# 捕获合法支付回调
# 重放相同请求，应返回成功但不重复处理
curl -X POST "http://localhost:8000/api/v1/payment/notify/epay" \
  -d "out_trade_no=PAY123&trade_status=TRADE_SUCCESS&money=100.00&sign=xxx"
```

### 2. CSRF 攻击测试

```bash
# 不带 CSRF token 的请求应被拒绝
curl -X POST "http://localhost:8000/api/v1/orders" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"package_id":1}'
```

### 3. 订阅枚举测试

```bash
# 快速访问不存在的订阅地址
for i in {1..100}; do
  curl "http://localhost:8000/api/v1/sub/invalid$i"
done
# 检查日志是否记录异常访问
```

---

## 部署清单

- [ ] 运行数据库迁移：`sqlite3 cboard.db < migrations/add_payment_nonces.sql`
- [ ] 重新编译后端：`go build -o cboard cmd/server/main.go`
- [ ] 重启服务：`./start.sh`
- [ ] 验证支付回调防重放功能
- [ ] 配置订阅访问频率限制
- [ ] 启用 HTTPS
- [ ] 配置日志监控告警
- [ ] 定期备份数据库

---

## 联系方式

如发现新的安全问题，请立即联系管理员。

**修复完成时间：** 2026-03-02
**修复人员：** Claude (AI Security Audit)
