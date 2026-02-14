package models

import (
	"time"
)

type RedeemCode struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Code      string     `gorm:"type:varchar(32);uniqueIndex;not null" json:"code"`
	Type      string     `gorm:"type:varchar(20);not null" json:"type"`
	Value     float64    `gorm:"type:decimal(10,2);not null" json:"value"`
	PackageID *uint      `json:"package_id,omitempty"`
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	Status    string     `gorm:"type:varchar(20);default:unused" json:"status"`
	MaxUses   int        `gorm:"default:1" json:"max_uses"`
	UsedCount int        `gorm:"default:0" json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedBy uint       `gorm:"not null" json:"created_by"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (RedeemCode) TableName() string { return "redeem_codes" }

type RedeemRecord struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	RedeemCodeID uint           `gorm:"index;not null" json:"redeem_code_id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	Code         string         `gorm:"type:varchar(32);not null" json:"code"`
	Type         string         `gorm:"type:varchar(20);not null" json:"type"`
	Value        float64        `gorm:"type:decimal(10,2);not null" json:"value"`
	IPAddress    *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

func (RedeemRecord) TableName() string { return "redeem_records" }
