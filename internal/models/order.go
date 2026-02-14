package models

import (
	"time"
)

type Order struct {
	ID                   uint            `gorm:"primaryKey" json:"id"`
	OrderNo              string          `gorm:"type:varchar(50);uniqueIndex" json:"order_no"`
	UserID               uint            `gorm:"index" json:"user_id"`
	PackageID            uint            `gorm:"index" json:"package_id"`
	Amount               float64         `gorm:"type:decimal(10,2)" json:"amount"`
	Status               string          `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	PaymentMethodID      *int64     `json:"payment_method_id"`
	PaymentMethodName    *string    `gorm:"type:varchar(100)" json:"payment_method_name"`
	PaymentTime          *time.Time `json:"payment_time"`
	PaymentTransactionID *string    `gorm:"type:varchar(100)" json:"payment_transaction_id"`
	ExpireTime           *time.Time `json:"expire_time"`
	CouponID             *int64     `gorm:"index" json:"coupon_id"`
	DiscountAmount       *float64   `gorm:"type:decimal(10,2);default:0" json:"discount_amount"`
	FinalAmount          *float64   `gorm:"type:decimal(10,2)" json:"final_amount"`
	ExtraData            *string    `gorm:"type:text" json:"extra_data"`
	CreatedAt            time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt            time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Order) TableName() string {
	return "orders"
}
