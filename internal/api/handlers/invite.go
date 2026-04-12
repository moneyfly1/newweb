package handlers

import (
	"fmt"
	"strconv"
	"strings"
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
		// 转义 LIKE 通配符防止信息泄露
		escapedSearch := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
		likePattern := "%" + escapedSearch + "%"
		query = query.Where("code LIKE ? ESCAPE '\\' OR user_id IN (SELECT id FROM users WHERE username LIKE ? ESCAPE '\\' OR email LIKE ? ESCAPE '\\')",
			likePattern, likePattern, likePattern)
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
	// 批量查询用户名，避免 N+1
	userIDs := make([]uint, 0, len(codes))
	for _, code := range codes {
		userIDs = append(userIDs, code.UserID)
	}
	userMap := make(map[uint]string)
	if len(userIDs) > 0 {
		var users []models.User
		db.Select("id, username").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u.Username
		}
	}
	var result []R
	for _, code := range codes {
		item := R{InviteCode: code}
		item.Username = userMap[code.UserID]
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
	if err := db.Save(&code).Error; err != nil {
		utils.InternalError(c, "更新邀请码状态失败")
		return
	}
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
		Code            string `json:"invite_code"`
	}
	// 批量查询用户名和邀请码，避免 N+1
	allUserIDs := make([]uint, 0, len(rels)*2)
	codeIDs := make([]uint, 0, len(rels))
	for _, rel := range rels {
		allUserIDs = append(allUserIDs, rel.InviterID, rel.InviteeID)
		codeIDs = append(codeIDs, rel.InviteCodeID)
	}
	userNameMap := make(map[uint]string)
	if len(allUserIDs) > 0 {
		var users []models.User
		db.Select("id, username").Where("id IN ?", allUserIDs).Find(&users)
		for _, u := range users {
			userNameMap[u.ID] = u.Username
		}
	}
	codeMap := make(map[uint]string)
	if len(codeIDs) > 0 {
		var inviteCodes []models.InviteCode
		db.Select("id, code").Where("id IN ?", codeIDs).Find(&inviteCodes)
		for _, ic := range inviteCodes {
			codeMap[ic.ID] = ic.Code
		}
	}
	var result []RR
	for _, rel := range rels {
		item := RR{InviteRelation: rel}
		item.InviterUsername = userNameMap[rel.InviterID]
		item.InviteeUsername = userNameMap[rel.InviteeID]
		item.Code = codeMap[rel.InviteCodeID]
		result = append(result, item)
	}
	utils.Success(c, gin.H{"items": result, "total": total})
}

// ── User: invite codes ──

func ListInviteCodes(c *gin.Context) {
	userID := c.GetUint("user_id")
	var codes []models.InviteCode
	database.GetDB().Where("user_id = ?", userID).Order("created_at DESC").Limit(100).Find(&codes)
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
	// Bounds validation - 非管理员限制奖励金额
	user := c.MustGet("user").(*models.User)
	maxReward := 100.0 // 普通用户最大奖励
	if user.IsAdmin {
		maxReward = 10000.0
	}
	if req.InviterReward < 0 || req.InviterReward > maxReward {
		utils.BadRequest(c, fmt.Sprintf("邀请人奖励需在 0 ~ %.0f 之间", maxReward))
		return
	}
	if req.InviteeReward < 0 || req.InviteeReward > maxReward {
		utils.BadRequest(c, fmt.Sprintf("受邀人奖励需在 0 ~ %.0f 之间", maxReward))
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
	if err := db.Create(&code).Error; err != nil {
		utils.InternalError(c, "创建邀请码失败")
		return
	}
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
	// 批量查询受邀用户信息，避免 N+1
	inviteeIDs := make([]uint, 0, len(relations))
	for _, r := range relations {
		inviteeIDs = append(inviteeIDs, r.InviteeID)
	}
	inviteeMap := make(map[uint]models.User)
	if len(inviteeIDs) > 0 {
		var invitees []models.User
		db.Select("id, username, email").Where("id IN ?", inviteeIDs).Find(&invitees)
		for _, u := range invitees {
			inviteeMap[u.ID] = u
		}
	}
	var recentInvites []recentInvite
	for _, r := range relations {
		user := inviteeMap[r.InviteeID]
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
	if err := db.Delete(&code).Error; err != nil {
		utils.InternalError(c, "删除邀请码失败")
		return
	}
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
		utils.BadRequest(c, "邀请码已过期，请联系邀请人获取新邀请码")
		return
	}
	if invite.MaxUses != nil && invite.UsedCount >= int(*invite.MaxUses) {
		utils.BadRequest(c, "邀请码使用次数已达上限，请联系邀请人获取新邀请码")
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
