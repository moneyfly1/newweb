# 🚨 严重安全漏洞发现报告

## 发现时间
2026-03-02 (第三轮深度审计)

---

## 🔴 严重漏洞清单

### 1. 签到重放攻击 - CRITICAL ⚠️⚠️⚠️

**漏洞位置：** `/internal/api/handlers/checkin.go:27-34`

**问题描述：**
```go
// 仅检查日期，没有防重放机制
today := time.Now().Format("2006-01-02")
var count int64
db.Model(&models.CheckIn{}).Where("user_id = ? AND DATE(created_at) = ?", userID, today).Count(&count)
if count > 0 {
    utils.BadRequest(c, "今天已经签到过了")
    return
}
```

**攻击场景：**
1. 攻击者在 23:59:59 签到成功
2. 立即修改系统时间到 00:00:01（或等待 1 秒）
3. 再次签到，绕过日期检查
4. **更严重：** 攻击者可以抓包重放请求，在事务提交前的短暂时间窗口内并发发送多个请求

**影响：**
- 无限刷余额
- 数据库余额被恶意增加
- 经济损失

**风险等级：** 🔴 严重（CRITICAL）

---

### 2. 管理员延期订阅无审计日志 - HIGH ⚠️⚠️

**漏洞位置：** `/internal/api/handlers/admin.go:1450-1480`

**问题描述：**
```go
func AdminExtendSubscription(c *gin.Context) {
    // ... 直接修改到期时间，无审计日志
    db.Model(&sub).Updates(map[string]interface{}{
        "expire_time": newExpire,
        "is_active":   true,
        "status":      "active",
    })
    utils.SuccessMessage(c, "延期成功")
}
```

**攻击场景：**
1. 恶意管理员或被入侵的管理员账户
2. 无限延长自己或他人的订阅
3. 无法追踪谁做了什么操作

**影响：**
- 无法审计管理员操作
- 内部人员滥用权限
- 合规风险

**风险等级：** 🟠 高危（HIGH）

---

### 3. 管理员设置到期时间无审计 - HIGH ⚠️⚠️

**漏洞位置：** `/internal/api/handlers/admin.go:2618-2648`

**问题描述：**
```go
func AdminSetSubscriptionExpireTime(c *gin.Context) {
    // 可以设置任意到期时间（最多 10 年）
    // 无审计日志
    // 无二次确认
}
```

**攻击场景：**
同上，更严重的是可以直接设置到期时间，而不是增加天数

**风险等级：** 🟠 高危（HIGH）

---

### 4. 管理员退款无事务保护 - HIGH ⚠️⚠️

**漏洞位置：** `/internal/api/handlers/admin.go:644-720`

**问题描述：**
```go
func AdminRefundOrder(c *gin.Context) {
    // 退款操作分多步：
    // 1. 更新订单状态
    // 2. 增加用户余额
    // 3. 记录余额日志
    // 但没有使用数据库事务！

    db.Model(&order).Updates(map[string]interface{}{
        "status": "refunded",
        "refund_time": &now,
    })

    // 如果这里失败，订单已标记为退款但余额未增加
    db.Model(&models.User{}).Where("id = ?", order.UserID).
        Update("balance", gorm.Expr("balance + ?", refundAmount))
}
```

**攻击场景：**
1. 网络中断或服务器崩溃
2. 订单标记为已退款，但余额未增加
3. 或反之：余额增加了，但订单未标记

**影响：**
- 数据不一致
- 用户损失或平台损失
- 财务对账困难

**风险等级：** 🟠 高危（HIGH）

---

### 5. 卡密兑换无频率限制 - MEDIUM ⚠️

**漏洞位置：** `/internal/api/handlers/redeem.go:15-122`

**问题描述：**
- 卡密兑换接口无频率限制
- 攻击者可以暴力枚举卡密

**攻击场景：**
1. 攻击者编写脚本暴力尝试卡密
2. 卡密格式可预测（如 6 位数字）
3. 成功率 = 1/1000000

**风险等级：** 🟡 中危（MEDIUM）

---

### 6. 签到余额更新竞态条件 - MEDIUM ⚠️

**漏洞位置：** `/internal/api/handlers/checkin.go:56-90`

