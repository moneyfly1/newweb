# 🚨 第四轮安全审计 - 优惠券、性能与系统漏洞

## 发现时间
2026-03-02 (第四轮全面审计)

---

## 🔴 严重漏洞清单

### 1. 优惠券竞态条件 - CRITICAL ⚠️⚠️⚠️

**漏洞位置：** `/internal/api/handlers/order.go:94-151`

**问题描述：**
```go
// 在订单创建时检查优惠券
var coupon models.Coupon
if err := db.Where("code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err == nil {
    // 检查数量
    if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
        utils.BadRequest(c, "优惠券已被领完")
        return
    }
    // 检查用户使用次数
    var usageCount int64
    db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).Count(&usageCount)
    // ...
}

// 创建订单后才记录使用
db.Create(&models.CouponUsage{...})
db.Model(&models.Coupon{}).Where("id = ?", *couponID).UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1"))
```

**攻击场景：**
1. 优惠券总数量 = 100
2. 当前已使用 = 99
3. 攻击者并发发送 10 个请求
4. 所有请求都通过检查（因为都看到 99 < 100）
5. 结果：10 个订单都使用了优惠券，实际使用 109 次

**影响：**
- 优惠券超发
- 经济损失
- 数据不一致

**风险等级：** 🔴 严重（CRITICAL）

---

### 2. 优惠券过期时间检查有误 - HIGH ⚠️⚠️

**漏洞位置：** `/internal/api/handlers/coupon.go:38`

**问题描述：**
```go
// 检查日期范围
now := time.Now()
if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil.AddDate(0, 0, 1)) {
    utils.BadRequest(c, "优惠券不在有效期内")
    return
}
```

**问题：**
- `AddDate(0, 0, 1)` 会将到期时间延长 1 天
- 如果优惠券到期时间是 2026-03-01 23:59:59
- 实际可用到 2026-03-02 23:59:59
- **过期的优惠券还能用 1 天！**

**风险等级：** 🟠 高危（HIGH）

---

### 3. 优惠券使用记录在订单外创建 - HIGH ⚠️⚠️

**漏洞位置：** `/internal/api/handlers/order.go:148-150`

**问题描述：**
```go
// 创建订单
if err := db.Create(&order).Error; err != nil {
    utils.InternalError(c, "创建订单失败")
    return
}

// 记录优惠券使用（在订单创建事务外！）
if couponID != nil {
    db.Create(&models.CouponUsage{...})
    db.Model(&models.Coupon{}).Where("id = ?", *couponID).UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1"))
}
```

**问题：**
- 优惠券使用记录不在事务内
- 如果记录失败，订单已创建但优惠券未标记使用
- 用户可以重复使用同一优惠券

**风险等级：** 🟠 高危（HIGH）

---

### 4. 页码无上限 - MEDIUM ⚠️

**漏洞位置：** `/internal/utils/pagination.go:20-41`

**问题描述：**
```go
func GetPagination(c *gin.Context) Pagination {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    if page < 1 {
        page = 1
    }
    // 没有检查 page 的最大值！
    if pageSize > 100 {
        pageSize = 100
    }
    // ...
}
```

**攻击场景：**
```bash
# 攻击者请求超大页码
curl "http://api.com/users?page=999999999&page_size=100"

# 计算 offset = (999999999 - 1) * 100 = 99999999900
# 数据库执行: SELECT * FROM users LIMIT 100 OFFSET 99999999900
# 导致数据库扫描海量数据，CPU 100%，内存耗尽
```

**影响：**
- 数据库性能下降
- 内存耗尽
- DoS 攻击

**风险等级：** 🟡 中危（MEDIUM）

---

### 5. N+1 查询问题 - MEDIUM ⚠️

**漏洞位置：** `/internal/api/handlers/order.go:31-67`

**问题描述：**
```go
// 查询订单列表
db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&orders)

// 循环查询套餐名称（N+1 问题）
for _, o := range orders {
    var pkg models.Package
    if err := dbConn.Select("name").First(&pkg, o.PackageID).Error; err == nil {
        item.PackageName = pkg.Name
        pkgNameCache[o.PackageID] = pkg.Name
    }
}
```

**问题：**
- 查询 20 个订单 = 1 次查询
- 查询 20 个套餐名称 = 20 次查询
- 总计 21 次数据库查询
- 应该使用 JOIN 或预加载

**影响：**
- 数据库连接数增加
- 响应时间变慢
- 高并发时性能急剧下降

**风险等级：** 🟡 中危（MEDIUM）

---

### 6. 硬编码 Limit 无法配置 - LOW ⚠️

**漏洞位置：** 多处

**问题描述：**
```go
// 硬编码 200
db.Where("user_id = ?", userID).Order("used_at DESC").Limit(200).Find(&usages)

// 硬编码 20
db.Where("inviter_id = ?", userID).Order("created_at DESC").Limit(20).Find(&relations)

// 硬编码 365
db.Model(&models.CheckIn{}).Limit(365).Find(&dates)
```

**问题：**
- 无法动态调整
- 可能返回过多数据
- 内存占用高

**风险等级：** 🟢 低危（LOW）

---

### 7. 优惠券验证接口无频率限制 - MEDIUM ⚠️

**漏洞位置：** `/internal/api/router/router.go`

**问题描述：**
- `/api/v1/coupons/verify` 无频率限制
- 攻击者可以暴力枚举优惠券代码

**攻击场景：**
```bash
# 暴力枚举优惠券
for code in {AAAA..ZZZZ}; do
  curl -X POST /api/v1/coupons/verify -d "{\"code\":\"$code\"}"
done
```

**风险等级：** 🟡 中危（MEDIUM）

---

