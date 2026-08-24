package handlers

import (
	"time"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func AdminGetCheckInStats(c *gin.Context) {
	db := database.GetDB()
	today := time.Now().Format("2006-01-02")

	var todayCount, totalCount int64
	db.Model(&models.CheckIn{}).Where("DATE(created_at) = ?", today).Count(&todayCount)
	db.Model(&models.CheckIn{}).Count(&totalCount)

	var todayTotalReward float64
	db.Model(&models.CheckIn{}).Where("DATE(created_at) = ?", today).
		Select("COALESCE(SUM(amount), 0)").Scan(&todayTotalReward)

	enabled := utils.IsBoolSettingDefault("checkin_enabled", true)
	minReward := utils.GetIntSetting("checkin_min_reward", 10)
	maxReward := utils.GetIntSetting("checkin_max_reward", 50)

	utils.Success(c, gin.H{
		"today_count":        todayCount,
		"total_count":        totalCount,
		"today_total_reward": todayTotalReward,
		"settings": gin.H{
			"enabled":    enabled,
			"min_reward": minReward,
			"max_reward": maxReward,
		},
	})
}
