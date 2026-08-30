package handlers

import (
	"errors"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 兑换结果哨兵错误（事务内不写 HTTP 响应，统一在事务外处理）
var (
	errRedeemNotFound  = errors.New("redeem_not_found")
	errRedeemUsed      = errors.New("redeem_used")
	errRedeemExpired   = errors.New("redeem_expired")
	errRedeemLimit     = errors.New("redeem_limit")
	errRedeemConflict  = errors.New("redeem_conflict")
	errRedeemDuplicate = errors.New("redeem_duplicate")
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
		if err := tx.Where("code = ?", req.Code).First(&code).Error; err != nil {
			return errRedeemNotFound
		}
		if code.Status != models.RedeemStatusUnused && code.Status != models.RedeemStatusActive {
			return errRedeemUsed
		}
		if code.ExpiresAt != nil && time.Now().After(*code.ExpiresAt) {
			return errRedeemExpired
		}
		if code.UsedCount >= code.MaxUses {
			return errRedeemLimit
		}

		// 原子抢占使用次数（条件更新 + 行数校验）。
		// SQLite 不支持 SELECT ... FOR UPDATE，原先的行锁在默认数据库下无效；
		// 改为按「旧 used_count」条件更新，并发兑换同一卡密时只有一方 RowsAffected=1。
		newCount := code.UsedCount + 1
		newStatus := code.Status
		if newCount >= code.MaxUses {
			newStatus = models.RedeemStatusUsed
		}
		claimRes := tx.Model(&models.RedeemCode{}).
			Where("id = ? AND status IN ? AND used_count = ?", code.ID, []string{models.RedeemStatusUnused, models.RedeemStatusActive}, code.UsedCount).
			Updates(map[string]interface{}{"used_count": newCount, "status": newStatus})
		if claimRes.Error != nil {
			return claimRes.Error
		}
		if claimRes.RowsAffected == 0 {
			return errRedeemConflict
		}

		if code.Type == "balance" {
			// Get user balance before update
			var user models.User
			if err := tx.First(&user, userID).Error; err != nil {
				return err
			}
			balanceBefore := user.Balance

			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				UpdateColumn("balance", gorm.Expr("balance + ?", code.Value)).Error; err != nil {
				return err
			}

			// Create balance log
			desc := "兑换卡密获得余额"
			if err := tx.Create(&models.BalanceLog{
				UserID:        userID,
				ChangeType:    "redeem",
				Amount:        code.Value,
				BalanceBefore: balanceBefore,
				BalanceAfter:  balanceBefore + code.Value,
				Description:   &desc,
			}).Error; err != nil {
				return err
			}
		}

		if code.Type == "duration" || code.Type == "package" {
			// Duration: value = days to add; Package: use linked package
			var sub models.Subscription
			if err := tx.Where("user_id = ?", userID).First(&sub).Error; err != nil {
				// Create new subscription
				subURL := utils.GenerateHexToken()
				sub = models.Subscription{
					UserID:          userID,
					SubscriptionURL: subURL,
					DeviceLimit:     3,
					IsActive:        true,
					Status:          models.SubStatusActive,
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
				// Extend existing subscription（原子延长，防并发兑换丢更新）
				days := int(code.Value)
				if code.Type == "package" && code.PackageID != nil {
					var pkg models.Package
					if tx.First(&pkg, *code.PackageID).Error == nil {
						days = pkg.DurationDays
					}
				}
				if _, err := services.ExtendSubscriptionExpiry(tx, sub.ID, sub.ExpireTime, days); err != nil {
					if errors.Is(err, services.ErrSubscriptionConflict) {
						return errRedeemConflict
					}
					return err
				}
				if err := tx.Model(&sub).Updates(map[string]interface{}{
					"is_active": true, "status": models.SubStatusActive,
				}).Error; err != nil {
					return err
				}
			}
		}

		ip := c.ClientIP()
		return tx.Create(&models.RedeemRecord{
			RedeemCodeID: code.ID, UserID: userID, Code: code.Code,
			Type: code.Type, Value: code.Value,
			IPAddress: &ip,
		}).Error
	})

	if err != nil {
		switch err {
		case errRedeemNotFound:
			utils.NotFound(c, "卡密不存在")
		case errRedeemUsed:
			utils.BadRequest(c, "卡密已使用或已失效")
		case errRedeemExpired:
			utils.BadRequest(c, "卡密已过期")
		case errRedeemLimit:
			utils.BadRequest(c, "卡密使用次数已达上限")
		case errRedeemConflict:
			utils.BadRequest(c, "卡密正在被使用，请稍后重试")
		default:
			utils.InternalError(c, "兑换失败")
		}
		return
	}
	utils.SuccessMessage(c, "兑换成功")
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