### 8. 订单取消无优惠券回滚 - HIGH ⚠️⚠️

**漏洞位置：** `/internal/api/handlers/order.go:366-377`

**问题描述：**
```go
func CancelOrder(c *gin.Context) {
    // 取消订单
    db.Model(&order).Update("status", "cancelled")
    utils.SuccessMessage(c, "订单已取消")
}
```

**问题：**
- 订单取消后，优惠券使用记录未删除
- 优惠券 used_quantity 未减少
- 用户无法再次使用该优惠券

**风险等级：** 🟠 高危（HIGH）

---

## 📊 漏洞统计

| 等级 | 数量 | 漏洞 |
|------|------|------|
| 🔴 严重 | 1 | 优惠券竞态条件 |
| 🟠 高危 | 3 | 过期检查错误、使用记录事务、取消无回滚 |
| 🟡 中危 | 3 | 页码无上限、N+1 查询、验证无限制 |
| 🟢 低危 | 1 | 硬编码 Limit |
| **总计** | **8** | |

---

## 🔧 修复方案

### 修复 1: 优惠券竞态条件

**方案：** 使用数据库行锁 + 事务

```go
func CreateOrder(c *gin.Context) {
    // ... 前置检查

    err := db.Transaction(func(tx *gorm.DB) error {
        // 在事务内加锁查询优惠券
        var coupon models.Coupon
        if req.CouponCode != "" {
            if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
                Where("code = ? AND status = ?", req.CouponCode, "active").
                First(&coupon).Error; err != nil {
                return fmt.Errorf("coupon_not_found")
            }

            // 检查数量（在锁内）
            if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
                return fmt.Errorf("coupon_exhausted")
            }

            // 检查用户使用次数（在锁内）
            var usageCount int64
            tx.Model(&models.CouponUsage{}).
                Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).
                Count(&usageCount)
            if int(usageCount) >= coupon.MaxUsesPerUser {
                return fmt.Errorf("coupon_limit_exceeded")
            }

            // 计算折扣
            // ...
        }

        // 创建订单
        if err := tx.Create(&order).Error; err != nil {
            return err
        }

        // 记录优惠券使用（在同一事务内）
        if couponID != nil {
            if err := tx.Create(&models.CouponUsage{...}).Error; err != nil {
                return err
            }
            if err := tx.Model(&models.Coupon{}).Where("id = ?", *couponID).
                UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1")).Error; err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        // 处理错误
        return
    }
}
```

---

### 修复 2: 优惠券过期时间检查

```go
// 修复前
if now.After(coupon.ValidUntil.AddDate(0, 0, 1)) {

// 修复后
if now.After(coupon.ValidUntil) {
```

---

### 修复 3: 页码上限

```go
func GetPagination(c *gin.Context) Pagination {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

    if page < 1 {
        page = 1
    }
    // 添加页码上限（防止超大 offset）
    if page > 10000 {
        page = 10000
    }

    if pageSize < 1 {
        pageSize = 20
    }
    if pageSize > 100 {
        pageSize = 100
    }

    return Pagination{Page: page, PageSize: pageSize, Sort: sort, Order: order}
}
```

---

### 修复 4: N+1 查询优化

```go
// 方案 A: 使用 JOIN
db.Table("orders").
    Select("orders.*, packages.name as package_name").
    Joins("LEFT JOIN packages ON packages.id = orders.package_id").
    Where("orders.user_id = ?", userID).
    Order(p.OrderClause()).
    Offset(p.Offset()).
    Limit(p.PageSize).
    Find(&items)

// 方案 B: 预加载所有套餐
var packageIDs []uint
for _, o := range orders {
    if o.PackageID > 0 {
        packageIDs = append(packageIDs, o.PackageID)
    }
}
var packages []models.Package
db.Where("id IN ?", packageIDs).Find(&packages)
packageMap := make(map[uint]string)
for _, pkg := range packages {
    packageMap[pkg.ID] = pkg.Name
}
```

---

### 修复 5: 优惠券验证频率限制

```go
// 在路由中添加
authorized.POST("/coupons/verify", middleware.RateLimit(10, time.Minute), handlers.VerifyCoupon)
```

---

### 修复 6: 订单取消回滚优惠券

```go
func CancelOrder(c *gin.Context) {
    // ... 查询订单

    err := db.Transaction(func(tx *gorm.DB) error {
        // 取消订单
        if err := tx.Model(&order).Update("status", "cancelled").Error; err != nil {
            return err
        }

        // 回滚优惠券
        if order.CouponID != nil {
            // 删除使用记录
            if err := tx.Where("order_id = ?", order.ID).Delete(&models.CouponUsage{}).Error; err != nil {
                return err
            }
            // 减少使用次数
            if err := tx.Model(&models.Coupon{}).Where("id = ?", *order.CouponID).
                UpdateColumn("used_quantity", gorm.Expr("CASE WHEN used_quantity > 0 THEN used_quantity - 1 ELSE 0 END")).Error; err != nil {
                return err
            }
        }

        return nil
    })
}
```

---

## 🎯 优先级

### P0 - 立即修复（24小时内）
1. ✅ 优惠券竞态条件
2. ✅ 优惠券过期时间检查
3. ✅ 优惠券使用记录事务

### P1 - 高优先级（3天内）
4. ✅ 订单取消回滚优惠券
5. ✅ 页码上限
6. ✅ 优惠券验证频率限制

### P2 - 中优先级（1周内）
7. ✅ N+1 查询优化
8. ⚠️ 硬编码 Limit 配置化

---

**报告生成时间：** 2026-03-02
**审计人员：** Claude (AI Security Audit - Round 4)
**严重程度：** 🔴 CRITICAL
**建议：** 立即修复 P0 级别漏洞
