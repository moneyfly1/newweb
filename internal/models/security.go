package models

import (
	"time"

	"gorm.io/gorm"
)

type LoginAttempt struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"type:varchar(100);index;not null;index:idx_login_lookup,priority:1" json:"username"`
	IPAddress *string   `gorm:"type:varchar(45);index" json:"ip_address,omitempty"`
	Success   bool      `gorm:"default:false;index:idx_login_lookup,priority:2" json:"success"`
	UserAgent *string   `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime;not null;index:idx_login_lookup,priority:3" json:"created_at"`
}

func (LoginAttempt) TableName() string { return "login_attempts" }

type VerificationAttempt struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	Email     string  `gorm:"type:varchar(100);index;not null" json:"email"`
	IPAddress *string `gorm:"type:varchar(45);index" json:"ip_address,omitempty"`
	Success   bool           `gorm:"default:false" json:"success"`
	Purpose   string         `gorm:"type:varchar(50);default:register" json:"purpose"`
	CreatedAt time.Time      `gorm:"autoCreateTime;not null" json:"created_at"`
}

func (VerificationAttempt) TableName() string { return "verification_attempts" }

type VerificationCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"type:varchar(100);index;not null;index:idx_verification_lookup,priority:1" json:"email"`
	Code      string    `gorm:"type:varchar(6);not null" json:"code"`
	CreatedAt time.Time `gorm:"autoCreateTime;not null" json:"created_at"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      int       `gorm:"default:0;index:idx_verification_lookup,priority:3" json:"used"`
	Purpose   string    `gorm:"type:varchar(50);default:register;index:idx_verification_lookup,priority:2" json:"purpose"`
}

func (VerificationCode) TableName() string { return "verification_codes" }
func (v *VerificationCode) IsExpired() bool { return time.Now().After(v.ExpiresAt) }
func (v *VerificationCode) IsUsed() bool    { return v.Used == 1 }
func (v *VerificationCode) MarkAsUsed()     { v.Used = 1 }

type TokenBlacklist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TokenHash string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"token_hash"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (TokenBlacklist) TableName() string    { return "token_blacklist" }
func (t *TokenBlacklist) IsExpired() bool   { return time.Now().After(t.ExpiresAt) }

func IsTokenBlacklisted(db *gorm.DB, tokenHash string) bool {
	var bl TokenBlacklist
	return db.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).First(&bl).Error == nil
}

func AddToBlacklist(db *gorm.DB, tokenHash string, userID uint, expiresAt time.Time) error {
	return db.Create(&TokenBlacklist{TokenHash: tokenHash, UserID: userID, ExpiresAt: expiresAt}).Error
}

func CleanExpiredTokens(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&TokenBlacklist{}).Error
}
