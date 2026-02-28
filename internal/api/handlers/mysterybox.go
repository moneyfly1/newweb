package handlers

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ── Admin: mystery box pools & prizes ──

// AdminListMysteryBoxPools GET /admin/mystery-box/pools
func AdminListMysteryBoxPools(c *gin.Context) {
	db := database.GetDB()
	var pools []models.MysteryBoxPool
	db.Preload("Prizes").Order("sort_order ASC, id ASC").Find(&pools)
	utils.Success(c, pools)
}

// AdminCreateMysteryBoxPool POST /admin/mystery-box/pools
func AdminCreateMysteryBoxPool(c *gin.Context) {
	var pool models.MysteryBoxPool
	if err := c.ShouldBindJSON(&pool); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if err := database.GetDB().Create(&pool).Error; err != nil {
		utils.InternalError(c, "创建奖池失败")
		return
	}
	utils.Success(c, pool)
}

// AdminUpdateMysteryBoxPool PUT /admin/mystery-box/pools/:id
func AdminUpdateMysteryBoxPool(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	var pool models.MysteryBoxPool
	if err := db.First(&pool, id).Error; err != nil {
		utils.NotFound(c, "奖池不存在")
		return
	}
	var req models.MysteryBoxPool
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	updates := map[string]interface{}{
		"name": req.Name, "description": req.Description, "price": req.Price,
		"is_active": req.IsActive, "sort_order": req.SortOrder, "min_level": req.MinLevel,
		"min_balance": req.MinBalance, "max_opens_per_day": req.MaxOpensPerDay,
		"max_opens_total": req.MaxOpensTotal, "start_time": req.StartTime, "end_time": req.EndTime,
	}
	if err := db.Model(&pool).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新奖池失败")
		return
	}
	db.Preload("Prizes").First(&pool, id)
	utils.Success(c, pool)
}

// AdminDeleteMysteryBoxPool DELETE /admin/mystery-box/pools/:id
func AdminDeleteMysteryBoxPool(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("pool_id = ?", id).Delete(&models.MysteryBoxPrize{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.MysteryBoxPool{}, id).Error
	}); err != nil {
		utils.InternalError(c, "删除奖池失败")
		return
	}
	utils.SuccessMessage(c, "删除成功")
}

// AdminAddPrize POST /admin/mystery-box/pools/:id/prizes
func AdminAddPrize(c *gin.Context) {
	poolID, err := strconv.Atoi(c.Param("id"))
	if err != nil || poolID <= 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	var pool models.MysteryBoxPool
	if err := db.First(&pool, poolID).Error; err != nil {
		utils.NotFound(c, "奖池不存在")
		return
	}
	var prize models.MysteryBoxPrize
	if err := c.ShouldBindJSON(&prize); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	prize.PoolID = uint(poolID)
	if err := db.Create(&prize).Error; err != nil {
		utils.InternalError(c, "添加奖品失败")
		return
	}
	utils.Success(c, prize)
}

// AdminUpdatePrize PUT /admin/mystery-box/prizes/:id
func AdminUpdatePrize(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	var prize models.MysteryBoxPrize
	if err := db.First(&prize, id).Error; err != nil {
		utils.NotFound(c, "奖品不存在")
		return
	}
	var req models.MysteryBoxPrize
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	updates := map[string]interface{}{
		"name": req.Name, "type": req.Type, "value": req.Value,
		"weight": req.Weight, "stock": req.Stock, "image_url": req.ImageURL,
	}
	if err := db.Model(&prize).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新奖品失败")
		return
	}
	db.First(&prize, id)
	utils.Success(c, prize)
}

// AdminDeletePrize DELETE /admin/mystery-box/prizes/:id
func AdminDeletePrize(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.BadRequest(c, "无效的ID")
		return
	}
	if err := database.GetDB().Delete(&models.MysteryBoxPrize{}, id).Error; err != nil {
		utils.InternalError(c, "删除奖品失败")
		return
	}
	utils.SuccessMessage(c, "删除成功")
}

// AdminGetMysteryBoxStats GET /admin/mystery-box/stats
func AdminGetMysteryBoxStats(c *gin.Context) {
	db := database.GetDB()
	var totalOpens int64
	db.Model(&models.MysteryBoxRecord{}).Count(&totalOpens)
	var totalRevenue float64
	db.Model(&models.MysteryBoxRecord{}).Select("COALESCE(SUM(cost), 0)").Scan(&totalRevenue)
	var totalPrizeValue float64
	db.Model(&models.MysteryBoxRecord{}).Select("COALESCE(SUM(prize_value), 0)").Scan(&totalPrizeValue)
	type PrizeDist struct {
		PrizeType string  `json:"prize_type"`
		Count     int64   `json:"count"`
		TotalVal  float64 `json:"total_value"`
	}
	var distribution []PrizeDist
	db.Model(&models.MysteryBoxRecord{}).
		Select("prize_type, COUNT(*) as count, COALESCE(SUM(prize_value), 0) as total_val").
		Group("prize_type").Find(&distribution)
	utils.Success(c, gin.H{
		"total_opens": totalOpens, "total_revenue": totalRevenue,
		"total_prize_value": totalPrizeValue, "prize_distribution": distribution,
	})
}

