# 安全漏洞修复报告

## 修复日期
2026-03-03

## 修复的高优先级漏洞

### 1. ✅ CSRF 时序攻击漏洞（中风险）
**文件**: `internal/api/middleware/csrf.go`

**问题**: 使用简单的字符串比较而非常量时间比较，可能导致时序攻击

**修复**:
```go
// 之前
return token == stored.token

// 修复后
return subtle.ConstantTimeCompare([]byte(token), []byte(stored.token)) == 1
```

**影响**: 防止攻击者通过时序差异推断 CSRF token 内容

---

### 2. ✅ .env 文件备份泄露（高风险）
**文件**: `internal/api/handlers/admin.go`

**问题**: 数据库备份包含 `.env` 文件，可能泄露数据库密码、API 密钥、支付密钥等敏感信息

**修复**: 移除了备份 `.env` 文件的代码，添加了安全注释

**影响**: 防止敏感凭证通过备份文件泄露

---

### 3. ✅ CSP 配置过于宽松（高风险）
**文件**: `internal/api/middleware/security.go`

**问题**:
- `'unsafe-inline'` 在 `script-src` 和 `style-src` 中削弱了 XSS 防护
- `data:` 在 `img-src` 中可能被滥用

**修复**:
```go
// 之前
"Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://telegram.org; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; ..."

// 修复后
"Content-Security-Policy", "default-src 'self'; script-src 'self' https://telegram.org; style-src 'self'; img-src 'self' https:; connect-src 'self' https:; frame-src https://telegram.org; object-src 'none'; base-uri 'self'; form-action 'self'"
```

**影响**: 显著提高了 XSS 攻击防护能力

**注意**: 如果前端使用内联脚本/样式，需要使用 nonce 或 hash 方案

---

### 4. ✅ 请求体大小无限制（中风险）
**文件**: `internal/api/handlers/payment.go`

**问题**: 无限制读取请求体可导致内存耗尽（DoS 攻击）

**修复**:
```go
// 添加 10MB 限制
c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10*1024*1024)
```

**影响**: 防止大型恶意请求导致服务器崩溃

---

### 5. ✅ 随机数生成错误未检查（中风险）
**文件**: `internal/utils/crypto.go`, `internal/api/middleware/csrf.go`

**问题**: `rand.Read()` 和 `rand.Int()` 的错误被忽略，可能导致不安全的随机数

**修复**:
```go
// 之前
n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))

// 修复后
n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
if err != nil {
    panic(fmt.Sprintf("随机数生成失败: %v", err))
}
```

**影响**: 确保随机数生成失败时能够及时发现和处理

---

### 6. ✅ 订阅 URL 枚举攻击（中风险）
**文件**: `internal/api/router/router.go`

**问题**: 频率限制为 120 请求/分钟，对于枚举攻击来说过高

**修复**:
```go
// 之前
subRL := middleware.RateLimit(120, time.Minute)

// 修复后
subRL := middleware.RateLimit(20, time.Minute)
```

**影响**: 显著降低了订阅 URL 被枚举的风险

---

## 仍需关注的问题

### ⚠️ 支付回调验证不足（高风险）
**文件**: `internal/api/handlers/payment.go`

**问题**: 通用支付回调处理器缺少签名验证，可能被伪造

**建议**:
- 对所有支付回调实施强制签名验证
- 验证回调来源 IP
- 实施幂等性检查防止重复处理

---

### ⚠️ 管理员操作缺少 CSRF 保护（中风险）
**文件**: `internal/api/router/router.go`

**问题**: 许多管理员操作未应用 CSRF 保护中间件

**建议**: 对所有状态改变操作应用 CSRF 保护

---

### ⚠️ IP 白名单绕过风险（中风险）
**文件**: `internal/api/middleware/security.go`

**问题**: 依赖可被伪造的 `X-Forwarded-For` 头部

**建议**:
- 仅信任受信任的代理
- 配置反向代理正确设置 `X-Forwarded-For`
- 使用 `X-Real-IP` 而非 `X-Forwarded-For`

---

### ⚠️ 登录尝试锁定可被绕过（中风险）
**文件**: `internal/api/handlers/auth.go`

**问题**: 锁定仅基于用户名，攻击者可通过改变 IP 地址绕过

**建议**:
- 同时基于 IP 和用户名进行锁定
- 添加全局登录速率限制
- 在多次失败后要求验证码

---

### ⚠️ Token 黑名单清理不足（低风险）
**文件**: `internal/models/security.go`

**问题**: 清理函数存在但未被调用，黑名单表会无限增长

**建议**: 在应用启动时启动后台清理任务

---

## 安全最佳实践建议

### 1. 前端安全
- ✅ 已实施严格的 CSP 策略
- ⚠️ 需要确保前端代码不使用内联脚本/样式
- ⚠️ 建议实施 Subresource Integrity (SRI)

### 2. 后端安全
- ✅ 使用 bcrypt 进行密码哈希
- ✅ 使用 `crypto/rand` 生成随机数
- ✅ 实施了 CSRF 保护
- ⚠️ 需要加强支付回调验证
- ⚠️ 需要实施更严格的速率限制

### 3. API 安全
- ✅ 实施了请求体大小限制
- ✅ 实施了频率限制
- ⚠️ 需要对所有管理员操作应用 CSRF 保护
- ⚠️ 需要实施更严格的输入验证

### 4. 数据安全
- ✅ 不再备份 .env 文件
- ⚠️ 建议对备份文件进行加密
- ⚠️ 建议实施更严格的文件权限控制

---

## 测试建议

### 1. 安全测试
- [ ] 测试 CSRF 保护是否有效
- [ ] 测试 CSP 策略是否正常工作
- [ ] 测试频率限制是否生效
- [ ] 测试请求体大小限制

### 2. 功能测试
- [ ] 测试前端是否因 CSP 策略而出现问题
- [ ] 测试支付流程是否正常
- [ ] 测试订阅链接访问是否正常
- [ ] 测试备份功能是否正常

### 3. 性能测试
- [ ] 测试频率限制对正常用户的影响
- [ ] 测试请求体大小限制对正常请求的影响

---

## 部署注意事项

1. **CSP 策略变更**: 如果前端使用内联脚本/样式，需要：
   - 将内联代码移到外部文件
   - 或使用 nonce/hash 方案

2. **频率限制变更**: 订阅链接频率限制从 120/分钟降到 20/分钟
   - 正常用户不应受影响
   - 如有问题可适当调整

3. **备份功能变更**: 不再备份 .env 文件
   - 需要单独备份配置信息
   - 建议使用加密存储

---

## 总结

本次修复解决了 **6 个高/中风险安全漏洞**，显著提升了系统的安全性：

✅ **已修复**:
1. CSRF 时序攻击
2. .env 文件泄露
3. CSP 配置过于宽松
4. 请求体大小无限制
5. 随机数生成错误未检查
6. 订阅 URL 枚举攻击

⚠️ **仍需关注**:
1. 支付回调验证不足
2. 管理员操作 CSRF 保护
3. IP 白名单绕过
4. 登录锁定绕过
5. Token 黑名单清理

建议在下一个迭代中继续完善剩余的安全问题。
