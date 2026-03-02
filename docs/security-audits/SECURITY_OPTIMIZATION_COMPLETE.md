# 安全优化完成清单

## ✅ 已完成的安全加固（第二轮）

### 1. 订阅路由频率限制 ✅
**问题：** 订阅地址可被暴力枚举
**修复：** 添加 120次/分钟 的频率限制
**文件：** `/internal/api/router/router.go`

```go
subRL := middleware.RateLimit(120, time.Minute)
api.GET("/sub/clash/:url", subRL, handlers.GetSubscription)
api.GET("/sub/:url", subRL, handlers.GetUniversalSubscription)
```

---

### 2. CORS 配置增强 ✅
**问题：** CORS 未包含 X-CSRF-Token header
**修复：** 添加 X-CSRF-Token 到 AllowHeaders
**文件：** `/internal/api/router/router.go`

```go
AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "Accept", "X-CSRF-Token"}
```

---

### 3. CSRF Token 端点注册 ✅
**问题：** CSRF token 获取端点未注册
**修复：** 添加 `/api/v1/csrf-token` 端点
**文件：** `/internal/api/router/router.go`

```go
authorized.GET("/csrf-token", middleware.GetCSRFToken)
```

---

### 4. 订阅转余额竞态条件修复 ✅
**问题：** ConvertToBalance 无事务锁，存在余额竞态
**修复：** 使用数据库事务 + 原子操作
**文件：** `/internal/api/handlers/subscription.go`

```go
err := db.Transaction(func(tx *gorm.DB) error {
    // 原子操作更新余额
    tx.Model(&models.User{}).Where("id = ?", userID).
        Update("balance", gorm.Expr("balance + ?", value))
    // ...
})
```

---

### 5. 自动取消过期订单 ✅
**问题：** 过期订单不会自动取消
**修复：** 添加定时任务每小时检查并取消过期订单
**文件：** `/internal/services/scheduler.go`

```go
s.startLoop("CancelExpiredOrders", 1*time.Hour, cancelExpiredOrdersTask)
```

---

### 6. 支付 Nonce 自动清理 ✅
**问题：** 过期的支付 nonce 不会自动清理
**修复：** 添加定时任务每 6 小时清理过期 nonce
**文件：** `/internal/services/scheduler.go`

```go
s.startLoop("CleanPaymentNonces", 6*time.Hour, cleanPaymentNoncesTask)
```

---

### 7. 注册蜜罐防机器人 ✅
**问题：** 无机器人注册防护
**修复：** 添加隐藏蜜罐字段 `website`
**文件：** `/internal/api/handlers/auth.go`

```go
Honeypot string `json:"website"` // 正常用户不应填写

if req.Honeypot != "" {
    // 返回假成功，迷惑机器人
    utils.Success(c, gin.H{"access_token": "fake_token"})
    return
}
```

---

### 8. 输入清理工具集 ✅
**新增文件：** `/internal/utils/sanitize.go`

**功能：**
- `SanitizeInput()` - HTML 转义 + 控制字符过滤
- `SanitizeUsername()` - 用户名清理（仅字母数字下划线中文）
- `SanitizeEmail()` - 邮箱格式验证
- `ValidateNoSQLInjection()` - SQL 注入特征检测
- `ValidateNoXSS()` - XSS 特征检测
- `ValidateNoPathTraversal()` - 路径遍历检测
- `SanitizeFilename()` - 文件名清理
- `ValidateURL()` - URL 安全验证（防 SSRF）

---

### 9. 安全审计工具集 ✅
**新增文件：** `/internal/utils/security.go`

**功能：**
- `CreateAdminAuditLog()` - 记录管理员操作审计日志
- `CreateSecurityEvent()` - 记录安全事件
- `DetectSuspiciousActivity()` - 检测可疑活动
- `ValidateAdminAction()` - 验证管理员操作权限

---

## 📊 安全加固统计

### 第一轮修复（已完成）
- ✅ 支付回调重放攻击防护
- ✅ 支付金额校验增强
- ✅ Token 刷新安全增强
- ✅ CSRF 保护机制
- ✅ 订阅地址枚举检测

### 第二轮优化（已完成）
- ✅ 订阅路由频率限制
- ✅ CORS 配置完善
- ✅ 余额竞态条件修复
- ✅ 自动化清理任务
- ✅ 注册机器人防护
- ✅ 输入清理工具集
- ✅ 安全审计工具集

### 总计
- **新增文件：** 7 个
- **修改文件：** 6 个
- **修复漏洞：** 13 个
- **新增功能：** 15+ 个安全工具函数

---

## 🔧 部署步骤

