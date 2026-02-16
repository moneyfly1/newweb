package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
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

	// Auto-create PaymentConfig for Stripe if enabled
	stripeConfigured := cfgMap["pay_stripe_secret_key"] != "" && cfgMap["pay_stripe_publishable_key"] != ""
	if isEnabled(cfgMap["pay_stripe_enabled"]) && stripeConfigured {
		if !hasPayType("stripe") {
			pc := models.PaymentConfig{PayType: "stripe", Status: 1, SortOrder: 103}
			db.Create(&pc)
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "stripe", "sort_order": 103})
		}
	}

	// Auto-create PaymentConfig for Crypto USDT if enabled
	cryptoConfigured := cfgMap["pay_crypto_wallet_address"] != ""
	if isEnabled(cfgMap["pay_crypto_enabled"]) && cryptoConfigured {
		if !hasPayType("crypto") {
			pc := models.PaymentConfig{PayType: "crypto", Status: 1, SortOrder: 104}
			db.Create(&pc)
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "crypto", "sort_order": 104})
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
		IsMobile        bool `json:"is_mobile"`
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
					var paymentURL string
					if req.IsMobile {
						// Use WAP payment for mobile
						paymentURL, err = services.AlipayCreateWapOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", payAmount), notifyURL, returnURL)
					} else {
						// Use QR code payment for desktop
						paymentURL, err = services.AlipayCreateOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", payAmount), notifyURL, returnURL)
					}
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

	// Stripe payment
	if payConfig.PayType == "stripe" {
		stripeCfg, err := services.GetStripeConfig()
		if err != nil {
			utils.BadRequest(c, "Stripe 未配置")
			return
		}

		var pkg models.Package
		orderName := "订单-" + order.OrderNo
		if err := db.First(&pkg, order.PackageID).Error; err == nil {
			orderName = pkg.Name
		}

		// Read exchange rate (CNY to USD)
		rate := 7.2
		if r := utils.GetSetting("pay_stripe_exchange_rate"); r != "" {
			if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 {
				rate = parsed
			}
		}
		amountUSD := payAmount / rate
		amountCents := int64(amountUSD * 100)
		if amountCents < 50 {
			amountCents = 50 // Stripe minimum is $0.50
		}

		siteURL := utils.GetSetting("site_url")
		if siteURL == "" {
			siteURL = "http://localhost:8000"
		}
		successURL := siteURL + "/payment/return?order_no=" + order.OrderNo
		cancelURL := siteURL + "/order"

		_, checkoutURL, err := services.StripeCreateCheckoutSession(stripeCfg, txID, orderName, amountCents, "usd", successURL, cancelURL)
		if err != nil {
			utils.InternalError(c, "创建 Stripe 支付失败: "+err.Error())
			return
		}

		utils.Success(c, gin.H{
			"message":        "支付创建成功",
			"order_no":       order.OrderNo,
			"transaction_id": txID,
			"amount":         payAmount,
			"pay_type":       "stripe",
			"payment_url":    checkoutURL,
		})
		return
	}

	// Crypto payment
	if payConfig.PayType == "crypto" {
		cryptoCfg, err := services.GetCryptoConfig()
		if err != nil {
			utils.BadRequest(c, "加密货币支付未配置")
			return
		}

		rate := 7.2
		if r := utils.GetSetting("pay_crypto_exchange_rate"); r != "" {
			if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 {
				rate = parsed
			}
		}
		amountUSDT := payAmount / rate

		utils.Success(c, gin.H{
			"message":        "请转账到以下地址",
			"order_no":       order.OrderNo,
			"transaction_id": txID,
			"amount":         payAmount,
			"pay_type":       "crypto",
			"crypto_info": gin.H{
				"wallet_address": cryptoCfg.WalletAddress,
				"network":        cryptoCfg.Network,
				"currency":       cryptoCfg.Currency,
				"amount_usdt":    fmt.Sprintf("%.2f", amountUSDT),
			},
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
		IsMobile        bool `json:"is_mobile"`
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
					var paymentURL string
					if req.IsMobile {
						// Use WAP payment for mobile
						paymentURL, err = services.AlipayCreateWapOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
					} else {
						// Use QR code payment for desktop
						paymentURL, err = services.AlipayCreateOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
					}
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

	// Stripe recharge payment
	if payConfig.PayType == "stripe" {
		stripeCfg, err := services.GetStripeConfig()
		if err != nil {
			utils.BadRequest(c, "Stripe 未配置")
			return
		}

		orderName := "充值-" + record.OrderNo

		rate := 7.2
		if r := utils.GetSetting("pay_stripe_exchange_rate"); r != "" {
			if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 {
				rate = parsed
			}
		}
		amountUSD := record.Amount / rate
		amountCents := int64(amountUSD * 100)
		if amountCents < 50 {
			amountCents = 50
		}

		siteURL := utils.GetSetting("site_url")
		if siteURL == "" {
			siteURL = "http://localhost:8000"
		}
		successURL := siteURL + "/payment/return?order_no=" + record.OrderNo
		cancelURL := siteURL + "/recharge"

		_, checkoutURL, err := services.StripeCreateCheckoutSession(stripeCfg, txID, orderName, amountCents, "usd", successURL, cancelURL)
		if err != nil {
			utils.InternalError(c, "创建 Stripe 支付失败: "+err.Error())
			return
		}

		db.Model(&record).Update("payment_url", &checkoutURL)

		utils.Success(c, gin.H{
			"message":        "支付创建成功",
			"order_no":       record.OrderNo,
			"transaction_id": txID,
			"amount":         record.Amount,
			"pay_type":       "stripe",
			"payment_url":    checkoutURL,
		})
		return
	}

	// Crypto recharge payment
	if payConfig.PayType == "crypto" {
		cryptoCfg, err := services.GetCryptoConfig()
		if err != nil {
			utils.BadRequest(c, "加密货币支付未配置")
			return
		}

		rate := 7.2
		if r := utils.GetSetting("pay_crypto_exchange_rate"); r != "" {
			if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 {
				rate = parsed
			}
		}
		amountUSDT := record.Amount / rate

		utils.Success(c, gin.H{
			"message":        "请转账到以下地址",
			"order_no":       record.OrderNo,
			"transaction_id": txID,
			"amount":         record.Amount,
			"pay_type":       "crypto",
			"crypto_info": gin.H{
				"wallet_address": cryptoCfg.WalletAddress,
				"network":        cryptoCfg.Network,
				"currency":       cryptoCfg.Currency,
				"amount_usdt":    fmt.Sprintf("%.2f", amountUSDT),
			},
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
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var tx models.PaymentTransaction
	if err := database.GetDB().Where("id = ? AND user_id = ?", id, userID).First(&tx).Error; err != nil {
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

	// Stripe webhook callback
	if payType == "stripe" {
		handleStripeWebhook(c, db)
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
		err := db.Transaction(func(tx *gorm.DB) error {
			// Re-fetch with status check inside transaction to prevent double-spend
			var txn models.PaymentTransaction
			if err := tx.Where("id = ? AND status = ?", transaction.ID, "pending").First(&txn).Error; err != nil {
				return err // Already processed or not found
			}

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
			tx.Model(&txn).Updates(updates)

			// Mark order as paid and activate subscription
			var order models.Order
			if err := tx.First(&order, txn.OrderID).Error; err == nil && order.Status == "pending" {
				now := time.Now()
				pmName := payType
				tx.Model(&order).Updates(map[string]interface{}{
					"status":              "paid",
					"payment_method_name": &pmName,
					"payment_time":        &now,
				})
				services.ActivateSubscription(tx, &order, payType)
			}

			return nil
		})
		if err == nil {
			result := "success"
			callback.ProcessingResult = &result
		}
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
		err := db.Transaction(func(tx *gorm.DB) error {
			// Re-fetch with status check inside transaction to prevent double-spend
			var txn models.PaymentTransaction
			if err := tx.Where("id = ? AND status = ?", transaction.ID, "pending").First(&txn).Error; err != nil {
				return err // Already processed or not found
			}

			// 金额校验
			if callbackMoney != "" {
				expectedAmount := fmt.Sprintf("%.2f", txn.Amount)
				if callbackMoney != expectedAmount {
					return fmt.Errorf("金额不匹配: 期望 %s, 实际 %s", expectedAmount, callbackMoney)
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
			tx.Model(&txn).Updates(updates)

			// Determine if this is a recharge or order payment by transaction ID prefix
			if strings.HasPrefix(outTradeNo, "RCH") {
				handleEpayRechargeCallback(tx, &txn, outTradeNo)
			} else {
				handleEpayOrderCallback(tx, &txn)
			}

			return nil
		})
		if err != nil {
			// Check if it was an amount mismatch (not a duplicate)
			if strings.Contains(err.Error(), "金额不匹配") {
				s := err.Error()
				callback.ProcessingResult = &s
				callback.Processed = false
				db.Create(&callback)
				c.String(200, "success")
				return
			}
			// Already processed (duplicate callback) - just log and return success
		} else {
			result := "success"
			callback.ProcessingResult = &result
		}
	}

	db.Create(&callback)
	c.String(200, "success")
}

func handleEpayRechargeCallback(db *gorm.DB, transaction *models.PaymentTransaction, txID string) {
	// Use a transaction to atomically check status + update balance (prevents double-spend)
	var notifyUser *models.User
	var notifyRecord *models.RechargeRecord

	err := db.Transaction(func(tx *gorm.DB) error {
		var record models.RechargeRecord
		if err := tx.Where("payment_transaction_id = ? AND status = ?", txID, "pending").First(&record).Error; err != nil {
			return err // Already processed or not found
		}

		now := time.Now()
		tx.Model(&record).Updates(map[string]interface{}{
			"status":  "paid",
			"paid_at": &now,
		})

		// Add balance to user atomically
		tx.Model(&models.User{}).Where("id = ?", record.UserID).
			Update("balance", gorm.Expr("balance + ?", record.Amount))

		// Fetch user for notification (after balance update)
		var user models.User
		if tx.First(&user, record.UserID).Error == nil {
			notifyUser = &user
			notifyRecord = &record
		}

		return nil
	})

	if err != nil {
		return // Already processed or not found
	}

	// Send notifications outside the transaction
	if notifyUser != nil && notifyRecord != nil {
		amountStr := fmt.Sprintf("%.2f", notifyRecord.Amount)
		utils.CreateBalanceLogSimple(notifyRecord.UserID, "recharge", notifyRecord.Amount, notifyUser.Balance-notifyRecord.Amount, notifyUser.Balance, nil, fmt.Sprintf("充值到账: %s", notifyRecord.OrderNo))
		emailSubject, emailBody := services.RenderEmail("recharge_success", map[string]string{
			"order_no": notifyRecord.OrderNo, "amount": amountStr,
		})
		go services.QueueEmail(notifyUser.Email, emailSubject, emailBody, "recharge_success")
		go services.NotifyAdmin("recharge_success", map[string]string{
			"username": notifyUser.Username, "order_no": notifyRecord.OrderNo, "amount": amountStr,
		})
	}
}

func handleEpayOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) {
	db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Where("id = ? AND status = ?", transaction.OrderID, "pending").First(&order).Error; err != nil {
			return err // Already processed or not found
		}

		now := time.Now()
		pmName := "epay"
		tx.Model(&order).Updates(map[string]interface{}{
			"status":              "paid",
			"payment_method_name": &pmName,
			"payment_time":        &now,
		})
		services.ActivateSubscription(tx, &order, "epay")
		return nil
	})
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
		err := db.Transaction(func(tx *gorm.DB) error {
			// Re-fetch with status check inside transaction to prevent double-spend
			var txn models.PaymentTransaction
			if err := tx.Where("id = ? AND status = ?", transaction.ID, "pending").First(&txn).Error; err != nil {
				return err // Already processed or not found
			}

			// 金额校验
			if notification.TotalAmount != "" {
				expectedAmount := fmt.Sprintf("%.2f", txn.Amount)
				if notification.TotalAmount != expectedAmount {
					return fmt.Errorf("金额不匹配: 期望 %s, 实际 %s", expectedAmount, notification.TotalAmount)
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
			tx.Model(&txn).Updates(updates)

			// Determine if this is a recharge or order payment
			if strings.HasPrefix(outTradeNo, "RCH") {
				handleEpayRechargeCallback(tx, &txn, outTradeNo)
			} else {
				handleAlipayOrderCallback(tx, &txn)
			}

			return nil
		})
		if err != nil {
			if strings.Contains(err.Error(), "金额不匹配") {
				s := err.Error()
				callback.ProcessingResult = &s
				callback.Processed = false
				db.Create(&callback)
				c.String(200, "success")
				return
			}
		} else {
			result := "success"
			callback.ProcessingResult = &result
		}
	}

	db.Create(&callback)
	c.String(200, "success")
}

func handleAlipayOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) {
	db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Where("id = ? AND status = ?", transaction.OrderID, "pending").First(&order).Error; err != nil {
			return err // Already processed or not found
		}

		now := time.Now()
		pmName := "alipay"
		tx.Model(&order).Updates(map[string]interface{}{
			"status":              "paid",
			"payment_method_name": &pmName,
			"payment_time":        &now,
		})
		services.ActivateSubscription(tx, &order, "alipay")
		return nil
	})
}

func handleStripeWebhook(c *gin.Context, db *gorm.DB) {
	// Read raw body
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(400, "fail")
		return
	}

	// Get Stripe config for webhook secret
	stripeCfg, err := services.GetStripeConfig()
	if err != nil {
		c.String(400, "stripe not configured")
		return
	}

	// Verify webhook signature (required)
	if stripeCfg.WebhookSecret == "" {
		fmt.Printf("[stripe] webhook secret 未配置，拒绝处理\n")
		c.String(400, "webhook secret not configured")
		return
	}
	sigHeader := c.GetHeader("Stripe-Signature")
	if !services.StripeVerifyWebhook(rawBody, sigHeader, stripeCfg.WebhookSecret) {
		fmt.Printf("[stripe] webhook 签名验证失败\n")
		c.String(400, "signature verification failed")
		return
	}

	// Parse event JSON
	var event map[string]interface{}
	if err := json.Unmarshal(rawBody, &event); err != nil {
		c.String(400, "invalid json")
		return
	}

	eventType, _ := event["type"].(string)
	rawStr := string(rawBody)

	// Only handle checkout.session.completed
	if eventType != "checkout.session.completed" {
		c.String(200, "ok")
		return
	}

	// Extract session data
	data, _ := event["data"].(map[string]interface{})
	if data == nil {
		c.String(200, "ok")
		return
	}
	obj, _ := data["object"].(map[string]interface{})
	if obj == nil {
		c.String(200, "ok")
		return
	}

	// Get transaction_id from metadata
	metadata, _ := obj["metadata"].(map[string]interface{})
	txIDVal, _ := metadata["transaction_id"].(string)
	if txIDVal == "" {
		// Log unmatched callback
		callback := models.PaymentCallback{
			CallbackType: "stripe",
			CallbackData: rawStr,
			RawRequest:   &rawStr,
			Processed:    false,
		}
		db.Create(&callback)
		c.String(200, "ok")
		return
	}

	// Find transaction
	var transaction models.PaymentTransaction
	if err := db.Where("transaction_id = ?", txIDVal).First(&transaction).Error; err != nil {
		callback := models.PaymentCallback{
			CallbackType: "stripe",
			CallbackData: rawStr,
			RawRequest:   &rawStr,
			Processed:    false,
		}
		db.Create(&callback)
		c.String(200, "ok")
		return
	}

	// Log callback
	callback := models.PaymentCallback{
		PaymentTransactionID: transaction.ID,
		CallbackType:         "stripe",
		CallbackData:         rawStr,
		RawRequest:           &rawStr,
		Processed:            true,
	}

	if transaction.Status == "pending" {
		err := db.Transaction(func(tx *gorm.DB) error {
			// Re-fetch with status check inside transaction to prevent double-spend
			var txn models.PaymentTransaction
			if err := tx.Where("id = ? AND status = ?", transaction.ID, "pending").First(&txn).Error; err != nil {
				return err // Already processed or not found
			}

			// 金额校验: Stripe amount_total 单位为分
			if amountTotal, ok := obj["amount_total"].(float64); ok {
				expectedCents := int64(txn.Amount * 100)
				actualCents := int64(amountTotal)
				if actualCents != expectedCents {
					return fmt.Errorf("金额不匹配: 期望 %d, 实际 %d (分)", expectedCents, actualCents)
				}
			}

			// Extract Stripe payment intent ID as external transaction ID
			paymentIntent, _ := obj["payment_intent"].(string)
			sessionID, _ := obj["id"].(string)
			extTxID := paymentIntent
			if extTxID == "" {
				extTxID = sessionID
			}

			callbackJSON := rawStr
			updates := map[string]interface{}{
				"status":        "paid",
				"callback_data": &callbackJSON,
			}
			if extTxID != "" {
				updates["external_transaction_id"] = &extTxID
			}
			tx.Model(&txn).Updates(updates)

			// Determine if this is a recharge or order payment
			if strings.HasPrefix(txIDVal, "RCH") {
				handleStripeRechargeCallback(tx, &txn, txIDVal)
			} else {
				handleStripeOrderCallback(tx, &txn)
			}

			return nil
		})
		if err == nil {
			result := "success"
			callback.ProcessingResult = &result
		}
	}

	db.Create(&callback)
	c.String(200, "ok")
}

func handleStripeOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) {
	db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Where("id = ? AND status = ?", transaction.OrderID, "pending").First(&order).Error; err != nil {
			return err // Already processed or not found
		}

		now := time.Now()
		pmName := "stripe"
		tx.Model(&order).Updates(map[string]interface{}{
			"status":              "paid",
			"payment_method_name": &pmName,
			"payment_time":        &now,
		})
		services.ActivateSubscription(tx, &order, "stripe")
		return nil
	})
}

func handleStripeRechargeCallback(db *gorm.DB, transaction *models.PaymentTransaction, txID string) {
	// Reuse the same recharge callback logic as epay
	handleEpayRechargeCallback(db, transaction, txID)
}