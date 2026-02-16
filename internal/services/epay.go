package services

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cboard/v2/internal/utils"
)

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
	return sign == expected
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u.String())
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
