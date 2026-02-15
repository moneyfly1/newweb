package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
)

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
}

// GetStripeConfig reads Stripe config from system_configs
func GetStripeConfig() (*StripeConfig, error) {
	db := database.GetDB()
	keys := []string{"pay_stripe_secret_key", "pay_stripe_publishable_key", "pay_stripe_webhook_secret"}
	var configs []models.SystemConfig
	db.Where("`key` IN ?", keys).Find(&configs)

	m := make(map[string]string)
	for _, c := range configs {
		m[c.Key] = c.Value
	}

	if m["pay_stripe_secret_key"] == "" || m["pay_stripe_publishable_key"] == "" {
		return nil, fmt.Errorf("Stripe 未配置")
	}

	return &StripeConfig{
		SecretKey:      strings.TrimSpace(m["pay_stripe_secret_key"]),
		PublishableKey: strings.TrimSpace(m["pay_stripe_publishable_key"]),
		WebhookSecret:  strings.TrimSpace(m["pay_stripe_webhook_secret"]),
	}, nil
}

// IsStripeConfigured checks if Stripe keys are set
func IsStripeConfigured() bool {
	cfg, err := GetStripeConfig()
	if err != nil {
		return false
	}
	return cfg.SecretKey != "" && cfg.PublishableKey != ""
}

// StripeCreateCheckoutSession creates a Stripe Checkout Session via HTTP API.
// POST https://api.stripe.com/v1/checkout/sessions
// Uses Basic auth with secret key as username, empty password.
func StripeCreateCheckoutSession(cfg *StripeConfig, txID, orderName string, amountCents int64, currency, successURL, cancelURL string) (sessionID string, checkoutURL string, err error) {
	data := url.Values{}
	data.Set("mode", "payment")
	data.Set("success_url", successURL)
	data.Set("cancel_url", cancelURL)
	data.Set("line_items[0][price_data][currency]", currency)
	data.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(amountCents, 10))
	data.Set("line_items[0][price_data][product_data][name]", orderName)
	data.Set("line_items[0][quantity]", "1")
	data.Set("metadata[transaction_id]", txID)

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.SetBasicAuth(cfg.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求 Stripe API 失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("Stripe API 返回错误 (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("解析 Stripe 响应失败: %v", err)
	}

	sid, _ := result["id"].(string)
	curl, _ := result["url"].(string)
	if sid == "" || curl == "" {
		return "", "", fmt.Errorf("Stripe 返回数据不完整")
	}

	return sid, curl, nil
}

// StripeVerifyWebhook verifies Stripe webhook signature.
// Stripe-Signature header format: t=timestamp,v1=signature
// Signed payload: "{timestamp}.{payload}"
// HMAC-SHA256 with webhook secret
func StripeVerifyWebhook(payload []byte, sigHeader, webhookSecret string) bool {
	if sigHeader == "" || webhookSecret == "" {
		return false
	}

	var timestamp, signature string
	parts := strings.Split(sigHeader, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signature = kv[1]
		}
	}

	if timestamp == "" || signature == "" {
		return false
	}

	// Check timestamp is not too old (5 minutes tolerance)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return false
	}

	// Compute expected signature
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}
