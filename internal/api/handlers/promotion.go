package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// ==================== 促销活动（对齐 myweb promotions）====================

// GetActivePromotions 用户端：当前进行中的活动。
func GetActivePromotions(c *gin.Context) {
	now := time.Now()
	var promos []models.Promotion
	database.GetDB().
		Where("is_active = ? AND start_time <= ? AND end_time >= ?", true, now, now).
		Order("start_time DESC").
		Find(&promos)
	utils.Success(c, promos)
}

// GetMyPromotionParticipations 用户端：我的参与记录。
func GetMyPromotionParticipations(c *gin.Context) {
	userID := c.GetUint("user_id")
	var rows []models.PromotionParticipation
	database.GetDB().Where("user_id = ?", userID).Order("id DESC").Limit(100).Find(&rows)
	utils.Success(c, rows)
}

// AdminListPromotions 管理端：活动列表（含停用）。
func AdminListPromotions(c *gin.Context) {
	var promos []models.Promotion
	database.GetDB().Order("id DESC").Find(&promos)
	utils.Success(c, promos)
}

type promotionRequest struct {
	Name          string  `json:"name" binding:"required"`
	Type          string  `json:"type" binding:"required"`
	DiscountType  string  `json:"discount_type" binding:"required"`
	DiscountValue float64 `json:"discount_value" binding:"required"`
	MinAmount     float64 `json:"min_amount"`
	MaxDiscount   float64 `json:"max_discount"`
	StartTime     string  `json:"start_time" binding:"required"`
	EndTime       string  `json:"end_time" binding:"required"`
	IsActive      *bool   `json:"is_active"`
	Description   string  `json:"description"`
}

func parsePromotionTime(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

// AdminCreatePromotion 管理端：创建活动。
func AdminCreatePromotion(c *gin.Context) {
	var req promotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	start, err1 := parsePromotionTime(req.StartTime)
	end, err2 := parsePromotionTime(req.EndTime)
	if err1 != nil || err2 != nil || !end.After(start) {
		utils.BadRequest(c, "活动时间无效（结束时间必须晚于开始时间）")
		return
	}
	if req.DiscountType != "percentage" && req.DiscountType != "fixed" {
		utils.BadRequest(c, "折扣类型只支持 percentage / fixed")
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	promo := models.Promotion{
		Name: req.Name, Type: req.Type, DiscountType: req.DiscountType,
		DiscountValue: req.DiscountValue, MinAmount: req.MinAmount, MaxDiscount: req.MaxDiscount,
		StartTime: start, EndTime: end, IsActive: active, Description: req.Description,
	}
	if err := database.GetDB().Create(&promo).Error; err != nil {
		utils.InternalError(c, "创建活动失败")
		return
	}
	utils.CreateAuditLog(c, "create_promotion", "promotion", promo.ID, fmt.Sprintf("创建促销活动: %s", promo.Name))
	utils.Success(c, promo)
}

// AdminUpdatePromotion 管理端：更新活动。
func AdminUpdatePromotion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的活动ID")
		return
	}
	db := database.GetDB()
	var promo models.Promotion
	if err := db.First(&promo, id).Error; err != nil {
		utils.NotFound(c, "活动不存在")
		return
	}
	var req promotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	start, err1 := parsePromotionTime(req.StartTime)
	end, err2 := parsePromotionTime(req.EndTime)
	if err1 != nil || err2 != nil || !end.After(start) {
		utils.BadRequest(c, "活动时间无效（结束时间必须晚于开始时间）")
		return
	}
	updates := map[string]interface{}{
		"name": req.Name, "type": req.Type, "discount_type": req.DiscountType,
		"discount_value": req.DiscountValue, "min_amount": req.MinAmount,
		"max_discount": req.MaxDiscount, "start_time": start, "end_time": end,
		"description": req.Description,
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if err := db.Model(&models.Promotion{}).Where("id = ?", promo.ID).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新活动失败")
		return
	}
	utils.CreateAuditLog(c, "update_promotion", "promotion", uint(id), fmt.Sprintf("更新促销活动: %s", req.Name))
	utils.SuccessMessage(c, "更新成功")
}

// AdminDeletePromotion 管理端：删除活动。
func AdminDeletePromotion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的活动ID")
		return
	}
	db := database.GetDB()
	if err := db.Delete(&models.Promotion{}, id).Error; err != nil {
		utils.InternalError(c, "删除活动失败")
		return
	}
	utils.CreateAuditLog(c, "delete_promotion", "promotion", uint(id), "删除促销活动")
	utils.SuccessMessage(c, "已删除")
}

// 确保 gorm 引用（避免误删 import）
var _ = gorm.ErrRecordNotFound
