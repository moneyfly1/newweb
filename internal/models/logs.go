package models

import (
	"time"
)

type RegistrationLog struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	Username       string         `gorm:"type:varchar(50);not null" json:"username"`
	Email          string         `gorm:"type:varchar(100);not null" json:"email"`
	IPAddress      *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent      *string   `gorm:"type:text" json:"user_agent,omitempty"`
	Location       *string   `gorm:"type:varchar(255)" json:"location,omitempty"`
	RegisterSource *string   `gorm:"type:varchar(50)" json:"register_source,omitempty"`
	InviteCode     *string   `gorm:"type:varchar(20)" json:"invite_code,omitempty"`
	InviterID      *int64    `gorm:"index" json:"inviter_id,omitempty"`
	Status         string    `gorm:"type:varchar(20);default:success" json:"status"`
	FailureReason  *string   `gorm:"type:text" json:"failure_reason,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}

func (RegistrationLog) TableName() string { return "registration_logs" }

type SubscriptionLog struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	SubscriptionID uint           `gorm:"index;not null" json:"subscription_id"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	ActionType     string         `gorm:"type:varchar(50);not null;index" json:"action_type"`
	ActionBy       *string   `gorm:"type:varchar(50)" json:"action_by,omitempty"`
	ActionByUserID *int64    `gorm:"index" json:"action_by_user_id,omitempty"`
	BeforeData     *string   `gorm:"type:text" json:"before_data,omitempty"`
	AfterData      *string   `gorm:"type:text" json:"after_data,omitempty"`
	Description    *string   `gorm:"type:text" json:"description,omitempty"`
	IPAddress      *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	Location       *string   `gorm:"type:varchar(255)" json:"location,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}

func (SubscriptionLog) TableName() string { return "subscription_logs" }

type BalanceLog struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	ChangeType      string         `gorm:"type:varchar(50);not null;index" json:"change_type"`
	Amount          float64        `gorm:"type:decimal(10,2);not null" json:"amount"`
	BalanceBefore   float64        `gorm:"type:decimal(10,2);not null" json:"balance_before"`
	BalanceAfter    float64        `gorm:"type:decimal(10,2);not null" json:"balance_after"`
	RelatedOrderID  *int64    `gorm:"index" json:"related_order_id,omitempty"`
	RelatedRecordID *int64    `gorm:"index" json:"related_record_id,omitempty"`
	Description     *string   `gorm:"type:text" json:"description,omitempty"`
	Operator        *string   `gorm:"type:varchar(50)" json:"operator,omitempty"`
	OperatorUserID  *int64    `gorm:"index" json:"operator_user_id,omitempty"`
	IPAddress       *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	Location        *string   `gorm:"type:varchar(255)" json:"location,omitempty"`
	CreatedAt       time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}

func (BalanceLog) TableName() string { return "balance_logs" }

type CommissionLog struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	InviterID        uint           `gorm:"index;not null" json:"inviter_id"`
	InviteeID        uint           `gorm:"index;not null" json:"invitee_id"`
	InviteRelationID *int64     `gorm:"index" json:"invite_relation_id,omitempty"`
	CommissionType   string     `gorm:"type:varchar(50);not null;index" json:"commission_type"`
	Amount           float64    `gorm:"type:decimal(10,2);not null" json:"amount"`
	RelatedOrderID   *int64     `gorm:"index" json:"related_order_id,omitempty"`
	Status           string     `gorm:"type:varchar(20);default:pending;index" json:"status"`
	SettledAt        *time.Time `json:"settled_at,omitempty"`
	Description      *string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}

func (CommissionLog) TableName() string { return "commission_logs" }
