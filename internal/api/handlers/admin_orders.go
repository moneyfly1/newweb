package handlers

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// flexTime 兼容 SQLite 字符串时间与 NULL 的扫描（db.Raw().Scan() 场景）。
// SQLite 驱动把 datetime 列返回为 string，database/sql 无法直接扫描进 *time.Time，
// 导致 AdminListOrders 的 UNION 查询报 "unsupported Scan ... string into type *time.Time"。
type flexTime struct {
	Time  time.Time
	Valid bool
}

// parseSQLiteTime 解析 SQLite/GORM 存储的多种时间字符串格式
func parseSQLiteTime(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间字符串: %q", s)
}

// Scan 实现 sql.Scanner
func (f *flexTime) Scan(value interface{}) error {
	if value == nil {
		f.Valid = false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		f.Time, f.Valid = v, true
		return nil
	case string:
		t, err := parseSQLiteTime(v)
		if err != nil {
			return err
		}
		f.Time, f.Valid = t, true
		return nil
	case []byte:
		return f.Scan(string(v))
	default:
		return fmt.Errorf("无法扫描时间类型: %T", value)
	}
}

// MarshalJSON 保持 time.Time 的 JSON 输出格式（RFC3339 / null）
func (f flexTime) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(f.Time)
}

// Value 实现 driver.Valuer（GORM schema 要求自定义类型同时实现 Scanner/Valuer）
func (f flexTime) Value() (driver.Value, error) {
	if !f.Valid {
		return nil, nil
	}
	return f.Time, nil
}

