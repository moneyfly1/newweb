package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetPaymentMethods(c *gin.Context) {
	db := database.GetDB()
	var configs []models.PaymentConfig
	db.Where("status = ?", 1).Order("sort_order ASC").Find(&configs)
	var methods []gin.H
	for _, cfg := range configs {
		methods = append(methods, gin.H{"id": cfg.ID, "pay_type": cfg.PayType, "sort_order": cfg.SortOrder})
	}

	// Also check if EasyPay is enabled in system_configs (auto-create PaymentConfig if needed)
	var sysConfigs []models.SystemConfig
	db.Where("`key` IN ?", []string{"pay_epay_enabled", "pay_epay_gateway", "pay_epay_merchant_id", "pay_epay_secret_key", "pay_balance_enabled"}).Find(&sysConfigs)
	cfgMap := make(map[string]string)
	for _, sc := range sysConfigs {
		cfgMap[sc.Key] = sc.Value
	}

	// Check balance enabled (default true)
	balanceEnabled := cfgMap["pay_balance_enabled"] != "false" && cfgMap["pay_balance_enabled"] != "0"

	if (cfgMap["pay_epay_enabled"] == "true" || cfgMap["pay_epay_enabled"] == "1") &&
		cfgMap["pay_epay_gateway"] != "" && cfgMap["pay_epay_merchant_id"] != "" && cfgMap["pay_epay_secret_key"] != "" {
		// Check if epay PaymentConfig already exists
		hasEpay := false
		for _, m := range methods {
			if m["pay_type"] == "epay" {
				hasEpay = true
				break
			}
		}
		if !hasEpay {
			// Auto-create a PaymentConfig for epay
			pc := models.PaymentConfig{PayType: "epay", Status: 1, SortOrder: 100}
			db.Create(&pc)
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "epay", "sort_order": 100})
		}
	}

	utils.Success(c, gin.H{
		"methods":         methods,
		"balance_enabled": balanceEnabled,
	})
}

