package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListMysteryBoxPools GET /mystery-box/pools
func ListMysteryBoxPools(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.InternalError(c, "用户不存在")
		return
	}

	now := time.Now()
	var pools []models.MysteryBoxPool
	query := db.Where("is_active = ?", true).Order("sort_order ASC, id ASC")
	query.Preload("Prizes").Find(&pools)

	// 过滤条件
	var result []models.MysteryBoxPool
	for _, pool := range pools {
		if pool.StartTime != nil && now.Before(*pool.StartTime) {
			continue
		}
		if pool.EndTime != nil && now.After(*pool.EndTime) {
			continue
		}
		if pool.MinLevel != nil {
			var userLevel models.UserLevel
			if user.UserLevelID != nil {
				db.First(&userLevel, *user.UserLevelID)
			}
			if userLevel.LevelOrder < int(*pool.MinLevel) {
				continue
			}
		}
		result = append(result, pool)
	}

	utils.Success(c, result)
}

// OpenMysteryBox POST /mystery-box/open
func OpenMysteryBox(c *gin.Context) {
	var req struct {
		PoolID uint `json:"pool_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	userID := c.GetUint("user_id")
	db := database.GetDB()

	var pool models.MysteryBoxPool
	if err := db.Preload("Prizes").First(&pool, req.PoolID).Error; err != nil {
		utils.NotFound(c, "奖池不存在")
		return
	}
	if !pool.IsActive {
		utils.BadRequest(c, "该奖池已关闭")
		return
	}

	now := time.Now()
	if pool.StartTime != nil && now.Before(*pool.StartTime) {
		utils.BadRequest(c, "该奖池尚未开放")
		return
	}
	if pool.EndTime != nil && now.After(*pool.EndTime) {
		utils.BadRequest(c, "该奖池已结束")
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.InternalError(c, "用户不存在")
		return
	}

	// 检查等级
	if pool.MinLevel != nil {
		var userLevel models.UserLevel
		if user.UserLevelID != nil {
			db.First(&userLevel, *user.UserLevelID)
		}
		if userLevel.LevelOrder < int(*pool.MinLevel) {
			utils.BadRequest(c, "用户等级不足")
			return
		}
	}

	// 检查最低余额
	if pool.MinBalance != nil && user.Balance < *pool.MinBalance {
		utils.BadRequest(c, fmt.Sprintf("余额不足，需要至少 %.2f", *pool.MinBalance))
		return
	}

	// 检查余额是否够支付价格
	if user.Balance < pool.Price {
		utils.BadRequest(c, "余额不足，无法开启盲盒")
		return
	}

	// 检查每日限制
	if pool.MaxOpensPerDay != nil {
		today := now.Format("2006-01-02")
		var todayCount int64
		db.Model(&models.MysteryBoxRecord{}).
			Where("user_id = ? AND pool_id = ? AND DATE(created_at) = ?", userID, pool.ID, today).
			Count(&todayCount)
		if int(todayCount) >= *pool.MaxOpensPerDay {
			utils.BadRequest(c, "今日开启次数已达上限")
			return
		}
	}

	// 检查总次数限制
	if pool.MaxOpensTotal != nil {
		var totalCount int64
		db.Model(&models.MysteryBoxRecord{}).
			Where("user_id = ? AND pool_id = ?", userID, pool.ID).
			Count(&totalCount)
		if int(totalCount) >= *pool.MaxOpensTotal {
			utils.BadRequest(c, "开启次数已达上限")
			return
		}
	}

	// 加权随机抽奖
	prize, err := weightedRandomPrize(pool.Prizes)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 事务：扣费 + 发奖 + 记录
	var couponCode string
	txErr := db.Transaction(func(tx *gorm.DB) error {
		// 读取事务内最新余额
		var lockedUser models.User
		if err := tx.First(&lockedUser, userID).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}

		balanceBefore := lockedUser.Balance

		// 原子扣款：WHERE balance >= price 保证不会超扣
		result := tx.Model(&models.User{}).Where("id = ? AND balance >= ?", userID, pool.Price).
			Update("balance", gorm.Expr("balance - ?", pool.Price))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("余额不足")
		}

		// 扣费日志
		deductDesc := fmt.Sprintf("开启盲盒「%s」", pool.Name)
		if err := tx.Create(&models.BalanceLog{
			UserID:        userID,
			ChangeType:    "mystery_box",
			Amount:        -pool.Price,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceBefore - pool.Price,
			Description:   &deductDesc,
		}).Error; err != nil {
			return err
		}

		// 发放奖品
		switch prize.Type {
		case "balance":
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("balance", gorm.Expr("balance + ?", prize.Value)).Error; err != nil {
				return err
			}
			rewardDesc := fmt.Sprintf("盲盒奖品「%s」余额奖励", prize.Name)
			if err := tx.Create(&models.BalanceLog{
				UserID:        userID,
				ChangeType:    "mystery_box_reward",
				Amount:        prize.Value,
				BalanceBefore: balanceBefore - pool.Price,
				BalanceAfter:  balanceBefore - pool.Price + prize.Value,
				Description:   &rewardDesc,
			}).Error; err != nil {
				return err
			}

		case "subscription_days":
			days := int(prize.Value)
			var sub models.Subscription
			if err := tx.Where("user_id = ?", userID).First(&sub).Error; err != nil {
				// 没有订阅，创建新的
				subURL := utils.GenerateRandomString(32)
				sub = models.Subscription{
					UserID:          userID,
					SubscriptionURL: subURL,
					DeviceLimit:     3,
					IsActive:        true,
					Status:          "active",
					ExpireTime:      time.Now().AddDate(0, 0, days),
				}
				if err := tx.Create(&sub).Error; err != nil {
					return err
				}
			} else {
				// 已有订阅，延长到期时间
				newExpire := sub.ExpireTime
				if newExpire.Before(time.Now()) {
					newExpire = time.Now()
				}
				newExpire = newExpire.AddDate(0, 0, days)
				if err := tx.Model(&sub).Updates(map[string]interface{}{
					"expire_time": newExpire, "is_active": true, "status": "active",
				}).Error; err != nil {
					return err
				}
			}

		case "coupon":
			couponCode = "MB" + utils.GenerateRandomString(10)
			validFrom := time.Now()
			validUntil := validFrom.AddDate(0, 1, 0)
			coupon := models.Coupon{
				Code:           couponCode,
				Name:           fmt.Sprintf("盲盒奖品-%s", prize.Name),
				Description:    fmt.Sprintf("盲盒「%s」获得的优惠券", pool.Name),
				Type:           "fixed",
				DiscountValue:  prize.Value,
				ValidFrom:      validFrom,
				ValidUntil:     validUntil,
				MaxUsesPerUser: 1,
				Status:         "active",
			}
			qty := int64(1)
			coupon.TotalQuantity = &qty
			if err := tx.Create(&coupon).Error; err != nil {
				return err
			}

		case "nothing":
			// 谢谢参与
		}

		// 减少库存（行级条件更新，防止超卖）
		if prize.Stock != nil {
			result := tx.Model(&models.MysteryBoxPrize{}).Where("id = ? AND stock > 0", prize.ID).
				Update("stock", gorm.Expr("stock - 1"))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("奖品库存不足")
			}
		}

		// 创建开启记录
		return tx.Create(&models.MysteryBoxRecord{
			UserID:     userID,
			PoolID:     pool.ID,
			PrizeID:    prize.ID,
			PrizeName:  prize.Name,
			PrizeType:  prize.Type,
			PrizeValue: prize.Value,
			Cost:       pool.Price,
		}).Error
	})

	if txErr != nil {
		log.Printf("[mystery_box] 开启失败 user=%d pool=%d: %v", userID, pool.ID, txErr)
		errMsg := txErr.Error()
		if errMsg == "余额不足" {
			utils.BadRequest(c, "余额不足，无法开启盲盒")
		} else if errMsg == "奖品库存不足" {
			utils.BadRequest(c, "奖品库存不足，请稍后再试")
		} else {
			utils.InternalError(c, "开启盲盒失败")
		}
		return
	}

	resp := gin.H{
		"prize_name":  prize.Name,
		"prize_type":  prize.Type,
		"prize_value": prize.Value,
		"cost":        pool.Price,
	}
	// 优惠券奖品返回券码，让用户能看到并使用
	if prize.Type == "coupon" && couponCode != "" {
		resp["coupon_code"] = couponCode
	}
	utils.Success(c, resp)
}

// GetMysteryBoxHistory GET /mystery-box/history
func GetMysteryBoxHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	p := utils.GetPagination(c)
	db := database.GetDB()

	var total int64
	db.Model(&models.MysteryBoxRecord{}).Where("user_id = ?", userID).Count(&total)

	var items []models.MysteryBoxRecord
	db.Where("user_id = ?", userID).Order("created_at DESC").
		Offset(p.Offset()).Limit(p.PageSize).Find(&items)

	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

// weightedRandomPrize 加权随机选择奖品
func weightedRandomPrize(prizes []models.MysteryBoxPrize) (*models.MysteryBoxPrize, error) {
	// 过滤掉库存为0的奖品
	var available []models.MysteryBoxPrize
	for _, p := range prizes {
		if p.Stock != nil && *p.Stock <= 0 {
			continue
		}
		available = append(available, p)
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("奖池中没有可用奖品")
	}

	totalWeight := 0
	for _, p := range available {
		totalWeight += p.Weight
	}
	if totalWeight <= 0 {
		return nil, fmt.Errorf("奖池权重配置异常")
	}

	r := rand.Intn(totalWeight)
	for i := range available {
		r -= available[i].Weight
		if r < 0 {
			return &available[i], nil
		}
	}
	return &available[len(available)-1], nil
}
