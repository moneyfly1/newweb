package handlers

import (
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// AdminListPaymentGateways 获取所有支付方式信息
func AdminListPaymentGateways(c *gin.Context) {
	gateways := services.GetAllGatewaysInfo()
	utils.Success(c, gin.H{"gateways": gateways})
}

// AdminGetPaymentGateway 获取单个支付方式详情
func AdminGetPaymentGateway(c *gin.Context) {
	gatewayType := c.Param("type")
	info := services.GetGatewayInfo(gatewayType)
	utils.Success(c, info)
}

// AdminTestPaymentGateway 测试支付方式配置
func AdminTestPaymentGateway(c *gin.Context) {
	gatewayType := c.Param("type")

	gateway, err := services.GetPaymentGateway(gatewayType)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if !gateway.IsConfigured() {
		utils.BadRequest(c, gateway.GetDisplayName()+" 未配置")
		return
	}

	if err := gateway.ValidateConfig(); err != nil {
		utils.BadRequest(c, "配置验证失败: "+err.Error())
		return
	}

	// 获取配置信息（不包含敏感信息）
	config, err := gateway.GetConfig()
	if err != nil {
		utils.BadRequest(c, "获取配置失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"message":     gateway.GetDisplayName() + " 配置有效",
		"gateway":     gateway.GetName(),
		"configured":  true,
		"config_type": getConfigType(config),
	})
}

// getConfigType 获取配置类型（不返回敏感信息）
func getConfigType(config interface{}) string {
	switch config.(type) {
	case *services.AlipayConfig:
		return "alipay"
	case *services.StripeConfig:
		return "stripe"
	case *services.EpayConfig:
		return "epay"
	default:
		return "unknown"
	}
}

// AdminGetAvailableGateways 获取可用的支付方式
func AdminGetAvailableGateways(c *gin.Context) {
	gateways := services.GetAvailableGateways()

	var result []map[string]interface{}
	for _, gateway := range gateways {
		result = append(result, map[string]interface{}{
			"name":         gateway.GetName(),
			"display_name": gateway.GetDisplayName(),
			"configured":   gateway.IsConfigured(),
			"valid":        gateway.ValidateConfig() == nil,
		})
	}

	utils.Success(c, gin.H{
		"gateways": result,
		"count":    len(result),
	})
}
