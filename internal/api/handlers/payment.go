package handlers

import (
	"encoding/json"
	"fmt"
	"io"
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
	methods := make([]gin.H, 0, len(configs))
	for _, cfg := range configs {
		methods = append(methods, gin.H{"id": cfg.ID, "pay_type": cfg.PayType, "sort_order": cfg.SortOrder})
	}

	// Read all payment-related system_configs
	var sysConfigs []models.SystemConfig
	db.Where("`key` LIKE ?", "pay_%").Find(&sysConfigs)
	cfgMap := make(map[string]string)
	for _, sc := range sysConfigs {
		cfgMap[sc.Key] = sc.Value
	}

	// Check balance enabled (default true)
	balanceEnabled := cfgMap["pay_balance_enabled"] != "false" && cfgMap["pay_balance_enabled"] != "0"

	// Helper: check if a pay_type already exists in methods
	hasPayType := func(payType string) bool {
		for _, m := range methods {
			if m["pay_type"] == payType {
				return true
			}
		}
		return false
	}

	// Check if epay gateway is configured (required for all online payment types)
	epayConfigured := cfgMap["pay_epay_gateway"] != "" && cfgMap["pay_epay_merchant_id"] != "" && cfgMap["pay_epay_secret_key"] != ""

	// Auto-create PaymentConfig for EasyPay if enabled
	if isEnabled(cfgMap["pay_epay_enabled"]) && epayConfigured {
		if !hasPayType("epay") {
			pc := models.PaymentConfig{PayType: "epay", Status: 1, SortOrder: 100}
			db.Create(&pc)
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "epay", "sort_order": 100})
		}
	}

	// Auto-create PaymentConfig for Alipay if enabled (epay gateway OR direct Alipay keys)
	alipayDirectConfigured := cfgMap["pay_alipay_app_id"] != "" && cfgMap["pay_alipay_private_key"] != ""
	if isEnabled(cfgMap["pay_alipay_enabled"]) && (epayConfigured || alipayDirectConfigured) {
		if !hasPayType("alipay") {
			pc := models.PaymentConfig{PayType: "alipay", Status: 1, SortOrder: 101}
			db.Create(&pc)
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "alipay", "sort_order": 101})
		}
	}

	// Auto-create PaymentConfig for WeChat Pay if enabled (routes through epay gateway)
	if isEnabled(cfgMap["pay_wechat_enabled"]) && epayConfigured {
		if !hasPayType("wxpay") {
			pc := models.PaymentConfig{PayType: "wxpay", Status: 1, SortOrder: 102}
			db.Create(&pc)
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "wxpay", "sort_order": 102})
		}
	}

	utils.Success(c, gin.H{
		"methods":         methods,
		"balance_enabled": balanceEnabled,
	})
}

