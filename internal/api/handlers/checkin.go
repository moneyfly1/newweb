package handlers

import (
	"math/rand"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserCheckIn handles POST /api/v1/checkin
func UserCheckIn(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	// Check if checkin is enabled
	if !utils.IsBoolSettingDefault("checkin_enabled", true) {
		utils.BadRequest(c, "签到功能未开启")
		return
	}

	// Check if already checked in today
	today := time.Now().Format("2006-01-02")
	var count int64
	db.Model(&models.CheckIn{}).Where("user_id = ? AND DATE(created_at) = ?", userID, today).Count(&count)
	if count > 0 {
		utils.BadRequest(c, "今天已经签到过了")
		return
	}

	minReward := utils.GetIntSetting("checkin_min_reward", 10)
	maxReward := utils.GetIntSetting("checkin_max_reward", 50)
	if minReward > maxReward {
		minReward = maxReward
	}

	amount := float64(minReward + rand.Intn(maxReward-minReward+1))

	// Get user's current balance for the log
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.InternalError(c, "用户不存在")
		return
	}
	balanceBefore := user.Balance

	// Create check-in record and update balance in transaction
	tx := db.Begin()
	checkIn := models.CheckIn{
		UserID: userID,
		Amount: amount,
	}
	if err := tx.Create(&checkIn).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "签到失败")
		return
	}

	if err := tx.Model(&models.User{}).Where("id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "更新余额失败")
		return
	}

	desc := "每日签到奖励"
	balanceLog := models.BalanceLog{
		UserID:        userID,
		ChangeType:    "checkin",
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceBefore + amount,
		Description:   &desc,
	}
	if err := tx.Create(&balanceLog).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "记录余额日志失败")
		return
	}

	tx.Commit()

	// Calculate consecutive days
	consecutiveDays := calcConsecutiveDays(db, userID)

	utils.Success(c, gin.H{
		"amount":          amount,
		"consecutive_days": consecutiveDays,
	})
}

// GetCheckInStatus handles GET /api/v1/checkin/status
func GetCheckInStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	today := time.Now().Format("2006-01-02")
	var todayCount int64
	db.Model(&models.CheckIn{}).Where("user_id = ? AND DATE(created_at) = ?", userID, today).Count(&todayCount)

	var totalCheckIns int64
	db.Model(&models.CheckIn{}).Where("user_id = ?", userID).Count(&totalCheckIns)

	var lastCheckIn *time.Time
	var last models.CheckIn
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").First(&last).Error; err == nil {
		lastCheckIn = &last.CreatedAt
	}

	consecutiveDays := calcConsecutiveDays(db, userID)

	utils.Success(c, gin.H{
		"checked_in_today": todayCount > 0,
		"consecutive_days": consecutiveDays,
		"total_check_ins":  totalCheckIns,
		"last_check_in":    lastCheckIn,
	})
}

// GetCheckInHistory handles GET /api/v1/checkin/history
func GetCheckInHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	p := utils.GetPagination(c)
	db := database.GetDB()

	var total int64
	db.Model(&models.CheckIn{}).Where("user_id = ?", userID).Count(&total)

	var items []models.CheckIn
	db.Where("user_id = ?", userID).Order("created_at DESC").
		Offset(p.Offset()).Limit(p.PageSize).Find(&items)

	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

// calcConsecutiveDays calculates consecutive check-in days ending today or yesterday
func calcConsecutiveDays(db *gorm.DB, userID uint) int {
	type DateRow struct {
		D string
	}
	var dates []DateRow
	db.Model(&models.CheckIn{}).
		Select("DATE(created_at) as d").
		Where("user_id = ?", userID).
		Group("DATE(created_at)").
		Order("d DESC").
		Limit(365).
		Find(&dates)

	if len(dates) == 0 {
		return 0
	}

	consecutive := 0
	// Start from today
	checkDate := time.Now().Truncate(24 * time.Hour)

	for _, row := range dates {
		d, err := time.Parse("2006-01-02", row.D)
		if err != nil {
			break
		}
		d = d.Truncate(24 * time.Hour)

		if d.Equal(checkDate) {
			consecutive++
			checkDate = checkDate.AddDate(0, 0, -1)
		} else if consecutive == 0 && d.Equal(checkDate.AddDate(0, 0, -1)) {
			// If not checked in today, start from yesterday
			checkDate = checkDate.AddDate(0, 0, -1)
			if d.Equal(checkDate) {
				consecutive++
				checkDate = checkDate.AddDate(0, 0, -1)
			}
		} else {
			break
		}
	}

	return consecutive
}
