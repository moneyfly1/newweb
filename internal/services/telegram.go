package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"cboard/v2/internal/utils"
)

// TelegramLoginData represents data from Telegram Login Widget
type TelegramLoginData struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// VerifyTelegramLogin verifies the Telegram Login Widget data
// See: https://core.telegram.org/widgets/login#checking-authorization
func VerifyTelegramLogin(data *TelegramLoginData) bool {
	botToken := utils.GetSetting("notify_telegram_bot_token")
	if botToken == "" {
		return false
	}

	// Reject stale login data (older than 24 hours) to prevent replay attacks
	if time.Now().Unix()-data.AuthDate > 86400 {
		return false
	}

	// Create data-check-string
	pairs := []string{}
	if data.AuthDate > 0 {
		pairs = append(pairs, fmt.Sprintf("auth_date=%d", data.AuthDate))
	}
	if data.FirstName != "" {
		pairs = append(pairs, fmt.Sprintf("first_name=%s", data.FirstName))
	}
	if data.ID > 0 {
		pairs = append(pairs, fmt.Sprintf("id=%d", data.ID))
	}
	if data.LastName != "" {
		pairs = append(pairs, fmt.Sprintf("last_name=%s", data.LastName))
	}
	if data.PhotoURL != "" {
		pairs = append(pairs, fmt.Sprintf("photo_url=%s", data.PhotoURL))
	}
	if data.Username != "" {
		pairs = append(pairs, fmt.Sprintf("username=%s", data.Username))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// SHA256 of bot token as secret key
	secretKey := sha256.Sum256([]byte(botToken))

	// HMAC-SHA256 of data-check-string
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	return expectedHash == data.Hash
}
