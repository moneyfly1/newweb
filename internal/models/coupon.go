package models

import (
	"time"
)

const (
	CouponStatusActive   = "active"
	CouponStatusInactive = "inactive"
	CouponStatusExpired  = "expired"

	CouponTypeDiscount = "discount"
	CouponTypeFixed    = "fixed"
	CouponTypeFreeDays = "free_days"
)

type Coupon struct {
	ID                 uint            `gorm:"primaryKey" json:"id"`
	Code               string          `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	Name               string          `gorm:"type:varchar(100)" json:"name"`
	Description        string          `gorm:"type:text" json:"description"`
	Type               string          `gorm:"type:varchar(20)" json:"type"`
	DiscountValue      float64    `gorm:"type:decimal(10,2)" json:"discount_value"`
	MinAmount          *float64   `gorm:"type:decimal(10,2);default:0" json:"min_amount"`
	MaxDiscount        *float64   `gorm:"type:decimal(10,2)" json:"max_discount"`
	ValidFrom          time.Time  `json:"valid_from"`
	ValidUntil         time.Time  `json:"valid_until"`
	TotalQuantity      *int64     `json:"total_quantity"`
	UsedQuantity       int        `gorm:"default:0" json:"used_quantity"`
	MaxUsesPerUser     int        `gorm:"default:1" json:"max_uses_per_user"`
	Status             string     `gorm:"type:varchar(20);default:'active'" json:"status"`
	ApplicablePackages string     `gorm:"type:text" json:"applicable_packages"`
	CreatedBy          *int64     `gorm:"index" json:"created_by"`
	CreatedAt          time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Coupon) TableName() string {
	return "coupons"
}

type CouponUsage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CouponID       uint      `gorm:"index" json:"coupon_id"`
	UserID         uint      `gorm:"index" json:"user_id"`
	OrderID        *int64    `gorm:"index" json:"order_id"`
	DiscountAmount float64   `gorm:"type:decimal(10,2)" json:"discount_amount"`
	UsedAt         time.Time `gorm:"autoCreateTime" json:"used_at"`
}

func (CouponUsage) TableName() string {
	return "coupon_usages"
}
