package handlers

import (
	"strconv"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListDevices returns all devices belonging to the current user's subscriptions.
func ListDevices(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	// Find user's subscription IDs
	var subIDs []uint
	db.Model(&models.Subscription{}).Where("user_id = ?", userID).Pluck("id", &subIDs)
	if len(subIDs) == 0 {
		utils.Success(c, []interface{}{})
		return
	}

	var devices []models.Device
	db.Where("subscription_id IN ? AND is_active = ?", subIDs, true).
		Order("last_access DESC").Limit(200).Find(&devices)

	utils.Success(c, devices)
}

// DeleteDevice removes a device and decrements the subscription's CurrentDevices.
func DeleteDevice(c *gin.Context) {
	userID := c.GetUint("user_id")
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的设备ID")
		return
	}

	db := database.GetDB()

	var device models.Device
	if err := db.First(&device, deviceID).Error; err != nil {
		utils.NotFound(c, "设备不存在")
		return
	}

	// Verify ownership through subscription
	var sub models.Subscription
	if err := db.Where("id = ? AND user_id = ?", device.SubscriptionID, userID).First(&sub).Error; err != nil {
		utils.Forbidden(c, "无权操作此设备")
		return
	}

	// Soft-deactivate the device
	if err := db.Model(&device).Update("is_active", false).Error; err != nil {
		utils.InternalError(c, "删除设备失败")
		return
	}

	// Decrement current devices count (atomic, floor at 0)
	if sub.CurrentDevices > 0 {
		db.Model(&sub).UpdateColumn("current_devices", gorm.Expr("CASE WHEN current_devices > 0 THEN current_devices - 1 ELSE 0 END"))
	}

	utils.SuccessMessage(c, "设备已删除")
}
