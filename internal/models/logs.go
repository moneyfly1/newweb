package models

import (
	"time"
)

// AuditLog 审计日志（系统操作审计）
type AuditLog struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            *int64    `gorm:"index" json:"user_id,omitempty"`
	ActionType        string    `gorm:"type:varchar(50);index;not null" json:"action_type"`
	ResourceType      *string   `gorm:"type:varchar(50);index" json:"resource_type,omitempty"`
	ResourceID        *int64    `gorm:"index" json:"resource_id,omitempty"`
	ActionDescription *string   `gorm:"type:text" json:"action_description,omitempty"`
	IPAddress         *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         *string   `gorm:"type:text" json:"user_agent,omitempty"`
	Location          *string   `gorm:"type:varchar(255)" json:"location,omitempty"`
	RequestMethod     *string   `gorm:"type:varchar(10)" json:"request_method,omitempty"`
	RequestPath       *string   `gorm:"type:varchar(255)" json:"request_path,omitempty"`
	RequestParams     *string   `gorm:"type:text" json:"request_params,omitempty"`
	ResponseStatus    *int64    `json:"response_status,omitempty"`
	BeforeData        *string   `gorm:"type:text" json:"before_data,omitempty"`
	AfterData         *string   `gorm:"type:text" json:"after_data,omitempty"`
	CreatedAt         time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// RegistrationLog 用户注册日志
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

// SubscriptionLog 订阅变更日志
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

// BalanceLog 余额变动日志
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

// CommissionLog 佣金日志
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

// SystemLog 系统运行日志（邮件发送、定时任务、错误等）
type SystemLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Level     string    `gorm:"type:varchar(10);index;not null" json:"level"` // info, warn, error
	Module    string    `gorm:"type:varchar(50);index;not null" json:"module"` // scheduler, email, payment, notify, system
	Message   string    `gorm:"type:text;not null" json:"message"`
	Detail    *string   `gorm:"type:text" json:"detail,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (SystemLog) TableName() string { return "system_logs" }
