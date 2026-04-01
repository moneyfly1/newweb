package services

import (
	"crypto/hmac"
	"crypto/md5" // #nosec G501 -- EasyPay protocol mandates MD5 signing.
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cboard/v2/internal/utils"
)

var epayHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// EpayConfig holds the EasyPay gateway configuration
type EpayConfig struct {
	Gateway    string // e.g. https://pay.example.com
	MerchantID string
	SecretKey  string
}

// GetEpayConfig reads EasyPay gateway settings from system_configs.
func GetEpayConfig() (*EpayConfig, error) {
	m := utils.GetSettings("pay_epay_gateway", "pay_epay_merchant_id", "pay_epay_secret_key")

	if m["pay_epay_gateway"] == "" || m["pay_epay_merchant_id"] == "" || m["pay_epay_secret_key"] == "" {
		return nil, fmt.Errorf("易支付网关未配置")
	}

	return &EpayConfig{
		Gateway:    strings.TrimRight(m["pay_epay_gateway"], "/"),
		MerchantID: m["pay_epay_merchant_id"],
		SecretKey:  m["pay_epay_secret_key"],
	}, nil
}

// EpayCreateOrder calls the EasyPay API to create a payment order
// payType: "alipay" or "wxpay" or "qqpay"
// Returns the payment page URL that the user should be redirected to
func EpayCreateOrder(cfg *EpayConfig, payType, outTradeNo, name string, money string, notifyURL, returnURL string) (string, error) {
	params := map[string]string{
		"pid":          cfg.MerchantID,
		"type":         payType,
		"out_trade_no": outTradeNo,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         name,
		"money":        money,
	}

	// Generate sign
	sign := epaySign(params, cfg.SecretKey)
	params["sign"] = sign
	params["sign_type"] = "MD5"

	// Build the payment page URL (GET redirect mode)
	u, _ := url.Parse(cfg.Gateway + "/submit.php")
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// EpayVerifySign verifies the callback signature from EasyPay
func EpayVerifySign(params map[string]string, secretKey string) bool {
	sign := params["sign"]
	if sign == "" {
		return false
	}
	expected := epaySign(params, secretKey)
	return hmac.Equal([]byte(sign), []byte(expected))
}

// epaySign generates MD5 signature for EasyPay API
// Standard EasyPay sign algorithm: sort params by key, join as key=value&, append secret key, md5
func epaySign(params map[string]string, secretKey string) string {
	// Filter out sign, sign_type, and empty values
	var keys []string
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	str := strings.Join(parts, "&") + secretKey

	// #nosec G401 -- EasyPay requires MD5 as part of its signature specification.
	hash := md5.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// EpayQueryOrder queries order status from EasyPay API (optional, for verification)
func EpayQueryOrder(cfg *EpayConfig, outTradeNo string) (map[string]string, error) {
	params := map[string]string{
		"act":          "order",
		"pid":          cfg.MerchantID,
		"out_trade_no": outTradeNo,
	}
	sign := epaySign(params, cfg.SecretKey)
	params["sign"] = sign
	params["sign_type"] = "MD5"

	u, _ := url.Parse(cfg.Gateway + "/api.php")
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := epayHTTPClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Parse simple key=value response
	result := make(map[string]string)
	for _, line := range strings.Split(string(body), "&") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, nil
}

// EpayGateway 易支付网关实现
type EpayGateway struct {
	config *EpayConfig
}

// NewEpayGateway 创建易支付网关实例
func NewEpayGateway() (*EpayGateway, error) {
	config, err := GetEpayConfig()
	if err != nil {
		return nil, err
	}
	return &EpayGateway{config: config}, nil
}

// GetConfig 获取支付配置
func (g *EpayGateway) GetConfig() (interface{}, error) {
	if g.config == nil {
		config, err := GetEpayConfig()
		if err != nil {
			return nil, err
		}
		g.config = config
	}
	return g.config, nil
}

// IsConfigured 检查是否已配置
func (g *EpayGateway) IsConfigured() bool {
	_, err := GetEpayConfig()
	return err == nil
}

// CreatePayment 创建支付
// 注意：Epay 需要指定支付类型（alipay/wxpay/qqpay）
func (g *EpayGateway) CreatePayment(orderNo string, amount float64, subject, returnURL, notifyURL string) (interface{}, error) {
	if g.config == nil {
		config, err := GetEpayConfig()
		if err != nil {
			return nil, err
		}
		g.config = config
	}

	// 默认使用支付宝
	payType := "alipay"
	amountStr := fmt.Sprintf("%.2f", amount)

	payURL, err := EpayCreateOrder(g.config, payType, orderNo, subject, amountStr, notifyURL, returnURL)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pay_url":  payURL,
		"order_no": orderNo,
		"amount":   amount,
		"pay_type": payType,
	}, nil
}

// CreatePaymentWithType 创建指定类型的支付
func (g *EpayGateway) CreatePaymentWithType(payType, orderNo string, amount float64, subject, returnURL, notifyURL string) (interface{}, error) {
	if g.config == nil {
		config, err := GetEpayConfig()
		if err != nil {
			return nil, err
		}
		g.config = config
	}

	amountStr := fmt.Sprintf("%.2f", amount)
	payURL, err := EpayCreateOrder(g.config, payType, orderNo, subject, amountStr, notifyURL, returnURL)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pay_url":  payURL,
		"order_no": orderNo,
		"amount":   amount,
		"pay_type": payType,
	}, nil
}

// VerifyCallback 验证回调签名
func (g *EpayGateway) VerifyCallback(data map[string]interface{}) bool {
	// Epay 回调验证在 handleEpayNotify 中处理
	// 这里返回 true，实际验证在回调处理函数中
	return true
}

// GetName 获取网关名称
func (g *EpayGateway) GetName() string {
	return "epay"
}

// GetDisplayName 获取显示名称
func (g *EpayGateway) GetDisplayName() string {
	return "易支付"
}

// ValidateConfig 验证配置
func (g *EpayGateway) ValidateConfig() error {
	if g.config == nil {
		return fmt.Errorf("易支付配置未初始化")
	}
	if g.config.Gateway == "" {
		return fmt.Errorf("易支付网关地址未配置")
	}
	if g.config.MerchantID == "" {
		return fmt.Errorf("易支付商户ID未配置")
	}
	if g.config.SecretKey == "" {
		return fmt.Errorf("易支付密钥未配置")
	}
	return nil
}
