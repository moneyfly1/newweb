package models

import "time"

type User struct {
	ID                          uint       `gorm:"primaryKey" json:"id"`
	Username                    string     `gorm:"type:varchar(50);uniqueIndex" json:"username"`
	Email                       string     `gorm:"type:varchar(100);uniqueIndex" json:"email"`
	Password                    string     `gorm:"type:varchar(255)" json:"-"`
	IsActive                    bool       `gorm:"default:true" json:"is_active"`
	IsVerified                  bool       `gorm:"default:false" json:"is_verified"`
	IsAdmin                     bool       `gorm:"default:false" json:"is_admin"`
	Nickname                    *string    `gorm:"type:varchar(50)" json:"nickname"`
	Avatar                      *string    `gorm:"type:varchar(255)" json:"avatar"`
	Theme                       string     `gorm:"type:varchar(20);default:'light'" json:"theme"`
	Language                    string     `gorm:"type:varchar(10);default:'zh-CN'" json:"language"`
	Timezone                    string     `gorm:"type:varchar(50);default:'Asia/Shanghai'" json:"timezone"`
	EmailNotifications          bool       `gorm:"default:true" json:"email_notifications"`
	AbnormalLoginAlertEnabled   bool       `gorm:"default:true" json:"abnormal_login_alert_enabled"`
	NotificationTypes           string     `gorm:"type:text" json:"notification_types"`
	PushNotifications           bool       `gorm:"default:true" json:"push_notifications"`
	DataSharing                 bool       `gorm:"default:true" json:"data_sharing"`
	Analytics                   bool       `gorm:"default:true" json:"analytics"`
	Balance                     float64    `gorm:"type:decimal(10,2);default:0" json:"balance"`
	InvitedBy                   *uint      `gorm:"index" json:"invited_by"`
	InviteCodeUsed              *string    `gorm:"type:varchar(20)" json:"invite_code_used"`
	TotalInviteCount            int        `gorm:"default:0" json:"total_invite_count"`
	TotalInviteReward           float64    `gorm:"type:decimal(10,2);default:0" json:"total_invite_reward"`
	UserLevelID                 *uint      `gorm:"index" json:"user_level_id"`
	TotalConsumption            float64    `gorm:"type:decimal(10,2);default:0" json:"total_consumption"`
	LevelExpiresAt              *time.Time `json:"level_expires_at"`
	SpecialNodeSubscriptionType string     `gorm:"type:varchar(20);default:'both'" json:"special_node_subscription_type"`
	SpecialNodeExpiresAt        *time.Time `json:"special_node_expires_at"`
	TelegramID                  *int64     `gorm:"uniqueIndex" json:"telegram_id"`
	TelegramUsername            *string    `gorm:"type:varchar(100)" json:"telegram_username"`
	Notes                       *string    `gorm:"type:text" json:"notes"`
	VerificationToken           *string    `gorm:"type:varchar(255)" json:"-"`
	VerificationExpires         *time.Time `json:"-"`
	ResetToken                  *string    `gorm:"type:varchar(255)" json:"-"`
	ResetExpires                *time.Time `json:"-"`
	CreatedAt                   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt                   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	LastLogin                   *time.Time `json:"last_login"`
}

func (User) TableName() string {
	return "users"
}
