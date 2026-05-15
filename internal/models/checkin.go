package models

import "time"

// CheckIn 签到记录
type CheckIn struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Amount    float64   `gorm:"type:decimal(10,2)" json:"amount"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (CheckIn) TableName() string {
	return "check_ins"
}

// RedeemCode 兑换码
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

// RedeemRecord 兑换记录
type RedeemRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RedeemCodeID uint      `gorm:"index;not null" json:"redeem_code_id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	Code         string    `gorm:"type:varchar(32);not null" json:"code"`
	Type         string    `gorm:"type:varchar(20);not null" json:"type"`
	Value        float64   `gorm:"type:decimal(10,2);not null" json:"value"`
	IPAddress    *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RedeemRecord) TableName() string { return "redeem_records" }
