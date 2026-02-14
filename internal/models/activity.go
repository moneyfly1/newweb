package models

import (
	"time"
)

type UserActivity struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	ActivityType     string         `gorm:"type:varchar(50);not null" json:"activity_type"`
	Description      *string   `gorm:"type:text" json:"description,omitempty"`
	IPAddress        *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        *string   `gorm:"type:text" json:"user_agent,omitempty"`
	Location         *string   `gorm:"type:varchar(100)" json:"location,omitempty"`
	ActivityMetadata *string   `gorm:"type:text" json:"activity_metadata,omitempty"`
	CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

func (UserActivity) TableName() string { return "user_activities" }

type LoginHistory struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	LoginTime         time.Time      `gorm:"autoCreateTime" json:"login_time"`
	LogoutTime        *time.Time `json:"logout_time,omitempty"`
	IPAddress         *string    `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         *string    `gorm:"type:text" json:"user_agent,omitempty"`
	Location          *string    `gorm:"type:varchar(100)" json:"location,omitempty"`
	DeviceFingerprint *string    `gorm:"type:varchar(255)" json:"device_fingerprint,omitempty"`
	LoginStatus       string     `gorm:"type:varchar(20);default:success" json:"login_status"`
	FailureReason     *string    `gorm:"type:text" json:"failure_reason,omitempty"`
	SessionDuration   *int64     `json:"session_duration,omitempty"`
}

func (LoginHistory) TableName() string { return "login_history" }