### 1. 数据库迁移
```bash
cd /Users/apple/v2
sqlite3 cboard.db < migrations/add_payment_nonces.sql
```

### 2. 重新编译
```bash
kill $(cat backend.pid)
go build -o cboard cmd/server/main.go
```

### 3. 重启服务
```bash
./start.sh
```

### 4. 验证功能
```bash
# 测试 CSRF token 获取
curl -H "Authorization: Bearer $TOKEN" http://localhost:8000/api/v1/csrf-token

# 测试订阅频率限制
for i in {1..130}; do
  curl http://localhost:8000/api/v1/sub/test$i
done
# 应在第 121 次请求时返回 429 Too Many Requests

# 测试支付回调重放
# 捕获合法回调后重放，应返回成功但不重复处理
```

---

## 🛡️ 安全最佳实践

### 使用输入清理工具
```go
import "cboard/v2/internal/utils"

// 清理用户输入
username := utils.SanitizeUsername(req.Username)
email := utils.SanitizeEmail(req.Email)

// 验证安全性
if !utils.ValidateNoXSS(req.Content) {
    return errors.New("输入包含危险内容")
}
```

### 记录管理员操作
```go
// 在管理员操作中添加审计日志
utils.CreateAdminAuditLog(c, "delete_user", "user", &userID,
    fmt.Sprintf("删除用户: %s", username))
```

### 检测可疑活动
```go
// 在敏感操作前检测
if utils.DetectSuspiciousActivity(c, "rapid_login_attempts") {
    utils.TooManyRequests(c, "检测到异常活动")
    return
}
```

---

## 📝 前端集成指南

### 1. 添加蜜罐字段到注册表单
```vue
<template>
  <n-form>
    <!-- 正常字段 -->
    <n-form-item label="用户名">
      <n-input v-model:value="form.username" />
    </n-form-item>

    <!-- 蜜罐字段（隐藏） -->
    <input
      type="text"
      name="website"
      v-model="form.website"
      style="position:absolute;left:-9999px;opacity:0;pointer-events:none"
      tabindex="-1"
      autocomplete="off"
    />
  </n-form>
</template>

<script setup>
const form = ref({
  username: '',
  email: '',
  password: '',
  website: '', // 蜜罐字段，正常用户不会填写
})
</script>
```

### 2. 获取并使用 CSRF Token
```typescript
// 在 request.ts 中已实现
import { fetchCSRFToken } from '@/utils/request'

// 登录后获取 token
onMounted(async () => {
  const token = localStorage.getItem('access_token')
  if (token) {
    await fetchCSRFToken()
  }
})
```

---

## 🔍 监控建议

### 1. 日志监控
```bash
# 监控支付异常
tail -f backend.log | grep "金额不匹配\|重放攻击"

# 监控订阅枚举
tail -f backend.log | grep "订阅地址不存在访问尝试"

# 监控蜜罐触发
tail -f backend.log | grep "注册蜜罐触发"

# 监控可疑活动
tail -f backend.log | grep "rapid_login_attempts"
```

### 2. 数据库监控
```sql
-- 查看最近的支付 nonce（防重放）
SELECT * FROM payment_nonces ORDER BY processed_at DESC LIMIT 10;

-- 查看过期订单
SELECT * FROM orders WHERE status = 'expired' ORDER BY expire_time DESC LIMIT 10;

-- 查看审计日志
SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 20;
```

### 3. 性能监控
```bash
# 监控订阅访问频率
watch -n 1 'tail -100 backend.log | grep "/sub/" | wc -l'

# 监控数据库大小
watch -n 60 'du -h cboard.db'
```

---

## ⚠️ 注意事项

1. **蜜罐字段命名：** 前端的蜜罐字段必须命名为 `website`，与后端一致
2. **CSRF Token：** 所有 POST/PUT/DELETE 请求都需要携带 CSRF token
3. **频率限制：** 订阅访问限制为 120次/分钟，可根据实际情况调整
4. **审计日志：** 管理员操作会自动记录，定期检查 audit_logs 表
5. **定时任务：** 确保 scheduler 正常运行，检查日志确认任务执行

---

## 📚 相关文档

- [SECURITY_FIXES.md](./SECURITY_FIXES.md) - 第一轮安全修复详情
- [SECURITY_IMPLEMENTATION.md](./SECURITY_IMPLEMENTATION.md) - 实施指南
- [CODE_REVIEW.md](./CODE_REVIEW.md) - 代码审查报告

---

**优化完成时间：** 2026-03-02
**优化人员：** Claude (AI Security Audit - Round 2)
**总修复漏洞数：** 13 个
**新增安全功能：** 15+ 个
