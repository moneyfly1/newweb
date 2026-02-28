package handlers

import (
	"strconv"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ── Admin: invite codes ──

func AdminListInviteCodes(c *gin.Context) {
	db := database.GetDB()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	query := db.Model(&models.InviteCode{})
	if search != "" {
		query = query.Where("code LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	var total int64
	query.Count(&total)
	var codes []models.InviteCode
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&codes)
	type R struct {
		models.InviteCode
		Username string `json:"username"`
		Status   string `json:"status"`
	}
	var result []R
	for _, code := range codes {
		item := R{InviteCode: code}
		var u models.User
		if db.Select("username").First(&u, code.UserID).Error == nil {
			item.Username = u.Username
		}
		item.Status = inviteCodeStatus(code)
		result = append(result, item)
	}
	utils.Success(c, gin.H{"items": result, "total": total})
}

func AdminGetInviteStats(c *gin.Context) {
	db := database.GetDB()
	var totalCodes, activeCodes, totalRelations int64
	db.Model(&models.InviteCode{}).Count(&totalCodes)
	db.Model(&models.InviteCode{}).Where("is_active = ?", true).Count(&activeCodes)
	db.Model(&models.InviteRelation{}).Count(&totalRelations)
	var inviterReward, inviteeReward float64
	db.Model(&models.InviteRelation{}).Where("inviter_reward_given = ?", true).Select("COALESCE(SUM(inviter_reward_amount), 0)").Scan(&inviterReward)
	db.Model(&models.InviteRelation{}).Where("invitee_reward_given = ?", true).Select("COALESCE(SUM(invitee_reward_amount), 0)").Scan(&inviteeReward)
	utils.Success(c, gin.H{
		"total_codes": totalCodes, "active_codes": activeCodes,
		"total_invites": totalRelations, "total_inviter_reward": inviterReward, "total_invitee_reward": inviteeReward,
	})
}

func AdminDeleteInviteCode(c *gin.Context) {
	id := c.Param("id")
	if err := database.GetDB().Delete(&models.InviteCode{}, id).Error; err != nil {
		utils.InternalError(c, "删除失败")
		return
	}
	utils.SuccessMessage(c, "邀请码已删除")
}

func AdminToggleInviteCode(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var code models.InviteCode
	if err := db.First(&code, id).Error; err != nil {
		utils.NotFound(c, "邀请码不存在")
		return
	}
	code.IsActive = !code.IsActive
	db.Save(&code)
	msg := "已启用"
	if !code.IsActive {
		msg = "已禁用"
	}
	utils.SuccessMessage(c, "邀请码"+msg)
}

func AdminListInviteRelations(c *gin.Context) {
	db := database.GetDB()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	var total int64
	db.Model(&models.InviteRelation{}).Count(&total)
	var rels []models.InviteRelation
	db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rels)
	type RR struct {
		models.InviteRelation
		InviterUsername string `json:"inviter_username"`
		InviteeUsername string `json:"invitee_username"`
		Code           string `json:"invite_code"`
	}
	var result []RR
	for _, rel := range rels {
		item := RR{InviteRelation: rel}
		var inviter, invitee models.User
		if db.Select("username").First(&inviter, rel.InviterID).Error == nil {
			item.InviterUsername = inviter.Username
		}
		if db.Select("username").First(&invitee, rel.InviteeID).Error == nil {
			item.InviteeUsername = invitee.Username
		}
		var ic models.InviteCode
		if db.Select("code").First(&ic, rel.InviteCodeID).Error == nil {
			item.Code = ic.Code
		}
		result = append(result, item)
	}
	utils.Success(c, gin.H{"items": result, "total": total})
}

// ── User: invite codes ──

func ListInviteCodes(c *gin.Context) {
	userID := c.GetUint("user_id")
	var codes []models.InviteCode
	database.GetDB().Where("user_id = ?", userID).Order("created_at DESC").Find(&codes)
	type codeResponse struct {
		models.InviteCode
		Status string `json:"status"`
	}
	var result []codeResponse
	for _, code := range codes {
		result = append(result, codeResponse{InviteCode: code, Status: inviteCodeStatus(code)})
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
	// Bounds validation
	if req.InviterReward < 0 || req.InviterReward > 10000 {
		utils.BadRequest(c, "邀请人奖励需在 0 ~ 10000 之间")
		return
	}
	if req.InviteeReward < 0 || req.InviteeReward > 10000 {
		utils.BadRequest(c, "受邀人奖励需在 0 ~ 10000 之间")
		return
	}
	if req.MaxUses != nil && (*req.MaxUses < 1 || *req.MaxUses > 100000) {
		utils.BadRequest(c, "最大使用次数需在 1 ~ 100000 之间")
		return
	}
	if req.ExpiresInDays != nil && (*req.ExpiresInDays < 1 || *req.ExpiresInDays > 3650) {
		utils.BadRequest(c, "有效天数需在 1 ~ 3650 之间")
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
		Code: codeStr, UserID: userID, RewardType: "balance",
		InviterReward: req.InviterReward, InviteeReward: req.InviteeReward,
		NewUserOnly: true, IsActive: true,
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
	var totalInvites int64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ?", userID).Count(&totalInvites)
	var purchasedInvites int64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ? AND invitee_first_order_id IS NOT NULL", userID).Count(&purchasedInvites)
	var totalReward float64
	db.Model(&models.InviteRelation{}).Where("inviter_id = ? AND inviter_reward_given = ?", userID, true).
		Select("COALESCE(SUM(inviter_reward_amount), 0)").Scan(&totalReward)
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
			ID: r.ID, InviteeUsername: user.Username, InviteeEmail: user.Email,
			RegisteredAt: r.CreatedAt.Format("2006-01-02T15:04:05Z"),
			HasPurchased: r.InviteeFirstOrderID != nil, ConsumptionAmount: r.InviteeTotalConsumption,
			RewardStatus: status, RewardAmount: r.InviterRewardAmount,
		})
	}
	utils.Success(c, gin.H{
		"total_invites": totalInvites, "registered_invites": totalInvites,
		"purchased_invites": purchasedInvites, "total_reward": totalReward, "recent_invites": recentInvites,
	})
}

func GetMyCodes(c *gin.Context) { ListInviteCodes(c) }

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
	codeStr := c.Param("code")
	var invite models.InviteCode
	if err := database.GetDB().Where("UPPER(code) = UPPER(?) AND is_active = ?", codeStr, true).First(&invite).Error; err != nil {
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

// ── Helpers ──

func inviteCodeStatus(code models.InviteCode) string {
	if code.ExpiresAt != nil && time.Now().After(*code.ExpiresAt) {
		return "expired"
	}
	if code.MaxUses != nil && code.UsedCount >= int(*code.MaxUses) {
		return "exhausted"
	}
	if !code.IsActive {
		return "disabled"
	}
	return "active"
}