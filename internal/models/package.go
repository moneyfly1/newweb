package models

import (
	"time"
)

type Package struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"type:varchar(100)" json:"name"`
	Description   *string   `gorm:"type:text" json:"description"`
	Price         float64        `gorm:"type:decimal(10,2)" json:"price"`
	DurationDays  int            `json:"duration_days"`
	DeviceLimit   int            `gorm:"default:3" json:"device_limit"`
	Features      *string   `gorm:"type:text" json:"features"`
	SortOrder     int            `gorm:"default:1" json:"sort_order"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	IsFeatured    bool           `gorm:"default:false" json:"is_featured"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Package) TableName() string {
	return "packages"
}
