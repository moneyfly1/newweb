package handlers

import (
	"strconv"
	"time"

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

	// 使用事务确保设备停用和计数更新的原子性
	//
	// 删除设备 = 软删 + kicked_at（保留指纹行）：物理删除会让该设备下次拉订阅
	// 时因指纹查不到被当成新设备重新登记，用户会看到「删了又回来」。
	// 保留指纹行后，订阅侧识别到 kicked_at 会明确拒绝（设备已被移除）。
	now := time.Now()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&device).Updates(map[string]interface{}{
			"is_active": false,
			"kicked_at": now,
		}).Error; err != nil {
			return err
		}
		// Decrement current devices count (atomic, floor at 0)
		return tx.Model(&sub).UpdateColumn("current_devices", gorm.Expr("CASE WHEN current_devices > 0 THEN current_devices - 1 ELSE 0 END")).Error
	}); err != nil {
		utils.InternalError(c, "删除设备失败")
		return
	}

	utils.SuccessMessage(c, "设备已删除")
}
