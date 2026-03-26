package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
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

func amountsMatch(expected float64, callbackAmount string) bool {
	if callbackAmount == "" {
		return false
	}
	actual, err := strconv.ParseFloat(strings.TrimSpace(callbackAmount), 64)
	if err != nil {
		return false
	}
	return math.Abs(actual-expected) < 0.005
}

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
			if err := db.Create(&pc).Error; err != nil {
				utils.SysError("payment", fmt.Sprintf("创建支付配置失败(epay): %v", err))
			}
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "epay", "sort_order": 100})
		}
	}

	// Auto-create PaymentConfig for Alipay if enabled (epay gateway OR direct Alipay keys)
	alipayDirectConfigured := cfgMap["pay_alipay_app_id"] != "" && cfgMap["pay_alipay_private_key"] != ""
	if isEnabled(cfgMap["pay_alipay_enabled"]) && (epayConfigured || alipayDirectConfigured) {
		if !hasPayType("alipay") {
			pc := models.PaymentConfig{PayType: "alipay", Status: 1, SortOrder: 101}
			if err := db.Create(&pc).Error; err != nil {
				utils.SysError("payment", fmt.Sprintf("创建支付配置失败(alipay): %v", err))
			}
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "alipay", "sort_order": 101})
		}
	}

	// Auto-create PaymentConfig for WeChat Pay if enabled (routes through epay gateway)
	if isEnabled(cfgMap["pay_wechat_enabled"]) && epayConfigured {
		if !hasPayType("wxpay") {
			pc := models.PaymentConfig{PayType: "wxpay", Status: 1, SortOrder: 102}
			if err := db.Create(&pc).Error; err != nil {
				utils.SysError("payment", fmt.Sprintf("创建支付配置失败(wxpay): %v", err))
			}
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "wxpay", "sort_order": 102})
		}
	}

	// Auto-create PaymentConfig for Stripe if enabled
	stripeConfigured := cfgMap["pay_stripe_secret_key"] != "" && cfgMap["pay_stripe_publishable_key"] != ""
	if isEnabled(cfgMap["pay_stripe_enabled"]) && stripeConfigured {
		if !hasPayType("stripe") {
			pc := models.PaymentConfig{PayType: "stripe", Status: 1, SortOrder: 103}
			if err := db.Create(&pc).Error; err != nil {
				utils.SysError("payment", fmt.Sprintf("创建支付配置失败(stripe): %v", err))
			}
			methods = append(methods, gin.H{"id": pc.ID, "pay_type": "stripe", "sort_order": 103})
		}
	}

	// Auto-create PaymentConfig for Crypto USDT if enabled
	cryptoConfigured := cfgMap["pay_crypto_wallet_address"] != ""
	if isEnabled(cfgMap["pay_crypto_enabled"]) && cryptoConfigured {
		if !hasPayType("crypto") {
			pc := models.PaymentConfig{PayType: "crypto", Status: 1, SortOrder: 104}
			if err := db.Create(&pc).Error; err != nil {
				utils.SysError("payment", fmt.Sprintf("创建支付配置失败(crypto): %v", err))
			}
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
		utils.LogError("[CreatePayment] 参数错误: %v", err)
		utils.BadRequest(c, "参数错误")
		return
	}

	userID := c.GetUint("user_id")
	utils.LogPayment("[CreatePayment] 开始创建支付 - user_id=%d, order_id=%d, payment_method_id=%d, is_mobile=%v",
		userID, req.OrderID, req.PaymentMethodID, req.IsMobile)

	db := database.GetDB()
	var order models.Order
	if err := db.Where("id = ? AND user_id = ? AND status = ?", req.OrderID, userID, "pending").First(&order).Error; err != nil {
		utils.LogError("[CreatePayment] 订单不存在或状态不正确 - order_id=%d, user_id=%d, error=%v", req.OrderID, userID, err)
		utils.NotFound(c, "订单不存在或状态不正确")
		return
	}

	utils.LogPayment("[CreatePayment] 找到订单 - order_no=%s, package_id=%d, amount=%.2f",
		order.OrderNo, order.PackageID, order.Amount)

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
					// 关键修复：使用 transaction.TransactionID (txID) 作为 out_trade_no
					// 这样回调时可以通过 out_trade_no 直接找到 payment_transaction
					outTradeNo := *transaction.TransactionID
					utils.LogPayment("[CreatePayment] 使用 txID 作为 out_trade_no: %s (order_no: %s)", outTradeNo, order.OrderNo)
					if req.IsMobile {
						// Use WAP payment for mobile
						paymentURL, err = services.AlipayCreateWapOrder(alipayCfg, outTradeNo, orderName, fmt.Sprintf("%.2f", payAmount), notifyURL, returnURL)
					} else {
						// Use QR code payment for desktop
						paymentURL, err = services.AlipayCreateOrder(alipayCfg, outTradeNo, orderName, fmt.Sprintf("%.2f", payAmount), notifyURL, returnURL)
					}
					if err == nil {
						// 更新订单的 payment_transaction_id，统一保存交易号字符串
						ptxID := outTradeNo
						if err := db.Model(&order).Update("payment_transaction_id", &ptxID).Error; err != nil {
							utils.InternalError(c, "更新订单支付信息失败")
							return
						}
						utils.LogPayment("[CreatePayment] ✅ 支付宝订单创建成功 - txID=%s, order_no=%s, order_id=%d", outTradeNo, order.OrderNo, order.ID)
						utils.Success(c, gin.H{
							"message":        "支付创建成功",
							"order_no":       order.OrderNo,
							"transaction_id": outTradeNo,
							"amount":         payAmount,
							"pay_type":       "alipay",
							"payment_url":    paymentURL,
						})
						return
					}
					utils.LogError("[payment] 直接支付宝失败: %v, 尝试易支付", err)
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
	if err := db.Model(&record).Updates(map[string]interface{}{
		"payment_method":         &pmName,
		"payment_transaction_id": &txID,
	}).Error; err != nil {
		utils.InternalError(c, "更新充值记录失败")
		return
	}

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
					// 使用 txID 作为 out_trade_no，这样回调时可以通过 out_trade_no 找到 payment_transaction
					outTradeNo := txID
					utils.LogPayment("[CreateRechargePayment] 使用 txID 作为 out_trade_no: %s (order_no: %s)", outTradeNo, record.OrderNo)
					if req.IsMobile {
						// Use WAP payment for mobile
						paymentURL, err = services.AlipayCreateWapOrder(alipayCfg, outTradeNo, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
					} else {
						// Use QR code payment for desktop
						paymentURL, err = services.AlipayCreateOrder(alipayCfg, outTradeNo, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
					}
					if err == nil {
						// 更新充值记录，关联支付事务ID和支付URL
						if err := db.Model(&record).Updates(map[string]interface{}{
							"payment_url":            &paymentURL,
							"payment_transaction_id": &txID,
						}).Error; err != nil {
							utils.InternalError(c, "更新充值支付信息失败")
							return
						}
						utils.LogPayment("[CreateRechargePayment] ✅ 充值支付宝订单创建成功 - txID=%s, order_no=%s", outTradeNo, record.OrderNo)
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
					utils.LogError("[payment] 充值直接支付宝失败: %v, 尝试易支付", err)
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
		if err := db.Model(&record).Update("payment_url", &paymentURL).Error; err != nil {
			utils.InternalError(c, "保存支付链接失败")
			return
		}

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

		if err := db.Model(&record).Update("payment_url", &checkoutURL).Error; err != nil {
			utils.InternalError(c, "保存支付链接失败")
			return
		}

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
	db := database.GetDB()
	var tx models.PaymentTransaction
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&tx).Error; err != nil {
		utils.NotFound(c, "支付记录不存在")
		return
	}
	if tx.Status == "pending" {
		if status, _, err := tryCompensateAlipayPayment(db, &tx, "status_poll"); err != nil {
			utils.LogError("[Alipay] 状态轮询补偿失败: tx_id=%s error=%v", safeTransactionID(tx.TransactionID), err)
		} else {
			tx.Status = status
		}
	}
	utils.Success(c, gin.H{"status": tx.Status})
}

func safeTransactionID(txID *string) string {
	if txID == nil {
		return ""
	}
	return *txID
}

func buildAlipayCallbackPayload(result *services.AlipayTradeQueryResult) string {
	return fmt.Sprintf(`{"out_trade_no":"%s","trade_no":"%s","trade_status":"%s","total_amount":"%s"}`,
		result.OutTradeNo, result.TradeNo, result.TradeStatus, result.TotalAmount)
}

func finalizeAlipayPayment(db *gorm.DB, transaction *models.PaymentTransaction, tradeNo, totalAmount, source string) (string, bool, error) {
	if transaction == nil || transaction.TransactionID == nil || *transaction.TransactionID == "" {
		return "pending", false, fmt.Errorf("支付事务缺少 transaction_id")
	}
	outTradeNo := *transaction.TransactionID

	if models.IsNonceProcessed(db, outTradeNo, "alipay") {
		utils.LogCallback("[Alipay] 幂等命中，已处理: source=%s out_trade_no=%s", source, outTradeNo)
		var latest models.PaymentTransaction
		if err := db.Where("id = ?", transaction.ID).First(&latest).Error; err == nil {
			return latest.Status, false, nil
		}
		return transaction.Status, false, nil
	}

	var finalStatus string
	err := db.Transaction(func(tx *gorm.DB) error {
		var txn models.PaymentTransaction
		if err := tx.Where("id = ?", transaction.ID).First(&txn).Error; err != nil {
			return err
		}
		finalStatus = txn.Status
		if txn.Status != "pending" {
			return nil
		}
		if !amountsMatch(txn.Amount, totalAmount) {
			expectedAmount := fmt.Sprintf("%.2f", txn.Amount)
			return fmt.Errorf("金额不匹配: 期望 %s, 实际 %s", expectedAmount, totalAmount)
		}
		if err := models.RecordNonce(tx, outTradeNo, "alipay", tradeNo); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "unique") {
				return fmt.Errorf("记录 nonce 失败: %w", err)
			}
		}

		callbackJSON := fmt.Sprintf(`{"out_trade_no":"%s","trade_no":"%s","trade_status":"TRADE_SUCCESS","total_amount":"%s","source":"%s"}`,
			outTradeNo, tradeNo, totalAmount, source)
		updates := map[string]interface{}{
			"status":        "paid",
			"callback_data": &callbackJSON,
		}
		if tradeNo != "" {
			updates["external_transaction_id"] = &tradeNo
		}
		if err := tx.Model(&txn).Updates(updates).Error; err != nil {
			return err
		}

		if strings.HasPrefix(outTradeNo, "RCH") {
			if err := handleEpayRechargeCallback(tx, &txn, outTradeNo); err != nil {
				return err
			}
		} else {
			if err := handleAlipayOrderCallback(tx, &txn); err != nil {
				return err
			}
		}
		finalStatus = "paid"
		return nil
	})
	if err != nil {
		return "pending", false, err
	}
	return finalStatus, finalStatus == "paid", nil
}

