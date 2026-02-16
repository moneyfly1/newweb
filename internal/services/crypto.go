package services

import (
	"fmt"
	"strings"

	"cboard/v2/internal/utils"
)

// CryptoConfig holds crypto payment configuration
type CryptoConfig struct {
	WalletAddress string
	Network       string // "TRC20", "ERC20"
	Currency      string // "USDT", "USDC"
}

// GetCryptoConfig reads crypto payment config from system_configs
func GetCryptoConfig() (*CryptoConfig, error) {
	m := utils.GetSettings("pay_crypto_wallet_address", "pay_crypto_network", "pay_crypto_currency")

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
