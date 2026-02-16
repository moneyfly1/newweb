package handlers

import (
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

func ListInviteCodes(c *gin.Context) {
	userID := c.GetUint("user_id")
	var codes []models.InviteCode
	database.GetDB().Where("user_id = ?", userID).Order("created_at DESC").Find(&codes)

	now := time.Now()
	type codeResponse struct {
		models.InviteCode
		Status string `json:"status"`
	}
	var result []codeResponse
	for _, code := range codes {
		status := "active"
		if !code.IsActive {
			status = "disabled"
		} else if code.ExpiresAt != nil && now.After(*code.ExpiresAt) {
			status = "expired"
		} else if code.MaxUses != nil && code.UsedCount >= int(*code.MaxUses) {
			status = "exhausted"
		}
		result = append(result, codeResponse{InviteCode: code, Status: status})
	}
	utils.Success(c, result)
}

func CreateInviteCode(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		MaxUses       *int64  `json:"max_uses"`
		ExpiresInDays *int    `json:"expires_in_days"`
		InviterReward float64 `json:"inviter_reward"`
		InviteeReward float64 `json:"invitee_reward"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()
	codeStr := ""
	for i := 0; i < 5; i++ {
		codeStr = utils.GenerateRandomString(8)
		var count int64
		db.Model(&models.InviteCode{}).Where("UPPER(code) = UPPER(?)", codeStr).Count(&count)
		if count == 0 {
			break
		}
	}
	code := models.InviteCode{
		Code:          codeStr,
		UserID:        userID,
		RewardType:    "balance",
		InviterReward: req.InviterReward,
		InviteeReward: req.InviteeReward,
		NewUserOnly:   true,
		IsActive:      true,
	}
	if req.MaxUses != nil {
		code.MaxUses = req.MaxUses
	}
	if req.ExpiresInDays != nil {
		exp := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		code.ExpiresAt = &exp
	}
	db.Create(&code)
	utils.Success(c, code)
}

func GetInviteStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	db := database.GetDB()

	// Total invited users
	var totalInvites int64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ?", userID).Count(&totalInvites)

	// Purchased invites (has first order)
	var purchasedInvites int64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ? AND invitee_first_order_id IS NOT NULL", userID).Count(&purchasedInvites)

	// Total reward earned
	var totalReward float64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ? AND inviter_reward_given = ?", userID, true).
		Select("COALESCE(SUM(inviter_reward_amount), 0)").Scan(&totalReward)

	// Recent invites with invitee info
	var relations []models.InviteRelation
	db.Where("inviter_id = ?", userID).Order("created_at DESC").Limit(20).Find(&relations)

	type recentInvite struct {
		ID                uint    `json:"id"`
		InviteeUsername   string  `json:"invitee_username"`
		InviteeEmail      string  `json:"invitee_email"`
		RegisteredAt      string  `json:"registered_at"`
		HasPurchased      bool    `json:"has_purchased"`
		ConsumptionAmount float64 `json:"consumption_amount"`
		RewardStatus      string  `json:"reward_status"`
		RewardAmount      float64 `json:"reward_amount"`
	}

	var recentInvites []recentInvite
	for _, r := range relations {
		var user models.User
		db.Select("username, email").Where("id = ?", r.InviteeID).First(&user)
		status := "pending"
		if r.InviterRewardGiven {
			status = "paid"
		}
		recentInvites = append(recentInvites, recentInvite{
			ID:                r.ID,
			InviteeUsername:   user.Username,
			InviteeEmail:      user.Email,
			RegisteredAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
			HasPurchased:      r.InviteeFirstOrderID != nil,
			ConsumptionAmount: r.InviteeTotalConsumption,
			RewardStatus:      status,
			RewardAmount:      r.InviterRewardAmount,
		})
	}

	utils.Success(c, gin.H{
		"total_invites":      totalInvites,
		"registered_invites": totalInvites,
		"purchased_invites":  purchasedInvites,
		"total_reward":       totalReward,
		"recent_invites":     recentInvites,
	})
}

func GetMyCodes(c *gin.Context) {
	ListInviteCodes(c)
}

func DeleteInviteCode(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	db := database.GetDB()
	var code models.InviteCode
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&code).Error; err != nil {
		utils.NotFound(c, "邀请码不存在")
		return
	}
	db.Delete(&code)
	utils.SuccessMessage(c, "邀请码已删除")
}

func ValidateInviteCode(c *gin.Context) {
	code := c.Param("code")
	var invite models.InviteCode
	if err := database.GetDB().Where("UPPER(code) = UPPER(?) AND is_active = ?", code, true).First(&invite).Error; err != nil {
		utils.NotFound(c, "邀请码无效")
		return
	}
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		utils.BadRequest(c, "邀请码已过期")
		return
	}
	if invite.MaxUses != nil && invite.UsedCount >= int(*invite.MaxUses) {
		utils.BadRequest(c, "邀请码已达使用上限")
		return
	}
	utils.Success(c, gin.H{"valid": true, "invitee_reward": invite.InviteeReward})
}
