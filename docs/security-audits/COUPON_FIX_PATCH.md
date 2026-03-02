# 优惠券与性能修复补丁

## 修复说明

由于优惠券相关代码涉及多个文件和大量代码，这里提供关键修复点的代码片段。

## 1. 修复优惠券过期时间检查

**文件：** `/internal/api/handlers/coupon.go:38`

**修复前：**
```go
if now.After(coupon.ValidUntil.AddDate(0, 0, 1)) {
```

**修复后：**
```go
if now.After(coupon.ValidUntil) {
```

## 2. 添加优惠券验证频率限制

**文件：** `/internal/api/router/router.go`

**修复：**
```go
// 优惠券（添加频率限制）
authorized.POST("/coupons/verify", middleware.RateLimit(10, time.Minute), handlers.VerifyCoupon)
authorized.GET("/coupons/my", handlers.GetMyCoupons)
```

## 3. 修复优惠券竞态条件（核心修复）

**文件：** `/internal/api/handlers/order.go`

**关键点：**
1. 在事务内使用行锁查询优惠券
2. 在事务内检查数量和使用次数
3. 在事务内创建订单和优惠券使用记录

**修复模式：**
```go
err := db.Transaction(func(tx *gorm.DB) error {
    var coupon models.Coupon
    if req.CouponCode != "" {
        // 使用行锁查询优惠券
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Where("code = ? AND status = ?", req.CouponCode, "active").
            First(&coupon).Error; err != nil {
            return fmt.Errorf("coupon_not_found")
        }

        // 在锁内检查有效期
        now := time.Now()
        if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
            return fmt.Errorf("coupon_expired")
        }

        // 在锁内检查数量
        if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
            return fmt.Errorf("coupon_exhausted")
        }

        // 在锁内检查用户使用次数
        var usageCount int64
        tx.Model(&models.CouponUsage{}).
            Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).
            Count(&usageCount)
        if int(usageCount) >= coupon.MaxUsesPerUser {
            return fmt.Errorf("coupon_limit_exceeded")
        }

        // 计算折扣...
    }

    // 创建订单
    if err := tx.Create(&order).Error; err != nil {
        return err
    }

    // 在同一事务内记录优惠券使用
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
```

## 4. 修复订单取消回滚优惠券

**文件：** `/internal/api/handlers/order.go`

**修复：**
```go
func CancelOrder(c *gin.Context) {
    userID := c.GetUint("user_id")
    orderNo := c.Param("orderNo")
    db := database.GetDB()

    err := db.Transaction(func(tx *gorm.DB) error {
        var order models.Order
        if err := tx.Where("order_no = ? AND user_id = ? AND status = ?", orderNo, userID, "pending").
            First(&order).Error; err != nil {
            return fmt.Errorf("order_not_found")
        }

        // 取消订单
        if err := tx.Model(&order).Update("status", "cancelled").Error; err != nil {
            return err
        }

        // 回滚优惠券
        if order.CouponID != nil {
            // 删除使用记录
            orderID := int64(order.ID)
            if err := tx.Where("order_id = ?", orderID).Delete(&models.CouponUsage{}).Error; err != nil {
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

    if err != nil {
        if err.Error() == "order_not_found" {
            utils.NotFound(c, "订单不存在")
        } else {
            utils.InternalError(c, "取消订单失败")
        }
        return
    }

    utils.SuccessMessage(c, "订单已取消")
}
```

## 5. N+1 查询优化示例

**文件：** `/internal/api/handlers/order.go`

**优化方案：**
```go
func ListOrders(c *gin.Context) {
    userID := c.GetUint("user_id")
    p := utils.GetPagination(c)
    var orders []models.Order
    var total int64

    db := database.GetDB().Model(&models.Order{}).Where("user_id = ?", userID)
    if status := c.Query("status"); status != "" {
        db = db.Where("status = ?", status)
    }
    db.Count(&total)
    db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&orders)

    // 优化：批量查询套餐名称
    packageIDs := make([]uint, 0)
    for _, o := range orders {
        if o.PackageID > 0 {
            packageIDs = append(packageIDs, o.PackageID)
        }
    }

    packageMap := make(map[uint]string)
    if len(packageIDs) > 0 {
        var packages []models.Package
        database.GetDB().Where("id IN ?", packageIDs).Find(&packages)
        for _, pkg := range packages {
            packageMap[pkg.ID] = pkg.Name
        }
    }

    // 组装结果
    type OrderItem struct {
        models.Order
        PackageName string `json:"package_name"`
    }
    items := make([]OrderItem, 0, len(orders))
    for _, o := range orders {
        item := OrderItem{Order: o}
        if name, ok := packageMap[o.PackageID]; ok {
            item.PackageName = name
        } else if o.PackageID == 0 && o.ExtraData != nil {
            // 处理自定义套餐...
        }
        items = append(items, item)
    }

    utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}
```

## 实施步骤

1. **立即修复（P0）：**
   - ✅ 修复 pagination.go 页码上限（已完成）
   - ⚠️ 修复 coupon.go 过期时间检查
   - ⚠️ 修复 order.go 优惠券竞态条件
   - ⚠️ 修复 order.go 订单取消回滚

2. **高优先级（P1）：**
   - ⚠️ 添加优惠券验证频率限制
   - ⚠️ 优化 N+1 查询

3. **测试验证：**
   ```bash
   # 测试页码上限
   curl "http://localhost:8000/api/v1/orders?page=999999"
   # 应返回 page=10000 的结果

   # 测试优惠券并发
   # 创建总数量=1的优惠券，并发10个请求
   # 应只有1个成功

   # 测试订单取消
   # 创建订单使用优惠券，取消订单
   # 优惠券应可再次使用
   ```

## 注意事项

1. **优惠券竞态修复需要修改 3 个函数：**
   - `CreateOrder` - 普通订单
   - `CreateCustomOrder` - 自定义订单
   - `CreateUpgradeOrder` - 升级订单

2. **所有修改都需要添加 `clause.Locking{Strength: "UPDATE"}` 行锁**

3. **测试时注意并发场景**

4. **部署前务必备份数据库**

---

**创建时间：** 2026-03-02
**修复人员：** Claude (AI Security Audit)
**优先级：** P0 - 立即修复
