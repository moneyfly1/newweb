package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"gorm.io/gorm"
)

// ── Subscription activation ──

// ActivateSubscription creates or extends a subscription after successful payment.
func ActivateSubscription(db *gorm.DB, order *models.Order, paymentMethod string) error {
	fmt.Printf("[subscription] 开始激活订阅: order_id=%d, order_no=%s, user_id=%d, package_id=%d\n",
		order.ID, order.OrderNo, order.UserID, order.PackageID)

	var deviceLimit int
	var durationDays int
	var pkgName string

	if order.PackageID == 0 && order.ExtraData != nil {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(*order.ExtraData), &extra); err != nil {
			return fmt.Errorf("解析订单额外数据失败: %w", err)
		}
		if extra["type"] == "subscription_upgrade" {
			var sub models.Subscription
			if err := db.Where("user_id = ?", order.UserID).First(&sub).Error; err != nil {
				return fmt.Errorf("查找用户订阅失败: %w", err)
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
			if err := db.Model(&sub).Updates(map[string]interface{}{
				"device_limit": newLimit,
				"expire_time":  newExpire,
				"is_active":    true,
				"status":       "active",
			}).Error; err != nil {
				return fmt.Errorf("更新升级订阅失败: %w", err)
			}
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
						subURL = siteURL + "/api/v1/client/subscribe?token=" + userSub.SubscriptionURL
					}
				}
				emailSubject, emailBody := RenderEmail("payment_success", map[string]string{
					"username": user.Username, "order_no": order.OrderNo, "amount": payAmount, "package_name": pkgName, "subscription_url": subURL,
				})
				go QueueEmail(user.Email, emailSubject, emailBody, "payment_success")
				go NotifyAdmin("payment_success", map[string]string{
					"username": user.Username, "order_no": order.OrderNo, "package_name": pkgName, "amount": payAmount,
				})
			}
			distributeInviteCommission(db, order)
			return nil
		}
		if extra["type"] != "custom_package" {
			return fmt.Errorf("未知的订单类型: %v", extra["type"])
		}
		devices, _ := extra["devices"].(float64)
		months, _ := extra["months"].(float64)
		deviceLimit = int(devices)
		durationDays = int(months) * 30
		pkgName = fmt.Sprintf("自定义套餐 (%d设备/%d月)", int(devices), int(months))
	} else {
		var pkg models.Package
		if err := db.First(&pkg, order.PackageID).Error; err != nil {
			return fmt.Errorf("查找套餐失败: %w", err)
		}
		deviceLimit = pkg.DeviceLimit
		durationDays = pkg.DurationDays
		pkgName = pkg.Name
	}

	var sub models.Subscription
	if err := db.Where("user_id = ?", order.UserID).First(&sub).Error; err != nil {
		// Create new subscription
		fmt.Printf("[subscription] 创建新订阅: user_id=%d, device_limit=%d, duration_days=%d\n",
			order.UserID, deviceLimit, durationDays)
		sub = models.Subscription{
			UserID:          order.UserID,
			SubscriptionURL: utils.GenerateHexToken(),
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
			return fmt.Errorf("创建订阅失败: %w", err)
		}
		utils.CreateSubscriptionLog(sub.ID, order.UserID, "activate", "system", nil, fmt.Sprintf("购买套餐激活订阅: %s", pkgName), nil, nil)
		fmt.Printf("[subscription] 订阅创建成功: subscription_id=%d\n", sub.ID)
	} else {
		// Extend existing subscription
		fmt.Printf("[subscription] 续期现有订阅: subscription_id=%d, old_expire=%s, add_days=%d\n",
			sub.ID, sub.ExpireTime.Format("2006-01-02"), durationDays)
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
		if err := db.Model(&sub).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新订阅失败: %w", err)
		}
		utils.CreateSubscriptionLog(sub.ID, order.UserID, "extend", "system", nil, fmt.Sprintf("购买套餐续期订阅: %s, +%d天", pkgName, durationDays), nil, nil)
		fmt.Printf("[subscription] 订阅续期成功: subscription_id=%d, new_expire=%s\n", sub.ID, newExpire.Format("2006-01-02"))
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
				subURL = siteURL + "/api/v1/client/subscribe?token=" + userSub.SubscriptionURL
			}
		}
		emailSubject, emailBody := RenderEmail("payment_success", map[string]string{
			"username": user.Username, "order_no": order.OrderNo, "amount": payAmount, "package_name": pkgName, "subscription_url": subURL,
		})
		go QueueEmail(user.Email, emailSubject, emailBody, "payment_success")
		go NotifyAdmin("payment_success", map[string]string{
			"username": user.Username, "order_no": order.OrderNo, "package_name": pkgName, "amount": payAmount,
		})
	}

	distributeInviteCommission(db, order)
	fmt.Printf("[subscription] 订阅激活完成: order_no=%s, package=%s\n", order.OrderNo, pkgName)
	return nil
}