func isEnabled(val string) bool {
	return val == "true" || val == "1"
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
	userID := c.GetUint("user_id")
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

	// If this is an EasyPay payment type, call the EasyPay gateway or direct Alipay
	if payConfig.PayType == "epay" || payConfig.PayType == "alipay" || payConfig.PayType == "wxpay" || payConfig.PayType == "qqpay" {
		// Get package name for the order description
		var pkg models.Package
		orderName := "订单-" + order.OrderNo
		if err := db.First(&pkg, order.PackageID).Error; err == nil {
			orderName = pkg.Name
		}

		// Try direct Alipay first if pay_type is alipay and direct keys are configured
		if payConfig.PayType == "alipay" {
			if services.IsDirectAlipayConfigured() {
				alipayCfg, err := services.GetAlipayConfig()
				if err == nil {
					notifyURL, returnURL := services.BuildPaymentURLs("alipay", order.OrderNo)
					paymentURL, err := services.AlipayCreateOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", payAmount), notifyURL, returnURL)
					if err == nil {
						utils.Success(c, gin.H{
							"message":        "支付创建成功",
							"order_no":       order.OrderNo,
							"transaction_id": txID,
							"amount":         payAmount,
							"pay_type":       "alipay",
							"payment_url":    paymentURL,
						})
						return
					}
					fmt.Printf("[payment] 直接支付宝失败: %v, 尝试易支付\n", err)
				}
			}
		}

		// Fall back to epay gateway
		epayCfg, err := services.GetEpayConfig()
		if err != nil {
			// Neither direct Alipay nor epay is available
			if payConfig.PayType == "alipay" {
				utils.BadRequest(c, "支付宝配置不完整，请检查 AppID、私钥等配置")
			} else {
				utils.BadRequest(c, "易支付网关未配置，请在系统设置中配置")
			}
			return
		}

		// Determine the epay pay type
		epayType := payConfig.PayType
		if epayType == "epay" {
			epayType = "alipay" // default to alipay
		}

		notifyURL, returnURL := services.BuildPaymentURLs("epay", order.OrderNo)

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

func CreateRechargePayment(c *gin.Context) {
	var req struct {
		RechargeID      uint `json:"recharge_id" binding:"required"`
		PaymentMethodID uint `json:"payment_method_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	userID := c.GetUint("user_id")
	db := database.GetDB()

	// Find pending recharge record owned by user
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ? AND status = ?", req.RechargeID, userID, "pending").First(&record).Error; err != nil {
		utils.NotFound(c, "充值记录不存在或状态不正确")
		return
	}

	// Verify payment method
	var payConfig models.PaymentConfig
	if err := db.Where("id = ? AND status = ?", req.PaymentMethodID, 1).First(&payConfig).Error; err != nil {
		utils.NotFound(c, "支付方式不存在或未启用")
		return
	}

	// Create payment transaction (OrderID=0 for recharge, store recharge ID in PaymentData)
	txID := fmt.Sprintf("RCH%d%s", time.Now().Unix(), utils.GenerateRandomString(8))
	paymentData := fmt.Sprintf(`{"recharge_id":%d}`, record.ID)
	transaction := models.PaymentTransaction{
		OrderID:         0,
		UserID:          userID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          record.Amount,
		Currency:        "CNY",
		TransactionID:   &txID,
		Status:          "pending",
		PaymentData:     &paymentData,
	}
	if err := db.Create(&transaction).Error; err != nil {
		utils.InternalError(c, "创建支付交易失败")
		return
	}

	// Update recharge record with payment info
	pmName := payConfig.PayType
	db.Model(&record).Updates(map[string]interface{}{
		"payment_method":         &pmName,
		"payment_transaction_id": &txID,
	})

	// Handle epay-type payments
	if payConfig.PayType == "epay" || payConfig.PayType == "alipay" || payConfig.PayType == "wxpay" || payConfig.PayType == "qqpay" {
		orderName := "充值-" + record.OrderNo

		// Try direct Alipay first
		if payConfig.PayType == "alipay" {
			if services.IsDirectAlipayConfigured() {
				alipayCfg, err := services.GetAlipayConfig()
				if err == nil {
					notifyURL, returnURL := services.BuildPaymentURLs("alipay", record.OrderNo)
					paymentURL, err := services.AlipayCreateOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
					if err == nil {
						db.Model(&record).Update("payment_url", &paymentURL)
						utils.Success(c, gin.H{
							"message":        "支付创建成功",
							"order_no":       record.OrderNo,
							"transaction_id": txID,
							"amount":         record.Amount,
							"pay_type":       "alipay",
							"payment_url":    paymentURL,
						})
						return
					}
					fmt.Printf("[payment] 充值直接支付宝失败: %v, 尝试易支付\n", err)
				}
			}
		}

		// Fall back to epay gateway
		epayCfg, err := services.GetEpayConfig()
		if err != nil {
			if payConfig.PayType == "alipay" {
				utils.BadRequest(c, "支付宝配置不完整，请检查 AppID、私钥等配置")
			} else {
				utils.BadRequest(c, "易支付网关未配置，请在系统设置中配置")
			}
			return
		}

		epayType := payConfig.PayType
		if epayType == "epay" {
			epayType = "alipay"
		}

		notifyURL, returnURL := services.BuildPaymentURLs("epay", record.OrderNo)

		paymentURL, err := services.EpayCreateOrder(epayCfg, epayType, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
		if err != nil {
			utils.InternalError(c, "创建支付订单失败: "+err.Error())
			return
		}

		// Store payment URL on the recharge record
		db.Model(&record).Update("payment_url", &paymentURL)

		utils.Success(c, gin.H{
			"message":        "支付创建成功",
			"order_no":       record.OrderNo,
			"transaction_id": txID,
			"amount":         record.Amount,
			"pay_type":       payConfig.PayType,
			"payment_url":    paymentURL,
		})
		return
	}

	utils.Success(c, gin.H{
		"message":        "支付创建成功",
		"order_no":       record.OrderNo,
		"transaction_id": txID,
		"amount":         record.Amount,
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

	// Direct Alipay callback
	if payType == "alipay" {
		handleAlipayNotify(c, db)
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
			services.ActivateSubscription(db, &order, payType)
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
	callbackMoney := params["money"]

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
		// 金额校验
		if callbackMoney != "" {
			expectedAmount := fmt.Sprintf("%.2f", transaction.Amount)
			if callbackMoney != expectedAmount {
				s := fmt.Sprintf("金额不匹配: 期望 %s, 实际 %s", expectedAmount, callbackMoney)
				callback.ProcessingResult = &s
				callback.Processed = false
				db.Create(&callback)
				c.String(200, "success")
				return
			}
		}
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

		// Determine if this is a recharge or order payment by transaction ID prefix
		if strings.HasPrefix(outTradeNo, "RCH") {
			// Recharge payment: find recharge record via PaymentData or payment_transaction_id
			handleEpayRechargeCallback(db, &transaction, outTradeNo)
		} else {
			// Order payment: existing logic
			handleEpayOrderCallback(db, &transaction)
		}

		result := "success"
		callback.ProcessingResult = &result
	}

	db.Create(&callback)
	c.String(200, "success")
}

func handleEpayRechargeCallback(db *gorm.DB, transaction *models.PaymentTransaction, txID string) {
	var record models.RechargeRecord
	if err := db.Where("payment_transaction_id = ? AND status = ?", txID, "pending").First(&record).Error; err != nil {
		return
	}

	now := time.Now()
	db.Model(&record).Updates(map[string]interface{}{
		"status":  "paid",
		"paid_at": &now,
	})

	// Add balance to user
	db.Model(&models.User{}).Where("id = ?", record.UserID).
		Update("balance", gorm.Expr("balance + ?", record.Amount))

	// Notify
	var user models.User
	if db.First(&user, record.UserID).Error == nil {
		amountStr := fmt.Sprintf("%.2f", record.Amount)
		utils.CreateBalanceLogSimple(record.UserID, "recharge", record.Amount, user.Balance-record.Amount, user.Balance, nil, fmt.Sprintf("充值到账: %s", record.OrderNo))
		emailSubject, emailBody := services.RenderEmail("recharge_success", map[string]string{
			"order_no": record.OrderNo, "amount": amountStr,
		})
		go services.QueueEmail(user.Email, emailSubject, emailBody, "recharge_success")
		go services.NotifyAdmin("recharge_success", map[string]string{
			"username": user.Username, "order_no": record.OrderNo, "amount": amountStr,
		})
	}
}

func handleEpayOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) {
	var order models.Order
	if err := db.First(&order, transaction.OrderID).Error; err == nil && order.Status == "pending" {
		now := time.Now()
		pmName := "epay"
		db.Model(&order).Updates(map[string]interface{}{
			"status":              "paid",
			"payment_method_name": &pmName,
			"payment_time":        &now,
		})
		services.ActivateSubscription(db, &order, "epay")
	}
}

func handleAlipayNotify(c *gin.Context, db *gorm.DB) {
	alipayCfg, err := services.GetAlipayConfig()
	if err != nil {
		c.String(400, "fail")
		return
	}

	notification, err := services.AlipayVerifyCallback(alipayCfg, c.Request)
	if err != nil {
		fmt.Printf("[alipay] 回调验证失败: %v\n", err)
		c.String(400, "verify fail")
		return
	}

	// Only process successful trades
	if notification.TradeStatus != "TRADE_SUCCESS" && notification.TradeStatus != "TRADE_FINISHED" {
		c.String(200, "success")
		return
	}

	outTradeNo := notification.OutTradeNo
	tradeNo := notification.TradeNo

	// Log raw callback
	rawStr := fmt.Sprintf(`{"out_trade_no":"%s","trade_no":"%s","trade_status":"%s","total_amount":"%s"}`,
		outTradeNo, tradeNo, notification.TradeStatus, notification.TotalAmount)

	// Find transaction
	var transaction models.PaymentTransaction
	if err := db.Where("transaction_id = ?", outTradeNo).First(&transaction).Error; err != nil {
		callback := models.PaymentCallback{
			CallbackType: "alipay",
			CallbackData: rawStr,
			RawRequest:   &rawStr,
			Processed:    false,
		}
		db.Create(&callback)
		c.String(200, "success")
		return
	}

	callback := models.PaymentCallback{
		PaymentTransactionID: transaction.ID,
		CallbackType:         "alipay",
		CallbackData:         rawStr,
		RawRequest:           &rawStr,
		Processed:            true,
	}

	if transaction.Status == "pending" {
		// 金额校验
		if notification.TotalAmount != "" {
			expectedAmount := fmt.Sprintf("%.2f", transaction.Amount)
			if notification.TotalAmount != expectedAmount {
				s := fmt.Sprintf("金额不匹配: 期望 %s, 实际 %s", expectedAmount, notification.TotalAmount)
				callback.ProcessingResult = &s
				callback.Processed = false
				db.Create(&callback)
				c.String(200, "success")
				return
			}
		}
		callbackJSON := rawStr
		updates := map[string]interface{}{
			"status":        "paid",
			"callback_data": &callbackJSON,
		}
		if tradeNo != "" {
			updates["external_transaction_id"] = &tradeNo
		}
		db.Model(&transaction).Updates(updates)

		// Determine if this is a recharge or order payment
		if strings.HasPrefix(outTradeNo, "RCH") {
			handleEpayRechargeCallback(db, &transaction, outTradeNo)
		} else {
			handleAlipayOrderCallback(db, &transaction)
		}

		result := "success"
		callback.ProcessingResult = &result
	}

	db.Create(&callback)
	c.String(200, "success")
}

func handleAlipayOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) {
	var order models.Order
	if err := db.First(&order, transaction.OrderID).Error; err != nil || order.Status != "pending" {
		return
	}

	now := time.Now()
	pmName := "alipay"
	db.Model(&order).Updates(map[string]interface{}{
		"status":              "paid",
		"payment_method_name": &pmName,
		"payment_time":        &now,
	})
	services.ActivateSubscription(db, &order, "alipay")
}