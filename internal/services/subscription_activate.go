package services

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"gorm.io/gorm"
)

// ActivateSubscription creates or extends a subscription after successful payment.
// It also sends payment success email and notifies admin.
func ActivateSubscription(db *gorm.DB, order *models.Order, paymentMethod string) {
	var pkg models.Package
	if err := db.First(&pkg, order.PackageID).Error; err != nil {
		return
	}

	var sub models.Subscription
	if err := db.Where("user_id = ?", order.UserID).First(&sub).Error; err != nil {
		// Create new subscription
		pkgID := int64(pkg.ID)
		sub = models.Subscription{
			UserID:          order.UserID,
			PackageID:       &pkgID,
			SubscriptionURL: utils.GenerateRandomString(32),
			DeviceLimit:     pkg.DeviceLimit,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      time.Now().AddDate(0, 0, pkg.DurationDays),
		}
		db.Create(&sub)
		utils.CreateSubscriptionLog(sub.ID, order.UserID, "activate", "system", nil, fmt.Sprintf("购买套餐激活订阅: %s", pkg.Name), nil, nil)
	} else {
		// Extend existing subscription
		newExpire := sub.ExpireTime
		if newExpire.Before(time.Now()) {
			newExpire = time.Now()
		}
		newExpire = newExpire.AddDate(0, 0, pkg.DurationDays)
		pkgID := int64(pkg.ID)
		db.Model(&sub).Updates(map[string]interface{}{
			"package_id":   &pkgID,
			"device_limit": pkg.DeviceLimit,
			"expire_time":  newExpire,
			"is_active":    true,
			"status":       "active",
		})
		utils.CreateSubscriptionLog(sub.ID, order.UserID, "extend", "system", nil, fmt.Sprintf("购买套餐续期订阅: %s, +%d天", pkg.Name, pkg.DurationDays), nil, nil)
	}

	// Send payment success email + notify admin
	var user models.User
	if db.First(&user, order.UserID).Error == nil {
		payAmount := fmt.Sprintf("%.2f", order.Amount)
		if order.FinalAmount != nil {
			payAmount = fmt.Sprintf("%.2f", *order.FinalAmount)
		}
		emailSubject, emailBody := RenderEmail("payment_success", map[string]string{
			"order_no": order.OrderNo, "amount": payAmount, "package_name": pkg.Name,
		})
		go QueueEmail(user.Email, emailSubject, emailBody, "payment_success")
		go NotifyAdmin("payment_success", map[string]string{
			"username": user.Username, "order_no": order.OrderNo, "package_name": pkg.Name, "amount": payAmount,
		})
	}

	// 邀请人返佣
	distributeInviteCommission(db, order)
}

// distributeInviteCommission gives commission to the inviter when invitee makes a purchase.
func distributeInviteCommission(db *gorm.DB, order *models.Order) {
	// Find invite relation for this user
	var relation models.InviteRelation
	if err := db.Where("invitee_id = ?", order.UserID).First(&relation).Error; err != nil {
		return
	}

	// Get commission rate from system config
	rateStr := utils.GetSetting("invite_commission_rate")
	if rateStr == "" {
		return
	}
	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil || rate <= 0 {
		return
	}

	// Calculate commission
	payAmount := order.Amount
	if order.FinalAmount != nil {
		payAmount = *order.FinalAmount
	}
	commission := math.Round(payAmount*rate/100*100) / 100
	if commission <= 0 {
		return
	}

	// Credit inviter
	var inviter models.User
	if err := db.First(&inviter, relation.InviterID).Error; err != nil {
		return
	}
	db.Model(&inviter).UpdateColumn("balance", gorm.Expr("balance + ?", commission))

	// Log balance change
	desc := fmt.Sprintf("邀请用户购买返佣 (订单: %s, 比例: %.1f%%)", order.OrderNo, rate)
	db.Create(&models.BalanceLog{
		UserID:         inviter.ID,
		ChangeType:     "invite_commission",
		Amount:         commission,
		BalanceBefore:  inviter.Balance,
		BalanceAfter:   inviter.Balance + commission,
		RelatedOrderID: func() *int64 { id := int64(order.ID); return &id }(),
		Description:    &desc,
	})

	// Log commission
	orderID := int64(order.ID)
	relationID := int64(relation.ID)
	db.Create(&models.CommissionLog{
		InviterID:        relation.InviterID,
		InviteeID:        relation.InviteeID,
		InviteRelationID: &relationID,
		CommissionType:   "purchase",
		Amount:           commission,
		RelatedOrderID:   &orderID,
		Status:           "settled",
		Description:      &desc,
	})

	// Update relation consumption total
	db.Model(&relation).Updates(map[string]interface{}{
		"invitee_total_consumption": gorm.Expr("invitee_total_consumption + ?", payAmount),
		"invitee_first_order_id":    gorm.Expr("COALESCE(invitee_first_order_id, ?)", order.ID),
	})
}
