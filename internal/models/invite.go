package models

import (
	"time"
)

type InviteCode struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	Code           string          `gorm:"type:varchar(20);uniqueIndex" json:"code"`
	UserID         uint            `gorm:"index" json:"user_id"`
	UsedCount      int             `gorm:"default:0" json:"used_count"`
	MaxUses        *int64     `json:"max_uses"`
	ExpiresAt      *time.Time `json:"expires_at"`
	RewardType     string     `gorm:"type:varchar(20);default:'balance'" json:"reward_type"`
	InviterReward  float64    `gorm:"type:decimal(10,2);default:0" json:"inviter_reward"`
	InviteeReward  float64    `gorm:"type:decimal(10,2);default:0" json:"invitee_reward"`
	PackageIDs     *string    `gorm:"type:text" json:"package_ids"`
	MinOrderAmount float64         `gorm:"type:decimal(10,2);default:0" json:"min_order_amount"`
	NewUserOnly    bool            `gorm:"default:true" json:"new_user_only"`
	IsActive       bool            `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (InviteCode) TableName() string {
	return "invite_codes"
}

type InviteRelation struct {
	ID                      uint          `gorm:"primaryKey" json:"id"`
	InviteCodeID            uint          `gorm:"index" json:"invite_code_id"`
	InviterID               uint          `gorm:"index" json:"inviter_id"`
	InviteeID               uint          `gorm:"index" json:"invitee_id"`
	InviterRewardGiven      bool          `gorm:"default:false" json:"inviter_reward_given"`
	InviteeRewardGiven      bool          `gorm:"default:false" json:"invitee_reward_given"`
	InviterRewardAmount     float64       `gorm:"type:decimal(10,2);default:0" json:"inviter_reward_amount"`
	InviteeRewardAmount     float64       `gorm:"type:decimal(10,2);default:0" json:"invitee_reward_amount"`
	InviteeFirstOrderID     *int64  `gorm:"index" json:"invitee_first_order_id"`
	InviteeTotalConsumption float64       `gorm:"type:decimal(10,2);default:0" json:"invitee_total_consumption"`
	CreatedAt               time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}

func (InviteRelation) TableName() string {
	return "invite_relations"
}