func tryCompensateAlipayPayment(db *gorm.DB, transaction *models.PaymentTransaction, source string) (string, bool, error) {
	if transaction == nil || transaction.Status != "pending" || transaction.TransactionID == nil || *transaction.TransactionID == "" {
		if transaction == nil {
			return "pending", false, nil
		}
		return transaction.Status, false, nil
	}
	if strings.HasPrefix(*transaction.TransactionID, "RCH") == false && transaction.OrderID == 0 {
		return transaction.Status, false, nil
	}

	cfg, err := services.GetAlipayConfig()
	if err != nil {
		return transaction.Status, false, err
	}

	outTradeNo := *transaction.TransactionID
	utils.LogCallback("[Alipay] 主动查单补偿: source=%s out_trade_no=%s", source, outTradeNo)
	result, err := services.AlipayQueryTrade(cfg, outTradeNo)
	if err != nil {
		return transaction.Status, false, err
	}
	if result.OutTradeNo != "" && result.OutTradeNo != outTradeNo {
		return transaction.Status, false, fmt.Errorf("查单返回的 out_trade_no 不匹配: expected=%s actual=%s", outTradeNo, result.OutTradeNo)
	}
	if result.TradeStatus != "TRADE_SUCCESS" && result.TradeStatus != "TRADE_FINISHED" {
		utils.LogCallback("[Alipay] 主动查单未确认支付成功: source=%s out_trade_no=%s status=%s", source, outTradeNo, result.TradeStatus)
		return transaction.Status, false, nil
	}

	status, compensated, err := finalizeAlipayPayment(db, transaction, result.TradeNo, result.TotalAmount, source)
	if err != nil {
		return transaction.Status, false, err
	}

	callbackJSON := buildAlipayCallbackPayload(result)
	callback := models.PaymentCallback{
		PaymentTransactionID: transaction.ID,
		CallbackType:         "alipay_query_" + source,
		CallbackData:         callbackJSON,
		RawRequest:           &callbackJSON,
		Processed:            compensated || status == "paid",
	}
	resultText := status
	callback.ProcessingResult = &resultText
	if err := db.Create(&callback).Error; err != nil {
		utils.SysError("payment", fmt.Sprintf("保存支付宝补偿查询日志失败: %v", err))
	}

	transaction.Status = status
	return status, compensated, nil
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

	// 限制请求体大小为 10MB，防止 DoS 攻击
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10*1024*1024)

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
	if err := json.Unmarshal(rawBody, &callbackData); err != nil {
		utils.SysError("payment", fmt.Sprintf("解析支付回调数据失败: %v", err))
		c.String(400, "invalid callback data")
		return
	}

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
			if err := tx.Model(&txn).Updates(updates).Error; err != nil {
				return err
			}

			// Mark order as paid and activate subscription
			var order models.Order
			if err := tx.First(&order, txn.OrderID).Error; err == nil && order.Status == "pending" {
				now := time.Now()
				pmName := payType
				if err := tx.Model(&order).Updates(map[string]interface{}{
					"status":              "paid",
					"payment_method_name": &pmName,
					"payment_time":        &now,
				}).Error; err != nil {
					return err
				}
				if err := services.ActivateSubscription(tx, &order, payType); err != nil {
					return err
				}
			}

			return nil
		})
		if err == nil {
			result := "success"
			callback.ProcessingResult = &result
		}
	}

	if err := db.Create(&callback).Error; err != nil {
		utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
	}
	c.String(200, "success")
}

