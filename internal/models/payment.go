package models

import (
	"time"

	"gorm.io/gorm"
)

type PaymentTransaction struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	OrderID               uint      `gorm:"index" json:"order_id"`
	UserID                uint      `gorm:"index" json:"user_id"`
	PaymentMethodID       uint      `gorm:"index" json:"payment_method_id"`
	Amount                float64   `gorm:"type:decimal(10,2)" json:"amount"`
	Currency              string    `gorm:"type:varchar(10);default:'CNY'" json:"currency"`
	TransactionID         *string   `gorm:"type:varchar(100);uniqueIndex" json:"transaction_id"`
	ExternalTransactionID *string   `gorm:"type:varchar(100)" json:"external_transaction_id"`
	Status                string    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentData           *string   `gorm:"type:json" json:"payment_data"`
	CallbackData          *string   `gorm:"type:json" json:"callback_data"`
	CreatedAt             time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}

type PaymentCallback struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	PaymentTransactionID uint      `json:"payment_transaction_id"`
	CallbackType         string    `gorm:"type:varchar(50)" json:"callback_type"`
	CallbackData         string    `gorm:"type:json" json:"callback_data"`
	RawRequest           *string   `gorm:"type:text" json:"raw_request"`
	Processed            bool      `gorm:"default:false" json:"processed"`
	ProcessingResult     *string   `gorm:"type:varchar(50)" json:"processing_result"`
	ErrorMessage         *string   `gorm:"type:text" json:"error_message"`
	CreatedAt            time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PaymentCallback) TableName() string {
	return "payment_callbacks"
}

type PaymentConfig struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	PayType              string    `gorm:"type:varchar(50)" json:"pay_type"`
	AppID                *string   `gorm:"type:text" json:"app_id"`
	MerchantPrivateKey   *string   `gorm:"type:text" json:"-"`
	AlipayPublicKey      *string   `gorm:"type:text" json:"-"`
	WechatAppID          *string   `gorm:"type:text" json:"wechat_app_id"`
	WechatMchID          *string   `gorm:"type:text" json:"wechat_mch_id"`
	WechatAPIKey         *string   `gorm:"type:text" json:"-"`
	PaypalClientID       *string   `gorm:"type:text" json:"paypal_client_id"`
	PaypalSecret         *string   `gorm:"type:text" json:"-"`
	StripePublishableKey *string   `gorm:"type:text" json:"stripe_publishable_key"`
	StripeSecretKey      *string   `gorm:"type:text" json:"-"`
	BankName             *string   `gorm:"type:text" json:"bank_name"`
	AccountName          *string   `gorm:"type:text" json:"account_name"`
	AccountNumber        *string   `gorm:"type:text" json:"-"`
	WalletAddress        *string   `gorm:"type:text" json:"wallet_address"`
	Status               int       `gorm:"default:1" json:"status"`
	ReturnURL            *string   `gorm:"type:text" json:"return_url"`
	NotifyURL            *string   `gorm:"type:text" json:"notify_url"`
	SortOrder            int       `gorm:"default:0" json:"sort_order"`
	ConfigJSON           *string   `gorm:"type:json" json:"config_json"`
	CreatedAt            time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PaymentConfig) TableName() string {
	return "payment_configs"
}

// PaymentNonce 支付回调防重放记录
type PaymentNonce struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TransactionID   string    `gorm:"type:varchar(100);uniqueIndex:idx_nonce_tx_type;not null" json:"transaction_id"`
	CallbackType    string    `gorm:"type:varchar(50);uniqueIndex:idx_nonce_tx_type;not null" json:"callback_type"`
	ExternalTradeNo string    `gorm:"type:varchar(100);index" json:"external_trade_no"`
	ProcessedAt     time.Time `gorm:"autoCreateTime;not null" json:"processed_at"`
	ExpiresAt       time.Time `gorm:"not null;index" json:"expires_at"`
}

func (PaymentNonce) TableName() string {
	return "payment_nonces"
}

// IsNonceProcessed 检查 nonce 是否已处理（防重放）
// 注意：不过滤 expires_at，确保已处理的 nonce 永远有效，不受过期时间影响
func IsNonceProcessed(db *gorm.DB, transactionID string, callbackType string) bool {
	var nonce PaymentNonce
	return db.Where("transaction_id = ? AND callback_type = ?",
		transactionID, callbackType).First(&nonce).Error == nil
}

// RecordNonce 记录已处理的 nonce
func RecordNonce(db *gorm.DB, transactionID string, callbackType string, externalTradeNo string) error {
	nonce := PaymentNonce{
		TransactionID:   transactionID,
		CallbackType:    callbackType,
		ExternalTradeNo: externalTradeNo,
		ExpiresAt:       time.Now().Add(30 * 24 * time.Hour), // 30天后才允许清理
	}
	return db.Create(&nonce).Error
}

// CleanExpiredNonces 清理过期的 nonce 记录
func CleanExpiredNonces(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&PaymentNonce{}).Error
}
