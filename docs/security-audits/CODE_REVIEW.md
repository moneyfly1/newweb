# 代码安全与质量审查报告

**审查日期**: 2026-02-14  
**审查范围**: 前端 (Vue 3 + TypeScript) + 后端 (Go + Gin + GORM)

---

## 🔴 严重安全问题 (Critical)

### 1. **密码明文发送到邮箱** ⚠️⚠️⚠️
**位置**: `internal/api/handlers/auth.go:264`

**问题**: 注册时密码明文包含在欢迎邮件中
```go
welcomeSubject, welcomeBody := services.RenderEmail("welcome", map[string]string{
    "username": user.Username,
    "email":    user.Email,
    "password": req.Password,  // ⚠️ 明文密码
})
```

**风险**: 
- 密码可能被邮件服务器、中间人、邮件客户端记录
- 违反安全最佳实践

**修复建议**:
```go
// 移除密码字段，或使用临时密码机制
welcomeSubject, welcomeBody := services.RenderEmail("welcome", map[string]string{
    "username": user.Username,
    "email":    user.Email,
    // 不发送密码，提示用户使用注册时的密码
})
```

### 2. **CSV导入缺少安全验证** ⚠️
**位置**: `internal/api/handlers/admin.go:3121`

**问题**: 
- 未验证文件大小（可能导致DoS）
- 未验证文件类型（仅检查文件名）
- 未限制CSV行数（可能导致内存耗尽）

**修复建议**:
```go
// 1. 验证文件大小
const maxCSVSize = 10 * 1024 * 1024 // 10MB
fileInfo, _ := file.Stat()
if fileInfo.Size() > maxCSVSize {
    utils.BadRequest(c, "文件过大")
    return
}

// 2. 验证MIME类型
contentType := fileHeader.Header.Get("Content-Type")
if !strings.Contains(contentType, "text/csv") && !strings.Contains(contentType, "application/vnd.ms-excel") {
    utils.BadRequest(c, "文件类型不正确")
    return
}

// 3. 限制行数
const maxRows = 10000
if total > maxRows {
    utils.BadRequest(c, fmt.Sprintf("CSV行数超过限制（最多%d行）", maxRows))
    return
}
```

### 3. **Stripe Webhook签名验证可选** ⚠️
**位置**: `internal/api/handlers/payment.go:1005`

**问题**: 如果未配置 `webhook_secret`，则不验证签名，可能被伪造回调

**修复建议**:
```go
// 强制要求webhook secret
if stripeCfg.WebhookSecret == "" {
    c.String(400, "stripe webhook secret not configured")
    return
}
```

### 4. **IP白名单绕过漏洞** ⚠️
**位置**: `internal/api/middleware/security.go:43`

**问题**: IP白名单检查存在逻辑错误，可能导致绕过
```go
if err == nil && cidr.Contains(net.ParseIP(clientIP)) {
```
如果 `net.ParseIP(clientIP)` 返回 `nil`（无效IP），`cidr.Contains(nil)` 会 panic。

**修复建议**:
```go
if strings.Contains(line, "/") {
    _, cidr, err := net.ParseCIDR(line)
    if err == nil {
        ip := net.ParseIP(clientIP)
        if ip != nil && cidr.Contains(ip) {
            c.Next()
            return
        }
    }
}
```

### 5. **XSS 漏洞 - v-html 使用** ⚠️
**位置**: `frontend/src/views/admin/email-queue/Index.vue:160`

**问题**: 直接使用 `v-html` 渲染用户内容，存在 XSS 风险
```vue
<div v-html="detailItem.content" />
```

**修复建议**:
- 使用 HTML 转义库（如 `DOMPurify`）清理 HTML
- 或使用纯文本显示 + 代码高亮库

### 6. **innerHTML 使用风险** ⚠️
**位置**: 
- `frontend/src/views/auth/Login.vue:118`
- `frontend/src/views/settings/Index.vue:276`

**问题**: 直接操作 `innerHTML`，虽然这里是清空操作，但应使用更安全的方式

**修复建议**:
```typescript
// 当前: telegramWidgetRef.value.innerHTML = ''
// 建议: 
while (telegramWidgetRef.value.firstChild) {
  telegramWidgetRef.value.removeChild(telegramWidgetRef.value.firstChild)
}
```

### 7. **CSP 策略过于宽松** ⚠️
**位置**: `internal/api/middleware/security.go:18`