func handleEpayNotify(c *gin.Context, db *gorm.DB) {
	utils.LogCallback("========== 开始处理易支付回调 ==========")
	utils.LogCallback("Method: %s, URL: %s, IP: %s", c.Request.Method, c.Request.URL.String(), utils.GetRealClientIP(c))
	// EasyPay sends params via GET query or POST form
	params := make(map[string]string)
	if c.Request.Method == "GET" {
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else {
		if err := c.Request.ParseForm(); err != nil {
			utils.SysError("payment", fmt.Sprintf("易支付回调解析表单失败: %v", err))
			c.String(400, "fail")
			return
		}
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
		utils.SysError("payment", "易支付回调签名验证失败")
		c.String(400, "sign error")
		return
	}

	// Check trade status
	tradeStatus := params["trade_status"]
	utils.LogCallback("[Epay] trade_status=%s out_trade_no=%s trade_no=%s money=%s",
		tradeStatus, params["out_trade_no"], params["trade_no"], params["money"])
	if tradeStatus != "TRADE_SUCCESS" {
		utils.LogCallback("[Epay] 交易状态非成功，忽略回调")
		c.String(200, "success")
		return
	}

	outTradeNo := params["out_trade_no"]
	tradeNo := params["trade_no"]
	callbackMoney := params["money"]

	// 防重放攻击：检查 nonce
	if models.IsNonceProcessed(db, outTradeNo, "epay") {
		utils.SysError("payment", fmt.Sprintf("易支付回调重放攻击检测: %s", outTradeNo))
		c.String(200, "success") // 返回成功避免重复回调
		return
	}

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
		if err := db.Create(&callback).Error; err != nil {
			utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
		}
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

			// 金额校验（数值比较，兼容 10 / 10.0 / 10.00）
			if callbackMoney != "" {
				utils.LogCallback("[Epay] 金额校验: expected=%.2f actual=%s out_trade_no=%s", txn.Amount, callbackMoney, outTradeNo)
				if !amountsMatch(txn.Amount, callbackMoney) {
					expectedAmount := fmt.Sprintf("%.2f", txn.Amount)
					utils.SysError("payment", fmt.Sprintf("易支付金额不匹配: 订单 %s, 期望 %s, 实际 %s", outTradeNo, expectedAmount, callbackMoney))
					return fmt.Errorf("金额不匹配: 期望 %s, 实际 %s", expectedAmount, callbackMoney)
				}
			} else {
				utils.SysError("payment", fmt.Sprintf("易支付回调缺少金额字段: %s", outTradeNo))
				return fmt.Errorf("回调数据缺少金额字段")
			}

			// 记录 nonce 防止重放
			if err := models.RecordNonce(tx, outTradeNo, "epay", tradeNo); err != nil {
				return fmt.Errorf("记录 nonce 失败: %w", err)
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
			if err := tx.Model(&txn).Updates(updates).Error; err != nil {
				return err
			}
			utils.LogCallback("[Epay] 支付事务已标记为 paid: out_trade_no=%s trade_no=%s", outTradeNo, tradeNo)

			// Determine if this is a recharge or order payment by transaction ID prefix
			if strings.HasPrefix(outTradeNo, "RCH") {
				if err := handleEpayRechargeCallback(tx, &txn, outTradeNo); err != nil {
					return err
				}
			} else {
				if err := handleEpayOrderCallback(tx, &txn); err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			// Check if it was an amount mismatch (not a duplicate)
			if strings.Contains(err.Error(), "金额不匹配") {
				s := err.Error()
				callback.ProcessingResult = &s
				callback.Processed = false
				if err := db.Create(&callback).Error; err != nil {
					utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
				}
				c.String(200, "success")
				return
			}
			// Already processed (duplicate callback) - just log and return success
			utils.LogCallback("[Epay] ⚠️  回调已处理（重复），忽略: out_trade_no=%s", outTradeNo)
		} else {
			result := "success"
			callback.ProcessingResult = &result
			utils.LogCallback("[Epay] ✅✅✅ 易支付回调处理成功: out_trade_no=%s trade_no=%s", outTradeNo, tradeNo)
		}
	} else {
		utils.LogCallback("[Epay] 幂等命中，支付事务已是 %s: out_trade_no=%s", transaction.Status, outTradeNo)
	}

	if err := db.Create(&callback).Error; err != nil {
		utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
	}
	c.String(200, "success")
}

func handleEpayRechargeCallback(db *gorm.DB, transaction *models.PaymentTransaction, txID string) error {
	// Use a transaction to atomically check status + update balance (prevents double-spend)
	var notifyUser *models.User
	var notifyRecord *models.RechargeRecord

	err := db.Transaction(func(tx *gorm.DB) error {
		var record models.RechargeRecord
		if err := tx.Where("payment_transaction_id = ? AND status = ?", txID, "pending").First(&record).Error; err != nil {
			return err // Already processed or not found
		}

		now := time.Now()
		if err := tx.Model(&record).Updates(map[string]interface{}{
			"status":  "paid",
			"paid_at": &now,
		}).Error; err != nil {
			return err
		}

		// Add balance to user atomically
		if err := tx.Model(&models.User{}).Where("id = ?", record.UserID).
			Update("balance", gorm.Expr("balance + ?", record.Amount)).Error; err != nil {
			return fmt.Errorf("充值余额失败: %w", err)
		}

		// Fetch user for notification (after balance update)
		var user models.User
		if tx.First(&user, record.UserID).Error == nil {
			notifyUser = &user
			notifyRecord = &record
		}

		return nil
	})

	if err != nil {
		utils.SysError("payment", fmt.Sprintf("易支付充值回调处理失败: txID=%s, err=%v", txID, err))
		return err
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
	return nil
}

func handleEpayOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Where("id = ? AND status = ?", transaction.OrderID, "pending").First(&order).Error; err != nil {
			return err // Already processed or not found
		}

		now := time.Now()
		pmName := "epay"
		txIDStr := fmt.Sprintf("%d", transaction.ID)
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":                 "paid",
			"payment_method_name":    &pmName,
			"payment_time":           &now,
			"payment_transaction_id": &txIDStr,
		}).Error; err != nil {
			return err
		}
		if err := services.ActivateSubscription(tx, &order, "epay"); err != nil {
			return fmt.Errorf("激活订阅失败: %w", err)
		}
		return nil
	})
	if err != nil {
		utils.SysError("payment", fmt.Sprintf("易支付订单回调处理失败: %v", err))
		return err
	}
	return nil
}

