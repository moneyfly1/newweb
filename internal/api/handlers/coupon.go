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
	"gorm.io/gorm"
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
	if err := db.Where("code = ? AND status = ?", code, models.CouponStatusActive).First(&coupon).Error; err != nil {
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
	case models.CouponTypeDiscount:
		// 百分比折扣：value 为百分数（如 10 = 10%），按分位舍入防浮点误差
		discountAmount = math.Round(orderAmount*coupon.DiscountValue) / 100
	case models.CouponTypeFixed:
		discountAmount = coupon.DiscountValue
	case models.CouponTypeFreeDays:
		discountAmount = 0
	}

	if coupon.MaxDiscount != nil && discountAmount > *coupon.MaxDiscount {
		discountAmount = *coupon.MaxDiscount
	}
	if discountAmount > orderAmount {
		discountAmount = orderAmount
	}
	discountAmount = utils.Round2(discountAmount)

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

	// 补充优惠券名称/面值/状态与订单号（批量查询避免 N+1）
	type MyCouponItem struct {
		models.CouponUsage
		CouponName string  `json:"coupon_name"`
		Code       string  `json:"code"`
		Type       string  `json:"coupon_type"`
		Value      float64 `json:"coupon_value"`
		Status     string  `json:"coupon_status"`
		OrderNo    string  `json:"order_no"`
	}

	items := make([]MyCouponItem, 0, len(usages))
	if len(usages) == 0 {
		utils.Success(c, items)
		return
	}

	couponIDs := make([]uint, 0, len(usages))
	orderIDs := make([]int64, 0, len(usages))
	for _, u := range usages {
		couponIDs = append(couponIDs, u.CouponID)
		if u.OrderID != nil {
			orderIDs = append(orderIDs, *u.OrderID)
		}
	}

	couponMap := make(map[uint]models.Coupon)
	var coupons []models.Coupon
	db.Select("id, name, code, type, discount_value, status").Where("id IN ?", couponIDs).Find(&coupons)
	for _, cp := range coupons {
		couponMap[cp.ID] = cp
	}

	orderMap := make(map[int64]string)
	if len(orderIDs) > 0 {
		var orders []models.Order
		db.Select("id, order_no").Where("id IN ?", orderIDs).Find(&orders)
		for _, o := range orders {
			orderMap[int64(o.ID)] = o.OrderNo
		}
	}

	for _, u := range usages {
		item := MyCouponItem{CouponUsage: u}
		if cp, ok := couponMap[u.CouponID]; ok {
			item.CouponName = cp.Name
			item.Code = cp.Code
			item.Type = cp.Type
			item.Value = cp.DiscountValue
			item.Status = cp.Status
		}
		if u.OrderID != nil {
			item.OrderNo = orderMap[*u.OrderID]
		}
		items = append(items, item)
	}

	utils.Success(c, items)
}

// checkCouponPerUserInTx 在事务内复核该用户对该优惠券的使用次数是否已达上限。
// 必须在创建 CouponUsage 前调用（同一事务），SQLite 写事务串行化保证并发下计数可靠，
// 防止并发下单绕过 MaxUsesPerUser 导致 0 元订单。
func checkCouponPerUserInTx(tx *gorm.DB, couponID uint, userID uint, maxUsesPerUser int) error {
	if maxUsesPerUser <= 0 {
		return nil
	}
	var used int64
	if err := tx.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", couponID, userID).Count(&used).Error; err != nil {
		return err
	}
	if int(used) >= maxUsesPerUser {
		return fmt.Errorf("您已达到该优惠券的使用上限")
	}
	return nil
}