func distributeInviteCommission(db *gorm.DB, order *models.Order) {
	var relation models.InviteRelation
	if err := db.Where("invitee_id = ?", order.UserID).First(&relation).Error; err != nil {
		return
	}
	// Only pay commission on first order - use atomic CAS to prevent double payout
	if relation.InviteeFirstOrderID != nil {
		return
	}
	// 原子性设置 first_order_id，仅当仍为 NULL 时更新（防止并发双重发放）
	result := db.Model(&models.InviteRelation{}).
		Where("id = ? AND invitee_first_order_id IS NULL", relation.ID).
		Update("invitee_first_order_id", order.ID)
	if result.Error != nil || result.RowsAffected == 0 {
		// 另一个并发请求已经设置了，跳过
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
	if err := db.First(&inviter, inviter.ID).Error; err != nil {
		utils.SysError("subscription", fmt.Sprintf("返佣后读取邀请人余额失败: inviter=%d err=%v", inviter.ID, err))
		return
	}
	desc := fmt.Sprintf("邀请用户购买返佣 (订单: %s, 比例: %.1f%%)", order.OrderNo, rate)
	if err := db.Create(&models.BalanceLog{
		UserID:         inviter.ID,
		ChangeType:     "invite_commission",
		Amount:         commission,
		BalanceBefore:  inviter.Balance - commission,
		BalanceAfter:   inviter.Balance,
		RelatedOrderID: func() *int64 { id := int64(order.ID); return &id }(),
		Description:    &desc,
	}).Error; err != nil {
		utils.SysError("subscription", fmt.Sprintf("记录返佣余额日志失败: inviter=%d order=%d err=%v", inviter.ID, order.ID, err))
		return
	}
	orderID := int64(order.ID)
	relationID := int64(relation.ID)
	if err := db.Create(&models.CommissionLog{
		InviterID:        relation.InviterID,
		InviteeID:        relation.InviteeID,
		InviteRelationID: &relationID,
		CommissionType:   "purchase",
		Amount:           commission,
		RelatedOrderID:   &orderID,
		Status:           "settled",
		Description:      &desc,
	}).Error; err != nil {
		utils.SysError("subscription", fmt.Sprintf("记录返佣日志失败: relation=%d order=%d err=%v", relation.ID, order.ID, err))
		return
	}
	if err := db.Model(&relation).Updates(map[string]interface{}{
		"invitee_total_consumption": gorm.Expr("invitee_total_consumption + ?", payAmount),
	}).Error; err != nil {
		utils.SysError("subscription", fmt.Sprintf("更新邀请关系返佣信息失败: relation=%d order=%d err=%v", relation.ID, order.ID, err))
	}
}

// ── Subscription format generators ──

func formatSafeNodeName(name string) string {
	cleaned := strings.TrimSpace(name)
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return "Node"
	}
	return cleaned
}

func formatSafeCommaName(name string) string {
	cleaned := formatSafeNodeName(name)
	cleaned = strings.ReplaceAll(cleaned, ",", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}

func formatSafeSingBoxTag(name string) string {
	cleaned := formatSafeNodeName(name)
	re := regexp.MustCompile(`[^\p{L}\p{N}_\-⭐ ]+`)
	cleaned = re.ReplaceAllString(cleaned, "_")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return "Node"
	}
	return cleaned
}

