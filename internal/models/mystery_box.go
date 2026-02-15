package models

import "time"

// MysteryBoxPool 盲盒奖池
type MysteryBoxPool struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Name           string     `gorm:"type:varchar(100)" json:"name"`
	Description    *string    `gorm:"type:text" json:"description"`
	Price          float64    `gorm:"type:decimal(10,2)" json:"price"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	SortOrder      int        `gorm:"default:0" json:"sort_order"`
	MinLevel       *uint      `json:"min_level"`
	MinBalance     *float64   `gorm:"type:decimal(10,2)" json:"min_balance"`
	MaxOpensPerDay *int       `json:"max_opens_per_day"`
	MaxOpensTotal  *int       `json:"max_opens_total"`
	StartTime      *time.Time `json:"start_time"`
	EndTime        *time.Time `json:"end_time"`
	Prizes         []MysteryBoxPrize `gorm:"foreignKey:PoolID" json:"prizes"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MysteryBoxPool) TableName() string { return "mystery_box_pools" }

// MysteryBoxPrize 盲盒奖品
type MysteryBoxPrize struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	PoolID    uint    `gorm:"index" json:"pool_id"`
	Name      string  `gorm:"type:varchar(100)" json:"name"`
	Type      string  `gorm:"type:varchar(20)" json:"type"` // balance, coupon, subscription_days, nothing
	Value     float64 `gorm:"type:decimal(10,2)" json:"value"`
	Weight    int     `gorm:"default:1" json:"weight"`
	Stock     *int    `json:"stock"`
	ImageURL  *string `gorm:"type:varchar(255)" json:"image_url"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MysteryBoxPrize) TableName() string { return "mystery_box_prizes" }

// MysteryBoxRecord 盲盒开启记录
type MysteryBoxRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index" json:"user_id"`
	PoolID     uint      `gorm:"index" json:"pool_id"`
	PrizeID    uint      `gorm:"index" json:"prize_id"`
	PrizeName  string    `gorm:"type:varchar(100)" json:"prize_name"`
	PrizeType  string    `gorm:"type:varchar(20)" json:"prize_type"`
	PrizeValue float64   `gorm:"type:decimal(10,2)" json:"prize_value"`
	Cost       float64   `gorm:"type:decimal(10,2)" json:"cost"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MysteryBoxRecord) TableName() string { return "mystery_box_records" }
