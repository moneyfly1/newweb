package models

import (
	"time"
)

type UserLevel struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	LevelName      string         `gorm:"type:varchar(50);uniqueIndex" json:"level_name"`
	LevelOrder     int            `gorm:"uniqueIndex" json:"level_order"`
	MinConsumption float64        `gorm:"type:decimal(10,2);default:0" json:"min_consumption"`
	DiscountRate   float64        `gorm:"type:decimal(5,2);default:1.0" json:"discount_rate"`
	DeviceLimit    int            `gorm:"default:3" json:"device_limit"`
	Benefits       *string `gorm:"type:text" json:"benefits"`
	IconURL        *string `gorm:"type:varchar(255)" json:"icon_url"`
	Color          string         `gorm:"type:varchar(20);default:'#909399'" json:"color"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserLevel) TableName() string {
	return "user_levels"
}
