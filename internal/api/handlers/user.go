package handlers

import (
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func GetCurrentUser(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	utils.Success(c, user)
}

func UpdateCurrentUser(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var req struct {
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Theme    string `json:"theme"`
		Language string `json:"language"`
		Timezone string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	updates := map[string]interface{}{}
	if req.Username != "" && req.Username != user.Username {
		var count int64
		db.Model(&models.User{}).Where("username = ? AND id != ?", req.Username, user.ID).Count(&count)
		if count > 0 {
			utils.BadRequest(c, "用户名已被使用")
			return
		}
		updates["username"] = req.Username
	}
	if req.Nickname != "" {
		updates["nickname"] = &req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = &req.Avatar
	}
	if req.Theme != "" {
		updates["theme"] = req.Theme
	}
	if req.Language != "" {
		updates["language"] = req.Language
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}
	if len(updates) > 0 {
		db.Model(user).Updates(updates)
	}
	db.First(user, user.ID)
	utils.Success(c, user)
}

func ChangePassword(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if !utils.CheckPassword(req.OldPassword, user.Password) {
		utils.BadRequest(c, "原密码错误")
		return
	}
	if err := utils.ValidatePasswordStrength(req.NewPassword); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	hashed, _ := utils.HashPassword(req.NewPassword)
	database.GetDB().Model(user).Update("password", hashed)
	utils.SuccessMessage(c, "密码修改成功")
}

func UpdatePreferences(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var req struct {
		Theme    string `json:"theme"`
		Language string `json:"language"`
		Timezone string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.Theme != "" { updates["theme"] = req.Theme }
	if req.Language != "" { updates["language"] = req.Language }
	if req.Timezone != "" { updates["timezone"] = req.Timezone }
	database.GetDB().Model(user).Updates(updates)
	utils.SuccessMessage(c, "偏好设置已更新")
}

func GetNotificationSettings(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	utils.Success(c, gin.H{
		"email_notifications": user.EmailNotifications,
		"abnormal_login_alert_enabled": user.AbnormalLoginAlertEnabled,
		"push_notifications": user.PushNotifications,
	})
}

func UpdateNotificationSettings(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var req struct {
		EmailNotifications        *bool `json:"email_notifications"`
		AbnormalLoginAlertEnabled *bool `json:"abnormal_login_alert_enabled"`
		PushNotifications         *bool `json:"push_notifications"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.EmailNotifications != nil { updates["email_notifications"] = *req.EmailNotifications }
	if req.AbnormalLoginAlertEnabled != nil { updates["abnormal_login_alert_enabled"] = *req.AbnormalLoginAlertEnabled }
	if req.PushNotifications != nil { updates["push_notifications"] = *req.PushNotifications }
	database.GetDB().Model(user).Updates(updates)
	utils.SuccessMessage(c, "通知设置已更新")
}

func GetPrivacySettings(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	utils.Success(c, gin.H{"data_sharing": user.DataSharing, "analytics": user.Analytics})
}

func UpdatePrivacySettings(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var req struct {
		DataSharing *bool `json:"data_sharing"`
		Analytics   *bool `json:"analytics"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.DataSharing != nil { updates["data_sharing"] = *req.DataSharing }
	if req.Analytics != nil { updates["analytics"] = *req.Analytics }
	database.GetDB().Model(user).Updates(updates)
	utils.SuccessMessage(c, "隐私设置已更新")
}

func GetMyLevel(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	if user.UserLevelID == nil {
		utils.Success(c, nil)
		return
	}
	var level models.UserLevel
	if err := database.GetDB().First(&level, *user.UserLevelID).Error; err != nil {
		utils.Success(c, nil)
		return
	}
	utils.Success(c, level)
}

func GetLoginHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	p := utils.GetPagination(c)
	var items []models.LoginHistory
	var total int64
	db := database.GetDB().Model(&models.LoginHistory{}).Where("user_id = ?", userID)
	db.Count(&total)
	db.Order("login_time DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func GetActivities(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	p := utils.GetPagination(c)
	var items []models.UserActivity
	var total int64
	db := database.GetDB().Model(&models.UserActivity{}).Where("user_id = ?", userID)
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func GetDashboardInfo(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	user := c.MustGet("user").(*models.User)
	db := database.GetDB()
	var sub models.Subscription
	hasSub := db.Where("user_id = ?", userID).First(&sub).Error == nil
	var orderCount int64
	db.Model(&models.Order{}).Where("user_id = ?", userID).Count(&orderCount)
	var deviceCount int64
	if hasSub {
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&deviceCount)
	}
	utils.Success(c, gin.H{
		"balance": user.Balance,
		"has_subscription": hasSub,
		"subscription": sub,
		"order_count": orderCount,
		"device_count": deviceCount,
	})
}

func GetSubscriptionResets(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	p := utils.GetPagination(c)
	var items []models.SubscriptionReset
	var total int64
	db := database.GetDB().Model(&models.SubscriptionReset{}).Where("user_id = ?", userID)
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func GetUserDevices(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var sub models.Subscription
	if err := database.GetDB().Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.Success(c, []interface{}{})
		return
	}
	var devices []models.Device
	database.GetDB().Where("subscription_id = ? AND is_active = ?", sub.ID, true).Find(&devices)
	utils.Success(c, devices)
}
