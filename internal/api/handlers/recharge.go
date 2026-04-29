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
	if status := c.Query("status"); status != "" {
		db = db.Where("status = ?", status)
	}
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
	userID := c.GetUint("user_id")
	db := database.GetDB()

	record := models.RechargeRecord{
		UserID:    userID,
		OrderNo:   fmt.Sprintf("R%d%s", time.Now().Unix(), utils.GenerateRandomString(6)),
		Amount:    req.Amount,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := db.Create(&record).Error; err != nil {
		utils.InternalError(c, "创建充值记录失败")
		return
	}

	if req.PaymentMethodID != 0 {
		var payConfig models.PaymentConfig
		if err := db.Where("id = ? AND status = ?", req.PaymentMethodID, 1).First(&payConfig).Error; err == nil {
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
			if err := db.Create(&transaction).Error; err == nil {
				pmName := payConfig.PayType
				_ = db.Model(&record).Updates(map[string]interface{}{
					"payment_method":         &pmName,
					"payment_transaction_id": &txID,
				}).Error

				if payConfig.PayType == "epay" || payConfig.PayType == "alipay" || payConfig.PayType == "wxpay" || payConfig.PayType == "qqpay" {
					orderName := "充值-" + record.OrderNo

					if payConfig.PayType == "alipay" && services.IsDirectAlipayConfigured() {
						alipayCfg, err := services.GetAlipayConfig()
						if err == nil {
							notifyURL, returnURL := services.BuildPaymentURLs("alipay", record.OrderNo)
							paymentURL, err := services.AlipayCreateOrder(alipayCfg, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
							if err == nil {
								_ = db.Model(&record).Update("payment_url", &paymentURL).Error
								_ = db.First(&record, record.ID).Error
								utils.Success(c, gin.H{
									"record":         record,
									"transaction_id": txID,
									"payment_url":    paymentURL,
								})
								return
							}
						}
					}

					epayCfg, err := services.GetEpayConfig()
					if err == nil {
						epayType := payConfig.PayType
						if epayType == "epay" {
							epayType = "alipay"
						}
						notifyURL, returnURL := services.BuildPaymentURLs("epay", record.OrderNo)
						paymentURL, err := services.EpayCreateOrder(epayCfg, epayType, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
						if err == nil {
							_ = db.Model(&record).Update("payment_url", &paymentURL).Error
							_ = db.First(&record, record.ID).Error
							utils.Success(c, gin.H{
								"record":         record,
								"transaction_id": txID,
								"payment_url":    paymentURL,
							})
							return
						}
					}
				}

				if payConfig.PayType == "codepay" || payConfig.PayType == "codepay_alipay" || payConfig.PayType == "codepay_wxpay" {
					codepayCfg, err := services.GetCodepayConfig()
					if err == nil {
						codepayType := "alipay"
						if payConfig.PayType == "codepay_wxpay" {
							codepayType = "wxpay"
						}
						orderName := "充值-" + record.OrderNo
						notifyURL, returnURL := services.BuildPaymentURLs("codepay", record.OrderNo)
						if codepayCfg.NotifyURL != "" || codepayCfg.ReturnURL != "" || codepayCfg.BaseURL != "" {
							notifyURL, returnURL = services.CodepayBuildURLs(codepayCfg, record.OrderNo)
						}
						codepayResult, err := services.CodepayCreateOrder(codepayCfg, codepayType, txID, orderName, fmt.Sprintf("%.2f", record.Amount), notifyURL, returnURL)
						if err == nil {
							_ = db.Model(&record).Update("payment_url", codepayResult.PaymentURL).Error
							_ = db.First(&record, record.ID).Error
							utils.Success(c, gin.H{
								"record":         record,
								"transaction_id": txID,
								"payment_url":    codepayResult.PaymentURL,
								"payment_mode":   codepayResult.Mode,
							})
							return
						}
					}
				}
			}
		}
	}

	utils.Success(c, gin.H{"record": record})
}

func GetRechargeStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error; err != nil {
		utils.NotFound(c, "充值记录不存在")
		return
	}

	if record.Status == "pending" && record.PaymentTransactionID != nil && *record.PaymentTransactionID != "" {
		var transaction models.PaymentTransaction
		if err := db.Where("transaction_id = ? AND user_id = ?", *record.PaymentTransactionID, userID).First(&transaction).Error; err == nil {
			if transaction.Status == "pending" {
				if _, _, err := tryCompensateAlipayPayment(db, &transaction, "recharge_status_poll"); err != nil {
					utils.LogError("[Recharge] 状态轮询补偿失败: tx_id=%s error=%v", *record.PaymentTransactionID, err)
				}
				_ = db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error
			}
		}
	}

	result := gin.H{
		"id":       record.ID,
		"order_no": record.OrderNo,
		"amount":   record.Amount,
		"status":   record.Status,
	}
	if record.PaidAt != nil {
		result["paid_at"] = record.PaidAt.Format("2006-01-02 15:04:05")
	}
	utils.Success(c, result)
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
