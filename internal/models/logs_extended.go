package models

import (
	"time"
)

// OrderLog 订单操作日志
type OrderLog struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	OrderID         uint       `gorm:"index" json:"order_id"`
	UserID          uint       `gorm:"index" json:"user_id"`
	ActionType      string     `gorm:"type:varchar(50)" json:"action_type"` // create, pay, cancel, refund, etc.
	ActionBy        *string    `gorm:"type:varchar(50)" json:"action_by"`   // user, admin, system
	ActionByUserID  *int64     `json:"action_by_user_id"`
	Description     *string    `gorm:"type:text" json:"description"`
	BeforeData      *string    `gorm:"type:text" json:"before_data"`
	AfterData       *string    `gorm:"type:text" json:"after_data"`
	CreatedAt       time.Time  `json:"created_at"`
}

// PaymentLog 支付操作日志
type PaymentLog struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	TransactionID uint       `gorm:"index" json:"transaction_id"`
	UserID        uint       `gorm:"index" json:"user_id"`
	PaymentMethod string     `gorm:"type:varchar(50)" json:"payment_method"`
	Amount        float64    `json:"amount"`
	Status        string     `gorm:"type:varchar(50)" json:"status"` // pending, success, failed
	Description   *string    `gorm:"type:text" json:"description"`
	IPAddress     *string    `gorm:"type:varchar(100)" json:"ip_address"`
	UserAgent     *string    `gorm:"type:text" json:"user_agent"`
	Location      *string    `gorm:"type:varchar(200)" json:"location"`
	CreatedAt     time.Time  `json:"created_at"`
}

// CouponLog 优惠券操作日志
type CouponLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	CouponID    uint       `gorm:"index" json:"coupon_id"`
	UserID      uint       `gorm:"index" json:"user_id"`
	ActionType  string     `gorm:"type:varchar(50)" json:"action_type"` // use, expire, cancel
	Description *string    `gorm:"type:text" json:"description"`
	IPAddress   *string    `gorm:"type:varchar(100)" json:"ip_address"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NodeLog 节点操作日志
type NodeLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	NodeID         uint       `gorm:"index" json:"node_id"`
	ActionType     string     `gorm:"type:varchar(50)" json:"action_type"` // create, update, delete, enable, disable
	ActionBy       *string    `gorm:"type:varchar(50)" json:"action_by"`   // admin, system
	ActionByUserID *int64     `json:"action_by_user_id"`
	Description    *string    `gorm:"type:text" json:"description"`
	BeforeData     *string    `gorm:"type:text" json:"before_data"`
	AfterData      *string    `gorm:"type:text" json:"after_data"`
	CreatedAt      time.Time  `json:"created_at"`
}

// UserActionLog 用户操作日志
type UserActionLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index" json:"user_id"`
	ActionType  string     `gorm:"type:varchar(50)" json:"action_type"` // login, logout, update_profile, change_password, etc.
	Module      string     `gorm:"type:varchar(50)" json:"module"`      // auth, profile, subscription, etc.
	Description *string    `gorm:"type:text" json:"description"`
	IPAddress   *string    `gorm:"type:varchar(100)" json:"ip_address"`
	UserAgent   *string    `gorm:"type:text" json:"user_agent"`
	Location    *string    `gorm:"type:varchar(200)" json:"location"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AdminActionLog 管理员操作日志
type AdminActionLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	AdminID     uint       `gorm:"index" json:"admin_id"`
	ActionType  string     `gorm:"type:varchar(50)" json:"action_type"` // create, update, delete, etc.
	Module      string     `gorm:"type:varchar(50)" json:"module"`      // user, node, package, config, etc.
	TargetType  string     `gorm:"type:varchar(50)" json:"target_type"` // user, node, package, etc.
	TargetID    *int64     `json:"target_id"`
	Description *string    `gorm:"type:text" json:"description"`
	BeforeData  *string    `gorm:"type:text" json:"before_data"`
	AfterData   *string    `gorm:"type:text" json:"after_data"`
	IPAddress   *string    `gorm:"type:varchar(100)" json:"ip_address"`
	UserAgent   *string    `gorm:"type:text" json:"user_agent"`
	CreatedAt   time.Time  `json:"created_at"`
}

// DeviceLog 设备操作日志
type DeviceLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	DeviceID    uint       `gorm:"index" json:"device_id"`
	UserID      uint       `gorm:"index" json:"user_id"`
	ActionType  string     `gorm:"type:varchar(50)" json:"action_type"` // connect, disconnect, delete
	Description *string    `gorm:"type:text" json:"description"`
	IPAddress   *string    `gorm:"type:varchar(100)" json:"ip_address"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TicketLog 工单操作日志
type TicketLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	TicketID    uint       `gorm:"index" json:"ticket_id"`
	UserID      uint       `gorm:"index" json:"user_id"`
	ActionType  string     `gorm:"type:varchar(50)" json:"action_type"` // create, reply, close, reopen
	ActionBy    *string    `gorm:"type:varchar(50)" json:"action_by"`   // user, admin
	Description *string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
}

// InviteLog 邀请操作日志
type InviteLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	InviterID   uint       `gorm:"index" json:"inviter_id"`
	InviteeID   uint       `gorm:"index" json:"invitee_id"`
	InviteCode  string     `gorm:"type:varchar(50)" json:"invite_code"`
	ActionType  string     `gorm:"type:varchar(50)" json:"action_type"` // register, reward
	Description *string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ConfigChangeLog 配置变更日志
type ConfigChangeLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	AdminID     uint       `gorm:"index" json:"admin_id"`
	ConfigKey   string     `gorm:"type:varchar(100);index" json:"config_key"`
	OldValue    *string    `gorm:"type:text" json:"old_value"`
	NewValue    *string    `gorm:"type:text" json:"new_value"`
	Description *string    `gorm:"type:text" json:"description"`
	IPAddress   *string    `gorm:"type:varchar(100)" json:"ip_address"`
	CreatedAt   time.Time  `json:"created_at"`
}

// SecurityLog 安全事件日志
type SecurityLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      *int64     `gorm:"index" json:"user_id"`
	EventType   string     `gorm:"type:varchar(50);index" json:"event_type"` // login_failed, suspicious_activity, brute_force, etc.
	Severity    string     `gorm:"type:varchar(20)" json:"severity"`         // low, medium, high, critical
	Description *string    `gorm:"type:text" json:"description"`
	IPAddress   *string    `gorm:"type:varchar(100)" json:"ip_address"`
	UserAgent   *string    `gorm:"type:text" json:"user_agent"`
	Location    *string    `gorm:"type:varchar(200)" json:"location"`
	CreatedAt   time.Time  `json:"created_at"`
}

// APILog API 调用日志
type APILog struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       *int64     `gorm:"index" json:"user_id"`
	Method       string     `gorm:"type:varchar(10)" json:"method"`
	Path         string     `gorm:"type:varchar(500);index" json:"path"`
	StatusCode   int        `json:"status_code"`
	ResponseTime int        `json:"response_time"` // milliseconds
	IPAddress    *string    `gorm:"type:varchar(100)" json:"ip_address"`
	UserAgent    *string    `gorm:"type:text" json:"user_agent"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DatabaseLog 数据库操作日志
type DatabaseLog struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	AdminID      uint       `gorm:"index" json:"admin_id"`
	Operation    string     `gorm:"type:varchar(50)" json:"operation"` // backup, restore, migrate, truncate
	TableName    string     `gorm:"type:varchar(100)" json:"table_name"`
	AffectedRows int        `json:"affected_rows"`
	Description  *string    `gorm:"type:text" json:"description"`
	CreatedAt    time.Time  `json:"created_at"`
}

// EmailLog 邮件发送日志
type EmailLog struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       *int64     `gorm:"index" json:"user_id"`
	EmailType    string     `gorm:"type:varchar(50)" json:"email_type"` // verification, reset_password, welcome, etc.
	Recipient    string     `gorm:"type:varchar(255)" json:"recipient"`
	Subject      string     `gorm:"type:varchar(500)" json:"subject"`
	Status       string     `gorm:"type:varchar(20)" json:"status"` // sent, failed, pending
	ErrorMessage *string    `gorm:"type:text" json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
}

// NotificationLog 通知发送日志
type NotificationLog struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           *int64     `gorm:"index" json:"user_id"`
	NotificationType string     `gorm:"type:varchar(50)" json:"notification_type"` // order, payment, subscription, etc.
	Channel          string     `gorm:"type:varchar(20)" json:"channel"`           // email, telegram, bark
	Status           string     `gorm:"type:varchar(20)" json:"status"`            // sent, failed
	Content          *string    `gorm:"type:text" json:"content"`
	CreatedAt        time.Time  `json:"created_at"`
}
