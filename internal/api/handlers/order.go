package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

	// Enrich with package_name
	type OrderItem struct {
		models.Order
		PackageName string `json:"package_name"`
	}
	items := make([]OrderItem, 0, len(orders))
	pkgNameCache := make(map[uint]string)
	dbConn := database.GetDB()
	for _, o := range orders {
		item := OrderItem{Order: o}
		if o.PackageID == 0 && o.ExtraData != nil {
			var extra map[string]interface{}
			if json.Unmarshal([]byte(*o.ExtraData), &extra) == nil {
				if extra["type"] == "custom_package" {
					devices, _ := extra["devices"].(float64)
					months, _ := extra["months"].(float64)
					item.PackageName = fmt.Sprintf("自定义套餐 (%d设备/%d月)", int(devices), int(months))
				} else if extra["type"] == "subscription_upgrade" {
					addDevices, _ := extra["add_devices"].(float64)
					extendMonths, _ := extra["extend_months"].(float64)
					item.PackageName = fmt.Sprintf("订阅升级: +%d设备", int(addDevices))
					if int(extendMonths) > 0 {
						item.PackageName = fmt.Sprintf("订阅升级: +%d设备, 续期%d月", int(addDevices), int(extendMonths))
					}
				}
			}
		} else if name, ok := pkgNameCache[o.PackageID]; ok {
			item.PackageName = name
		} else {
			var pkg models.Package
			if err := dbConn.Select("name").First(&pkg, o.PackageID).Error; err == nil {
				item.PackageName = pkg.Name
				pkgNameCache[o.PackageID] = pkg.Name
			}
		}
		items = append(items, item)
	}
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func CreateOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		PackageID  uint   `json:"package_id" binding:"required"`
		CouponCode string `json:"coupon_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	var pkg models.Package
	if err := db.First(&pkg, req.PackageID).Error; err != nil {
		utils.NotFound(c, "套餐不存在")
		return
	}
	if !pkg.IsActive {
		utils.BadRequest(c, "套餐已下架")
		return
	}
	amount := pkg.Price
	var discountAmount float64
	var couponID *int64
	if req.CouponCode != "" {
		var coupon models.Coupon
		if err := db.Where("code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err == nil {
			now := time.Now()
			if now.After(coupon.ValidFrom) && now.Before(coupon.ValidUntil.AddDate(0, 0, 1)) {
				// Check total quantity
				if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
					utils.BadRequest(c, "优惠券已被领完")
					return
				}
				// Check per-user usage
				var usageCount int64
				db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).Count(&usageCount)
				if int(usageCount) >= coupon.MaxUsesPerUser {
					utils.BadRequest(c, "您已达到该优惠券的使用上限")
					return
				}
				switch coupon.Type {
				case "discount":
					discountAmount = math.Round(amount*coupon.DiscountValue) / 100
				case "fixed":
					discountAmount = coupon.DiscountValue
				}
				if coupon.MaxDiscount != nil && discountAmount > *coupon.MaxDiscount {
					discountAmount = *coupon.MaxDiscount
				}
				if discountAmount > amount {
					discountAmount = amount
				}
				discountAmount = math.Round(discountAmount*100) / 100
				cid := int64(coupon.ID)
				couponID = &cid
			}
		}
	}
	finalAmount := amount - discountAmount
	orderNo := fmt.Sprintf("ORD%d%s", time.Now().Unix(), utils.GenerateRandomString(6))
	expireTime := time.Now().Add(30 * time.Minute)
	order := models.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		PackageID:      req.PackageID,
		Amount:         amount,
		Status:         "pending",
		CouponID:       couponID,
		DiscountAmount: &discountAmount,
		FinalAmount:    &finalAmount,
		ExpireTime:     &expireTime,
	}
	if err := db.Create(&order).Error; err != nil {
		utils.InternalError(c, "创建订单失败")
		return
	}
	// Record coupon usage
	if couponID != nil {
		db.Create(&models.CouponUsage{CouponID: uint(*couponID), UserID: userID, OrderID: func() *int64 { id := int64(order.ID); return &id }(), DiscountAmount: discountAmount})
		db.Model(&models.Coupon{}).Where("id = ?", *couponID).UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1"))
	}

	// 通知用户新订单 + 通知管理员
	user := c.MustGet("user").(*models.User)
	go services.NotifyUser(userID, "new_order", map[string]string{
		"order_no": orderNo, "package_name": pkg.Name, "amount": fmt.Sprintf("%.2f", finalAmount),
	})
	go services.NotifyAdmin("new_order", map[string]string{
		"username": user.Username, "order_no": orderNo, "package_name": pkg.Name, "amount": fmt.Sprintf("%.2f", finalAmount),
	})

	utils.Success(c, order)
}

func PayOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderNo := c.Param("orderNo")
	var req struct {
		PaymentMethod string `json:"payment_method"`
	}
	c.ShouldBindJSON(&req)
	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ? AND status = ?", orderNo, userID, "pending").First(&order).Error; err != nil {
		utils.NotFound(c, "订单不存在或状态不正确")
		return
	}
	// 余额支付
	if req.PaymentMethod == "balance" {
		// 检查余额支付是否启用
		balEnabled := utils.GetSetting("pay_balance_enabled")
		if balEnabled == "false" || balEnabled == "0" {
			utils.BadRequest(c, "余额支付已关闭")
			return
		}
		user := c.MustGet("user").(*models.User)
		payAmount := order.Amount
		if order.FinalAmount != nil {
			payAmount = *order.FinalAmount
		}
		if user.Balance < payAmount {
			utils.BadRequest(c, "余额不足")
			return
		}
		tx := db.Begin()
		// Re-check order status inside transaction to prevent double-spend
		var freshOrder models.Order
		if err := tx.Where("id = ? AND status = ?", order.ID, "pending").First(&freshOrder).Error; err != nil {
			tx.Rollback()
			utils.BadRequest(c, "订单已支付或已取消")
			return
		}
		if err := tx.Model(user).UpdateColumn("balance", gorm.Expr("balance - ?", payAmount)).Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "扣减余额失败")
			return
		}
		// 记录余额消费日志
		orderID := order.ID
		utils.CreateBalanceLogEntry(userID, "consume", -payAmount, user.Balance, user.Balance-payAmount, &orderID, fmt.Sprintf("余额支付订单: %s", orderNo), c)
		now := time.Now()
		balanceStr := "balance"
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":              "paid",
			"payment_method_name": &balanceStr,
			"payment_time":        &now,
		}).Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "更新订单状态失败")
			return
		}
		// 创建或续期订阅
		var deviceLimit int
		var durationDays int
		var pkgName string

		isUpgradeOrder := false
		var upgradeAddDevices int
		var upgradeExtendMonths int
		if order.PackageID == 0 && order.ExtraData != nil {
			var extra map[string]interface{}
			if err := json.Unmarshal([]byte(*order.ExtraData), &extra); err != nil {
				tx.Rollback()
				utils.InternalError(c, "订单数据异常")
				return
			}
			if extra["type"] == "subscription_upgrade" {
				isUpgradeOrder = true
				if v, ok := extra["add_devices"].(float64); ok {
					upgradeAddDevices = int(v)
				}
				if v, ok := extra["extend_months"].(float64); ok {
					upgradeExtendMonths = int(v)
				}
			} else {
				devices, _ := extra["devices"].(float64)
				months, _ := extra["months"].(float64)
				deviceLimit = int(devices)
				durationDays = int(months) * 30
				pkgName = fmt.Sprintf("自定义套餐 (%d设备/%d月)", int(devices), int(months))
			}
		} else {
			var pkg models.Package
			if err := tx.First(&pkg, order.PackageID).Error; err != nil {
				tx.Rollback()
				utils.InternalError(c, "套餐不存在")
				return
			}
			deviceLimit = pkg.DeviceLimit
			durationDays = pkg.DurationDays
			pkgName = pkg.Name
		}

		var sub models.Subscription
		if err := tx.Where("user_id = ?", userID).First(&sub).Error; err != nil {
			if isUpgradeOrder {
				tx.Rollback()
				utils.BadRequest(c, "升级订单需要已有订阅")
				return
			}
			sub = models.Subscription{
				UserID:          userID,
				SubscriptionURL: utils.GenerateRandomString(32),
				DeviceLimit:     deviceLimit,
				IsActive:        true,
				Status:          "active",
				ExpireTime:      time.Now().AddDate(0, 0, durationDays),
			}
			if order.PackageID > 0 {
				pkgID := int64(order.PackageID)
				sub.PackageID = &pkgID
			}
			if err := tx.Create(&sub).Error; err != nil {
				tx.Rollback()
				utils.InternalError(c, "创建订阅失败")
				return
			}
		} else {
			if isUpgradeOrder {
				pkgName = fmt.Sprintf("订阅升级: +%d设备", upgradeAddDevices)
				if upgradeExtendMonths > 0 {
					pkgName = fmt.Sprintf("订阅升级: +%d设备, 续期%d月", upgradeAddDevices, upgradeExtendMonths)
				}
				newLimit := sub.DeviceLimit + upgradeAddDevices
				newExpire := sub.ExpireTime
				if upgradeExtendMonths > 0 {
					newExpire = newExpire.AddDate(0, upgradeExtendMonths, 0)
				}
				if err := tx.Model(&sub).Updates(map[string]interface{}{
					"device_limit": newLimit,
					"expire_time":  newExpire,
					"is_active":    true,
					"status":       "active",
				}).Error; err != nil {
					tx.Rollback()
					utils.InternalError(c, "订阅升级失败")
					return
				}
			} else {
				newExpire := sub.ExpireTime
				if newExpire.Before(time.Now()) {
					newExpire = time.Now()
				}
				newExpire = newExpire.AddDate(0, 0, durationDays)
				updates := map[string]interface{}{
					"device_limit": deviceLimit,
					"expire_time":  newExpire,
					"is_active":    true,
					"status":       "active",
				}
				if order.PackageID > 0 {
					pkgID := int64(order.PackageID)
					updates["package_id"] = &pkgID
				}
				if err := tx.Model(&sub).Updates(updates).Error; err != nil {
					tx.Rollback()
					utils.InternalError(c, "续期订阅失败")
					return
				}
			}
		}
		if err := tx.Commit().Error; err != nil {
			utils.InternalError(c, "支付事务提交失败")
			return
		}
		// 发送支付成功邮件 + 通知管理员
		payAmountStr := fmt.Sprintf("%.2f", payAmount)
		var subURL string
		var userSub models.Subscription
		if database.GetDB().Where("user_id = ?", user.ID).First(&userSub).Error == nil {
			settings := utils.GetSettings("site_url", "domain_name")
			siteURL := settings["site_url"]
			if siteURL == "" {
				siteURL = settings["domain_name"]
			}
			if siteURL != "" && !strings.HasPrefix(siteURL, "http") {
				siteURL = "https://" + siteURL
			}
			siteURL = strings.TrimRight(siteURL, "/")
			subURL = siteURL + "/api/v1/subscribe/" + userSub.SubscriptionURL
		}
		emailSubject, emailBody := services.RenderEmail("payment_success", map[string]string{
			"order_no": orderNo, "amount": payAmountStr, "package_name": pkgName, "subscription_url": subURL,
		})
		go services.QueueEmail(user.Email, emailSubject, emailBody, "payment_success")
		go services.NotifyAdmin("payment_success", map[string]string{
			"username": user.Username, "order_no": orderNo, "package_name": pkgName, "amount": payAmountStr,
		})
		utils.Success(c, gin.H{"message": "支付成功", "order_no": orderNo})
		return
	}
	// Other payment methods are handled via CreatePayment in payment.go
	utils.BadRequest(c, "暂不支持该支付方式，请使用余额支付或通过支付接口创建支付")
}

func CancelOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderNo := c.Param("orderNo")
	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ? AND status = ?", orderNo, userID, "pending").First(&order).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	db.Model(&order).Update("status", "cancelled")
	utils.SuccessMessage(c, "订单已取消")
}

func GetOrderStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderNo := c.Param("orderNo")
	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, userID).First(&order).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	result := gin.H{"status": order.Status, "order_no": order.OrderNo, "final_amount": order.Amount}
	if order.FinalAmount != nil {
		result["final_amount"] = *order.FinalAmount
	}
	if order.PaymentTime != nil {
		result["paid_at"] = order.PaymentTime.Format("2006-01-02 15:04:05")
	}
	// Get package name
	if order.PackageID == 0 && order.ExtraData != nil {
		var extra map[string]interface{}
		if json.Unmarshal([]byte(*order.ExtraData), &extra) == nil {
			if extra["type"] == "custom_package" {
				devices, _ := extra["devices"].(float64)
				months, _ := extra["months"].(float64)
				result["package_name"] = fmt.Sprintf("自定义套餐 (%d设备/%d月)", int(devices), int(months))
			} else if extra["type"] == "subscription_upgrade" {
				addDevices, _ := extra["add_devices"].(float64)
				extendMonths, _ := extra["extend_months"].(float64)
				result["package_name"] = fmt.Sprintf("订阅升级: +%d设备", int(addDevices))
				if int(extendMonths) > 0 {
					result["package_name"] = fmt.Sprintf("订阅升级: +%d设备, 续期%d月", int(addDevices), int(extendMonths))
				}
			}
		}
	} else {
		var pkg models.Package
		if err := db.First(&pkg, order.PackageID).Error; err == nil {
			result["package_name"] = pkg.Name
		}
	}
	utils.Success(c, result)
}

// CreateCustomOrder POST /orders/custom
func CreateCustomOrder(c *gin.Context) {
	var req struct {
		Devices    int    `json:"devices" binding:"required,min=1"`
		Months     int    `json:"months" binding:"required,min=1"`
		CouponCode string `json:"coupon_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	if !utils.IsBoolSetting("custom_package_enabled") {
		utils.BadRequest(c, "自定义套餐功能未启用")
		return
	}

	pricePerDeviceYear := utils.GetFloatSetting("custom_package_price_per_device_year", 40)
	minDevices := utils.GetIntSetting("custom_package_min_devices", 1)
	maxDevices := utils.GetIntSetting("custom_package_max_devices", 20)
	minMonths := utils.GetIntSetting("custom_package_min_months", 6)

	if req.Devices < minDevices || req.Devices > maxDevices {
		utils.BadRequest(c, fmt.Sprintf("设备数量需在 %d ~ %d 之间", minDevices, maxDevices))
		return
	}
	if req.Months < minMonths {
		utils.BadRequest(c, fmt.Sprintf("最少购买 %d 个月", minMonths))
		return
	}

	// Parse duration discounts
	var discountTiers []struct {
		Months   int     `json:"months"`
		Discount float64 `json:"discount"`
	}
	discountsJSON := utils.GetSetting("custom_package_duration_discounts")
	if discountsJSON != "" {
		json.Unmarshal([]byte(discountsJSON), &discountTiers)
	}

	// Calculate price
	basePrice := pricePerDeviceYear * float64(req.Devices) * (float64(req.Months) / 12.0)
	basePrice = math.Round(basePrice*100) / 100

	// Find best matching discount
	var discountPercent float64
	for _, tier := range discountTiers {
		if req.Months >= tier.Months && tier.Discount > discountPercent {
			discountPercent = tier.Discount
		}
	}
	finalPrice := basePrice * (1 - discountPercent/100)
	finalPrice = math.Round(finalPrice*100) / 100

	// Apply coupon
	userID := c.GetUint("user_id")
	db := database.GetDB()
	var couponDiscount float64
	var couponID *int64
	if req.CouponCode != "" {
		var coupon models.Coupon
		if err := db.Where("code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err == nil {
			now := time.Now()
			if now.After(coupon.ValidFrom) && now.Before(coupon.ValidUntil.AddDate(0, 0, 1)) {
				if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
					utils.BadRequest(c, "优惠券已被领完")
					return
				}
				var usageCount int64
				db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).Count(&usageCount)
				if int(usageCount) >= coupon.MaxUsesPerUser {
					utils.BadRequest(c, "您已达到该优惠券的使用上限")
					return
				}
				switch coupon.Type {
				case "discount":
					couponDiscount = math.Round(finalPrice*coupon.DiscountValue) / 100
				case "fixed":
					couponDiscount = coupon.DiscountValue
				}
				if coupon.MaxDiscount != nil && couponDiscount > *coupon.MaxDiscount {
					couponDiscount = *coupon.MaxDiscount
				}
				if couponDiscount > finalPrice {
					couponDiscount = finalPrice
				}
				couponDiscount = math.Round(couponDiscount*100) / 100
				cid := int64(coupon.ID)
				couponID = &cid
			}
		}
	}
	finalPrice = finalPrice - couponDiscount
	if finalPrice < 0 {
		finalPrice = 0
	}

	// Build extra data
	extraData, _ := json.Marshal(map[string]interface{}{
		"type": "custom_package", "devices": req.Devices,
		"months": req.Months, "discount_percent": discountPercent,
	})
	extraStr := string(extraData)

	orderNo := fmt.Sprintf("ORD%d%s", time.Now().Unix(), utils.GenerateRandomString(6))
	expireTime := time.Now().Add(30 * time.Minute)
	totalDiscount := (basePrice - finalPrice)
	order := models.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		PackageID:      0,
		Amount:         basePrice,
		Status:         "pending",
		CouponID:       couponID,
		DiscountAmount: &totalDiscount,
		FinalAmount:    &finalPrice,
		ExpireTime:     &expireTime,
		ExtraData:      &extraStr,
	}
	if err := db.Create(&order).Error; err != nil {
		utils.InternalError(c, "创建订单失败")
		return
	}
	if couponID != nil {
		db.Create(&models.CouponUsage{CouponID: uint(*couponID), UserID: userID, OrderID: func() *int64 { id := int64(order.ID); return &id }(), DiscountAmount: couponDiscount})
		db.Model(&models.Coupon{}).Where("id = ?", *couponID).UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1"))
	}

	pkgName := fmt.Sprintf("自定义套餐 (%d设备/%d月)", req.Devices, req.Months)
	user := c.MustGet("user").(*models.User)
	go services.NotifyUser(userID, "new_order", map[string]string{
		"order_no": orderNo, "package_name": pkgName, "amount": fmt.Sprintf("%.2f", finalPrice),
	})
	go services.NotifyAdmin("new_order", map[string]string{
		"username": user.Username, "order_no": orderNo, "package_name": pkgName, "amount": fmt.Sprintf("%.2f", finalPrice),
	})

	utils.Success(c, order)
}

