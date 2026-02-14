package handlers

import (
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// VerifyCoupon checks whether a coupon code is valid and returns discount info.
func VerifyCoupon(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	db := database.GetDB()
	var coupon models.Coupon
	if err := db.Where("code = ?", req.Code).First(&coupon).Error; err != nil {
		utils.NotFound(c, "优惠券不存在")
		return
	}

	// Check status
	if coupon.Status != string(models.CouponStatusActive) {
		utils.BadRequest(c, "优惠券已失效")
		return
	}

	// Check date range
	now := time.Now()
	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		utils.BadRequest(c, "优惠券不在有效期内")
		return
	}

	// Check quantity
	if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
		utils.BadRequest(c, "优惠券已被领完")
		return
	}

	// Check per-user usage if authenticated
	userID := c.GetUint("user_id")
	if userID > 0 {
		var usageCount int64
		db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).Count(&usageCount)
		if int(usageCount) >= coupon.MaxUsesPerUser {
			utils.BadRequest(c, "您已达到该优惠券的使用上限")
			return
		}
	}

	utils.Success(c, gin.H{
		"id":             coupon.ID,
		"code":           coupon.Code,
		"name":           coupon.Name,
		"type":           coupon.Type,
		"discount_value": coupon.DiscountValue,
		"valid_from":     coupon.ValidFrom,
		"valid_until":    coupon.ValidUntil,
	})
}

// GetMyCoupons lists coupon usage records for the current user.
func GetMyCoupons(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var usages []models.CouponUsage
	db.Where("user_id = ?", userID).Order("used_at DESC").Find(&usages)

	utils.Success(c, usages)
}
