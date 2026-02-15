package models

import "time"

type CheckIn struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Amount    float64   `gorm:"type:decimal(10,2)" json:"amount"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (CheckIn) TableName() string {
	return "check_ins"
}
