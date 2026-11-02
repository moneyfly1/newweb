package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// XBoard 兼容接口（对齐 myweb xboard_compat）。
//
// 一些第三方客户端/面板（XBoard 系）按固定路径与字段读取用户与订阅信息：
//   GET /api/v1/user/info      → 用户基础信息
//   GET /api/v1/user/subscribe → 订阅地址与到期信息
// 这里按它们的字段名输出，避免用户在这些客户端里看到「账号异常」。

// GetCurrentUserXBoardCompat /user/info。
func GetCurrentUserXBoardCompat(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user models.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	resp := gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
		"is_admin":    user.IsAdmin,
		"balance":     user.Balance,
		"created_at":  user.CreatedAt.Format(time.RFC3339),
	}
	if user.LastLogin != nil {
		resp["last_login"] = user.LastLogin.Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, resp)
}

// GetUserSubscriptionXBoardCompat /user/subscribe。
func GetUserSubscriptionXBoardCompat(c *gin.Context) {
	// 订阅内容必须实时反映节点增删，禁止任何客户端/代理缓存
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")

	userID := c.GetUint("user_id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", userID).First(&sub).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	urls := buildSubscriptionURLs(sub.SubscriptionURL)

	expiryDate := ""
	if !sub.ExpireTime.IsZero() {
		expiryDate = sub.ExpireTime.Format(time.RFC3339)
	}
	remainingDays := 0
	isExpired := false
	if !sub.ExpireTime.IsZero() {
		d := time.Until(sub.ExpireTime)
		remainingDays = int(d.Hours() / 24)
		isExpired = d <= 0
	}

	resp := gin.H{
		"subscribe_url":   urls["clash_url"],     // XBoard 期望的字段名
		"universal_url":   urls["universal_url"], // 通用订阅 URL
		"expire_time":     expiryDate,            // ISO 8601
		"expiryDate":      expiryDate,            // 兼容字段
		"device_limit":    sub.DeviceLimit,
		"current_devices": sub.CurrentDevices,
		"remaining_days":  remainingDays,
		"is_expired":      isExpired,
		"status":          sub.Status,
		"is_active":       sub.IsActive,
	}
	if !sub.IsActive {
		resp["status"] = "disabled"
		resp["is_active"] = false
	}
	c.JSON(http.StatusOK, resp)
}

// GetClientSubscribeXBoardCompat /client/subscribe。
//
// 直接复用主订阅入口：访问控制（脚本/预览拦截）、设备登记、占位节点、
// 协议过滤都在 GetSubscription 内完成，无需重复实现。
func GetClientSubscribeXBoardCompat(c *gin.Context) {
	GetSubscription(c)
}
