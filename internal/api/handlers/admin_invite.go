package handlers

import (
	"strconv"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

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
		if code.ExpiresAt != nil && time.Now().After(*code.ExpiresAt) {
			item.Status = "expired"
		} else if code.MaxUses != nil && code.UsedCount >= int(*code.MaxUses) {
			item.Status = "exhausted"
		} else if !code.IsActive {
			item.Status = "disabled"
		} else {
			item.Status = "active"
		}
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
		var code models.InviteCode
		if db.Select("code").First(&code, rel.InviteCodeID).Error == nil {
			item.Code = code.Code
		}
		result = append(result, item)
	}
	utils.Success(c, gin.H{"items": result, "total": total})
}
