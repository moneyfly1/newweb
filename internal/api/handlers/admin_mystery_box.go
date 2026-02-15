package handlers

import (
	"strconv"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	db := database.GetDB()
	if err := db.Create(&pool).Error; err != nil {
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
		"name":               req.Name,
		"description":        req.Description,
		"price":              req.Price,
		"is_active":          req.IsActive,
		"sort_order":         req.SortOrder,
		"min_level":          req.MinLevel,
		"min_balance":        req.MinBalance,
		"max_opens_per_day":  req.MaxOpensPerDay,
		"max_opens_total":    req.MaxOpensTotal,
		"start_time":         req.StartTime,
		"end_time":           req.EndTime,
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
		"name":      req.Name,
		"type":      req.Type,
		"value":     req.Value,
		"weight":    req.Weight,
		"stock":     req.Stock,
		"image_url": req.ImageURL,
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
	db := database.GetDB()
	if err := db.Delete(&models.MysteryBoxPrize{}, id).Error; err != nil {
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
		"total_opens":       totalOpens,
		"total_revenue":     totalRevenue,
		"total_prize_value": totalPrizeValue,
		"prize_distribution": distribution,
	})
}