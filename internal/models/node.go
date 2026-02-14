package models

import "time"

type Node struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"type:varchar(100)" json:"name"`
	Region        string    `gorm:"type:varchar(50)" json:"region"`
	Type          string    `gorm:"type:varchar(20)" json:"type"`
	Status        string    `gorm:"type:varchar(20);default:'offline'" json:"status"`
	Load          float64   `gorm:"default:0" json:"load"`
	Speed         float64   `gorm:"default:0" json:"speed"`
	Uptime        int       `gorm:"default:0" json:"uptime"`
	Latency       int       `gorm:"default:0" json:"latency"`
	Description   *string   `gorm:"type:text" json:"description"`
	Config        *string   `gorm:"type:text" json:"config"`
	IsRecommended bool      `gorm:"default:false" json:"is_recommended"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	IsManual      bool      `gorm:"default:false" json:"is_manual"`
	OrderIndex    int       `gorm:"default:0;index" json:"order_index"`
	LastTest      *time.Time `json:"last_test"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Node) TableName() string {
	return "nodes"
}

type CustomNode struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"type:varchar(100)" json:"name"`
	DisplayName      string     `gorm:"type:varchar(100)" json:"display_name"`
	Protocol         string     `gorm:"type:varchar(20)" json:"protocol"`
	Domain           string     `gorm:"type:varchar(255)" json:"domain"`
	Port             int        `gorm:"default:443" json:"port"`
	Config           string     `gorm:"type:text" json:"config"`
	Status           string     `gorm:"type:varchar(20);default:'inactive'" json:"status"`
	IsActive         bool       `gorm:"default:true" json:"is_active"`
	Latency          int        `gorm:"default:0" json:"latency"`
	LastTest         *time.Time `json:"last_test"`
	ExpireTime       *time.Time `json:"expire_time"`
	FollowUserExpire bool       `gorm:"default:false" json:"follow_user_expire"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CustomNode) TableName() string {
	return "custom_nodes"
}

type UserCustomNode struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index:idx_user_node" json:"user_id"`
	CustomNodeID uint      `gorm:"index:idx_user_node" json:"custom_node_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserCustomNode) TableName() string {
	return "user_custom_nodes"
}