func AdminListOrders(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	type AdminOrderItem struct {
		ID                   uint       `json:"id"`
		OrderNo              string     `json:"order_no"`
		UserID               uint       `json:"user_id"`
		UserEmail            string     `json:"user_email"`
		PackageID            uint       `json:"package_id"`
		Amount               float64    `json:"amount"`
		Status               string     `json:"status"`
		PaymentMethodID      *int64     `json:"payment_method_id"`
		PaymentMethodName    *string    `json:"payment_method_name"`
		PaymentTime          *flexTime  `json:"payment_time"`
		PaymentTransactionID *string    `json:"payment_transaction_id"`
		GatewayTradeNo       *string    `json:"gateway_trade_no,omitempty"`
		ExpireTime           *flexTime  `json:"expire_time"`
		CouponID             *int64     `json:"coupon_id"`
		DiscountAmount       *float64   `json:"discount_amount"`
		FinalAmount          *float64   `json:"final_amount"`
		ExtraData            *string    `json:"extra_data"`
		CreatedAt            flexTime   `json:"created_at"`
		UpdatedAt            flexTime   `json:"updated_at"`
		OrderType            string     `json:"order_type"`
		OrderTypeText        string     `json:"order_type_text"`
		OrderSummary         string     `json:"order_summary"`
		PackageName          string     `json:"package_name"`
		Devices              *int       `json:"devices,omitempty"`
		Months               *int       `json:"months,omitempty"`
		AddDevices           *int       `json:"add_devices,omitempty"`
		ExtendMonths         *int       `json:"extend_months,omitempty"`
		CurrentDeviceLimit   *int       `json:"current_device_limit,omitempty"`
		NewDeviceLimit       *int       `json:"new_device_limit,omitempty"`
		CurrentExpireTime    *string    `json:"current_expire_time,omitempty"`
		NewExpireTime        *string    `json:"new_expire_time,omitempty"`
		BalanceAmount        *float64   `json:"balance_amount,omitempty"`
	}

	statusFilter := c.Query("status")
	userIDFilter := c.Query("user_id")
	orderNoFilter := c.Query("order_no")
	searchFilter := c.Query("search")

	// SQL 层 UNION ALL 合并订单与充值记录，排序、过滤、分页全部下沉到数据库，
	// 避免订单量大时全表载入内存（OOM/超时）。
	unionBody := `SELECT 'order' AS src, o.id, o.order_no, o.user_id,
			u.email AS user_email, u.username AS user_username,
			o.package_id, o.amount, o.status, o.payment_method_id, o.payment_method_name, o.payment_time,
			o.payment_transaction_id, o.expire_time, o.coupon_id, o.discount_amount, o.final_amount,
			o.extra_data, o.created_at, o.updated_at
		FROM orders o LEFT JOIN users u ON u.id = o.user_id
		UNION ALL
		SELECT 'recharge' AS src, r.id, r.order_no, r.user_id,
			u.email AS user_email, u.username AS user_username,
			0, r.amount, r.status, NULL, r.payment_method, r.paid_at,
			r.payment_transaction_id, NULL, NULL, NULL, r.amount,
			NULL, r.created_at, r.updated_at
		FROM recharge_records r LEFT JOIN users u ON u.id = r.user_id`

	whereClauses := []string{"1=1"}
	var whereArgs []interface{}
	if statusFilter != "" {
		whereClauses = append(whereClauses, "status = ?")
		whereArgs = append(whereArgs, statusFilter)
	}
	if userIDFilter != "" {
		whereClauses = append(whereClauses, "user_id = ?")
		whereArgs = append(whereArgs, userIDFilter)
	}
	if orderNoFilter != "" {
		whereClauses = append(whereClauses, "order_no LIKE ?")
		whereArgs = append(whereArgs, "%"+orderNoFilter+"%")
	}
	if searchFilter != "" {
		pattern := "%" + searchFilter + "%"
		whereClauses = append(whereClauses, "(order_no LIKE ? OR user_email LIKE ? OR user_username LIKE ? OR CAST(user_id AS CHAR) LIKE ?)")
		whereArgs = append(whereArgs, pattern, pattern, pattern, pattern)
	}
	whereSQL := strings.Join(whereClauses, " AND ")

	var items []AdminOrderItem
	pageArgs := append(append([]interface{}{}, whereArgs...), p.PageSize, p.Offset())
	if err := db.Raw(`SELECT * FROM (`+unionBody+`) merged WHERE `+whereSQL+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, pageArgs...).Scan(&items).Error; err != nil {
		utils.InternalError(c, "查询订单失败")
		return
	}
	var total int64
	if err := db.Raw(`SELECT COUNT(*) FROM (`+unionBody+`) merged WHERE `+whereSQL, whereArgs...).Scan(&total).Error; err != nil {
		utils.InternalError(c, "查询订单失败")
		return
	}

	packageIDs := make([]uint, 0, len(items))
	orderIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.PackageID > 0 {
			packageIDs = append(packageIDs, item.PackageID)
		}
		if len(item.OrderNo) < 3 || item.OrderNo[:3] != "RCH" {
			orderIDs = append(orderIDs, item.ID)
		}
	}

	pkgNameCache := make(map[uint]string)
	if len(packageIDs) > 0 {
		var packages []models.Package
		db.Select("id, name").Where("id IN ?", packageIDs).Find(&packages)
		for _, pkg := range packages {
			pkgNameCache[pkg.ID] = pkg.Name
		}
	}

	gatewayTradeCache := make(map[uint]*string)
	if len(orderIDs) > 0 {
		var transactions []models.PaymentTransaction
		db.Select("id, order_id, external_transaction_id").
			Where("order_id IN ? AND external_transaction_id IS NOT NULL AND external_transaction_id <> ''", orderIDs).
			Order("id DESC").
			Find(&transactions)
		for _, txn := range transactions {
			if _, exists := gatewayTradeCache[txn.OrderID]; !exists {
				gatewayTradeCache[txn.OrderID] = txn.ExternalTransactionID
			}
		}
	}

	for i := range items {
		item := &items[i]

		// Check if this is a recharge record (order_no starts with "RCH")
		if len(item.OrderNo) >= 3 && item.OrderNo[:3] == "RCH" {
			item.OrderType = "recharge"
			item.OrderTypeText = "余额充值"
			item.OrderSummary = fmt.Sprintf("充值 %.2f 元", item.Amount)
			item.PackageName = "余额充值"
			continue
		}

		item.OrderType = "package"
		item.OrderTypeText = "套餐订单"
		item.OrderSummary = "标准套餐"
		if gatewayTradeNo, ok := gatewayTradeCache[item.ID]; ok {
			item.GatewayTradeNo = gatewayTradeNo
		}
		if name, ok := pkgNameCache[item.PackageID]; ok {
			item.PackageName = name
			item.OrderSummary = name
		}

		if item.PackageID == 0 && item.ExtraData != nil {
			var extra map[string]interface{}
			if json.Unmarshal([]byte(*item.ExtraData), &extra) == nil {
				switch extra["type"] {
				case "custom_package":
					item.OrderType = "custom_package"
					item.OrderTypeText = "自定义套餐"
					if devices, ok := extra["devices"].(float64); ok {
						v := int(devices)
						item.Devices = &v
					}
					if months, ok := extra["months"].(float64); ok {
						v := int(months)
						item.Months = &v
					}
					if item.Devices != nil && item.Months != nil {
						item.OrderSummary = fmt.Sprintf("%d设备 / %d个月", *item.Devices, *item.Months)
						item.PackageName = fmt.Sprintf("自定义套餐（%d设备/%d个月）", *item.Devices, *item.Months)
					}
				case "subscription_upgrade":
					item.OrderType = "subscription_upgrade"
					item.OrderTypeText = "设备升级"
					if addDevices, ok := extra["add_devices"].(float64); ok {
						v := int(addDevices)
						item.AddDevices = &v
					}
					if extendMonths, ok := extra["extend_months"].(float64); ok {
						v := int(extendMonths)
						item.ExtendMonths = &v
					}
					if item.AddDevices != nil {
						item.OrderSummary = fmt.Sprintf("+%d设备", *item.AddDevices)
						item.PackageName = item.OrderSummary
					}
					if item.AddDevices != nil && item.ExtendMonths != nil && *item.ExtendMonths > 0 {
						item.OrderSummary = fmt.Sprintf("+%d设备 / 续期%d个月", *item.AddDevices, *item.ExtendMonths)
						item.PackageName = item.OrderSummary
					}
					if v, ok := extra["current_device_limit"].(float64); ok {
						cd := int(v)
						item.CurrentDeviceLimit = &cd
					}
					if v, ok := extra["new_device_limit"].(float64); ok {
						nd := int(v)
						item.NewDeviceLimit = &nd
					}
					if v, ok := extra["current_expire_time"].(string); ok {
						s := formatUpgradeTime(v)
						item.CurrentExpireTime = &s
					}
					if v, ok := extra["new_expire_time"].(string); ok {
						s := formatUpgradeTime(v)
						item.NewExpireTime = &s
					}
				}
			}
		}
	}

	utils.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func AdminGetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	utils.Success(c, order)
}

