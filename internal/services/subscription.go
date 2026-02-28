package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"gorm.io/gorm"
)

// ── Subscription activation ──

// ActivateSubscription creates or extends a subscription after successful payment.
func ActivateSubscription(db *gorm.DB, order *models.Order, paymentMethod string) {
	var deviceLimit int
	var durationDays int
	var pkgName string

	if order.PackageID == 0 && order.ExtraData != nil {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(*order.ExtraData), &extra); err != nil {
			return
		}
		if extra["type"] == "subscription_upgrade" {
			var sub models.Subscription
			if err := db.Where("user_id = ?", order.UserID).First(&sub).Error; err != nil {
				return
			}
			addDevices := 0
			extendMonths := 0
			if v, ok := extra["add_devices"].(float64); ok {
				addDevices = int(v)
			}
			if v, ok := extra["extend_months"].(float64); ok {
				extendMonths = int(v)
			}
			newLimit := sub.DeviceLimit + addDevices
			newExpire := sub.ExpireTime
			if extendMonths > 0 {
				newExpire = newExpire.AddDate(0, extendMonths, 0)
			}
			db.Model(&sub).Updates(map[string]interface{}{
				"device_limit": newLimit,
				"expire_time":  newExpire,
				"is_active":    true,
				"status":       "active",
			})
			pkgName = fmt.Sprintf("订阅升级: +%d设备", addDevices)
			if extendMonths > 0 {
				pkgName = fmt.Sprintf("订阅升级: +%d设备, 续期%d月", addDevices, extendMonths)
			}
			var user models.User
			if db.First(&user, order.UserID).Error == nil {
				payAmount := fmt.Sprintf("%.2f", order.Amount)
				if order.FinalAmount != nil {
					payAmount = fmt.Sprintf("%.2f", *order.FinalAmount)
				}
				var subURL string
				var userSub models.Subscription
				if db.Where("user_id = ?", order.UserID).First(&userSub).Error == nil {
					if siteURL := GetSiteURL(); siteURL != "" {
						subURL = siteURL + "/api/v1/subscribe/" + userSub.SubscriptionURL
					}
				}
				emailSubject, emailBody := RenderEmail("payment_success", map[string]string{
					"order_no": order.OrderNo, "amount": payAmount, "package_name": pkgName, "subscription_url": subURL,
				})
				go QueueEmail(user.Email, emailSubject, emailBody, "payment_success")
				go NotifyAdmin("payment_success", map[string]string{
					"username": user.Username, "order_no": order.OrderNo, "package_name": pkgName, "amount": payAmount,
				})
			}
			distributeInviteCommission(db, order)
			return
		}
		if extra["type"] != "custom_package" {
			return
		}
		devices, _ := extra["devices"].(float64)
		months, _ := extra["months"].(float64)
		deviceLimit = int(devices)
		durationDays = int(months) * 30
		pkgName = fmt.Sprintf("自定义套餐 (%d设备/%d月)", int(devices), int(months))
	} else {
		var pkg models.Package
		if err := db.First(&pkg, order.PackageID).Error; err != nil {
			return
		}
		deviceLimit = pkg.DeviceLimit
		durationDays = pkg.DurationDays
		pkgName = pkg.Name
	}

	var sub models.Subscription
	if err := db.Where("user_id = ?", order.UserID).First(&sub).Error; err != nil {
		sub = models.Subscription{
			UserID:          order.UserID,
			SubscriptionURL: utils.GenerateRandomString(32),
			DeviceLimit:     deviceLimit,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      time.Now().AddDate(0, 0, durationDays),
		}
		if order.PackageID > 0 {
			pkgID := int64(order.PackageID)
			sub.PackageID = &pkgID
		}
		if err := db.Create(&sub).Error; err != nil {
			utils.SysError("subscription", fmt.Sprintf("创建订阅失败: userID=%d, orderNo=%s, err=%v", order.UserID, order.OrderNo, err))
			return
		}
		utils.CreateSubscriptionLog(sub.ID, order.UserID, "activate", "system", nil, fmt.Sprintf("购买套餐激活订阅: %s", pkgName), nil, nil)
	} else {
		newExpire := sub.ExpireTime
		if newExpire.Before(time.Now()) {
			newExpire = time.Now()
		}
		newExpire = newExpire.AddDate(0, 0, durationDays)
		updates := map[string]interface{}{
			"device_limit": deviceLimit,
			"expire_time":  newExpire,
			"is_active":    true,
			"status":       "active",
		}
		if order.PackageID > 0 {
			pkgID := int64(order.PackageID)
			updates["package_id"] = &pkgID
		}
		db.Model(&sub).Updates(updates)
		utils.CreateSubscriptionLog(sub.ID, order.UserID, "extend", "system", nil, fmt.Sprintf("购买套餐续期订阅: %s, +%d天", pkgName, durationDays), nil, nil)
	}

	var user models.User
	if db.First(&user, order.UserID).Error == nil {
		payAmount := fmt.Sprintf("%.2f", order.Amount)
		if order.FinalAmount != nil {
			payAmount = fmt.Sprintf("%.2f", *order.FinalAmount)
		}
		var subURL string
		var userSub models.Subscription
		if db.Where("user_id = ?", order.UserID).First(&userSub).Error == nil {
			if siteURL := GetSiteURL(); siteURL != "" {
				subURL = siteURL + "/api/v1/subscribe/" + userSub.SubscriptionURL
			}
		}
		emailSubject, emailBody := RenderEmail("payment_success", map[string]string{
			"order_no": order.OrderNo, "amount": payAmount, "package_name": pkgName, "subscription_url": subURL,
		})
		go QueueEmail(user.Email, emailSubject, emailBody, "payment_success")
		go NotifyAdmin("payment_success", map[string]string{
			"username": user.Username, "order_no": order.OrderNo, "package_name": pkgName, "amount": payAmount,
		})
	}

	distributeInviteCommission(db, order)
}

