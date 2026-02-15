package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/smartwalle/alipay/v3"
)

// AlipayConfig holds direct Alipay API configuration
type AlipayConfig struct {
	AppID        string
	PrivateKey   string
	PublicKey    string
	NotifyURL   string
	ReturnURL   string
	IsProduction bool
}

// GetAlipayConfig reads direct Alipay settings from system_configs.
func GetAlipayConfig() (*AlipayConfig, error) {
	db := database.GetDB()
	keys := []string{"pay_alipay_app_id", "pay_alipay_private_key", "pay_alipay_public_key",
		"pay_alipay_notify_url", "pay_alipay_return_url", "pay_alipay_sandbox"}
	var configs []models.SystemConfig
	db.Where("`key` IN ?", keys).Find(&configs)

	m := make(map[string]string)
	for _, c := range configs {
		m[c.Key] = c.Value
	}

	appID := strings.TrimSpace(m["pay_alipay_app_id"])
	if appID == "" {
		return nil, fmt.Errorf("支付宝 AppID 未配置")
	}
	privateKey := strings.TrimSpace(m["pay_alipay_private_key"])
	if privateKey == "" {
		return nil, fmt.Errorf("支付宝应用私钥未配置")
	}
	publicKey := strings.TrimSpace(m["pay_alipay_public_key"])

	isProduction := m["pay_alipay_sandbox"] != "true" && m["pay_alipay_sandbox"] != "1"

	return &AlipayConfig{
		AppID:        appID,
		PrivateKey:   privateKey,
		PublicKey:    publicKey,
		NotifyURL:   strings.TrimSpace(m["pay_alipay_notify_url"]),
		ReturnURL:   strings.TrimSpace(m["pay_alipay_return_url"]),
		IsProduction: isProduction,
	}, nil
}

// IsDirectAlipayConfigured checks if direct Alipay keys are configured (not epay gateway).
func IsDirectAlipayConfigured() bool {
	cfg, err := GetAlipayConfig()
	if err != nil {
		return false
	}
	return cfg.AppID != "" && cfg.PrivateKey != ""
}

// AlipayCreateOrder creates a payment via direct Alipay API.
// notifyURL/returnURL are auto-generated defaults; config overrides take priority.
func AlipayCreateOrder(cfg *AlipayConfig, outTradeNo, subject, amount, notifyURL, returnURL string) (string, error) {
	privateKey := normalizePrivateKey(cfg.PrivateKey)
	if privateKey == "" {
		return "", fmt.Errorf("支付宝私钥格式错误")
	}

	client, err := alipay.New(cfg.AppID, privateKey, cfg.IsProduction)
	if err != nil {
		return "", fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}

	if cfg.PublicKey != "" {
		pubKey := normalizePublicKey(cfg.PublicKey)
		if pubKey != "" {
			if err := client.LoadAliPayPublicKey(pubKey); err != nil {
				log.Printf("[alipay] 加载支付宝公钥失败: %v", err)
			}
		}
	}

	// Config overrides take priority over auto-generated URLs
	if cfg.NotifyURL != "" {
		notifyURL = cfg.NotifyURL
	}
	if cfg.ReturnURL != "" {
		returnURL = cfg.ReturnURL
	}

	log.Printf("[alipay] 创建订单: out_trade_no=%s, amount=%s, notify=%s, return=%s, production=%v",
		outTradeNo, amount, notifyURL, returnURL, cfg.IsProduction)

	// Try TradePreCreate (QR code) first
	preCreate := alipay.TradePreCreate{}
	preCreate.NotifyURL = notifyURL
	preCreate.ReturnURL = returnURL
	preCreate.Subject = subject
	preCreate.OutTradeNo = outTradeNo
	preCreate.TotalAmount = amount

	ctx := context.Background()
	rsp, err := client.TradePreCreate(ctx, preCreate)
	if err == nil && !rsp.IsFailure() && rsp.QRCode != "" {
		log.Printf("[alipay] TradePreCreate成功, QR: %s (订单: %s)", rsp.QRCode, outTradeNo)
		return rsp.QRCode, nil
	}

	// Log the pre-create failure
	if err != nil {
		log.Printf("[alipay] TradePreCreate失败: %v, 尝试页面支付", err)
	} else if rsp.IsFailure() {
		log.Printf("[alipay] TradePreCreate业务失败: Code=%s Msg=%s SubMsg=%s, 尝试页面支付", rsp.Code, rsp.Msg, rsp.SubMsg)
	}

	// Fallback to TradePagePay (redirect)
	pagePay := alipay.TradePagePay{}
	pagePay.NotifyURL = notifyURL
	pagePay.ReturnURL = returnURL
	pagePay.Subject = subject
	pagePay.OutTradeNo = outTradeNo
	pagePay.TotalAmount = amount
	pagePay.ProductCode = "FAST_INSTANT_TRADE_PAY"

	payURL, err := client.TradePagePay(pagePay)
	if err != nil {
		return "", fmt.Errorf("创建支付宝订单失败: %v", err)
	}
	if payURL == nil {
		return "", fmt.Errorf("支付宝返回的支付URL为空")
	}

	log.Printf("[alipay] TradePagePay成功 (订单: %s)", outTradeNo)
	return payURL.String(), nil
}

