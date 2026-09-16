package models

import (
	"time"

	"gorm.io/gorm"
)

// 设备在线判定窗口
const (
	// HeartbeatOnlineWindow 自有客户端（Mclash/MoneyFly/ClashMi）心跳窗口：
	// 客户端每 60~120 秒上报一次，取 3 分钟容忍抖动/丢包。
	HeartbeatOnlineWindow = 3 * time.Minute
	// SubscriptionOnlineWindow 第三方客户端（Clash Verge/Shadowrocket/v2rayN 等）无法上报心跳，
	// 只能以「24 小时内拉取过订阅」近似为在线。
	SubscriptionOnlineWindow = 24 * time.Hour
)

type Device struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            *int64     `gorm:"index" json:"user_id"`
	SubscriptionID    uint       `gorm:"index;index:idx_device_lookup,priority:1;index:idx_sub_active_access,priority:1" json:"subscription_id"`
	DeviceFingerprint string     `gorm:"type:varchar(255);index;index:idx_device_lookup,priority:2" json:"device_fingerprint"`
	DeviceHash        *string    `gorm:"type:varchar(255)" json:"device_hash"`
	DeviceUA          *string    `gorm:"type:varchar(255)" json:"device_ua"`
	DeviceName        *string    `gorm:"type:varchar(100)" json:"device_name"`
	DeviceType        *string    `gorm:"type:varchar(50)" json:"device_type"`
	IPAddress         *string    `gorm:"type:varchar(45)" json:"ip_address"`
	Region            string     `gorm:"type:varchar(100)" json:"region"`
	UserAgent         *string    `gorm:"type:text" json:"user_agent"`
	SoftwareName      *string    `gorm:"type:varchar(100)" json:"software_name"`
	SoftwareVersion   *string    `gorm:"type:varchar(50)" json:"software_version"`
	OSName            *string    `gorm:"type:varchar(50)" json:"os_name"`
	OSVersion         *string    `gorm:"type:varchar(50)" json:"os_version"`
	DeviceModel       *string    `gorm:"type:varchar(100)" json:"device_model"`
	DeviceBrand       *string    `gorm:"type:varchar(50)" json:"device_brand"`
	SubscriptionType  *string    `gorm:"type:varchar(20);index" json:"subscription_type"`
	Remark            *string    `gorm:"type:varchar(200)" json:"remark"`
	IsActive          bool       `gorm:"default:true;index;index:idx_device_lookup,priority:3;index:idx_sub_active_access,priority:2" json:"is_active"`
	IsAllowed         bool       `gorm:"default:true" json:"is_allowed"`
	FirstSeen         *time.Time `json:"first_seen"`
	LastAccess        time.Time  `gorm:"autoCreateTime;index:idx_sub_active_access,priority:3" json:"last_access"`
	LastSeen          *time.Time `json:"last_seen"`
	// LastHeartbeat 客户端在线心跳时间（自有客户端定时上报）。
	// 订阅拉取是周期性行为，无法反映「此刻是否在用」；心跳才是真实在线依据。
	LastHeartbeat *time.Time `gorm:"index" json:"last_heartbeat,omitempty"`
	AccessCount   int        `gorm:"default:0" json:"access_count"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// IsOnline 是否在线（非持久化，查询后由 AfterFind 自动计算）：
	// 1) 自有客户端：3 分钟内有心跳；
	// 2) 其他客户端：24 小时内拉取过订阅。
	IsOnline bool `gorm:"-" json:"is_online"`
}

func (Device) TableName() string {
	return "devices"
}

// AfterFind 查询后自动计算在线状态，使所有返回设备的接口都带 is_online，无需逐个 handler 处理。
func (d *Device) AfterFind(*gorm.DB) error {
	d.IsOnline = d.ComputeOnline()
	return nil
}

// ComputeOnline 在线判定：心跳优先（自有客户端），否则以 24 小时内的订阅拉取近似。
func (d Device) ComputeOnline() bool {
	if d.LastHeartbeat != nil && time.Since(*d.LastHeartbeat) < HeartbeatOnlineWindow {
		return true
	}
	return !d.LastAccess.IsZero() && time.Since(d.LastAccess) < SubscriptionOnlineWindow
}