func handleAlipayNotify(c *gin.Context, db *gorm.DB) {
	// Log incoming request
	utils.LogCallback("========== 开始处理支付宝回调 ==========")
	utils.LogCallback("Method: %s", c.Request.Method)
	utils.LogCallback("URL: %s", c.Request.URL.String())
	utils.LogCallback("Remote: %s", c.ClientIP())

	alipayCfg, err := services.GetAlipayConfig()
	if err != nil {
		utils.LogError("[Alipay] ❌ 获取配置失败: %v", err)
		c.String(400, "fail")
		return
	}

	notification, err := services.AlipayVerifyCallback(alipayCfg, c.Request)
	if err != nil {
		utils.LogError("[Alipay] ❌ 回调验证失败: %v", err)
		utils.LogError("[Alipay] 验签上下文: app_id=%s production=%v sandbox_raw=%q notify=%s return=%s public_key=%s",
			alipayCfg.AppID, alipayCfg.IsProduction, alipayCfg.SandboxRaw, alipayCfg.NotifyURL, alipayCfg.ReturnURL, alipayCfg.PublicKeyHint)
		utils.SysError("payment", fmt.Sprintf("支付宝回调验证失败: %v", err))
		c.String(400, "verify fail")
		return
	}

	utils.LogCallback("[Alipay] ✅ 回调验证成功")
	utils.LogCallback("  - out_trade_no: %s", notification.OutTradeNo)
	utils.LogCallback("  - trade_no: %s", notification.TradeNo)
	utils.LogCallback("  - trade_status: %s", notification.TradeStatus)
	utils.LogCallback("  - total_amount: %s", notification.TotalAmount)

	// Only process successful trades
	if notification.TradeStatus != "TRADE_SUCCESS" && notification.TradeStatus != "TRADE_FINISHED" {
		fmt.Printf("[alipay] 交易状态非成功: %s\n", notification.TradeStatus)
		c.String(200, "success")
		return
	}

	outTradeNo := notification.OutTradeNo
	tradeNo := notification.TradeNo

	// 防重放攻击：检查 nonce
	if models.IsNonceProcessed(db, outTradeNo, "alipay") {
		utils.SysError("payment", fmt.Sprintf("支付宝回调重放攻击检测: %s", outTradeNo))
		c.String(200, "success")
		return
	}

	// Log raw callback
	rawStr := fmt.Sprintf(`{"out_trade_no":"%s","trade_no":"%s","trade_status":"%s","total_amount":"%s"}`,
		outTradeNo, tradeNo, notification.TradeStatus, notification.TotalAmount)

	// Find transaction by out_trade_no (which should be txID)
	// First try to find by transaction_id
	var transaction models.PaymentTransaction
	utils.LogCallback("[Alipay] 🔍 查找支付事务 - out_trade_no=%s", outTradeNo)
	err = db.Where("transaction_id = ?", outTradeNo).First(&transaction).Error

	// If not found, try to find by order_no (fallback for old orders)
	if err != nil {
		utils.LogCallback("[Alipay] ⚠️  通过 transaction_id 未找到，尝试通过 order_no 查找")
		var order models.Order
		if err := db.Where("order_no = ?", outTradeNo).First(&order).Error; err != nil {
			utils.LogError("[alipay] ❌ 找不到订单: out_trade_no=%s, error=%v", outTradeNo, err)
			callback := models.PaymentCallback{
				CallbackType: "alipay",
				CallbackData: rawStr,
				RawRequest:   &rawStr,
				Processed:    false,
			}
			errMsg := fmt.Sprintf("找不到订单: %v", err)
			callback.ErrorMessage = &errMsg
			if err := db.Create(&callback).Error; err != nil {
				utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
			}
			c.String(200, "success")
			return
		}

		utils.LogCallback("[Alipay] ✓ 通过 order_no 找到订单: order_id=%d, order_no=%s", order.ID, order.OrderNo)
		// Find transaction by order_id
		if err := db.Where("order_id = ?", order.ID).First(&transaction).Error; err != nil {
			utils.LogError("[alipay] ❌ 找不到支付事务: order_id=%d, error=%v", order.ID, err)
			callback := models.PaymentCallback{
				CallbackType: "alipay",
				CallbackData: rawStr,
				RawRequest:   &rawStr,
				Processed:    false,
			}
			errMsg := fmt.Sprintf("找不到支付事务: %v", err)
			callback.ErrorMessage = &errMsg
			if err := db.Create(&callback).Error; err != nil {
				utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
			}
			c.String(200, "success")
			return
		}
	}

	utils.LogCallback("[Alipay] ✅ 找到支付事务: transaction_id=%d, order_id=%d, status=%s, amount=%.2f",
		transaction.ID, transaction.OrderID, transaction.Status, transaction.Amount)

	callback := models.PaymentCallback{
		PaymentTransactionID: transaction.ID,
		CallbackType:         "alipay",
		CallbackData:         rawStr,
		RawRequest:           &rawStr,
		Processed:            true,
	}

	if transaction.Status == "pending" {
		status, _, finalizeErr := finalizeAlipayPayment(db, &transaction, tradeNo, notification.TotalAmount, "notify")
		if finalizeErr != nil {
			if strings.Contains(finalizeErr.Error(), "金额不匹配") {
				s := finalizeErr.Error()
				callback.ProcessingResult = &s
				callback.Processed = false
				callback.ErrorMessage = &s
				if err := db.Create(&callback).Error; err != nil {
					utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
				}
				c.String(200, "success")
				return
			}
			errMsg := finalizeErr.Error()
			callback.Processed = false
			callback.ErrorMessage = &errMsg
			callback.ProcessingResult = &errMsg
			utils.LogError("[Alipay] ❌ 回调处理失败: out_trade_no=%s trade_no=%s error=%v", outTradeNo, tradeNo, finalizeErr)
		} else {
			transaction.Status = status
			result := "success"
			callback.ProcessingResult = &result
		}
	}

	if err := db.Create(&callback).Error; err != nil {
		utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
	}
	c.String(200, "success")
}

func handleAlipayOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) error {
	utils.LogCallback("[Alipay] 📦 开始处理订单回调 - transaction_id=%d, order_id=%d", transaction.ID, transaction.OrderID)

	// 检查是否为充值订单（order_id = 0）
	if transaction.OrderID == 0 {
		utils.LogCallback("[Alipay] 💰 这是充值订单，调用充值处理逻辑")
		if transaction.TransactionID != nil {
			return handleEpayRechargeCallback(db, transaction, *transaction.TransactionID)
		}
		utils.LogError("[Alipay] ❌ 充值订单缺少 transaction_id")
		return fmt.Errorf("充值订单缺少 transaction_id")
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Where("id = ? AND status = ?", transaction.OrderID, "pending").First(&order).Error; err != nil {
			utils.LogError("[Alipay] ❌ 查找订单失败: order_id=%d, error=%v", transaction.OrderID, err)
			return err // Already processed or not found
		}

		utils.LogCallback("[Alipay] ✓ 找到待支付订单: order_no=%s, user_id=%d, package_id=%d, amount=%.2f",
			order.OrderNo, order.UserID, order.PackageID, order.Amount)

		now := time.Now()
		pmName := "alipay"
		txIDStr := ""
		if transaction.TransactionID != nil {
			txIDStr = *transaction.TransactionID
		}
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":                 "paid",
			"payment_method_name":    &pmName,
			"payment_time":           &now,
			"payment_transaction_id": &txIDStr,
		}).Error; err != nil {
			utils.LogError("[Alipay] ❌ 更新订单状态失败: error=%v", err)
			return err
		}

		utils.LogCallback("[Alipay] ✅ 订单状态已更新为 paid - order_no=%s", order.OrderNo)

		if err := services.ActivateSubscription(tx, &order, "alipay"); err != nil {
			utils.LogError("[Alipay] ❌ 激活订阅失败: error=%v", err)
			return fmt.Errorf("激活订阅失败: %w", err)
		}

		utils.LogCallback("[Alipay] ✅ 订阅激活成功 - order_no=%s", order.OrderNo)
		return nil
	})
	if err != nil {
		utils.SysError("payment", fmt.Sprintf("支付宝订单回调处理失败: %v", err))
		utils.LogError("[Alipay] ❌ 订单回调处理失败: %v", err)
		return err
	}
	utils.LogCallback("[Alipay] ✅✅✅ 订单回调处理完成")
	return nil
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
		if err := db.Create(&callback).Error; err != nil {
			utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
		}
		c.String(200, "ok")
		return
	}

	// 防重放攻击：检查 nonce
	if models.IsNonceProcessed(db, txIDVal, "stripe") {
		utils.SysError("payment", fmt.Sprintf("Stripe 回调重放攻击检测: %s", txIDVal))
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
		if err := db.Create(&callback).Error; err != nil {
			utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
		}
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

			// 金额校验: Stripe amount_total 单位为分，需严格匹配
			if amountTotal, ok := obj["amount_total"].(float64); ok {
				// 将数据库金额（元）转换为分，考虑汇率
				rate := 7.2
				if r := utils.GetSetting("pay_stripe_exchange_rate"); r != "" {
					if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 {
						rate = parsed
					}
				}
				amountUSD := txn.Amount / rate
				expectedCents := int64(math.Round(amountUSD * 100))
				actualCents := int64(amountTotal)

				// 允许 1 分的误差（汇率精度问题）
				if math.Abs(float64(actualCents-expectedCents)) > 1 {
					utils.SysError("payment", fmt.Sprintf("Stripe 金额不匹配: 订单 %s, 期望 %d 分, 实际 %d 分", txIDVal, expectedCents, actualCents))
					return fmt.Errorf("金额不匹配: 期望 %d, 实际 %d (分)", expectedCents, actualCents)
				}
			} else {
				utils.SysError("payment", fmt.Sprintf("Stripe 回调缺少金额字段: %s", txIDVal))
				return fmt.Errorf("回调数据缺少金额字段")
			}

			// 记录 nonce 防止重放
			paymentIntent, _ := obj["payment_intent"].(string)
			sessionID, _ := obj["id"].(string)
			extTxID := paymentIntent
			if extTxID == "" {
				extTxID = sessionID
			}
			if err := models.RecordNonce(tx, txIDVal, "stripe", extTxID); err != nil {
				return fmt.Errorf("记录 nonce 失败: %w", err)
			}

			callbackJSON := rawStr
			updates := map[string]interface{}{
				"status":        "paid",
				"callback_data": &callbackJSON,
			}
			if extTxID != "" {
				updates["external_transaction_id"] = &extTxID
			}
			if err := tx.Model(&txn).Updates(updates).Error; err != nil {
				return err
			}

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

	if err := db.Create(&callback).Error; err != nil {
		utils.SysError("payment", fmt.Sprintf("保存支付回调日志失败: %v", err))
	}
	c.String(200, "ok")
}

