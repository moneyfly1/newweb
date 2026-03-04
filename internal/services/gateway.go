package services

import (
	"fmt"

	"cboard/v2/internal/utils"
)

// PaymentGateway 支付网关接口
type PaymentGateway interface {
	// GetConfig 获取支付配置
	GetConfig() (interface{}, error)

	// IsConfigured 检查是否已配置
	IsConfigured() bool

	// CreatePayment 创建支付
	CreatePayment(orderNo string, amount float64, subject, returnURL, notifyURL string) (interface{}, error)

	// VerifyCallback 验证回调签名
	VerifyCallback(data map[string]interface{}) bool

	// GetName 获取网关名称
	GetName() string

	// GetDisplayName 获取显示名称
	GetDisplayName() string

	// ValidateConfig 验证配置
	ValidateConfig() error
}

// BasePaymentConfig 基础支付配置
type BasePaymentConfig struct {
	Enabled      bool
	IsProduction bool
}

// GetPaymentSettings 获取支付配置（公共函数）
func GetPaymentSettings(prefix string, keys ...string) map[string]string {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = prefix + key
	}
	return utils.GetSettings(fullKeys...)
}

// IsPaymentEnabled 检查支付方式是否启用
func IsPaymentEnabled(enabledKey string) bool {
	value := utils.GetSetting(enabledKey)
	return value == "true" || value == "1"
}

// ValidatePaymentConfig 验证支付配置是否完整
func ValidatePaymentConfig(configs map[string]string, requiredKeys []string) error {
	for _, key := range requiredKeys {
		if configs[key] == "" {
			return ErrPaymentNotConfigured
		}
	}
	return nil
}

// ErrPaymentNotConfigured 支付未配置错误
var ErrPaymentNotConfigured = &PaymentError{Code: "NOT_CONFIGURED", Message: "支付方式未配置"}

// PaymentError 支付错误
type PaymentError struct {
	Code    string
	Message string
}

func (e *PaymentError) Error() string {
	return e.Message
}

// GetPaymentGateway 根据类型获取支付网关实例
func GetPaymentGateway(gatewayType string) (PaymentGateway, error) {
	switch gatewayType {
	case "alipay":
		return NewAlipayGateway()
	case "stripe":
		return NewStripeGateway()
	case "epay":
		return NewEpayGateway()
	default:
		return nil, &PaymentError{
			Code:    "UNKNOWN_GATEWAY",
			Message: fmt.Sprintf("未知的支付网关类型: %s", gatewayType),
		}
	}
}

// GetAvailableGateways 获取所有已配置的支付网关
func GetAvailableGateways() []PaymentGateway {
	var gateways []PaymentGateway

	// 尝试初始化所有支付网关
	if gateway, err := NewAlipayGateway(); err == nil && gateway.IsConfigured() {
		gateways = append(gateways, gateway)
	}

	if gateway, err := NewStripeGateway(); err == nil && gateway.IsConfigured() {
		gateways = append(gateways, gateway)
	}

	if gateway, err := NewEpayGateway(); err == nil && gateway.IsConfigured() {
		gateways = append(gateways, gateway)
	}

	return gateways
}

// GetGatewayInfo 获取支付网关信息
func GetGatewayInfo(gatewayType string) map[string]interface{} {
	gateway, err := GetPaymentGateway(gatewayType)
	if err != nil {
		return map[string]interface{}{
			"name":         gatewayType,
			"display_name": gatewayType,
			"configured":   false,
			"error":        err.Error(),
		}
	}

	return map[string]interface{}{
		"name":         gateway.GetName(),
		"display_name": gateway.GetDisplayName(),
		"configured":   gateway.IsConfigured(),
		"valid":        gateway.ValidateConfig() == nil,
	}
}

// GetAllGatewaysInfo 获取所有支付网关信息
func GetAllGatewaysInfo() []map[string]interface{} {
	gatewayTypes := []string{"alipay", "stripe", "epay"}
	var infos []map[string]interface{}

	for _, gatewayType := range gatewayTypes {
		infos = append(infos, GetGatewayInfo(gatewayType))
	}

	return infos
}
