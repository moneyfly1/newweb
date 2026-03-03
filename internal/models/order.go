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

// Package 套餐
type Package struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(100)" json:"name"`
	Description  *string   `gorm:"type:text" json:"description"`
	Price        float64   `gorm:"type:decimal(10,2)" json:"price"`
	DurationDays int       `json:"duration_days"`
	DeviceLimit  int       `gorm:"default:3" json:"device_limit"`
	Features     *string   `gorm:"type:text" json:"features"`
	SortOrder    int       `gorm:"default:1" json:"sort_order"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	IsFeatured   bool      `gorm:"default:false" json:"is_featured"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Package) TableName() string {
	return "packages"
}

// RechargeRecord 充值记录
type RechargeRecord struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	UserID               uint       `gorm:"index" json:"user_id"`
	OrderNo              string     `gorm:"type:varchar(50);uniqueIndex" json:"order_no"`
	Amount               float64    `gorm:"type:decimal(10,2)" json:"amount"`
	Status               string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentMethod        *string    `gorm:"type:varchar(50)" json:"payment_method"`
	PaymentTransactionID *string    `gorm:"type:varchar(100)" json:"payment_transaction_id"`
	PaymentQRCode        *string    `gorm:"type:text" json:"payment_qr_code"`
	PaymentURL           *string    `gorm:"type:text" json:"payment_url"`
	IPAddress            *string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent            *string    `gorm:"type:text" json:"user_agent"`
	PaidAt               *time.Time `json:"paid_at"`
	CreatedAt            time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (RechargeRecord) TableName() string {
	return "recharge_records"
}
