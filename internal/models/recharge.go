package models

import (
	"time"
)

type RechargeRecord struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	UserID               uint           `gorm:"index" json:"user_id"`
	OrderNo              string         `gorm:"type:varchar(50);uniqueIndex" json:"order_no"`
	Amount               float64        `gorm:"type:decimal(10,2)" json:"amount"`
	Status               string         `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentMethod        *string    `gorm:"type:varchar(50)" json:"payment_method"`
	PaymentTransactionID *string    `gorm:"type:varchar(100)" json:"payment_transaction_id"`
	PaymentQRCode        *string    `gorm:"type:text" json:"payment_qr_code"`
	PaymentURL           *string    `gorm:"type:text" json:"payment_url"`
	IPAddress            *string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent            *string    `gorm:"type:text" json:"user_agent"`
	PaidAt               *time.Time `json:"paid_at"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (RechargeRecord) TableName() string {
	return "recharge_records"
}