// CalcUpgradePrice 计算「增加设备 + 可选续期」应付金额（按剩余时间与续期分别计费）
// 单价：custom_package_price_per_device_year 元/设备/年（如 40）
// - 仅增加设备：新增设备数 × 单价 × (剩余天数/365)
// - 仅续期：原设备数 × 单价 × (续期月数/12)
// - 增加设备且续期：原设备续期费 + 新增设备 × 单价 × (剩余天数/365 + 续期月数/12)
func CalcUpgradePrice(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.NotFound(c, "暂无有效订阅")
		return
	}
	var req struct {
		AddDevices   int `json:"add_devices" binding:"required,min=1"`
		ExtendMonths int `json:"extend_months"` // 0 表示不续期
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if req.AddDevices%5 != 0 {
		utils.BadRequest(c, "增加设备数只能为 5 的倍数")
		return
	}

	pricePerDeviceYear := utils.GetFloatSetting("custom_package_price_per_device_year", 40)
	now := time.Now()
	remainingDays := 0.0
	if sub.ExpireTime.After(now) {
		remainingDays = math.Max(0, sub.ExpireTime.Sub(now).Hours()/24)
	}

	currentDevices := sub.DeviceLimit
	currentExpire := sub.ExpireTime

	var feeExtend float64
	if req.ExtendMonths > 0 {
		feeExtend = float64(currentDevices) * pricePerDeviceYear * (float64(req.ExtendMonths) / 12.0)
		feeExtend = math.Round(feeExtend*100) / 100
	}

	remainingYears := remainingDays / 365.0
	extendYears := float64(req.ExtendMonths) / 12.0
	totalYearsForNewDevices := remainingYears + extendYears
	feeNewDevices := float64(req.AddDevices) * pricePerDeviceYear * totalYearsForNewDevices
	feeNewDevices = math.Round(feeNewDevices*100) / 100

	total := feeExtend + feeNewDevices
	total = math.Round(total*100) / 100

	utils.Success(c, gin.H{
		"price_per_device_year": pricePerDeviceYear,
		"current_device_limit":  currentDevices,
		"current_expire_time":   currentExpire.Format("2006-01-02 15:04:05"),
		"remaining_days":        int(math.Ceil(remainingDays)),
		"add_devices":           req.AddDevices,
		"extend_months":         req.ExtendMonths,
		"fee_extend":            feeExtend,
		"fee_new_devices":       feeNewDevices,
		"total":                 total,
	})
}

