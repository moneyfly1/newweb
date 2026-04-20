package models

import "time"

type Subscription struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"index:idx_user_status" json:"user_id"`
	PackageID         *int64    `gorm:"index" json:"package_id"`
	SubscriptionURL   string    `gorm:"type:varchar(100);uniqueIndex" json:"subscription_url"`
	DeviceLimit       int       `json:"device_limit"`
	CurrentDevices    int       `gorm:"default:0;index:idx_device_check" json:"current_devices"`
	UniversalCount    int       `gorm:"default:0" json:"universal_count"`
	ClashCount        int       `gorm:"default:0" json:"clash_count"`
	SurgeCount        int       `gorm:"default:0" json:"surge_count"`
	QuanXCount        int       `gorm:"default:0" json:"quanx_count"`
	ShadowrocketCount int       `gorm:"default:0" json:"shadowrocket_count"`
	ProtocolFilter    string    `gorm:"type:text" json:"protocol_filter"` // JSON: {"clash":["vmess",...], "universal":["vmess",...]}; empty = use global
	IsActive          bool      `gorm:"default:true;index:idx_active_expire" json:"is_active"`
	Status            string    `gorm:"type:varchar(20);default:'active';index:idx_user_status" json:"status"`
	ExpireTime        time.Time `gorm:"index:idx_active_expire" json:"expire_time"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

type SubscriptionReset struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	UserID             uint      `gorm:"index" json:"user_id"`
	SubscriptionID     uint      `gorm:"index" json:"subscription_id"`
	ResetType          string    `gorm:"type:varchar(50)" json:"reset_type"`
	Reason             string    `gorm:"type:text" json:"reason"`
	OldSubscriptionURL *string   `gorm:"type:varchar(255);index" json:"old_subscription_url"`
	NewSubscriptionURL *string   `gorm:"type:varchar(255);index" json:"new_subscription_url"`
	DeviceCountBefore  int       `gorm:"default:0" json:"device_count_before"`
	DeviceCountAfter   int       `gorm:"default:0" json:"device_count_after"`
	ResetBy            *string   `gorm:"type:varchar(50)" json:"reset_by"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (SubscriptionReset) TableName() string {
	return "subscription_resets"
}