// AlipayVerifyCallback verifies and parses an Alipay async notification.
func AlipayVerifyCallback(cfg *AlipayConfig, req *http.Request) (*AlipayNotification, error) {
	privateKey := normalizePrivateKey(cfg.PrivateKey)
	if privateKey == "" {
		return nil, fmt.Errorf("支付宝私钥格式错误")
	}

	client, err := alipay.New(cfg.AppID, privateKey, cfg.IsProduction)
	if err != nil {
		return nil, fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}

	if cfg.PublicKey != "" {
		pubKey := normalizePublicKey(cfg.PublicKey)
		if pubKey != "" {
			client.LoadAliPayPublicKey(pubKey)
		}
	}

	notification, err := client.GetTradeNotification(req)
	if err != nil {
		return nil, fmt.Errorf("验证支付宝通知失败: %v", err)
	}

	return &AlipayNotification{
		TradeNo:     notification.TradeNo,
		OutTradeNo:  notification.OutTradeNo,
		TradeStatus: string(notification.TradeStatus),
		TotalAmount: notification.TotalAmount,
		BuyerID:     notification.BuyerId,
	}, nil
}

// AlipayNotification holds parsed Alipay callback data
type AlipayNotification struct {
	TradeNo     string
	OutTradeNo  string
	TradeStatus string
	TotalAmount string
	BuyerID     string
}

// normalizePrivateKey ensures the private key is in proper PEM format
func normalizePrivateKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if strings.Contains(key, "BEGIN") {
		key = strings.ReplaceAll(key, "\r\n", "\n")
		key = strings.ReplaceAll(key, "\r", "\n")
		return key
	}
	// Strip whitespace
	clean := strings.NewReplacer("\n", "", "\r", "", " ", "", "\t", "").Replace(key)
	if len(clean) < 100 {
		return ""
	}
	// Determine key type by prefix
	keyType := "RSA PRIVATE KEY"
	if strings.HasPrefix(clean, "MIIE") {
		keyType = "PRIVATE KEY"
	}
	return formatPEM(clean, keyType)
}

// normalizePublicKey ensures the public key is in proper PEM format
func normalizePublicKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if strings.Contains(key, "BEGIN") {
		key = strings.ReplaceAll(key, "\r\n", "\n")
		key = strings.ReplaceAll(key, "\r", "\n")
		return key
	}
	clean := strings.NewReplacer("\n", "", "\r", "", " ", "", "\t", "").Replace(key)
	if len(clean) < 50 {
		return ""
	}
	return formatPEM(clean, "PUBLIC KEY")
}

// formatPEM wraps a base64 key body into PEM format with 64-char lines
func formatPEM(body, keyType string) string {
	begin := fmt.Sprintf("-----BEGIN %s-----", keyType)
	end := fmt.Sprintf("-----END %s-----", keyType)
	// Remove any existing markers
	body = strings.TrimPrefix(body, begin)
	body = strings.TrimSuffix(body, end)
	body = strings.TrimSpace(body)
	body = strings.NewReplacer("\n", "", "\r", "", " ", "").Replace(body)

	var b strings.Builder
	b.WriteString(begin)
	b.WriteByte('\n')
	for i := 0; i < len(body); i += 64 {
		e := i + 64
		if e > len(body) {
			e = len(body)
		}
		b.WriteString(body[i:e])
		b.WriteByte('\n')
	}
	b.WriteString(end)
	return b.String()
}