// GenerateSurgeConfig generates Surge-compatible proxy list
func GenerateSurgeConfig(nodes []models.Node, siteName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s Surge Config\n", siteName))
	sb.WriteString("[Proxy]\n")
	sb.WriteString("DIRECT = direct\n")
	var names []string
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		if strings.Contains(*node.Config, "baidu.com") {
			continue
		}
		line := convertNodeToSurgeLine(node)
		if line != "" {
			sb.WriteString(line + "\n")
			names = append(names, formatSafeCommaName(node.Name))
		}
	}
	sb.WriteString("\n[Proxy Group]\n")
	if len(names) > 0 {
		sb.WriteString("Proxy = select, AutoTest, DIRECT, " + strings.Join(names, ", ") + "\n")
		sb.WriteString("AutoTest = url-test, " + strings.Join(names, ", ") + ", url=http://www.gstatic.com/generate_204, interval=300, tolerance=50\n")
	} else {
		sb.WriteString("Proxy = select, DIRECT\n")
	}
	sb.WriteString("\n[Rule]\n")
	sb.WriteString("DOMAIN-SET,https://cdn.jsdelivr.net/gh/Loyalsoldier/surge-rules@release/reject.txt,REJECT\n")
	sb.WriteString("DOMAIN-SET,https://cdn.jsdelivr.net/gh/Loyalsoldier/surge-rules@release/proxy.txt,Proxy\n")
	sb.WriteString("DOMAIN-SET,https://cdn.jsdelivr.net/gh/Loyalsoldier/surge-rules@release/direct.txt,DIRECT\n")
	sb.WriteString("RULE-SET,https://cdn.jsdelivr.net/gh/Loyalsoldier/surge-rules@release/cncidr.txt,DIRECT\n")
	sb.WriteString("GEOIP,CN,DIRECT,no-resolve\n")
	sb.WriteString("FINAL,Proxy,dns-failed\n")
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
	name = formatSafeCommaName(name)
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
	name = formatSafeCommaName(name)
	return fmt.Sprintf("%s = trojan, %s, %s, password=%s, sni=%s", name, host, port, password, sni)
}

// GenerateShadowrocketBase64 generates Shadowrocket-compatible base64 subscription
func GenerateShadowrocketBase64(nodes []models.Node) string {
	return GenerateUniversalBase64(nodes)
}

// GenerateQuantumultXConfig generates a full QuantumultX configuration profile.
func GenerateQuantumultXConfig(nodes []models.Node) string {
	var proxyLines []string
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		if strings.Contains(*node.Config, "baidu.com") {
			continue
		}
		config := *node.Config
		name := formatSafeCommaName(node.Name)
		var line string
		if strings.HasPrefix(config, "ss://") {
			line = convertSSToQuantumultX(name, config)
		} else if strings.HasPrefix(config, "trojan://") {
			line = convertTrojanToQuantumultX(name, config)
		} else {
			m, err := NodeConfigToClashMap(node.Type, config, node.Name)
			if err == nil {
				line = clashMapToQuantumultXLine(name, m)
			}
		}
		if line != "" {
			proxyLines = append(proxyLines, line)
		}
	}

	var sb strings.Builder
	sb.WriteString("[general]\n")
	sb.WriteString("network_check_url = http://www.gstatic.com/generate_204\n")
	sb.WriteString("server_check_url = http://www.gstatic.com/generate_204\n")
	sb.WriteString("geo_location_checker = http://www.gstatic.com/generate_204\n\n")

	sb.WriteString("[server_local]\n")
	for _, l := range proxyLines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString("\n")

	sb.WriteString("[filter_remote]\n")
	sb.WriteString("https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/QuantumultX/Advertising/Advertising.list, tag=Advertising, force-policy=REJECT, update-interval=86400, opt-parser=true, enabled=true\n")
	sb.WriteString("https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/QuantumultX/Global/Global.list, tag=Global, force-policy=Proxy, update-interval=86400, opt-parser=true, enabled=true\n")
	sb.WriteString("https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/QuantumultX/China/China.list, tag=China, force-policy=DIRECT, update-interval=86400, opt-parser=true, enabled=true\n\n")

	sb.WriteString("[filter_local]\n")
	sb.WriteString("GEOIP,CN,DIRECT\n")
	sb.WriteString("FINAL,Proxy\n\n")

	sb.WriteString("[rewrite_remote]\n\n")
	sb.WriteString("[rewrite_local]\n\n")
	sb.WriteString("[task_local]\n\n")
	sb.WriteString("[mitm]\n")
	sb.WriteString("skip_validating_cert = true\n")
	return sb.String()
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
	name = formatSafeCommaName(name)
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
	name = formatSafeCommaName(name)
	return fmt.Sprintf("trojan=%s:%s, password=%s, over-tls=true, tls-verification=false, tag=%s", host, port, password, name)
}