func CreatePayment(c *gin.Context) {
	var req struct {
		OrderID         uint `json:"order_id" binding:"required"`
		PaymentMethodID uint `json:"payment_method_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	userID := c.MustGet("user_id").(uint)
	db := database.GetDB()
	var order models.Order
	if err := db.Where("id = ? AND user_id = ? AND status = ?", req.OrderID, userID, "pending").First(&order).Error; err != nil {
		utils.NotFound(c, "订单不存在或状态不正确")
		return
	}

	// Verify payment method exists
	var payConfig models.PaymentConfig
	if err := db.Where("id = ? AND status = ?", req.PaymentMethodID, 1).First(&payConfig).Error; err != nil {
		utils.NotFound(c, "支付方式不存在或未启用")
		return
	}

	// Calculate payment amount
	payAmount := order.Amount
	if order.FinalAmount != nil {
		payAmount = *order.FinalAmount
	}

	// Create payment transaction
	txID := fmt.Sprintf("PAY%d%s", time.Now().Unix(), utils.GenerateRandomString(8))
	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          userID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          payAmount,
		Currency:        "CNY",
		TransactionID:   &txID,
		Status:          "pending",
	}
	if err := db.Create(&transaction).Error; err != nil {
		utils.InternalError(c, "创建支付交易失败")
		return
	}

	// If this is an EasyPay payment type, call the EasyPay gateway
	if payConfig.PayType == "epay" || payConfig.PayType == "alipay" || payConfig.PayType == "wxpay" || payConfig.PayType == "qqpay" {
		epayCfg, err := services.GetEpayConfig()
		if err != nil {
			utils.BadRequest(c, err.Error())
			return
		}

		// Determine the epay pay type
		epayType := payConfig.PayType
		if epayType == "epay" {
			epayType = "alipay" // default to alipay
		}

		// Build notify and return URLs
		// Get site_url from system_configs for return URL
		var siteURLConfig models.SystemConfig
		siteURL := ""
		if err := db.Where("`key` = ?", "site_url").First(&siteURLConfig).Error; err == nil {
			siteURL = strings.TrimRight(siteURLConfig.Value, "/")
		}

		// Build API base URL from site_url or fallback
		apiBase := siteURL
		if apiBase == "" {
			apiBase = "http://localhost:8000"
		}
		notifyURL := apiBase + "/api/v1/payment/notify/epay"
		returnURL := siteURL + "/payment/return?order_no=" + url.QueryEscape(order.OrderNo)

		// Get package name for the order description
		var pkg models.Package
		orderName := "订单-" + order.OrderNo
		if err := db.First(&pkg, order.PackageID).Error; err == nil {
			orderName = pkg.Name
		}

		paymentURL, err := services.EpayCreateOrder(epayCfg, epayType, txID, orderName, fmt.Sprintf("%.2f", payAmount), notifyURL, returnURL)
		if err != nil {
			utils.InternalError(c, "创建支付订单失败: "+err.Error())
			return
		}

		utils.Success(c, gin.H{
			"message":        "支付创建成功",
			"order_no":       order.OrderNo,
			"transaction_id": txID,
			"amount":         payAmount,
			"pay_type":       payConfig.PayType,
			"payment_url":    paymentURL,
		})
		return
	}

	utils.Success(c, gin.H{
		"message":        "支付创建成功",
		"order_no":       order.OrderNo,
		"transaction_id": txID,
		"amount":         payAmount,
		"pay_type":       payConfig.PayType,
	})
}

func GetPaymentStatus(c *gin.Context) {
	id := c.Param("id")
	var tx models.PaymentTransaction
	if err := database.GetDB().First(&tx, id).Error; err != nil {
		utils.NotFound(c, "支付记录不存在")
		return
	}
	utils.Success(c, gin.H{"status": tx.Status})
}

func PaymentNotify(c *gin.Context) {
	payType := c.Param("type")
	db := database.GetDB()

	// EasyPay uses form-encoded GET/POST params, not JSON body
	if payType == "epay" {
		handleEpayNotify(c, db)
		return
	}

	// Read raw callback body
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(400, "fail")
		return
	}
	rawStr := string(rawBody)

	// Find the payment config for this pay type
	var payConfig models.PaymentConfig
	if err := db.Where("pay_type = ? AND status = ?", payType, 1).First(&payConfig).Error; err != nil {
		c.String(400, "unknown payment type")
		return
	}

	// Parse callback data to find transaction/order
	var callbackData map[string]interface{}
	json.Unmarshal(rawBody, &callbackData)

	// Try to find transaction by common callback fields
	var transaction models.PaymentTransaction
	txFound := false

	// Try out_trade_no (common in Alipay/WeChat)
	if outTradeNo, ok := callbackData["out_trade_no"].(string); ok && outTradeNo != "" {
		if err := db.Where("transaction_id = ?", outTradeNo).First(&transaction).Error; err == nil {
			txFound = true
		}
	}

	// Try order_no field
	if !txFound {
		if orderNo, ok := callbackData["order_no"].(string); ok && orderNo != "" {
			var order models.Order
			if err := db.Where("order_no = ?", orderNo).First(&order).Error; err == nil {
				if err := db.Where("order_id = ?", order.ID).First(&transaction).Error; err == nil {
					txFound = true
				}
			}
		}
	}

	// Log the callback regardless
	callback := models.PaymentCallback{
		CallbackType: payType,
		CallbackData: string(rawBody),
		RawRequest:   &rawStr,
		Processed:    txFound,
	}
	if txFound {
		callback.PaymentTransactionID = transaction.ID
	}

	if txFound && transaction.Status == "pending" {
		// Mark transaction as paid
		extTxID := ""
		if v, ok := callbackData["trade_no"].(string); ok {
			extTxID = v
		}
		callbackJSON := string(rawBody)
		updates := map[string]interface{}{
			"status":        "paid",
			"callback_data": &callbackJSON,
		}
		if extTxID != "" {
			updates["external_transaction_id"] = &extTxID
		}
		db.Model(&transaction).Updates(updates)

		// Mark order as paid and activate subscription
		var order models.Order
		if err := db.First(&order, transaction.OrderID).Error; err == nil && order.Status == "pending" {
			now := time.Now()
			pmName := payType
			db.Model(&order).Updates(map[string]interface{}{
				"status":              "paid",
				"payment_method_name": &pmName,
				"payment_time":        &now,
			})

			// Activate/extend subscription
			var pkg models.Package
			if err := db.First(&pkg, order.PackageID).Error; err == nil {
				var sub models.Subscription
				if err := db.Where("user_id = ?", order.UserID).First(&sub).Error; err != nil {
					pkgID := int64(pkg.ID)
					sub = models.Subscription{
						UserID:          order.UserID,
						PackageID:       &pkgID,
						SubscriptionURL: utils.GenerateRandomString(32),
						DeviceLimit:     pkg.DeviceLimit,
						IsActive:        true,
						Status:          "active",
						ExpireTime:      time.Now().AddDate(0, 0, pkg.DurationDays),
					}
					db.Create(&sub)
				} else {
					newExpire := sub.ExpireTime
					if newExpire.Before(time.Now()) {
						newExpire = time.Now()
					}
					newExpire = newExpire.AddDate(0, 0, pkg.DurationDays)
					pkgID := int64(pkg.ID)
					db.Model(&sub).Updates(map[string]interface{}{
						"package_id":   &pkgID,
						"device_limit": pkg.DeviceLimit,
						"expire_time":  newExpire,
						"is_active":    true,
						"status":       "active",
					})
				}
			}
		}

		result := "success"
		callback.ProcessingResult = &result
	}

	db.Create(&callback)
	c.String(200, "success")
}

func handleEpayNotify(c *gin.Context, db *gorm.DB) {
	// EasyPay sends params via GET query or POST form
	params := make(map[string]string)
	if c.Request.Method == "GET" {
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else {
		c.Request.ParseForm()
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}

	// Get EasyPay config for signature verification
	epayCfg, err := services.GetEpayConfig()
	if err != nil {
		c.String(400, "fail")
		return
	}

	// Verify signature
	if !services.EpayVerifySign(params, epayCfg.SecretKey) {
		c.String(400, "sign error")
		return
	}

	// Check trade status
	tradeStatus := params["trade_status"]
	if tradeStatus != "TRADE_SUCCESS" {
		c.String(200, "success")
		return
	}

	outTradeNo := params["out_trade_no"]
	tradeNo := params["trade_no"]

	// Log raw callback
	rawJSON, _ := json.Marshal(params)
	rawStr := string(rawJSON)

	// Find transaction
	var transaction models.PaymentTransaction
	if err := db.Where("transaction_id = ?", outTradeNo).First(&transaction).Error; err != nil {
		// Log unmatched callback
		callback := models.PaymentCallback{
			CallbackType: "epay",
			CallbackData: rawStr,
			RawRequest:   &rawStr,
			Processed:    false,
		}
		db.Create(&callback)
		c.String(200, "success")
		return
	}

	// Log callback
	callback := models.PaymentCallback{
		PaymentTransactionID: transaction.ID,
		CallbackType:         "epay",
		CallbackData:         rawStr,
		RawRequest:           &rawStr,
		Processed:            true,
	}

	if transaction.Status == "pending" {
		// Mark transaction as paid
		callbackJSON := rawStr
		updates := map[string]interface{}{
			"status":        "paid",
			"callback_data": &callbackJSON,
		}
		if tradeNo != "" {
			updates["external_transaction_id"] = &tradeNo
		}
		db.Model(&transaction).Updates(updates)

		// Mark order as paid and activate subscription
		var order models.Order
		if err := db.First(&order, transaction.OrderID).Error; err == nil && order.Status == "pending" {
			now := time.Now()
			pmName := "epay"
			db.Model(&order).Updates(map[string]interface{}{
				"status":              "paid",
				"payment_method_name": &pmName,
				"payment_time":        &now,
			})

			// Activate/extend subscription
			var pkg models.Package
			if err := db.First(&pkg, order.PackageID).Error; err == nil {
				var sub models.Subscription
				if err := db.Where("user_id = ?", order.UserID).First(&sub).Error; err != nil {
					pkgID := int64(pkg.ID)
					sub = models.Subscription{
						UserID:          order.UserID,
						PackageID:       &pkgID,
						SubscriptionURL: utils.GenerateRandomString(32),
						DeviceLimit:     pkg.DeviceLimit,
						IsActive:        true,
						Status:          "active",
						ExpireTime:      time.Now().AddDate(0, 0, pkg.DurationDays),
					}
					db.Create(&sub)
				} else {
					newExpire := sub.ExpireTime
					if newExpire.Before(time.Now()) {
						newExpire = time.Now()
					}
					newExpire = newExpire.AddDate(0, 0, pkg.DurationDays)
					pkgID := int64(pkg.ID)
					db.Model(&sub).Updates(map[string]interface{}{
						"package_id":   &pkgID,
						"device_limit": pkg.DeviceLimit,
						"expire_time":  newExpire,
						"is_active":    true,
						"status":       "active",
					})
				}
			}
		}

		result := "success"
		callback.ProcessingResult = &result
	}

	db.Create(&callback)
	c.String(200, "success")
}