// getSiteURL reads site_url from system_configs
func getSiteURL() string {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("`key` IN ?", []string{"site_url", "domain_name"}).Find(&configs)
	var siteURL string
	for _, c := range configs {
		if c.Key == "site_url" && c.Value != "" {
			siteURL = c.Value
			break
		}
		if c.Key == "domain_name" && c.Value != "" && siteURL == "" {
			siteURL = c.Value
		}
	}
	if siteURL != "" && !strings.HasPrefix(siteURL, "http") {
		siteURL = "https://" + siteURL
	}
	return strings.TrimRight(siteURL, "/")
}

// AlipayCreateWapOrder creates a WAP payment for mobile browsers.
func AlipayCreateWapOrder(cfg *AlipayConfig, outTradeNo, subject, amount, notifyURL, returnURL string) (string, error) {
	privateKey := normalizePrivateKey(cfg.PrivateKey)
	if privateKey == "" {
		return "", fmt.Errorf("支付宝私钥格式错误")
	}

	client, err := alipay.New(cfg.AppID, privateKey, cfg.IsProduction)
	if err != nil {
		return "", fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}

	if cfg.PublicKey != "" {
		pubKey := normalizePublicKey(cfg.PublicKey)
		if pubKey != "" {
			if err := client.LoadAliPayPublicKey(pubKey); err != nil {
				log.Printf("[alipay] 加载支付宝公钥失败: %v", err)
			}
		}
	}

	if cfg.NotifyURL != "" {
		notifyURL = cfg.NotifyURL
	}
	if cfg.ReturnURL != "" {
		returnURL = cfg.ReturnURL
	}

	log.Printf("[alipay] 创建WAP订单: out_trade_no=%s, amount=%s", outTradeNo, amount)

	wapPay := alipay.TradeWapPay{}
	wapPay.NotifyURL = notifyURL
	wapPay.ReturnURL = returnURL
	wapPay.Subject = subject
	wapPay.OutTradeNo = outTradeNo
	wapPay.TotalAmount = amount
	wapPay.ProductCode = "QUICK_WAP_WAY"

	payURL, err := client.TradeWapPay(wapPay)
	if err != nil {
		return "", fmt.Errorf("创建WAP支付失败: %v", err)
	}
	if payURL == nil {
		return "", fmt.Errorf("支付宝返回的WAP支付URL为空")
	}

	log.Printf("[alipay] TradeWapPay成功 (订单: %s)", outTradeNo)
	return payURL.String(), nil
}

// BuildPaymentURLs builds notify and return URLs for payment callbacks
func BuildPaymentURLs(payType, orderNo string) (notifyURL, returnURL string) {
	siteURL := getSiteURL()
	apiBase := siteURL
	if apiBase == "" {
		apiBase = "http://localhost:8000"
	}
	notifyURL = apiBase + "/api/v1/payment/notify/" + payType
	returnURL = siteURL + "/payment/return?order_no=" + url.QueryEscape(orderNo)
	return
}

// AlipayRefund refunds a payment via direct Alipay API.
func AlipayRefund(tradeNo, outRequestNo, refundAmount string) error {
	cfg, err := GetAlipayConfig()
	if err != nil {
		return fmt.Errorf("获取支付宝配置失败: %v", err)
	}

	privateKey := normalizePrivateKey(cfg.PrivateKey)
	if privateKey == "" {
		return fmt.Errorf("支付宝私钥格式错误")
	}

	client, err := alipay.New(cfg.AppID, privateKey, cfg.IsProduction)
	if err != nil {
		return fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}

	if cfg.PublicKey != "" {
		pubKey := normalizePublicKey(cfg.PublicKey)
		if pubKey != "" {
			client.LoadAliPayPublicKey(pubKey)
		}
	}

	refund := alipay.TradeRefund{}
	refund.TradeNo = tradeNo
	refund.OutRequestNo = outRequestNo
	refund.RefundAmount = refundAmount
	refund.RefundReason = "管理员退款"

	ctx := context.Background()
	rsp, err := client.TradeRefund(ctx, refund)
	if err != nil {
		return fmt.Errorf("退款请求失败: %v", err)
	}

	if rsp.IsFailure() {
		return fmt.Errorf("退款失败: %s - %s", rsp.Msg, rsp.SubMsg)
	}

	log.Printf("[alipay] 退款成功: trade_no=%s, refund_amount=%s", tradeNo, refundAmount)
	return nil
}
