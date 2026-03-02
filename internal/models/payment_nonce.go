package models

import (
	"time"

	"gorm.io/gorm"
)

// PaymentNonce 支付回调防重放记录
type PaymentNonce struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TransactionID   string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"transaction_id"`
	CallbackType    string    `gorm:"type:varchar(50);not null" json:"callback_type"`
	ExternalTradeNo string    `gorm:"type:varchar(100);index" json:"external_trade_no"`
	ProcessedAt     time.Time `gorm:"autoCreateTime;not null" json:"processed_at"`
	ExpiresAt       time.Time `gorm:"not null;index" json:"expires_at"`
}

func (PaymentNonce) TableName() string {
	return "payment_nonces"
}

// IsNonceProcessed 检查 nonce 是否已处理（防重放）
func IsNonceProcessed(db *gorm.DB, transactionID string, callbackType string) bool {
	var nonce PaymentNonce
	return db.Where("transaction_id = ? AND callback_type = ? AND expires_at > ?",
		transactionID, callbackType, time.Now()).First(&nonce).Error == nil
}

// RecordNonce 记录已处理的 nonce
func RecordNonce(db *gorm.DB, transactionID string, callbackType string, externalTradeNo string) error {
	nonce := PaymentNonce{
		TransactionID:   transactionID,
		CallbackType:    callbackType,
		ExternalTradeNo: externalTradeNo,
		ExpiresAt:       time.Now().Add(24 * time.Hour), // 24小时后过期
	}
	return db.Create(&nonce).Error
}

// CleanExpiredNonces 清理过期的 nonce 记录
func CleanExpiredNonces(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&PaymentNonce{}).Error
}
