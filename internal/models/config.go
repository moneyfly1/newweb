package models

import "time"

type SystemConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"type:varchar(100);uniqueIndex:idx_key_category" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Type        string    `gorm:"type:varchar(50)" json:"type"`
	Category    string    `gorm:"type:varchar(50);uniqueIndex:idx_key_category" json:"category"`
	DisplayName string    `gorm:"type:varchar(100)" json:"display_name"`
	Description string    `gorm:"type:text" json:"description"`
	IsPublic    bool      `gorm:"default:false" json:"is_public"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SystemConfig) TableName() string {
	return "system_configs"
}

type Announcement struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"type:varchar(200)" json:"title"`
	Content     string     `gorm:"type:text" json:"content"`
	Type        string     `gorm:"type:varchar(50);default:'info'" json:"type"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	IsPinned    bool       `gorm:"default:false" json:"is_pinned"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	TargetUsers string     `gorm:"type:varchar(50);default:'all'" json:"target_users"`
	CreatedBy   uint       `json:"created_by"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Announcement) TableName() string {
	return "announcements"
}

type ThemeConfig struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	DisplayName  string    `gorm:"type:varchar(100);not null" json:"display_name"`
	IsActive     bool      `gorm:"default:false" json:"is_active"`
	IsDefault    bool      `gorm:"default:false" json:"is_default"`
	Config       string    `gorm:"type:text" json:"config"`
	PreviewImage string    `gorm:"type:varchar(200)" json:"preview_image"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ThemeConfig) TableName() string {
	return "theme_configs"
}