func findOrderPaymentTransaction(db *gorm.DB, order models.Order) (*models.PaymentTransaction, error) {
	var txn models.PaymentTransaction
	if order.PaymentTransactionID != nil && strings.TrimSpace(*order.PaymentTransactionID) != "" {
		if err := db.Where("transaction_id = ?", strings.TrimSpace(*order.PaymentTransactionID)).First(&txn).Error; err == nil {
			return &txn, nil
		}
	}
	if err := db.Where("order_id = ? AND status = ?", order.ID, "paid").Order("id DESC").First(&txn).Error; err == nil {
		return &txn, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func paymentCallbackType(db *gorm.DB, transactionID uint) string {
	var callback models.PaymentCallback
	if err := db.Select("callback_type").
		Where("payment_transaction_id = ? AND processed = ?", transactionID, true).
		Order("id DESC").
		First(&callback).Error; err == nil {
		return strings.ToLower(callback.CallbackType)
	}
	return ""
}

func paymentConfigPayType(db *gorm.DB, paymentMethodID uint) string {
	var paymentMethod models.PaymentConfig
	if err := db.Select("pay_type").First(&paymentMethod, paymentMethodID).Error; err == nil {
		return strings.ToLower(paymentMethod.PayType)
	}
	return ""
}

func classifyRefundChannel(db *gorm.DB, order models.Order, txn *models.PaymentTransaction) string {
	if order.PaymentMethodName != nil {
		method := strings.ToLower(strings.TrimSpace(*order.PaymentMethodName))
		if method == "balance" {
			return "balance"
		}
		if method == "管理员手动确认" || method == "admin_manual" || strings.HasPrefix(method, "admin") {
			return "manual"
		}
	}
	if txn == nil || txn.ID == 0 {
		return "unknown"
	}

	callbackType := paymentCallbackType(db, txn.ID)
	if callbackType == "epay" {
		return "epay"
	}
	if callbackType == "codepay" {
		return "codepay"
	}
	if callbackType == "alipay" || strings.HasPrefix(callbackType, "alipay_query") {
		return "alipay"
	}

	payType := paymentConfigPayType(db, txn.PaymentMethodID)
	switch payType {
	case "epay", "wxpay", "qqpay":
		return "epay"
	case "codepay", "codepay_alipay", "codepay_wxpay":
		return "codepay"
	case "alipay":
		return "alipay"
	default:
		return payType
	}
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func AdminRefundOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "paid" && order.Status != "completed" {
		utils.BadRequest(c, "只能退款已支付或已完成的订单")
		return
	}

	// 原子抢占退款状态：仅当订单仍为 paid/completed 时置为 refunding，
	// 防止并发退款请求同时通过状态检查，导致网关双退或余额双倍入账。
	claim := db.Model(&models.Order{}).
		Where("id = ? AND status IN ('paid','completed')", order.ID).
		Update("status", "refunding")
	if claim.Error != nil {
		utils.InternalError(c, "锁定订单退款状态失败")
		return
	}
	if claim.RowsAffected == 0 {
		utils.BadRequest(c, "订单已退款或状态不允许")
		return
	}
	refundSettled := false
	// 任一步骤失败时恢复原状态，允许重试
	defer func() {
		if !refundSettled {
			db.Model(&models.Order{}).Where("id = ? AND status = 'refunding'", order.ID).
				Update("status", order.Status)
		}
	}()

	refundAmount := order.Amount
	if order.FinalAmount != nil {
		refundAmount = *order.FinalAmount
	}

	txn, _ := findOrderPaymentTransaction(db, order)
	channel := classifyRefundChannel(db, order, txn)
	merchantOrderNo := ""
	gatewayTradeNo := ""
	if txn != nil {
		merchantOrderNo = stringValue(txn.TransactionID)
		gatewayTradeNo = stringValue(txn.ExternalTransactionID)
	}

	refundMethod := "余额"
	switch channel {
	case "alipay":
		if merchantOrderNo == "" || gatewayTradeNo == "" {
			utils.BadRequest(c, "支付宝原路退款缺少商户订单号或支付宝交易号")
			return
		}
		if err := services.AlipayRefund(gatewayTradeNo, merchantOrderNo, fmt.Sprintf("%.2f", refundAmount)); err != nil {
			utils.BadRequest(c, "支付宝退款失败: "+err.Error())
			return
		}
		refundMethod = "支付宝原路退回"
	case "epay":
		epayCfg, err := services.GetEpayConfig()
		if err != nil {
			utils.BadRequest(c, "易支付退款失败: "+err.Error())
			return
		}
		if merchantOrderNo == "" && gatewayTradeNo == "" {
			utils.BadRequest(c, "易支付原路退款缺少商户订单号或易支付订单号")
			return
		}
		if err := services.EpayRefund(epayCfg, merchantOrderNo, gatewayTradeNo, fmt.Sprintf("%.2f", refundAmount)); err != nil {
			utils.BadRequest(c, err.Error())
			return
		}
		refundMethod = "易支付原路退回"
	case "balance":
		refundMethod = "余额"
	case "manual":
		utils.BadRequest(c, "管理员手动确认的订单没有线上支付流水，不能原路退款；请线下退款后取消/调整用户权限")
		return
	case "codepay":
		codepayCfg, err := services.GetCodepayConfig()
		if err != nil {
			utils.BadRequest(c, "码支付退款失败: "+err.Error())
			return
		}
		if merchantOrderNo == "" && gatewayTradeNo == "" {
			utils.BadRequest(c, "码支付原路退款缺少商户订单号或平台订单号")
			return
		}
		if err := services.CodepayRefund(codepayCfg, merchantOrderNo, gatewayTradeNo, fmt.Sprintf("%.2f", refundAmount)); err != nil {
			utils.BadRequest(c, err.Error())
			return
		}
		refundMethod = "码支付原路退回"
	default:
		utils.BadRequest(c, "无法确认支付渠道，未执行余额退款；请检查支付流水和回调记录")
		return
	}

	// 防双退：网关退款已成功，先持久化订单退款状态（独立提交），
	// 即使后续余额/订阅回滚失败也不重试网关退款
	markRefunded := db.Model(&models.Order{}).
		Where("id = ? AND status = ?", order.ID, "refunding").
		Update("status", "refunded")
	if markRefunded.Error != nil {
		utils.InternalError(c, "更新订单退款状态失败")
		return
	}
	refundSettled = true

	tx := db.Begin()
	if tx.Error != nil {
		utils.InternalError(c, "创建事务失败")
		return
	}

	if channel == "balance" {
		if err := tx.Model(&models.User{}).Where("id = ?", order.UserID).
			UpdateColumn("balance", gorm.Expr("balance + ?", refundAmount)).Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "退款失败")
			return
		}
	}

	if txn != nil && txn.ID > 0 {
		if err := tx.Model(txn).Update("status", "refunded").Error; err != nil {
			tx.Rollback()
			utils.InternalError(c, "退款失败")
			return
		}
	}

	// Cancel/rollback the subscription that was activated by this order
	var sub models.Subscription
	if tx.Where("user_id = ?", order.UserID).First(&sub).Error == nil {
		shouldCancel := false
		if order.PackageID == 0 {
			// Custom package order — always cancel
			shouldCancel = true
		} else if sub.PackageID != nil && *sub.PackageID == int64(order.PackageID) {
			shouldCancel = true
		}
		if shouldCancel {
			if err := tx.Model(&sub).Updates(map[string]interface{}{
				"is_active": false,
				"status":    "cancelled",
			}).Error; err != nil {
				tx.Rollback()
				utils.InternalError(c, "退款失败")
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.InternalError(c, "退款事务提交失败（订单已标记退款，请勿重复退款）")
		return
	}

	var refundUser models.User
	if db.First(&refundUser, order.UserID).Error == nil {
		desc := fmt.Sprintf("管理员退款订单: %s (%s)", order.OrderNo, refundMethod)
		if channel == "balance" {
			utils.CreateBalanceLogEntry(order.UserID, "refund", refundAmount, refundUser.Balance-refundAmount, refundUser.Balance, func() *uint { id := uint(order.ID); return &id }(), desc, c)
		}
	}
	utils.CreateAuditLog(c, "refund_order", "order", uint(id), fmt.Sprintf("退款订单: %s, 金额: %.2f, 方式: %s, 商户单号: %s, 平台流水: %s", order.OrderNo, refundAmount, refundMethod, merchantOrderNo, gatewayTradeNo))
	utils.SuccessMessage(c, fmt.Sprintf("退款成功（%s）", refundMethod))
}

func AdminCancelOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "pending" && order.Status != "expired" {
		utils.BadRequest(c, "只能取消待支付或已过期的订单")
		return
	}

	// 条件更新：仅当订单仍为 pending/expired 时置为 cancelled，
	// 防止取消请求覆盖并发支付回调刚置为 paid 的订单（已扣款却显示已取消）。
	cancelRes := db.Model(&models.Order{}).
		Where("id = ? AND status IN ('pending','expired')", order.ID).
		Update("status", "cancelled")
	if cancelRes.Error != nil {
		utils.InternalError(c, "取消订单失败")
		return
	}
	if cancelRes.RowsAffected == 0 {
		utils.BadRequest(c, "订单状态已变化，无法取消")
		return
	}

	utils.CreateAuditLog(c, "cancel_order", "order", uint(id), fmt.Sprintf("取消订单: %s", order.OrderNo))
	utils.SuccessMessage(c, "订单已取消")
}

func AdminMarkOrderPaid(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "pending" && order.Status != "expired" && order.Status != "cancelled" {
		utils.BadRequest(c, "只能将待支付、已过期或已取消的套餐订单标记为已付款")
		return
	}

	tx := db.Begin()
	if tx.Error != nil {
		utils.InternalError(c, "创建事务失败")
		return
	}

	now := time.Now()
	paymentMethodName := "管理员手动确认"
	transactionID := fmt.Sprintf("ADMIN-%s-%d", order.OrderNo, now.Unix())
	updates := map[string]interface{}{
		"status":                 "paid",
		"payment_time":           &now,
		"payment_method_name":    &paymentMethodName,
		"payment_transaction_id": &transactionID,
	}
	// 条件更新 + 行数校验：仅当订单仍为 pending/expired/cancelled 时才置为 paid，
	// 防止并发重复标记导致订阅被多次叠加开通（免费权益）。
	markRes := tx.Model(&models.Order{}).
		Where("id = ? AND status IN ('pending','expired','cancelled')", order.ID).
		Updates(updates)
	if markRes.Error != nil {
		tx.Rollback()
		utils.InternalError(c, "更新订单状态失败")
		return
	}
	if markRes.RowsAffected == 0 {
		tx.Rollback()
		utils.BadRequest(c, "订单状态已变化，无法标记为已付款")
		return
	}
	order.Status = "paid"
	order.PaymentTime = &now
	order.PaymentMethodName = &paymentMethodName
	order.PaymentTransactionID = &transactionID

	if err := services.ActivateSubscription(tx, &order, "admin_manual"); err != nil {
		tx.Rollback()
		utils.InternalError(c, "开通订阅失败: "+err.Error())
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.InternalError(c, "提交事务失败")
		return
	}

	utils.CreateAuditLog(c, "mark_order_paid", "order", uint(id), fmt.Sprintf("管理员手动标记订单已付款并开通权限: %s", order.OrderNo))
	utils.SuccessMessage(c, "已标记付款并开通权限")
}

func AdminCompleteOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "paid" {
		utils.BadRequest(c, "只能完成已支付的订单")
		return
	}

	if err := db.Model(&order).Update("status", "completed").Error; err != nil {
		utils.InternalError(c, "完成订单失败")
		return
	}

	utils.CreateAuditLog(c, "complete_order", "order", uint(id), fmt.Sprintf("完成订单: %s", order.OrderNo))
	utils.SuccessMessage(c, "订单已完成")
}

