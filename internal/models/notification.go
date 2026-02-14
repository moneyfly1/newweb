package models

import (
	"time"
)

type Notification struct {
	ID        uint          `gorm:"primaryKey" json:"id"`
	UserID    *int64    `gorm:"index" json:"user_id"`
	Title     string        `gorm:"type:varchar(255)" json:"title"`
	Content   string        `gorm:"type:text" json:"content"`
	Type      string        `gorm:"type:varchar(50);default:'system'" json:"type"`
	IsRead    bool          `gorm:"default:false" json:"is_read"`
	IsActive  bool          `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	ReadAt    *time.Time `json:"read_at"`
}

func (Notification) TableName() string {
	return "notifications"
}

type EmailTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Subject   string    `gorm:"type:varchar(200)" json:"subject"`
	Content   string    `gorm:"type:text" json:"content"`
	Variables string    `gorm:"type:text" json:"variables"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (EmailTemplate) TableName() string {
	return "email_templates"
}

type EmailQueue struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ToEmail     string         `gorm:"type:varchar(100)" json:"to_email"`
	Subject     string         `gorm:"type:varchar(200)" json:"subject"`
	Content     string         `gorm:"type:text" json:"content"`
	ContentType string         `gorm:"type:varchar(20);default:'plain'" json:"content_type"`
	EmailType   string         `gorm:"type:varchar(50)" json:"email_type"`
	Attachments string         `gorm:"type:text" json:"attachments"`
	Status      string         `gorm:"type:varchar(20);default:'pending'" json:"status"`
	RetryCount  int            `gorm:"default:0" json:"retry_count"`
	MaxRetries  int            `gorm:"default:3" json:"max_retries"`
	SentAt      *time.Time `json:"sent_at"`
	ErrorMessage *string   `gorm:"type:text" json:"error_message"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (EmailQueue) TableName() string {
	return "email_queue"
}