// CreateUpgradeOrder 创建「增加设备 + 可选续期」订单
func CreateUpgradeOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.NotFound(c, "暂无有效订阅")
		return
	}
	var req struct {
		AddDevices   int    `json:"add_devices" binding:"required,min=1"`
		ExtendMonths int    `json:"extend_months"`
		CouponCode   string `json:"coupon_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if req.AddDevices%5 != 0 {
		utils.BadRequest(c, "增加设备数只能为 5 的倍数")
		return
	}

	pricePerDeviceYear := utils.GetFloatSetting("custom_package_price_per_device_year", 40)
	now := time.Now()
	remainingDays := 0.0
	if sub.ExpireTime.After(now) {
		remainingDays = math.Max(0, sub.ExpireTime.Sub(now).Hours()/24)
	}

	var feeExtend float64
	if req.ExtendMonths > 0 {
		feeExtend = float64(sub.DeviceLimit) * pricePerDeviceYear * (float64(req.ExtendMonths) / 12.0)
		feeExtend = math.Round(feeExtend*100) / 100
	}
	remainingYears := remainingDays / 365.0
	extendYears := float64(req.ExtendMonths) / 12.0
	feeNewDevices := float64(req.AddDevices) * pricePerDeviceYear * (remainingYears + extendYears)
	feeNewDevices = math.Round(feeNewDevices*100) / 100
	basePrice := feeExtend + feeNewDevices
	basePrice = math.Round(basePrice*100) / 100

	var couponDiscount float64
	var couponID *int64
	if req.CouponCode != "" {
		var coupon models.Coupon
		if err := db.Where("code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err == nil {
			if time.Now().After(coupon.ValidFrom) && time.Now().Before(coupon.ValidUntil.AddDate(0, 0, 1)) {
				if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
					utils.BadRequest(c, "优惠券已被领完")
					return
				}
				var usageCount int64
				db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).Count(&usageCount)
				if int(usageCount) >= coupon.MaxUsesPerUser {
					utils.BadRequest(c, "您已达到该优惠券的使用上限")
					return
				}
				switch coupon.Type {
				case "discount":
					couponDiscount = math.Round(basePrice*coupon.DiscountValue) / 100
				case "fixed":
					couponDiscount = coupon.DiscountValue
				}
				if coupon.MaxDiscount != nil && couponDiscount > *coupon.MaxDiscount {
					couponDiscount = *coupon.MaxDiscount
				}
				if couponDiscount > basePrice {
					couponDiscount = basePrice
				}
				couponDiscount = math.Round(couponDiscount*100) / 100
				cid := int64(coupon.ID)
				couponID = &cid
			}
		}
	}
	finalPrice := basePrice - couponDiscount
	if finalPrice < 0 {
		finalPrice = 0
	}
	finalPrice = math.Round(finalPrice*100) / 100
	totalDiscount := basePrice - finalPrice

	extraData, _ := json.Marshal(map[string]interface{}{
		"type":                 "subscription_upgrade",
		"add_devices":          req.AddDevices,
		"extend_months":        req.ExtendMonths,
		"current_device_limit": sub.DeviceLimit,
		"current_expire_time":  sub.ExpireTime.Format(time.RFC3339),
	})
	extraStr := string(extraData)

	orderNo := fmt.Sprintf("ORD%d%s", time.Now().Unix(), utils.GenerateRandomString(6))
	expireTime := time.Now().Add(30 * time.Minute)
	order := models.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		PackageID:      0,
		Amount:         basePrice,
		Status:         "pending",
		CouponID:       couponID,
		DiscountAmount: &totalDiscount,
		FinalAmount:    &finalPrice,
		ExpireTime:     &expireTime,
		ExtraData:      &extraStr,
	}
	if err := db.Create(&order).Error; err != nil {
		utils.InternalError(c, "创建订单失败")
		return
	}
	if couponID != nil {
		db.Create(&models.CouponUsage{CouponID: uint(*couponID), UserID: userID, OrderID: func() *int64 { id := int64(order.ID); return &id }(), DiscountAmount: couponDiscount})
		db.Model(&models.Coupon{}).Where("id = ?", *couponID).UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1"))
	}

	pkgName := fmt.Sprintf("订阅升级: +%d设备", req.AddDevices)
	if req.ExtendMonths > 0 {
		pkgName = fmt.Sprintf("订阅升级: +%d设备, 续期%d月", req.AddDevices, req.ExtendMonths)
	}
	user := c.MustGet("user").(*models.User)
	go services.NotifyUser(userID, "new_order", map[string]string{
		"order_no": orderNo, "package_name": pkgName, "amount": fmt.Sprintf("%.2f", finalPrice),
	})
	go services.NotifyAdmin("new_order", map[string]string{
		"username": user.Username, "order_no": orderNo, "package_name": pkgName, "amount": fmt.Sprintf("%.2f", finalPrice),
	})

	utils.Success(c, order)
}
