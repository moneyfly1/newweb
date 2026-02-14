package models

import (
	"time"
)

type AuditLog struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
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
	CreatedAt         time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
