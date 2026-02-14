package models

import (
	"time"
)

type PaymentTransaction struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	OrderID               uint           `gorm:"index" json:"order_id"`
	UserID                uint           `gorm:"index" json:"user_id"`
	PaymentMethodID       uint           `gorm:"index" json:"payment_method_id"`
	Amount                float64        `gorm:"type:decimal(10,2)" json:"amount"`
	Currency              string         `gorm:"type:varchar(10);default:'CNY'" json:"currency"`
	TransactionID         *string `gorm:"type:varchar(100);uniqueIndex" json:"transaction_id"`
	ExternalTransactionID *string `gorm:"type:varchar(100)" json:"external_transaction_id"`
	Status                string  `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentData           *string `gorm:"type:json" json:"payment_data"`
	CallbackData          *string `gorm:"type:json" json:"callback_data"`
	CreatedAt             time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}

type PaymentCallback struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	PaymentTransactionID uint           `json:"payment_transaction_id"`
	CallbackType         string         `gorm:"type:varchar(50)" json:"callback_type"`
	CallbackData         string         `gorm:"type:json" json:"callback_data"`
	RawRequest           *string `gorm:"type:text" json:"raw_request"`
	Processed            bool    `gorm:"default:false" json:"processed"`
	ProcessingResult     *string `gorm:"type:varchar(50)" json:"processing_result"`
	ErrorMessage         *string `gorm:"type:text" json:"error_message"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

func (PaymentCallback) TableName() string {
	return "payment_callbacks"
}

type PaymentConfig struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	PayType             string  `gorm:"type:varchar(50)" json:"pay_type"`
	AppID               *string `gorm:"type:text" json:"app_id"`
	MerchantPrivateKey  *string `gorm:"type:text" json:"merchant_private_key"`
	AlipayPublicKey     *string `gorm:"type:text" json:"alipay_public_key"`
	WechatAppID         *string `gorm:"type:text" json:"wechat_app_id"`
	WechatMchID         *string `gorm:"type:text" json:"wechat_mch_id"`
	WechatAPIKey        *string `gorm:"type:text" json:"wechat_api_key"`
	PaypalClientID      *string `gorm:"type:text" json:"paypal_client_id"`
	PaypalSecret        *string `gorm:"type:text" json:"paypal_secret"`
	StripePublishableKey *string `gorm:"type:text" json:"stripe_publishable_key"`
	StripeSecretKey     *string `gorm:"type:text" json:"stripe_secret_key"`
	BankName            *string `gorm:"type:text" json:"bank_name"`
	AccountName         *string `gorm:"type:text" json:"account_name"`
	AccountNumber       *string `gorm:"type:text" json:"account_number"`
	WalletAddress       *string `gorm:"type:text" json:"wallet_address"`
	Status              int     `gorm:"default:1" json:"status"`
	ReturnURL           *string `gorm:"type:text" json:"return_url"`
	NotifyURL           *string `gorm:"type:text" json:"notify_url"`
	SortOrder           int     `gorm:"default:0" json:"sort_order"`
	ConfigJSON          *string `gorm:"type:json" json:"config_json"`
	CreatedAt           time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PaymentConfig) TableName() string {
	return "payment_configs"
}