// clashMapToQuantumultXLine converts a clash map to a QuantumultX proxy line.
func clashMapToQuantumultXLine(name string, m map[string]interface{}) string {
	typ, _ := m["type"].(string)
	server, _ := m["server"].(string)
	port := clashMapPortStr(m)
	if server == "" || port == "" {
		return ""
	}
	switch typ {
	case "ss":
		cipher, _ := m["cipher"].(string)
		password, _ := m["password"].(string)
		return fmt.Sprintf("shadowsocks=%s:%s, method=%s, password=%s, tag=%s", server, port, cipher, password, name)
	case "trojan":
		password, _ := m["password"].(string)
		sni := loonGetSNI(m)
		skipVerify := "false"
		if sv, ok := m["skip-cert-verify"].(bool); ok && sv {
			skipVerify = "true"
		}
		line := fmt.Sprintf("trojan=%s:%s, password=%s, over-tls=true, tls-host=%s, tls-verification=%s, fast-open=false, udp-relay=false, tag=%s",
			server, port, password, sni, func() string {
				if skipVerify == "true" {
					return "false"
				}
				return "true"
			}(), name)
		return line
	case "vmess":
		uuid, _ := m["uuid"].(string)
		cipher, _ := m["cipher"].(string)
		if cipher == "" || cipher == "auto" {
			cipher = "chacha20-poly1305"
		}
		tls, _ := m["tls"].(bool)
		sni, _ := m["servername"].(string)
		network, _ := m["network"].(string)
		tlsStr := "false"
		if tls {
			tlsStr = "true"
		}
		line := fmt.Sprintf("vmess=%s:%s, method=%s, password=%s", server, port, cipher, uuid)
		if tls {
			line += fmt.Sprintf(", obfs=over-tls, obfs-host=%s", sni)
		}
		if network == "ws" {
			wsPath, wsHost := extractWSParams(m)
			if tls {
				line += fmt.Sprintf(", obfs=wss, obfs-uri=%s", wsPath)
			} else {
				line += fmt.Sprintf(", obfs=ws, obfs-uri=%s", wsPath)
			}
			if wsHost != "" {
				line += fmt.Sprintf(", obfs-host=%s", wsHost)
			}
			_ = tlsStr
		}
		line += fmt.Sprintf(", tag=%s", name)
		return line
	case "vless":
		uuid, _ := m["uuid"].(string)
		sni, _ := m["servername"].(string)
		return fmt.Sprintf("vless=%s:%s, password=%s, obfs=over-tls, obfs-host=%s, tag=%s", server, port, uuid, sni, name)
	}
	return ""
}

// GenerateLoonConfig generates Loon-compatible proxy configuration.
func GenerateLoonConfig(nodes []models.Node, siteName string) string {
	var proxyLines []string
	var proxyNames []string
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		// Skip info nodes (server = baidu.com placeholder)
		if strings.Contains(*node.Config, "baidu.com") {
			continue
		}
		m, err := NodeConfigToClashMap(node.Type, *node.Config, node.Name)
		if err != nil {
			continue
		}
		name := formatSafeCommaName(node.Name)
		line := clashMapToLoonLine(name, m)
		if line != "" {
			proxyLines = append(proxyLines, line)
			proxyNames = append(proxyNames, name)
		}
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s - Loon Config\n\n", siteName))
	sb.WriteString("[Proxy]\n")
	for _, l := range proxyLines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString("\n[Proxy Group]\n")
	if len(proxyNames) > 0 {
		sb.WriteString("Proxy = select, " + strings.Join(proxyNames, ", ") + "\n")
	} else {
		sb.WriteString("Proxy = select, DIRECT\n")
	}
	sb.WriteString("\n[Remote Rule]\n")
	sb.WriteString("https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Loon/Advertising/Advertising.list, policy=REJECT, tag=Advertising, enabled=true\n")
	sb.WriteString("https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Loon/Global/Global.list, policy=Proxy, tag=Global, enabled=true\n")
	sb.WriteString("https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Loon/ChinaMax/ChinaMax.list, policy=DIRECT, tag=ChinaMax, enabled=true\n")
	sb.WriteString("\n[Rule]\n")
	sb.WriteString("GEOIP,CN,DIRECT,no-resolve\n")
	sb.WriteString("FINAL,Proxy\n")
	return sb.String()
}

