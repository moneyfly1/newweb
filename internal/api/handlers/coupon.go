package handlers

import (
	"fmt"
	"math"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// CouponValidationResult holds the result of coupon validation.
type CouponValidationResult struct {
	Coupon         *models.Coupon
	DiscountAmount float64
	Error          string
}

// ValidateAndApplyCoupon validates a coupon and calculates the discount.
// It checks: existence, status, date range, quantity, per-user usage,
// min amount, applicable packages, and calculates discount based on type.
func ValidateAndApplyCoupon(code string, userID uint, orderAmount float64, packageID uint) *CouponValidationResult {
	if code == "" {
		return nil
	}

	db := database.GetDB()
	var coupon models.Coupon
	if err := db.Where("code = ? AND status = ?", code, "active").First(&coupon).Error; err != nil {
		return &CouponValidationResult{Error: "优惠券不存在或已失效"}
	}

	now := time.Now()
	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		return &CouponValidationResult{Error: "优惠券不在有效期内"}
	}

	if coupon.TotalQuantity != nil && coupon.UsedQuantity >= int(*coupon.TotalQuantity) {
		return &CouponValidationResult{Error: "优惠券已被领完"}
	}

	var usageCount int64
	db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).Count(&usageCount)
	if int(usageCount) >= coupon.MaxUsesPerUser {
		return &CouponValidationResult{Error: "您已达到该优惠券的使用上限"}
	}

	if coupon.MinAmount != nil && orderAmount < *coupon.MinAmount {
		return &CouponValidationResult{Error: fmt.Sprintf("订单金额需满 %.2f 元才可使用此优惠券", *coupon.MinAmount)}
	}

	if coupon.ApplicablePackages != "" && packageID > 0 {
		allowed := strings.Split(coupon.ApplicablePackages, ",")
		pkgStr := fmt.Sprintf("%d", packageID)
		matched := false
		for _, a := range allowed {
			if strings.TrimSpace(a) == pkgStr {
				matched = true
				break
			}
		}
		if !matched {
			return &CouponValidationResult{Error: "此优惠券不适用于该套餐"}
		}
	}

	var discountAmount float64
	switch coupon.Type {
	case "discount":
		discountAmount = math.Round(orderAmount*coupon.DiscountValue) / 100
	case "fixed":
		discountAmount = coupon.DiscountValue
	case "free_days":
		discountAmount = 0
	}

	if coupon.MaxDiscount != nil && discountAmount > *coupon.MaxDiscount {
		discountAmount = *coupon.MaxDiscount
	}
	if discountAmount > orderAmount {
		discountAmount = orderAmount
	}
	discountAmount = math.Round(discountAmount*100) / 100

	return &CouponValidationResult{
		Coupon:         &coupon,
		DiscountAmount: discountAmount,
	}
}

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
	db.Where("user_id = ?", userID).Order("used_at DESC").Limit(200).Find(&usages)

	utils.Success(c, usages)
}