func handleStripeOrderCallback(db *gorm.DB, transaction *models.PaymentTransaction) {
	err := db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Where("id = ? AND status = ?", transaction.OrderID, "pending").First(&order).Error; err != nil {
			return err // Already processed or not found
		}

		now := time.Now()
		pmName := "stripe"
		txIDStr := fmt.Sprintf("%d", transaction.ID)
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":                 "paid",
			"payment_method_name":    &pmName,
			"payment_time":           &now,
			"payment_transaction_id": &txIDStr,
		}).Error; err != nil {
			return err
		}
		if err := services.ActivateSubscription(tx, &order, "stripe"); err != nil {
			return fmt.Errorf("激活订阅失败: %w", err)
		}
		return nil
	})
	if err != nil {
		utils.SysError("payment", fmt.Sprintf("Stripe订单回调处理失败: %v", err))
	}
}

func handleStripeRechargeCallback(db *gorm.DB, transaction *models.PaymentTransaction, txID string) {
	// Reuse the same recharge callback logic as epay
	handleEpayRechargeCallback(db, transaction, txID)
}

// PaymentReturn handles synchronous return from payment gateway (e.g., Alipay return_url)
// Redirects to frontend payment result page
func PaymentReturn(c *gin.Context) {
	// Get order_no from query params (支付宝会传递 out_trade_no)
	// out_trade_no 现在是 txID (如 PAY1234567890ABC)，需要查找对应的订单号
	outTradeNo := c.Query("out_trade_no")
	if outTradeNo == "" {
		outTradeNo = c.Query("order_no")
	}

	utils.LogCallback("[PaymentReturn] 同步回调 - out_trade_no=%s, query=%v", outTradeNo, c.Request.URL.Query())

	// 查找订单号
	orderNo := outTradeNo
	if outTradeNo != "" {
		db := database.GetDB()
		// 先尝试通过 transaction_id 查找
		var transaction models.PaymentTransaction
		if err := db.Where("transaction_id = ?", outTradeNo).First(&transaction).Error; err == nil {
			if transaction.Status == "pending" {
				if _, _, err := tryCompensateAlipayPayment(db, &transaction, "sync_return"); err != nil {
					utils.LogError("[Alipay] 同步返回补偿失败: out_trade_no=%s error=%v", outTradeNo, err)
				}
			}
			// 找到支付事务，获取订单号
			if transaction.OrderID > 0 {
				var order models.Order
				if err := db.First(&order, transaction.OrderID).Error; err == nil {
					orderNo = order.OrderNo
					utils.LogCallback("[PaymentReturn] 通过 txID 找到订单: %s -> %s", outTradeNo, orderNo)
				}
			} else {
				// 充值订单，查找充值记录
				var record models.RechargeRecord
				if err := db.Where("payment_transaction_id = ?", outTradeNo).First(&record).Error; err == nil {
					orderNo = record.OrderNo
					utils.LogCallback("[PaymentReturn] 通过 txID 找到充值订单: %s -> %s", outTradeNo, orderNo)
				}
			}
		}
	}

	// Get site URL for redirect
	siteURL := services.GetSiteURL()
	if siteURL == "" {
		siteURL = "http://localhost:8000"
	}

	// Redirect to frontend payment return page
	redirectURL := siteURL + "/payment/return"
	if orderNo != "" {
		redirectURL += "?order_no=" + url.QueryEscape(orderNo)
	}

	utils.LogCallback("[PaymentReturn] 重定向到: %s", redirectURL)
	c.Redirect(302, redirectURL)
}
