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
	utils.Success(c, codes)
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
	var totalInvited int64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ?", userID).Count(&totalInvited)
	var totalReward float64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ? AND inviter_reward_given = ?", userID, true).
		Select("COALESCE(SUM(inviter_reward_amount), 0)").Scan(&totalReward)
	utils.Success(c, gin.H{"total_invited": totalInvited, "total_reward": totalReward})
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
