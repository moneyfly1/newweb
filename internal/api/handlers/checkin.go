package handlers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
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

	minReward := utils.GetIntSetting("checkin_min_reward", 10)
	maxReward := utils.GetIntSetting("checkin_max_reward", 50)
	if minReward > maxReward {
		minReward = maxReward
	}

	// Settings are in 分 (cents), convert to 元 (yuan) for balance
	rangeSize := maxReward - minReward + 1
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	rewardCents := minReward + int(n.Int64())
	amount := float64(rewardCents) / 100.0

	var newBalance float64

	// 使用事务防止重放攻击和竞态条件
	err := db.Transaction(func(tx *gorm.DB) error {
		// 在事务内再次检查是否已签到（防重放）
		today := time.Now().Format("2006-01-02")
		todayStart, _ := time.ParseInLocation("2006-01-02", today, time.Now().Location())
		tomorrowStart := todayStart.AddDate(0, 0, 1)
		var count int64
		tx.Model(&models.CheckIn{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, todayStart, tomorrowStart).Count(&count)
		if count > 0 {
			return fmt.Errorf("already_checked_in")
		}

		// 在事务内读取用户余额
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return fmt.Errorf("user_not_found")
		}
		balanceBefore := user.Balance

		// 创建签到记录
		checkIn := models.CheckIn{
			UserID: userID,
			Amount: amount,
		}
		if err := tx.Create(&checkIn).Error; err != nil {
			return err
		}

		// 原子更新余额
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}

		// 记录余额日志
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
			return err
		}

		newBalance = balanceBefore + amount
		return nil
	})

	if err != nil {
		if err.Error() == "already_checked_in" {
			utils.BadRequest(c, "今天已经签到过了")
		} else if err.Error() == "user_not_found" {
			utils.InternalError(c, "用户不存在")
		} else {
			utils.InternalError(c, "签到失败")
		}
		return
	}

	// 计算连续签到天数（事务外，确保新记录已提交）
	consecutiveDays := calcConsecutiveDays(db, userID)

	utils.Success(c, gin.H{
		"amount":           amount,
		"consecutive_days": consecutiveDays,
		"new_balance":      newBalance,
	})
}

// GetCheckInStatus handles GET /api/v1/checkin/status
func GetCheckInStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	today := time.Now().Format("2006-01-02")
	todayStart, _ := time.ParseInLocation("2006-01-02", today, time.Now().Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	var (
		todayCount      int64
		totalCheckIns   int64
		lastCheckIn     *time.Time
		consecutiveDays int
	)

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		db.Model(&models.CheckIn{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, todayStart, tomorrowStart).Count(&todayCount)
	}()
	go func() {
		defer wg.Done()
		db.Model(&models.CheckIn{}).Where("user_id = ?", userID).Count(&totalCheckIns)
	}()
	go func() {
		defer wg.Done()
		var last models.CheckIn
		if err := db.Where("user_id = ?", userID).Order("created_at DESC").First(&last).Error; err == nil {
			lastCheckIn = &last.CreatedAt
		}
	}()
	go func() {
		defer wg.Done()
		consecutiveDays = calcConsecutiveDays(db, userID)
	}()
	wg.Wait()

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
	// 查最近 365 天，覆盖最长连续签到
	since := time.Now().AddDate(0, 0, -365)
	var dates []DateRow
	db.Model(&models.CheckIn{}).
		Select("DATE(created_at) as d").
		Where("user_id = ? AND created_at >= ?", userID, since).
		Group("DATE(created_at)").
		Order("d DESC").
		Find(&dates)

	if len(dates) == 0 {
		return 0
	}

	consecutive := 0
	// Start from today (local time)
	now := time.Now()
	checkDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

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
