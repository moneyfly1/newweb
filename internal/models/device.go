package models

import "time"

type Device struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            *int64    `gorm:"index" json:"user_id"`
	SubscriptionID    uint      `gorm:"index;index:idx_device_lookup,priority:1" json:"subscription_id"`
	DeviceFingerprint string    `gorm:"type:varchar(255);index;index:idx_device_lookup,priority:2" json:"device_fingerprint"`
	DeviceHash        *string   `gorm:"type:varchar(255)" json:"device_hash"`
	DeviceUA          *string   `gorm:"type:varchar(255)" json:"device_ua"`
	DeviceName        *string   `gorm:"type:varchar(100)" json:"device_name"`
	DeviceType        *string   `gorm:"type:varchar(50)" json:"device_type"`
	IPAddress         *string   `gorm:"type:varchar(45)" json:"ip_address"`
	Region            string    `gorm:"type:varchar(100)" json:"region"`
	UserAgent         *string   `gorm:"type:text" json:"user_agent"`
	SoftwareName      *string   `gorm:"type:varchar(100)" json:"software_name"`
	SoftwareVersion   *string   `gorm:"type:varchar(50)" json:"software_version"`
	OSName            *string   `gorm:"type:varchar(50)" json:"os_name"`
	OSVersion         *string   `gorm:"type:varchar(50)" json:"os_version"`
	DeviceModel       *string   `gorm:"type:varchar(100)" json:"device_model"`
	DeviceBrand       *string   `gorm:"type:varchar(50)" json:"device_brand"`
	SubscriptionType  *string   `gorm:"type:varchar(20);index" json:"subscription_type"`
	IsActive          bool      `gorm:"default:true;index;index:idx_device_lookup,priority:3" json:"is_active"`
	IsAllowed         bool      `gorm:"default:true" json:"is_allowed"`
	FirstSeen         *time.Time `json:"first_seen"`
	LastAccess        time.Time  `gorm:"autoCreateTime" json:"last_access"`
	LastSeen          *time.Time `json:"last_seen"`
	AccessCount       int        `gorm:"default:0" json:"access_count"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Device) TableName() string {
	return "devices"
}