func clashMapToLoonLine(name string, m map[string]interface{}) string {
	typ, _ := m["type"].(string)
	server, _ := m["server"].(string)
	port := clashMapPortStr(m)
	if server == "" || port == "" {
		return ""
	}
	switch typ {
	case "ss":
		cipher, _ := m["cipher"].(string)
		password, _ := m["password"].(string)
		return fmt.Sprintf("%s = Shadowsocks,%s,%s,%s,%s,fast-open=false,udp=true", name, server, port, cipher, password)
	case "vmess":
		uuid, _ := m["uuid"].(string)
		cipher, _ := m["cipher"].(string)
		if cipher == "" || cipher == "auto" {
			cipher = "auto"
		}
		tls, _ := m["tls"].(bool)
		sni, _ := m["servername"].(string)
		tlsStr := "false"
		if tls {
			tlsStr = "true"
		}
		network, _ := m["network"].(string)
		if network == "ws" {
			wsPath, wsHost := extractWSParams(m)
			extra := ""
			if wsHost != "" {
				extra = ",host=" + wsHost
			}
			return fmt.Sprintf("%s = VMESS,%s,%s,%s,%s,over-tls=%s,tls-name=%s,transport=ws,path=%s%s",
				name, server, port, cipher, uuid, tlsStr, sni, wsPath, extra)
		}
		return fmt.Sprintf("%s = VMESS,%s,%s,%s,%s,over-tls=%s,tls-name=%s", name, server, port, cipher, uuid, tlsStr, sni)
	case "trojan":
		password, _ := m["password"].(string)
		sni := loonGetSNI(m)
		skipVerify := "false"
		if sv, ok := m["skip-cert-verify"].(bool); ok && sv {
			skipVerify = "true"
		}
		return fmt.Sprintf("%s = Trojan,%s,%s,%s,over-tls=true,tls-name=%s,skip-cert-verify=%s", name, server, port, password, sni, skipVerify)
	case "vless":
		uuid, _ := m["uuid"].(string)
		sni, _ := m["servername"].(string)
		return fmt.Sprintf("%s = VLESS,%s,%s,%s,over-tls=true,tls-name=%s", name, server, port, uuid, sni)
	}
	return ""
}