func distributeInviteCommission(db *gorm.DB, order *models.Order) {
	var relation models.InviteRelation
	if err := db.Where("invitee_id = ?", order.UserID).First(&relation).Error; err != nil {
		return
	}
	// Only pay commission on first order
	if relation.InviteeFirstOrderID != nil {
		return
	}
	rateStr := utils.GetSetting("invite_commission_rate")
	if rateStr == "" {
		return
	}
	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil || rate <= 0 {
		return
	}
	payAmount := order.Amount
	if order.FinalAmount != nil {
		payAmount = *order.FinalAmount
	}
	commission := math.Round(payAmount*rate/100*100) / 100
	if commission <= 0 {
		return
	}
	var inviter models.User
	if err := db.First(&inviter, relation.InviterID).Error; err != nil {
		return
	}
	if err := db.Model(&inviter).UpdateColumn("balance", gorm.Expr("balance + ?", commission)).Error; err != nil {
		return
	}
	// Re-read balance for accurate log
	db.First(&inviter, inviter.ID)
	desc := fmt.Sprintf("邀请用户购买返佣 (订单: %s, 比例: %.1f%%)", order.OrderNo, rate)
	db.Create(&models.BalanceLog{
		UserID:         inviter.ID,
		ChangeType:     "invite_commission",
		Amount:         commission,
		BalanceBefore:  inviter.Balance - commission,
		BalanceAfter:   inviter.Balance,
		RelatedOrderID: func() *int64 { id := int64(order.ID); return &id }(),
		Description:    &desc,
	})
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
	db.Model(&relation).Updates(map[string]interface{}{
		"invitee_total_consumption": gorm.Expr("invitee_total_consumption + ?", payAmount),
		"invitee_first_order_id":    order.ID,
	})
}

// ── Subscription format generators ──

