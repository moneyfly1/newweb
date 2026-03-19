package handlers

import (
	"fmt"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

func ListRechargeRecords(c *gin.Context) {
	userID := c.GetUint("user_id")
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
		Amount          float64 `json:"amount" binding:"required,gt=0"`
		PaymentMethodID uint    `json:"payment_method_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if req.Amount > 100000 {
		utils.BadRequest(c, "单次充值金额不能超过 100000")
		return
	}
	userID := c.GetUint("user_id")
	db := database.GetDB()
	orderNo := fmt.Sprintf("RCH%d%s", time.Now().Unix(), utils.GenerateRandomString(6))
	record := models.RechargeRecord{
		UserID:  userID,
		OrderNo: orderNo,
		Amount:  req.Amount,
		Status:  "pending",
	}
	if err := db.Create(&record).Error; err != nil {
		utils.InternalError(c, "创建充值记录失败")
		return
	}

	// If payment_method_id provided, create payment immediately
	if req.PaymentMethodID > 0 {
		var payConfig models.PaymentConfig
		if err := db.Where("id = ? AND status = ?", req.PaymentMethodID, 1).First(&payConfig).Error; err != nil {
			utils.BadRequest(c, "支付方式不可用")
			return
		}

		txID := fmt.Sprintf("RCH%d%s", time.Now().Unix(), utils.GenerateRandomString(8))
		paymentData := fmt.Sprintf(`{"recharge_id":%d}`, record.ID)
		transaction := models.PaymentTransaction{
			OrderID:         0,
			UserID:          userID,
			PaymentMethodID: req.PaymentMethodID,
			Amount:          record.Amount,
			Currency:        "CNY",
			TransactionID:   &txID,
			Status:          "pending",
			PaymentData:     &paymentData,
		}
		if err := db.Create(&transaction).Error; err != nil {
			utils.Success(c, record)
			return
		}

		pmName := payConfig.PayType
		if err := db.Model(&record).Updates(map[string]interface{}{
			"payment_method":         &pmName,
			"payment_transaction_id": &txID,
		}).Error; err != nil {
			utils.InternalError(c, "更新充值记录失败")
			return
		}

		if payConfig.PayType == "epay" || payConfig.PayType == "alipay" || payConfig.PayType == "wxpay" || payConfig.PayType == "qqpay" {
			orderName := "充值-" + record.OrderNo

			// Try direct Alipay first
			if payConfig.PayType == "alipay" {
				if services.IsDirectAlipayConfigured() {
					alipayCfg, err := services.GetAlipayConfig()
					if err == nil {
						notifyURL, returnURL := services.BuildPaymentURLs("alipay", record.OrderNo)
						paymentURL, err := services.AlipayCreateOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
						if err == nil {
							if err := db.Model(&record).Update("payment_url", &paymentURL).Error; err != nil {
								utils.InternalError(c, "更新支付链接失败")
								return
							}
							if err := db.First(&record, record.ID).Error; err != nil {
								utils.InternalError(c, "读取充值记录失败")
								return
							}
							utils.Success(c, gin.H{
								"record":         record,
								"transaction_id": txID,
								"payment_url":    paymentURL,
							})
							return
						}
					}
				}
			}

			// Fall back to epay gateway
			epayCfg, err := services.GetEpayConfig()
			if err != nil {
				utils.Success(c, record)
				return
			}

			epayType := payConfig.PayType
			if epayType == "epay" {
				epayType = "alipay"
			}

			notifyURL, returnURL := services.BuildPaymentURLs("epay", record.OrderNo)

			paymentURL, err := services.EpayCreateOrder(epayCfg, epayType, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
			if err != nil {
				utils.Success(c, record)
				return
			}

			if err := db.Model(&record).Update("payment_url", &paymentURL).Error; err != nil {
				utils.InternalError(c, "更新支付链接失败")
				return
			}
			if err := db.First(&record, record.ID).Error; err != nil {
				utils.InternalError(c, "读取充值记录失败")
				return
			}

			utils.Success(c, gin.H{
				"record":         record,
				"transaction_id": txID,
				"payment_url":    paymentURL,
			})
			return
		}
	}

	utils.Success(c, record)
}

func CancelRecharge(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ? AND status = ?", id, userID, "pending").First(&record).Error; err != nil {
		utils.NotFound(c, "充值记录不存在")
		return
	}
	if err := db.Model(&record).Update("status", "cancelled").Error; err != nil {
		utils.InternalError(c, "取消充值失败")
		return
	}
	utils.SuccessMessage(c, "充值已取消")
}
