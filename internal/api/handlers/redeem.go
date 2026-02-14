package handlers

import (
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func RedeemCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	userID := c.GetUint("user_id")
	db := database.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var code models.RedeemCode
		// Lock the row to prevent concurrent redemption
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", req.Code).First(&code).Error; err != nil {
			utils.NotFound(c, "卡密不存在")
			return err
		}
		if code.Status != "unused" && code.Status != "active" {
			utils.BadRequest(c, "卡密已使用或已失效")
			return gorm.ErrInvalidData
		}
		if code.ExpiresAt != nil && time.Now().After(*code.ExpiresAt) {
			utils.BadRequest(c, "卡密已过期")
			return gorm.ErrInvalidData
		}
		if code.UsedCount >= code.MaxUses {
			utils.BadRequest(c, "卡密使用次数已达上限")
			return gorm.ErrInvalidData
		}

		if code.Type == "balance" {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				UpdateColumn("balance", gorm.Expr("balance + ?", code.Value)).Error; err != nil {
				return err
			}
		}

		if code.Type == "duration" || code.Type == "package" {
			// Duration: value = days to add; Package: use linked package
			var sub models.Subscription
			if err := tx.Where("user_id = ?", userID).First(&sub).Error; err != nil {
				// Create new subscription
				subURL := uuid.New().String()[:8]
				sub = models.Subscription{
					UserID:          userID,
					SubscriptionURL: subURL,
					DeviceLimit:     3,
					IsActive:        true,
					Status:          "active",
					ExpireTime:      time.Now().AddDate(0, 0, int(code.Value)),
				}
				if code.PackageID != nil {
					pkgID := int64(*code.PackageID)
					sub.PackageID = &pkgID
					var pkg models.Package
					if tx.First(&pkg, *code.PackageID).Error == nil {
						sub.DeviceLimit = pkg.DeviceLimit
						if code.Type == "package" {
							sub.ExpireTime = time.Now().AddDate(0, 0, pkg.DurationDays)
						}
					}
				}
				if err := tx.Create(&sub).Error; err != nil {
					return err
				}
			} else {
				// Extend existing subscription
				newExpire := sub.ExpireTime
				if newExpire.Before(time.Now()) {
					newExpire = time.Now()
				}
				days := int(code.Value)
				if code.Type == "package" && code.PackageID != nil {
					var pkg models.Package
					if tx.First(&pkg, *code.PackageID).Error == nil {
						days = pkg.DurationDays
					}
				}
				newExpire = newExpire.AddDate(0, 0, days)
				if err := tx.Model(&sub).Updates(map[string]interface{}{
					"expire_time": newExpire, "is_active": true, "status": "active",
				}).Error; err != nil {
					return err
				}
			}
		}

		code.UsedCount++
		if code.UsedCount >= code.MaxUses {
			code.Status = "used"
		}
		if err := tx.Save(&code).Error; err != nil {
			return err
		}
		ip := c.ClientIP()
		return tx.Create(&models.RedeemRecord{
			RedeemCodeID: code.ID, UserID: userID, Code: code.Code,
			Type: code.Type, Value: code.Value,
			IPAddress: &ip,
		}).Error
	})

	if err == nil {
		utils.SuccessMessage(c, "兑换成功")
	}
}

func GetRedeemHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	p := utils.GetPagination(c)
	var items []models.RedeemRecord
	var total int64
	db := database.GetDB().Model(&models.RedeemRecord{}).Where("user_id = ?", userID)
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}
