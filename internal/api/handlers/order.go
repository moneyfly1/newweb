package handlers

import (
	"fmt"
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
		if name, ok := pkgNameCache[o.PackageID]; ok {
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
			if time.Now().After(coupon.ValidFrom) && time.Now().Before(coupon.ValidUntil) {
				switch coupon.Type {
				case "discount":
					discountAmount = amount * coupon.DiscountValue / 100
				case "fixed":
					discountAmount = coupon.DiscountValue
				}
				if discountAmount > amount {
					discountAmount = amount
				}
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
	db.Create(&order)

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
		var pkg models.Package
		if err := tx.First(&pkg, order.PackageID).Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "套餐不存在")
			return
		}
		var sub models.Subscription
		if err := tx.Where("user_id = ?", userID).First(&sub).Error; err != nil {
			// 创建新订阅
			pkgID := int64(pkg.ID)
			sub = models.Subscription{
				UserID:          userID,
				PackageID:       &pkgID,
				SubscriptionURL: utils.GenerateRandomString(32),
				DeviceLimit:     pkg.DeviceLimit,
				IsActive:        true,
				Status:          "active",
				ExpireTime:      time.Now().AddDate(0, 0, pkg.DurationDays),
			}
			if err := tx.Create(&sub).Error; err != nil {
				tx.Rollback()
				utils.InternalError(c, "创建订阅失败")
				return
			}
		} else {
			// 续期
			newExpire := sub.ExpireTime
			if newExpire.Before(time.Now()) {
				newExpire = time.Now()
			}
			newExpire = newExpire.AddDate(0, 0, pkg.DurationDays)
			pkgID := int64(pkg.ID)
			if err := tx.Model(&sub).Updates(map[string]interface{}{
				"package_id":   &pkgID,
				"device_limit": pkg.DeviceLimit,
				"expire_time":  newExpire,
				"is_active":    true,
				"status":       "active",
			}).Error; err != nil {
				tx.Rollback()
				utils.InternalError(c, "续期订阅失败")
				return
			}
		}
		if err := tx.Commit().Error; err != nil {
			utils.InternalError(c, "支付事务提交失败")
			return
		}
		// 发送支付成功邮件 + 通知管理员
		payAmountStr := fmt.Sprintf("%.2f", payAmount)
		var pkgName string
		if err := database.GetDB().Model(&models.Package{}).Where("id = ?", order.PackageID).Pluck("name", &pkgName).Error; err != nil {
			pkgName = "未知套餐"
		}
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
	var pkg models.Package
	if err := db.First(&pkg, order.PackageID).Error; err == nil {
		result["package_name"] = pkg.Name
	}
	utils.Success(c, result)
}