**问题描述：**
```go
// 虽然使用了事务，但在事务外读取了余额
var user models.User
if err := db.First(&user, userID).Error; err != nil {
    // ...
}
balanceBefore := user.Balance  // 事务外读取

tx := db.Begin()
// ... 事务内更新
```

**问题：**
- `balanceBefore` 可能不准确
- 如果有并发操作，余额日志记录的 before/after 可能错误

**风险等级：** 🟡 中危（MEDIUM）

---

### 7. 管理员操作无二次确认 - MEDIUM ⚠️

**问题描述：**
- 删除用户、退款、延期等敏感操作无二次确认
- 前端可能有确认，但后端无验证

**风险等级：** 🟡 中危（MEDIUM）

---

## 📊 漏洞统计

| 等级 | 数量 | 漏洞 |
|------|------|------|
| 🔴 严重 | 1 | 签到重放攻击 |
| 🟠 高危 | 3 | 管理员操作无审计、退款无事务 |
| 🟡 中危 | 3 | 卡密枚举、签到竞态、无二次确认 |
| **总计** | **7** | |

---

## 🛠️ 修复优先级

### P0 - 立即修复（24小时内）
1. ✅ 签到重放攻击
2. ✅ 管理员退款事务保护

### P1 - 高优先级（3天内）
3. ✅ 管理员操作审计日志
4. ✅ 卡密兑换频率限制

### P2 - 中优先级（1周内）
5. ✅ 签到余额竞态修复
6. ⚠️ 管理员操作二次确认（需前端配合）

---

## 🔧 修复方案

### 修复 1: 签到重放攻击

**方案 A：** 使用唯一索引（推荐）
```sql
CREATE UNIQUE INDEX idx_checkin_user_date ON check_ins(user_id, DATE(created_at));
```

**方案 B：** 使用分布式锁
```go
// 使用 Redis 或数据库行锁
lockKey := fmt.Sprintf("checkin:%d:%s", userID, today)
```

**方案 C：** 在事务内重新检查（最简单）
```go
tx := db.Begin()
// 在事务内再次检查
var count int64
tx.Model(&models.CheckIn{}).Where("user_id = ? AND DATE(created_at) = ?", userID, today).Count(&count)
if count > 0 {
    tx.Rollback()
    return
}
// 继续创建签到记录
```

---

### 修复 2: 管理员操作审计

```go
func AdminExtendSubscription(c *gin.Context) {
    // ... 原有逻辑

    // 添加审计日志
    adminID := c.GetUint("user_id")
    utils.CreateAdminAuditLog(c, "extend_subscription", "subscription", &sub.ID,
        fmt.Sprintf("延长订阅 %d 天，新到期时间: %s", req.Days, newExpire.Format("2006-01-02")))

    // 记录订阅日志
    utils.CreateSubscriptionLog(sub.ID, sub.UserID, "extend", "admin", &adminID,
        fmt.Sprintf("管理员延长订阅 %d 天", req.Days), nil, nil)
}
```

---

### 修复 3: 退款事务保护

```go
func AdminRefundOrder(c *gin.Context) {
    // 使用事务包裹所有操作
    err := db.Transaction(func(tx *gorm.DB) error {
        // 1. 更新订单状态
        if err := tx.Model(&order).Updates(...).Error; err != nil {
            return err
        }

        // 2. 增加余额
        if err := tx.Model(&models.User{}).Where("id = ?", order.UserID).
            Update("balance", gorm.Expr("balance + ?", refundAmount)).Error; err != nil {
            return err
        }

        // 3. 记录日志
        if err := tx.Create(&balanceLog).Error; err != nil {
            return err
        }

        return nil
    })
}
```

---

### 修复 4: 卡密兑换频率限制

```go
// 在路由中添加
authorized.POST("/redeem", middleware.RateLimit(5, time.Minute), handlers.RedeemCode)
```

---

## 🎯 下一步行动

1. **立即修复签到重放** - 最严重
2. **添加管理员审计日志** - 合规要求
3. **修复退款事务** - 财务安全
4. **添加频率限制** - 防暴力破解
5. **全面测试** - 确保修复有效

---

**报告生成时间：** 2026-03-02
**审计人员：** Claude (AI Security Audit - Round 3)
**严重程度：** 🔴 CRITICAL
**建议：** 立即修复 P0 级别漏洞