// GenerateSingBoxConfig generates SingBox JSON outbound configuration.
func GenerateSingBoxConfig(nodes []models.Node) string {
	var outbounds []map[string]interface{}
	var proxyNames []string
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		if strings.Contains(*node.Config, "baidu.com") {
			continue
		}
		m, err := NodeConfigToClashMap(node.Type, *node.Config, node.Name)
		if err != nil {
			continue
		}
		tagName := formatSafeSingBoxTag(node.Name)
		ob := clashMapToSingBoxOutbound(tagName, m)
		if ob != nil {
			outbounds = append(outbounds, ob)
			proxyNames = append(proxyNames, tagName)
		}
	}
	selectorOut := append([]string{}, proxyNames...)
	selectorOut = append(selectorOut, "direct")
	allOutbounds := []map[string]interface{}{
		{"type": "selector", "tag": "Proxy", "outbounds": selectorOut},
	}
	allOutbounds = append(allOutbounds, outbounds...)
	allOutbounds = append(allOutbounds,
		map[string]interface{}{"type": "direct", "tag": "direct"},
		map[string]interface{}{"type": "block", "tag": "block"},
		map[string]interface{}{"type": "dns", "tag": "dns-out"},
	)

	config := map[string]interface{}{
		"dns": map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{"tag": "dns-direct", "address": "223.5.5.5", "strategy": "ipv4_only"},
				map[string]interface{}{"tag": "dns-remote", "address": "8.8.8.8", "strategy": "ipv4_only", "detour": "Proxy"},
			},
			"rules": []interface{}{
				map[string]interface{}{"outbound": "any", "server": "dns-direct"},
				map[string]interface{}{"rule_set": "geosite-cn", "server": "dns-direct"},
			},
			"final": "dns-remote",
		},
		"outbounds": allOutbounds,
		"route": map[string]interface{}{
			"rule_set": []interface{}{
				map[string]interface{}{
					"tag": "geosite-ads", "type": "remote", "format": "binary",
					"url":              "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
					"download_detour": "direct",
				},
				map[string]interface{}{
					"tag": "geosite-cn", "type": "remote", "format": "binary",
					"url":              "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
					"download_detour": "direct",
				},
				map[string]interface{}{
					"tag": "geoip-cn", "type": "remote", "format": "binary",
					"url":              "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
					"download_detour": "direct",
				},
			},
			"rules": []interface{}{
				map[string]interface{}{"protocol": "dns", "outbound": "dns-out"},
				map[string]interface{}{"rule_set": []string{"geosite-ads"}, "outbound": "block"},
				map[string]interface{}{"rule_set": []string{"geosite-cn", "geoip-cn"}, "outbound": "direct"},
			},
			"final":                  "Proxy",
			"auto_detect_interface": true,
		},
		"cache_file": map[string]interface{}{"enabled": true},
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func clashMapToSingBoxOutbound(name string, m map[string]interface{}) map[string]interface{} {
	typ, _ := m["type"].(string)
	server, _ := m["server"].(string)
	port := clashMapPortInt(m)
	if server == "" || port == 0 {
		return nil
	}
	switch typ {
	case "ss":
		cipher, _ := m["cipher"].(string)
		password, _ := m["password"].(string)
		return map[string]interface{}{
			"type": "shadowsocks", "tag": name,
			"server": server, "server_port": port,
			"method": cipher, "password": password,
		}
	case "vmess":
		uuid, _ := m["uuid"].(string)
		alterId := clashMapIntField(m, "alterId")
		security, _ := m["cipher"].(string)
		if security == "" {
			security = "auto"
		}
		tls, _ := m["tls"].(bool)
		sni, _ := m["servername"].(string)
		ob := map[string]interface{}{
			"type": "vmess", "tag": name,
			"server": server, "server_port": port,
			"uuid": uuid, "alter_id": alterId, "security": security,
		}
		if tls {
			ob["tls"] = map[string]interface{}{"enabled": true, "server_name": sni}
		}
		network, _ := m["network"].(string)
		if network == "ws" {
			wsPath, wsHost := extractWSParams(m)
			transport := map[string]interface{}{"type": "ws", "path": wsPath}
			if wsHost != "" {
				transport["headers"] = map[string]interface{}{"Host": wsHost}
			}
			ob["transport"] = transport
		}
		return ob
	case "trojan":
		password, _ := m["password"].(string)
		sni := loonGetSNI(m)
		skipVerify, _ := m["skip-cert-verify"].(bool)
		return map[string]interface{}{
			"type": "trojan", "tag": name,
			"server": server, "server_port": port,
			"password": password,
			"tls": map[string]interface{}{"enabled": true, "server_name": sni, "insecure": skipVerify},
		}
	case "vless":
		uuid, _ := m["uuid"].(string)
		sni, _ := m["servername"].(string)
		return map[string]interface{}{
			"type": "vless", "tag": name,
			"server": server, "server_port": port,
			"uuid": uuid,
			"tls": map[string]interface{}{"enabled": true, "server_name": sni},
		}
	case "hysteria2":
		password, _ := m["password"].(string)
		sni := loonGetSNI(m)
		return map[string]interface{}{
			"type": "hysteria2", "tag": name,
			"server": server, "server_port": port,
			"password": password,
			"tls": map[string]interface{}{"enabled": true, "server_name": sni},
		}
	case "hysteria":
		authStr, _ := m["auth_str"].(string)
		sni := loonGetSNI(m)
		up, _ := m["up"].(string)
		down, _ := m["down"].(string)
		return map[string]interface{}{
			"type": "hysteria", "tag": name,
			"server": server, "server_port": port,
			"auth_str": authStr, "up": up, "down": down,
			"tls": map[string]interface{}{"enabled": true, "server_name": sni},
		}
	}
	return nil
}

// Helper: get port as string from clash map
func clashMapPortStr(m map[string]interface{}) string {
	switch v := m["port"].(type) {
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.Itoa(int(v))
	case string:
		return v
	}
	return ""
}

// Helper: get port as int from clash map
func clashMapPortInt(m map[string]interface{}) int {
	switch v := m["port"].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

// Helper: get an int field from clash map (handles float64 from JSON)
func clashMapIntField(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

// Helper: extract ws path and host from clash map ws-opts
func extractWSParams(m map[string]interface{}) (path, host string) {
	path = "/"
	wsOpts, _ := m["ws-opts"].(map[string]interface{})
	if wsOpts == nil {
		return
	}
	if p, ok := wsOpts["path"].(string); ok && p != "" {
		path = p
	}
	if h, ok := wsOpts["headers"].(map[string]interface{}); ok {
		if hv, ok := h["Host"].(string); ok {
			host = hv
		}
	}
	return
}

// Helper: get SNI from trojan/hysteria map (tries "sni" then "servername")
func loonGetSNI(m map[string]interface{}) string {
	if sni, ok := m["sni"].(string); ok && sni != "" {
		return sni
	}
	sni, _ := m["servername"].(string)
	return sni
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