**问题**: Content-Security-Policy 包含 `'unsafe-inline'` 和 `'unsafe-eval'`
```go
c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://telegram.org; ...")
```

**修复建议**:
- 移除 `'unsafe-eval'`（除非绝对必要）
- 使用 nonce 或 hash 替代 `'unsafe-inline'`

---

## 🟡 中等问题 (Medium)

### 8. **缺少 CSRF 保护**
**位置**: 全局

**问题**: 未发现 CSRF Token 保护机制

**修复建议**:
- 添加 CSRF Token 中间件
- 或使用 SameSite Cookie（需要 HTTPS）

### 9. **密码强度验证不够严格**
**位置**: `internal/utils/password_validate.go`

**问题**: 仅要求 8 位 + 字母数字，未检查常见弱密码

**修复建议**:
- 添加常见密码字典检查
- 增加特殊字符要求（可选）
- 检查密码复杂度（如不能全是数字）

### 10. **错误信息可能泄露敏感信息**
**位置**: 多个 handler 文件

**问题**: 部分错误信息可能暴露系统内部信息

**示例**:
```go
utils.InternalError(c, "密码加密失败")  // 暴露了内部错误
```

**修复建议**:
- 生产环境统一错误信息
- 详细错误仅记录日志，不返回给客户端

### 11. **JWT Token 刷新缺少频率限制** ✅ 已修复
**位置**: `internal/api/router/router.go:50`

**问题**: `/auth/refresh` 路由未应用 RateLimit

**修复建议**:
```go
auth.POST("/refresh", middleware.RateLimit(10, time.Minute), handlers.RefreshToken)
```

### 12. **Telegram 登录时间窗口过大**
**位置**: `internal/api/handlers/auth.go:589`

**问题**: 5 分钟的时间窗口可能过长

**修复建议**:
```go
if time.Now().Unix()-req.AuthDate > 180 { // 3分钟
```

### 13. **数据库查询缺少索引检查**
**位置**: 多个查询位置

**问题**: 部分查询字段可能未建立索引，影响性能和安全

**建议**: 检查以下字段是否有索引：
- `users.email`
- `users.telegram_id`
- `login_attempts.username + created_at`
- `verification_codes.email + code + purpose`

---

## 🟢 轻微问题 (Low)

### 14. **console.log 残留**
**位置**: 
- `frontend/src/views/invite/Index.vue:567,580`
- `frontend/src/views/redeem/Index.vue:99`
- `frontend/src/views/mystery-box/Index.vue:208`

**问题**: 生产代码中包含 `console.error`，应使用日志系统

**修复建议**: 移除或替换为日志库

### 15. **日志中可能泄露敏感信息**
**位置**: 多个位置

**问题**: 
- `internal/services/subscription.go`: 密码可能出现在日志中
- `internal/services/alipay.go`: 订单信息包含敏感数据
- `internal/api/handlers/payment.go`: 支付回调数据记录

**修复建议**:
- 使用脱敏函数处理敏感字段
- 避免在日志中记录完整密码、token、私钥

### 16. **console.log 残留**
**位置**: `internal/api/middleware/auth.go:81-82`

**问题**: 函数定义缺少 `func` 关键字
```go
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {  // ✅ 正确
```

**注意**: 实际代码是正确的，但需要确认是否有其他类似问题

### 17. **CORS 默认配置过于宽松**
**位置**: `internal/api/router/router.go:39`

**问题**: 开发环境默认允许多个 localhost 端口

**建议**: 生产环境必须配置 `CORS_ORIGINS`

### 18. **文件上传安全**
**位置**: 需要检查文件上传相关代码

**建议**: 
- 验证文件类型（MIME + 扩展名）
- 限制文件大小
- 防止路径遍历攻击
- 扫描恶意文件内容

### 19. **订阅URL熵值检查**
**位置**: `internal/utils/crypto.go:22`

**问题**: `GenerateRandomString` 使用62字符集，32位长度，熵值约为 190 bits，足够安全，但需要确认所有使用场景

**建议**: 
- 订阅URL使用32字符 ✅ 已足够
- 验证码使用6位数字，熵值较低但可接受（用于验证码）

### 20. **支付回调重放攻击防护**
**位置**: `internal/api/handlers/payment.go`

**问题**: 需要检查是否有防重放机制

**建议**:
- 检查交易状态（已处理则忽略）
- 使用唯一交易ID
- 记录回调时间戳，拒绝过旧的回调
**位置**: 需要检查日志记录

