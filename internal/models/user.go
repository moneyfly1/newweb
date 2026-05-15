package models

import "time"

type User struct {
	ID                          uint       `gorm:"primaryKey" json:"id"`
	Username                    string     `gorm:"type:varchar(50);uniqueIndex" json:"username"`
	Email                       string     `gorm:"type:varchar(100);uniqueIndex" json:"email"`
	Password                    string     `gorm:"type:varchar(255)" json:"-"`
	IsActive                    bool       `gorm:"default:true;index" json:"is_active"`
	IsVerified                  bool       `gorm:"default:false;index" json:"is_verified"`
	IsAdmin                     bool       `gorm:"default:false" json:"is_admin"`
	Nickname                    *string    `gorm:"type:varchar(50)" json:"nickname"`
	Avatar                      *string    `gorm:"type:varchar(255)" json:"avatar"`
	Theme                       string     `gorm:"type:varchar(20);default:'indigo'" json:"theme"`
	Language                    string     `gorm:"type:varchar(10);default:'zh-CN'" json:"language"`
	Timezone                    string     `gorm:"type:varchar(50);default:'Asia/Shanghai'" json:"timezone"`
	EmailNotifications          bool       `gorm:"default:true;index" json:"email_notifications"`
	AbnormalLoginAlertEnabled   bool       `gorm:"default:true" json:"abnormal_login_alert_enabled"`
	NotificationTypes           string     `gorm:"type:text" json:"notification_types"`
	PushNotifications           bool       `gorm:"default:true" json:"push_notifications"`
	NotifyOrder                 bool       `gorm:"default:true" json:"notify_order"`
	NotifyExpiry                bool       `gorm:"default:true" json:"notify_expiry"`
	NotifySubscription          bool       `gorm:"default:true" json:"notify_subscription"`
	DataSharing                 bool       `gorm:"default:true" json:"data_sharing"`
	Analytics                   bool       `gorm:"default:true" json:"analytics"`
	Balance                     float64    `gorm:"type:decimal(10,2);default:0;index" json:"balance"`
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

// UserLevel 用户等级
type UserLevel struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	LevelName      string    `gorm:"type:varchar(50);uniqueIndex" json:"level_name"`
	LevelOrder     int       `gorm:"uniqueIndex" json:"level_order"`
	MinConsumption float64   `gorm:"type:decimal(10,2);default:0" json:"min_consumption"`
	DiscountRate   float64   `gorm:"type:decimal(5,2);default:1.0" json:"discount_rate"`
	Benefits       *string   `gorm:"type:text" json:"benefits"`
	IconURL        *string   `gorm:"type:varchar(255)" json:"icon_url"`
	Color          string    `gorm:"type:varchar(20);default:'#909399'" json:"color"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserLevel) TableName() string {
	return "user_levels"
}

// UserActivity 用户活动记录
type UserActivity struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	ActivityType     string    `gorm:"type:varchar(50);not null" json:"activity_type"`
	Description      *string   `gorm:"type:text" json:"description,omitempty"`
	IPAddress        *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        *string   `gorm:"type:text" json:"user_agent,omitempty"`
	Location         *string   `gorm:"type:varchar(100)" json:"location,omitempty"`
	ActivityMetadata *string   `gorm:"type:text" json:"activity_metadata,omitempty"`
	CreatedAt        time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (UserActivity) TableName() string { return "user_activities" }

// LoginHistory 登录历史
type LoginHistory struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            uint       `gorm:"index:idx_user_login;index:idx_login_user_ip,priority:2;not null" json:"user_id"`
	LoginTime         time.Time  `gorm:"autoCreateTime;index:idx_login_time;index:idx_login_user_ip,priority:1" json:"login_time"`
	LogoutTime        *time.Time `json:"logout_time,omitempty"`
	IPAddress         *string    `gorm:"type:varchar(45);index:idx_ip_time;index:idx_login_user_ip,priority:3" json:"ip_address,omitempty"`
	UserAgent         *string    `gorm:"type:text" json:"user_agent,omitempty"`
	Location          *string    `gorm:"type:varchar(100)" json:"location,omitempty"`
	DeviceFingerprint *string    `gorm:"type:varchar(255)" json:"device_fingerprint,omitempty"`
	LoginStatus       string     `gorm:"type:varchar(20);default:success" json:"login_status"`
	FailureReason     *string    `gorm:"type:text" json:"failure_reason,omitempty"`
	SessionDuration   *int64     `json:"session_duration,omitempty"`
}

func (LoginHistory) TableName() string { return "login_history" }
