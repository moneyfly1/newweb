package handlers

import (
	"fmt"
	"time"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func ListRechargeRecords(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	p := utils.GetPagination(c)
	var items []models.RechargeRecord
	var total int64
	db := database.GetDB().Model(&models.RechargeRecord{}).Where("user_id = ?", userID)
	db.Count(&total)
	db.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&items)
	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func CreateRecharge(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	userID := c.MustGet("user_id").(uint)
	orderNo := fmt.Sprintf("RCH%d%s", time.Now().Unix(), utils.GenerateRandomString(6))
	record := models.RechargeRecord{
		UserID:  userID,
		OrderNo: orderNo,
		Amount:  req.Amount,
		Status:  "pending",
	}
	database.GetDB().Create(&record)
	utils.Success(c, record)
}

func CancelRecharge(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")
	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ? AND status = ?", id, userID, "pending").First(&record).Error; err != nil {
		utils.NotFound(c, "充值记录不存在")
		return
	}
	db.Model(&record).Update("status", "cancelled")
	utils.SuccessMessage(c, "充值已取消")
}
