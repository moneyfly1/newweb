package services

import (
	"crypto/md5" // #nosec G501 -- CodePay protocol mandates MD5 signing.
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cboard/v2/internal/utils"
)

var codepayHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// CodepayConfig holds the CodePay gateway configuration
type CodepayConfig struct {
	Gateway    string // e.g. https://mzf.akwl.net
	MerchantID string
	SecretKey  string
}

// CodepayResponse represents the mapi.php JSON response
type CodepayResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	TradeNo   string `json:"trade_no"`
	PayURL    string `json:"payurl"`
	QRCode    string `json:"qrcode"`
	URLScheme string `json:"urlscheme"`
	Money     string `json:"money"`
}

// GetCodepayConfig reads CodePay gateway settings from system_configs.
func GetCodepayConfig() (*CodepayConfig, error) {
	m := utils.GetSettings("pay_codepay_gateway", "pay_codepay_merchant_id", "pay_codepay_secret_key")

	if m["pay_codepay_gateway"] == "" || m["pay_codepay_merchant_id"] == "" || m["pay_codepay_secret_key"] == "" {
		return nil, fmt.Errorf("码支付网关未配置")
	}

	gateway := strings.TrimRight(m["pay_codepay_gateway"], "/")

	return &CodepayConfig{
		Gateway:    gateway,
		MerchantID: m["pay_codepay_merchant_id"],
		SecretKey:  m["pay_codepay_secret_key"],
	}, nil
}

// codepaySign generates MD5 signature for CodePay API
// Algorithm: filter empty/sign/sign_type → sort by key ASCII → join as key=value& → append secret key → md5
func codepaySign(params map[string]string, secretKey string) string {
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

	// #nosec G401 -- CodePay requires MD5 as part of its signature specification.
	hash := md5.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// codepayAPIURL builds the mapi.php URL from gateway address
func codepayAPIURL(gateway string) string {
	if strings.HasSuffix(gateway, "/xpay/epay") {
		return gateway + "/mapi.php"
	}
	return gateway + "/xpay/epay/mapi.php"
}

// codepaySubmitURL builds the submit.php URL from gateway address
func codepaySubmitURL(gateway string) string {
	if strings.HasSuffix(gateway, "/xpay/epay") {
		return gateway + "/submit.php"
	}
	return gateway + "/xpay/epay/submit.php"
}

// CodepayCreateOrder creates a payment order via CodePay
// First tries mapi.php for direct QR code, falls back to submit.php page redirect
func CodepayCreateOrder(cfg *CodepayConfig, payType, outTradeNo, name, money, notifyURL, returnURL string) (string, error) {
	params := map[string]string{
		"pid":          cfg.MerchantID,
		"type":         payType,
		"out_trade_no": outTradeNo,
		"name":         name,
		"money":        money,
		"notify_url":   notifyURL,
	}
	if returnURL != "" {
		params["return_url"] = returnURL
	}

	params["sign"] = codepaySign(params, cfg.SecretKey)
	params["sign_type"] = "MD5"

	// Try mapi.php first for direct QR code
	apiURL := codepayAPIURL(cfg.Gateway)
	utils.LogInfo("码支付发起mapi请求: URL=%s, Order=%s, Amount=%s, Type=%s", apiURL, outTradeNo, money, payType)

	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	resp, err := codepayHTTPClient.PostForm(apiURL, formData)
	if err == nil {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil && resp.StatusCode == http.StatusOK {
			respStr := strings.TrimSpace(string(body))
			utils.LogInfo("码支付mapi响应: %s", respStr)

			// Direct URL response
			if strings.HasPrefix(respStr, "http://") || strings.HasPrefix(respStr, "https://") {
				return respStr, nil
			}

			var codepayResp CodepayResponse
			if json.Unmarshal(body, &codepayResp) == nil && codepayResp.Code == 1 {
				utils.LogInfo("码支付mapi返回: code=%d, trade_no=%s, payurl=%s, qrcode=%s, urlscheme=%s",
					codepayResp.Code, codepayResp.TradeNo, codepayResp.PayURL, codepayResp.QRCode, codepayResp.URLScheme)

				if codepayResp.QRCode != "" {
					return codepayResp.QRCode, nil
				}
				if codepayResp.PayURL != "" {
					return codepayResp.PayURL, nil
				}
				if codepayResp.URLScheme != "" {
					return codepayResp.URLScheme, nil
				}
			}
		}
	}

	// Fallback to submit.php page redirect
	utils.LogInfo("码支付mapi未返回支付链接，使用submit.php页面方式: Order=%s", outTradeNo)
	submitURL := codepaySubmitURL(cfg.Gateway)
	submitParams := url.Values{}
	for k, v := range params {
		submitParams.Set(k, v)
	}
	return fmt.Sprintf("%s?%s", submitURL, submitParams.Encode()), nil
}

// CodepayVerifySign verifies the callback signature from CodePay
func CodepayVerifySign(params map[string]string, secretKey string) bool {
	sign := params["sign"]
	if sign == "" {
		return false
	}
	expected := codepaySign(params, secretKey)
	return strings.EqualFold(sign, expected)
}

// CodepayGateway 码支付网关实现
type CodepayGateway struct {
	config *CodepayConfig
}

// NewCodepayGateway 创建码支付网关实例
func NewCodepayGateway() (*CodepayGateway, error) {
	config, err := GetCodepayConfig()
	if err != nil {
		return nil, err
	}
	return &CodepayGateway{config: config}, nil
}

// GetConfig 获取支付配置
func (g *CodepayGateway) GetConfig() (interface{}, error) {
	if g.config == nil {
		config, err := GetCodepayConfig()
		if err != nil {
			return nil, err
		}
		g.config = config
	}
	return g.config, nil
}

// IsConfigured 检查是否已配置
func (g *CodepayGateway) IsConfigured() bool {
	_, err := GetCodepayConfig()
	return err == nil
}

// CreatePayment 创建支付（默认支付宝）
func (g *CodepayGateway) CreatePayment(orderNo string, amount float64, subject, returnURL, notifyURL string) (interface{}, error) {
	return g.CreatePaymentWithType("alipay", orderNo, amount, subject, returnURL, notifyURL)
}

// CreatePaymentWithType 创建指定类型的支付
func (g *CodepayGateway) CreatePaymentWithType(payType, orderNo string, amount float64, subject, returnURL, notifyURL string) (interface{}, error) {
	if g.config == nil {
		config, err := GetCodepayConfig()
		if err != nil {
			return nil, err
		}
		g.config = config
	}

	amountStr := fmt.Sprintf("%.2f", amount)
	payURL, err := CodepayCreateOrder(g.config, payType, orderNo, subject, amountStr, notifyURL, returnURL)
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
func (g *CodepayGateway) VerifyCallback(data map[string]interface{}) bool {
	return true
}

// GetName 获取网关名称
func (g *CodepayGateway) GetName() string {
	return "codepay"
}

// GetDisplayName 获取显示名称
func (g *CodepayGateway) GetDisplayName() string {
	return "码支付"
}

// ValidateConfig 验证配置
func (g *CodepayGateway) ValidateConfig() error {
	if g.config == nil {
		return fmt.Errorf("码支付配置未初始化")
	}
	if g.config.Gateway == "" {
		return fmt.Errorf("码支付网关地址未配置")
	}
	if g.config.MerchantID == "" {
		return fmt.Errorf("码支付商户ID未配置")
	}
	if g.config.SecretKey == "" {
		return fmt.Errorf("码支付密钥未配置")
	}
	return nil
}
