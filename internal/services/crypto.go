package services

import (
	"fmt"
	"strings"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
)

// CryptoConfig holds crypto payment configuration
type CryptoConfig struct {
	WalletAddress string
	Network       string // "TRC20", "ERC20"
	Currency      string // "USDT", "USDC"
}

// GetCryptoConfig reads crypto payment config from system_configs
func GetCryptoConfig() (*CryptoConfig, error) {
	db := database.GetDB()
	keys := []string{"pay_crypto_wallet_address", "pay_crypto_network", "pay_crypto_currency"}
	var configs []models.SystemConfig
	db.Where("`key` IN ?", keys).Find(&configs)

	m := make(map[string]string)
	for _, c := range configs {
		m[c.Key] = c.Value
	}

	if strings.TrimSpace(m["pay_crypto_wallet_address"]) == "" {
		return nil, fmt.Errorf("加密货币钱包地址未配置")
	}

	network := strings.TrimSpace(m["pay_crypto_network"])
	if network == "" {
		network = "TRC20"
	}
	currency := strings.TrimSpace(m["pay_crypto_currency"])
	if currency == "" {
		currency = "USDT"
	}

	return &CryptoConfig{
		WalletAddress: strings.TrimSpace(m["pay_crypto_wallet_address"]),
		Network:       network,
		Currency:      currency,
	}, nil
}

// IsCryptoConfigured checks if crypto wallet address is set
func IsCryptoConfigured() bool {
	cfg, err := GetCryptoConfig()
	if err != nil {
		return false
	}
	return cfg.WalletAddress != ""
}