func AdminBatchOrderAction(c *gin.Context) {
	var req struct {
		IDs    []uint `json:"ids"`
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		utils.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	success := 0
	failed := 0
	for _, id := range req.IDs {
		var order models.Order
		if err := db.First(&order, id).Error; err != nil {
			failed++
			continue
		}
		switch req.Action {
		case "mark_paid":
			if order.Status != "pending" && order.Status != "expired" && order.Status != "cancelled" {
				failed++
				continue
			}
			tx := db.Begin()
			now := time.Now()
			paymentMethodName := "管理员手动确认"
			transactionID := fmt.Sprintf("ADMIN-%s-%d", order.OrderNo, now.Unix())
			// 条件更新 + 行数校验，防止重复标记导致订阅叠加开通
			markRes := tx.Model(&models.Order{}).
				Where("id = ? AND status IN ('pending','expired','cancelled')", order.ID).
				Updates(map[string]interface{}{
					"status":                 "paid",
					"payment_time":           &now,
					"payment_method_name":    &paymentMethodName,
					"payment_transaction_id": &transactionID,
				})
			if markRes.Error != nil || markRes.RowsAffected == 0 {
				tx.Rollback()
				failed++
				continue
			}
			order.Status = "paid"
			order.PaymentTime = &now
			order.PaymentMethodName = &paymentMethodName
			order.PaymentTransactionID = &transactionID
			if err := services.ActivateSubscription(tx, &order, "admin_manual"); err != nil {
				tx.Rollback()
				failed++
				continue
			}
			if err := tx.Commit().Error; err != nil {
				failed++
				continue
			}
			success++
		case "cancel":
			if order.Status != "pending" && order.Status != "expired" {
				failed++
				continue
			}
			if err := db.Model(&order).Update("status", "cancelled").Error; err != nil {
				failed++
				continue
			}
			success++
		case "complete":
			if order.Status != "paid" {
				failed++
				continue
			}
			if err := db.Model(&order).Update("status", "completed").Error; err != nil {
				failed++
				continue
			}
			success++
		case "delete":
			if order.Status != "cancelled" && order.Status != "refunded" {
				failed++
				continue
			}
			if err := db.Delete(&order).Error; err != nil {
				failed++
				continue
			}
			success++
		default:
			utils.BadRequest(c, "不支持的批量操作")
			return
		}
	}

	utils.CreateAuditLog(c, "batch_order_action", "order", 0, fmt.Sprintf("批量订单操作: %s, 成功: %d, 失败: %d", req.Action, success, failed))
	utils.Success(c, gin.H{"success": success, "failed": failed})
}

func AdminDeleteOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的订单ID")
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		utils.NotFound(c, "订单不存在")
		return
	}
	if order.Status != "cancelled" && order.Status != "refunded" {
		utils.BadRequest(c, "只能删除已取消或已退款的订单")
		return
	}

	if err := db.Delete(&order).Error; err != nil {
		utils.InternalError(c, "删除订单失败")
		return
	}

	utils.CreateAuditLog(c, "delete_order", "order", uint(id), fmt.Sprintf("删除订单: %s", order.OrderNo))
	utils.SuccessMessage(c, "订单已删除")
}

// ==================== Package Management ====================