// GenerateSurgeConfig generates Surge-compatible proxy list
func GenerateSurgeConfig(nodes []models.Node, siteName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s Surge Config\n", siteName))
	sb.WriteString("[Proxy]\n")
	sb.WriteString("DIRECT = direct\n")
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		line := convertNodeToSurgeLine(node)
		if line != "" {
			sb.WriteString(line + "\n")
		}
	}
	sb.WriteString("\n[Proxy Group]\n")
	sb.WriteString("Proxy = select, ")
	var names []string
	for _, node := range nodes {
		if node.Config != nil && *node.Config != "" {
			names = append(names, node.Name)
		}
	}
	sb.WriteString(strings.Join(names, ", "))
	sb.WriteString("\n")
	return sb.String()
}

func convertNodeToSurgeLine(node models.Node) string {
	if node.Config == nil {
		return ""
	}
	config := *node.Config
	if strings.HasPrefix(config, "ss://") {
		return convertSSToSurge(node.Name, config)
	}
	if strings.HasPrefix(config, "trojan://") {
		return convertTrojanToSurge(node.Name, config)
	}
	return ""
}

func convertSSToSurge(name, config string) string {
	config = strings.TrimPrefix(config, "ss://")
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	method, password, host, port := parseSSConfig(config)
	if method == "" || host == "" {
		return ""
	}
	return fmt.Sprintf("%s = ss, %s, %s, encrypt-method=%s, password=%s", name, host, port, method, password)
}

func convertTrojanToSurge(name, config string) string {
	config = strings.TrimPrefix(config, "trojan://")
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	u, err := url.Parse("trojan://" + config)
	if err != nil {
		return ""
	}
	password := u.User.Username()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	sni := u.Query().Get("sni")
	if sni == "" {
		sni = host
	}
	return fmt.Sprintf("%s = trojan, %s, %s, password=%s, sni=%s", name, host, port, password, sni)
}

// GenerateShadowrocketBase64 generates Shadowrocket-compatible base64 subscription
func GenerateShadowrocketBase64(nodes []models.Node) string {
	return GenerateUniversalBase64(nodes)
}

// GenerateQuantumultXConfig generates QuantumultX server_remote format
func GenerateQuantumultXConfig(nodes []models.Node) string {
	var lines []string
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		config := *node.Config
		if strings.HasPrefix(config, "ss://") {
			line := convertSSToQuantumultX(node.Name, config)
			if line != "" {
				lines = append(lines, line)
			}
		} else if strings.HasPrefix(config, "trojan://") {
			line := convertTrojanToQuantumultX(node.Name, config)
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func convertSSToQuantumultX(name, config string) string {
	config = strings.TrimPrefix(config, "ss://")
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	method, password, host, port := parseSSConfig(config)
	if method == "" || host == "" {
		return ""
	}
	return fmt.Sprintf("shadowsocks=%s:%s, method=%s, password=%s, tag=%s", host, port, method, password, name)
}

func convertTrojanToQuantumultX(name, config string) string {
	config = strings.TrimPrefix(config, "trojan://")
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	u, err := url.Parse("trojan://" + config)
	if err != nil {
		return ""
	}
	password := u.User.Username()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	return fmt.Sprintf("trojan=%s:%s, password=%s, over-tls=true, tls-verification=false, tag=%s", host, port, password, name)
}

// parseSSConfig extracts method, password, host, port from an SS URI (without ss:// prefix and fragment).
func parseSSConfig(config string) (method, password, host, port string) {
	atIdx := strings.LastIndex(config, "@")
	if atIdx < 0 {
		return
	}
	userInfo := config[:atIdx]
	serverInfo := config[atIdx+1:]
	if decoded, err := base64.RawURLEncoding.DecodeString(userInfo); err == nil {
		userInfo = string(decoded)
	} else if decoded, err := base64.StdEncoding.DecodeString(userInfo); err == nil {
		userInfo = string(decoded)
	}
	parts := strings.SplitN(userInfo, ":", 2)
	if len(parts) == 2 {
		method = parts[0]
		password = parts[1]
	}
	hostPort := strings.SplitN(serverInfo, ":", 2)
	if len(hostPort) == 2 {
		host = hostPort[0]
		port = hostPort[1]
	}
	return
}