// ── User: mystery box ──

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
	db.Where("is_active = ?", true).Order("sort_order ASC, id ASC").Preload("Prizes").Find(&pools)
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
	if pool.MinBalance != nil && user.Balance < *pool.MinBalance {
		utils.BadRequest(c, fmt.Sprintf("余额不足，需要至少 %.2f", *pool.MinBalance))
		return
	}
	if user.Balance < pool.Price {
		utils.BadRequest(c, "余额不足，无法开启盲盒")
		return
	}
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

	prize, err := weightedRandomPrize(pool.Prizes)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var couponCode string
	txErr := db.Transaction(func(tx *gorm.DB) error {
		var lockedUser models.User
		if err := tx.First(&lockedUser, userID).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}
		balanceBefore := lockedUser.Balance
		result := tx.Model(&models.User{}).Where("id = ? AND balance >= ?", userID, pool.Price).
			Update("balance", gorm.Expr("balance - ?", pool.Price))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("余额不足")
		}
		deductDesc := fmt.Sprintf("开启盲盒「%s」", pool.Name)
		if err := tx.Create(&models.BalanceLog{
			UserID: userID, ChangeType: "mystery_box", Amount: -pool.Price,
			BalanceBefore: balanceBefore, BalanceAfter: balanceBefore - pool.Price, Description: &deductDesc,
		}).Error; err != nil {
			return err
		}

		switch prize.Type {
		case "balance":
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("balance", gorm.Expr("balance + ?", prize.Value)).Error; err != nil {
				return err
			}
			rewardDesc := fmt.Sprintf("盲盒奖品「%s」余额奖励", prize.Name)
			if err := tx.Create(&models.BalanceLog{
				UserID: userID, ChangeType: "mystery_box_reward", Amount: prize.Value,
				BalanceBefore: balanceBefore - pool.Price, BalanceAfter: balanceBefore - pool.Price + prize.Value, Description: &rewardDesc,
			}).Error; err != nil {
				return err
			}
		case "subscription_days":
			days := int(prize.Value)
			var sub models.Subscription
			if err := tx.Where("user_id = ?", userID).First(&sub).Error; err != nil {
				sub = models.Subscription{
					UserID: userID, SubscriptionURL: utils.GenerateRandomString(32),
					DeviceLimit: 3, IsActive: true, Status: "active",
					ExpireTime: time.Now().AddDate(0, 0, days),
				}
				if err := tx.Create(&sub).Error; err != nil {
					return err
				}
			} else {
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
			qty := int64(1)
			coupon := models.Coupon{
				Code: couponCode, Name: fmt.Sprintf("盲盒奖品-%s", prize.Name),
				Description: fmt.Sprintf("盲盒「%s」获得的优惠券", pool.Name),
				Type: "fixed", DiscountValue: prize.Value,
				ValidFrom: validFrom, ValidUntil: validUntil,
				MaxUsesPerUser: 1, Status: "active", TotalQuantity: &qty,
			}
			if err := tx.Create(&coupon).Error; err != nil {
				return err
			}
		case "nothing":
		}

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
		return tx.Create(&models.MysteryBoxRecord{
			UserID: userID, PoolID: pool.ID, PrizeID: prize.ID,
			PrizeName: prize.Name, PrizeType: prize.Type, PrizeValue: prize.Value, Cost: pool.Price,
		}).Error
	})

	if txErr != nil {
		log.Printf("[mystery_box] 开启失败 user=%d pool=%d: %v", userID, pool.ID, txErr)
		switch txErr.Error() {
		case "余额不足":
			utils.BadRequest(c, "余额不足，无法开启盲盒")
		case "奖品库存不足":
			utils.BadRequest(c, "奖品库存不足，请稍后再试")
		default:
			utils.InternalError(c, "开启盲盒失败")
		}
		return
	}
	resp := gin.H{"prize_name": prize.Name, "prize_type": prize.Type, "prize_value": prize.Value, "cost": pool.Price}
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

// ── Helpers ──

func weightedRandomPrize(prizes []models.MysteryBoxPrize) (*models.MysteryBoxPrize, error) {
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
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	r := int(n.Int64())
	for i := range available {
		r -= available[i].Weight
		if r < 0 {
			return &available[i], nil
		}
	}
	return &available[len(available)-1], nil
}