**建议**: 
- 确保密码、token、私钥等不会记录到日志
- 使用脱敏处理

---

## ✅ 做得好的地方

1. **SQL 注入防护**: ✅ 使用 GORM，参数化查询
2. **密码存储**: ✅ 使用 bcrypt 加密
3. **JWT 安全**: ✅ Token 黑名单机制
4. **速率限制**: ✅ 关键接口有 RateLimit
5. **输入验证**: ✅ 使用 Gin binding 验证
6. **排序字段验证**: ✅ 使用正则验证排序字段，防止 SQL 注入
7. **安全响应头**: ✅ 设置了多个安全响应头
8. **登录锁定**: ✅ 实现了登录失败锁定机制
9. **密码强度验证**: ✅ 有基本的密码强度检查
10. **敏感字段保护**: ✅ 最近修复了敏感字段脱敏问题

---

## 📋 建议的改进优先级

### 立即修复 (P0)
1. **密码明文发送到邮箱** (#1) ⚠️⚠️⚠️ 最严重
2. CSV导入安全验证 (#2)
3. Stripe Webhook签名验证 (#3)
4. IP白名单绕过漏洞 (#4) ✅ 已修复
5. XSS 漏洞 - v-html (#5) ✅ 已修复
6. CSP 策略优化 (#7)

### 近期修复 (P1)
7. 添加 CSRF 保护 (#8)
8. JWT 刷新频率限制 (#11) ✅ 已修复
9. 错误信息泄露 (#10)
10. 日志敏感信息泄露 (#15)
11. 支付回调重放攻击防护 (#20)

### 计划改进 (P2)
12. 密码强度增强 (#9)
13. Telegram 登录时间窗口 (#12)
14. 移除 console.log (#14)
15. 文件上传安全检查 (#18)
16. 订阅URL熵值检查 (#19)

---

## 🔧 代码质量建议

### 1. **错误处理**
- 统一错误处理机制
- 区分用户错误和系统错误
- 生产环境隐藏详细错误

### 2. **日志记录**
- 使用结构化日志
- 记录关键操作（登录、支付、敏感操作）
- 敏感信息脱敏

### 3. **测试覆盖**
- 添加单元测试
- 添加集成测试
- 安全测试（OWASP Top 10）

### 4. **代码审查**
- 建立代码审查流程
- 使用静态代码分析工具（gosec, eslint）
- 定期安全审计

### 5. **依赖管理**
- 定期更新依赖
- 检查已知漏洞（npm audit, go list -m -u）
- 使用依赖锁定文件

---

## 📊 安全评分

| 类别 | 评分 | 说明 |
|------|------|------|
| 认证与授权 | 8/10 | JWT 实现良好，但缺少 CSRF |
| 输入验证 | 9/10 | 使用 Gin binding，验证充分 |
| 输出编码 | 7/10 | v-html 存在风险 |
| 错误处理 | 7/10 | 需要统一和改进 |
| 加密与存储 | 9/10 | bcrypt 正确使用 |
| 会话管理 | 8/10 | JWT + 黑名单机制良好 |
| 安全配置 | 7/10 | CSP 需要优化 |
| 日志监控 | 6/10 | 需要结构化日志，敏感信息脱敏 |
| 数据保护 | 6/10 | 密码明文发送，需要改进 |

**总体评分**: 7.0/10 ⬇️ (发现密码明文发送问题后降低)

---

## 🆕 新增发现的漏洞总结

### 严重问题
1. **密码明文发送** - 注册时密码通过邮件明文发送
2. **CSV导入漏洞** - 缺少文件大小、类型、行数限制
3. **Stripe Webhook可选验证** - 未配置secret时不验证签名

### 中等问题
4. **日志敏感信息** - 密码、token可能出现在日志中
5. **支付回调重放** - 需要检查防重放机制

### 已修复问题 ✅
- IP白名单绕过漏洞
- XSS漏洞 (v-html)
- JWT刷新频率限制

---

## 📝 总结

代码整体质量良好，使用了现代的安全实践（JWT、bcrypt、参数化查询等）。主要需要关注：

1. **XSS 防护**: 修复 v-html 使用
2. **CSRF 保护**: 添加 CSRF Token
3. **错误处理**: 统一错误信息，避免泄露
4. **安全配置**: 优化 CSP 策略

建议优先修复 P0 级别的问题，然后逐步改进其他方面。
