package handlers

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// subscriptionStatus represents the state of a subscription access
type subscriptionStatus int

const (
	subStatusOK subscriptionStatus = iota
	subStatusNotFound
	subStatusExpired
	subStatusInactive
	subStatusDeviceOverLimit
)

// subscriptionContext holds all info needed for subscription generation
type subscriptionContext struct {
	Sub            *models.Subscription
	Nodes          []models.Node
	Status         subscriptionStatus
	SiteURL        string
	SupportContact string
	CurrentDevices int
	DeviceLimit    int
	ClientInfo     *services.ClientInfo
}

// getSubscriptionSiteConfig reads site URL and support contact from system_configs
func getSubscriptionSiteConfig() (siteURL, supportContact string) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("`key` IN ?",
		[]string{"site_url", "domain_name", "support_qq", "support_telegram", "support_email"}).Find(&configs)

	var contacts []string
	for _, c := range configs {
		switch c.Key {
		case "site_url":
			// site_url takes priority over domain_name
			if c.Value != "" {
				siteURL = c.Value
			}
		case "domain_name":
			// fallback if site_url is not set
			if siteURL == "" && c.Value != "" {
				siteURL = c.Value
			}
		case "support_qq":
			if c.Value != "" {
				contacts = append(contacts, "QQ:"+c.Value)
			}
		case "support_telegram":
			if c.Value != "" {
				contacts = append(contacts, "TG:@"+c.Value)
			}
		case "support_email":
			if c.Value != "" {
				contacts = append(contacts, c.Value)
			}
		}
	}
	supportContact = strings.Join(contacts, " | ")
	if siteURL != "" && !strings.HasPrefix(siteURL, "http") {
		siteURL = "https://" + siteURL
	}
	siteURL = strings.TrimRight(siteURL, "/")
	return
}

// createInfoNode creates a dummy SS node used to display messages in client
func createInfoNode(name string) models.Node {
	config := fmt.Sprintf("ss://%s@baidu.com:1234#%s",
		"YWVzLTEyOC1nY206aW5mbw==", // base64("aes-128-gcm:info")
		strings.ReplaceAll(name, " ", "%20"))
	return models.Node{
		Name:     name,
		Type:     "ss",
		Status:   "online",
		Config:   &config,
		IsActive: true,
	}
}

// buildSubscriptionContext validates subscription and prepares context
func buildSubscriptionContext(c *gin.Context) *subscriptionContext {
	url := c.Param("url")
	db := database.GetDB()

	siteURL, supportContact := getSubscriptionSiteConfig()
	ua := c.GetHeader("User-Agent")
	clientInfo := services.ParseUserAgent(ua)

	ctx := &subscriptionContext{SiteURL: siteURL, SupportContact: supportContact, ClientInfo: clientInfo}

	var sub models.Subscription
	if err := db.Where("subscription_url = ?", url).First(&sub).Error; err != nil {
		ctx.Status = subStatusNotFound
		return ctx
	}
	ctx.Sub = &sub
	ctx.DeviceLimit = sub.DeviceLimit
	ctx.CurrentDevices = sub.CurrentDevices

	if !sub.IsActive || sub.Status != "active" {
		ctx.Status = subStatusInactive
		return ctx
	}
	if time.Now().After(sub.ExpireTime) {
		ctx.Status = subStatusExpired
		return ctx
	}

	// Browser requests: allow access but don't count as device
	if clientInfo.IsBrowser {
		var nodes []models.Node
		db.Where("is_active = ? AND status = ?", true, "online").Order("order_index ASC").Find(&nodes)
		customNodes := fetchUserCustomNodes(db, sub.UserID, sub.ExpireTime)
		nodes = append(customNodes, nodes...)
		ctx.Nodes = nodes
		ctx.Status = subStatusOK
		return ctx
	}

	// Device fingerprint tracking using feature-based hash
	ip := utils.GetRealClientIP(c)
	fingerprint := services.GenerateDeviceFingerprint(ua, ip)

	var device models.Device
	err := db.Where("subscription_id = ? AND device_fingerprint = ? AND is_active = ?", sub.ID, fingerprint, true).First(&device).Error
	if err != nil {
		// New device — check limit
		if sub.CurrentDevices >= sub.DeviceLimit {
			ctx.Status = subStatusDeviceOverLimit
			return ctx
		}
		now := time.Now()
		softwareName := clientInfo.SoftwareName
		softwareVer := clientInfo.SoftwareVersion
		osName := clientInfo.OSName
		osVer := clientInfo.OSVersion
		deviceModel := clientInfo.DeviceModel
		deviceBrand := clientInfo.DeviceBrand
		deviceType := clientInfo.DeviceType
		subType := clientInfo.SubscriptionType
		deviceName := buildDeviceName(clientInfo)
		userID := int64(sub.UserID)
		device = models.Device{
			UserID:            &userID,
			SubscriptionID:    sub.ID,
			DeviceFingerprint: fingerprint,
			UserAgent:         &ua,
			IPAddress:         &ip,
			SoftwareName:      &softwareName,
			SoftwareVersion:   &softwareVer,
			OSName:            &osName,
			OSVersion:         &osVer,
			DeviceModel:       &deviceModel,
			DeviceBrand:       &deviceBrand,
			DeviceType:        &deviceType,
			DeviceName:        &deviceName,
			SubscriptionType:  &subType,
			IsActive:          true,
			IsAllowed:         true,
			FirstSeen:         &now,
			LastAccess:        now,
			AccessCount:       1,
		}
		db.Create(&device)
		db.Model(&sub).Update("current_devices", sub.CurrentDevices+1)
		ctx.CurrentDevices = sub.CurrentDevices + 1

		// Lookup and update region asynchronously
		go func(deviceID uint, ipAddr string) {
			region := utils.GetIPLocation(ipAddr)
			if region != "" {
				database.GetDB().Model(&models.Device{}).Where("id = ?", deviceID).Update("region", region)
			}
		}(device.ID, ip)
	} else {
		db.Model(&device).Updates(map[string]interface{}{
			"last_access":  time.Now(),
			"access_count": device.AccessCount + 1,
			"ip_address":   ip,
		})

		// Update region asynchronously if IP changed
		go func(deviceID uint, ipAddr string, oldIP *string) {
			if oldIP == nil || *oldIP != ipAddr {
				region := utils.GetIPLocation(ipAddr)
				if region != "" {
					database.GetDB().Model(&models.Device{}).Where("id = ?", deviceID).Update("region", region)
				}
			}
		}(device.ID, ip, device.IPAddress)
	}

	var nodes []models.Node
	db.Where("is_active = ? AND status = ?", true, "online").Order("order_index ASC").Find(&nodes)
	customNodes := fetchUserCustomNodes(db, sub.UserID, sub.ExpireTime)
	nodes = append(customNodes, nodes...)
	ctx.Nodes = nodes
	ctx.Status = subStatusOK
	return ctx
}

// fetchUserCustomNodes returns custom nodes assigned to a user, converted to models.Node format.
// subExpireTime is the user's subscription expiry, used for FollowUserExpire nodes.
func fetchUserCustomNodes(db *gorm.DB, userID uint, subExpireTime time.Time) []models.Node {
	var assignments []models.UserCustomNode
	db.Where("user_id = ?", userID).Find(&assignments)
	if len(assignments) == 0 {
		return nil
	}

	var customNodeIDs []uint
	for _, a := range assignments {
		customNodeIDs = append(customNodeIDs, a.CustomNodeID)
	}

	var customNodes []models.CustomNode
	db.Where("id IN ? AND is_active = ?", customNodeIDs, true).Find(&customNodes)

	now := time.Now()
	var nodes []models.Node
	for _, cn := range customNodes {
		// Check custom node's own expiry
		if cn.ExpireTime != nil && cn.ExpireTime.Before(now) {
			continue
		}
		// If FollowUserExpire is set, also check user's subscription expiry
		if cn.FollowUserExpire && now.After(subExpireTime) {
			continue
		}
		config := cn.Config
		displayName := cn.DisplayName
		if displayName == "" {
			displayName = cn.Name
		}
		nodes = append(nodes, models.Node{
			Name:     "⭐ " + displayName,
			Type:     cn.Protocol,
			Status:   "online",
			Config:   &config,
			IsActive: true,
		})
	}
	return nodes
}

func buildDeviceName(info *services.ClientInfo) string {
	parts := []string{}
	if info.SoftwareName != "Unknown" {
		parts = append(parts, info.SoftwareName)
	}
	if info.DeviceModel != "" {
		parts = append(parts, info.DeviceModel)
	} else if info.DeviceBrand != "" && info.DeviceBrand != "Android" {
		parts = append(parts, info.DeviceBrand)
	}
	if info.OSName != "Unknown" {
		osStr := info.OSName
		if info.OSVersion != "" {
			osStr += " " + info.OSVersion
		}
		parts = append(parts, osStr)
	}
	if info.SoftwareVersion != "" {
		parts = append(parts, "v"+info.SoftwareVersion)
	}
	if len(parts) == 0 {
		return "Unknown Device"
	}
	return strings.Join(parts, " - ")
}

// getInfoNodes returns informational nodes to prepend to the proxy list
func getInfoNodes(ctx *subscriptionContext) []models.Node {
	siteLabel := ctx.SiteURL
	if siteLabel == "" {
		siteLabel = "请在系统设置中配置域名"
	}

	var infoNodes []models.Node
	infoNodes = append(infoNodes, createInfoNode("📢 官网: "+siteLabel))

	if ctx.Sub != nil {
		expireStr := "无限期"
		if !ctx.Sub.ExpireTime.IsZero() {
			expireStr = ctx.Sub.ExpireTime.Format("2006-01-02")
		}
		infoNodes = append(infoNodes, createInfoNode("⏰ 到期: "+expireStr))
		infoNodes = append(infoNodes, createInfoNode(fmt.Sprintf("📱 设备: %d/%d", ctx.CurrentDevices, ctx.DeviceLimit)))
	}

	if ctx.SupportContact != "" {
		infoNodes = append(infoNodes, createInfoNode("💬 客服: "+ctx.SupportContact))
	}

	return infoNodes
}

// getErrorNodes returns error message nodes for abnormal subscription states
func getErrorNodes(ctx *subscriptionContext) []models.Node {
	siteLabel := ctx.SiteURL
	if siteLabel == "" {
		siteLabel = "请在系统设置中配置域名"
	}

	var reason, solution string
	switch ctx.Status {
	case subStatusNotFound:
		reason = "订阅不存在"
		solution = "请检查订阅地址是否正确"
	case subStatusExpired:
		reason = "订阅已过期"
		expireStr := ""
		if ctx.Sub != nil {
			expireStr = ctx.Sub.ExpireTime.Format("2006-01-02")
		}
		solution = fmt.Sprintf("请前往官网续费 (过期时间: %s)", expireStr)
	case subStatusInactive:
		reason = "订阅已失效"
		solution = "请联系管理员检查订阅状态"
	case subStatusDeviceOverLimit:
		reason = "设备数量超限"
		solution = fmt.Sprintf("当前设备 %d/%d，请在官网删除不使用的设备", ctx.CurrentDevices, ctx.DeviceLimit)
	}

	nodes := []models.Node{
		createInfoNode("📢 官网: " + siteLabel),
		createInfoNode("❌ 原因: " + reason),
		createInfoNode("💡 解决: " + solution),
	}
	if ctx.SupportContact != "" {
		nodes = append(nodes, createInfoNode("💬 客服: "+ctx.SupportContact))
	}
	return nodes
}

// generateSubscriptionName generates the subscription display name for Clash/Sparkle 等客户端（显示为配置名称，含站点与到期时间）.
func generateSubscriptionName(ctx *subscriptionContext) string {
	if ctx.Status != subStatusOK {
		switch ctx.Status {
		case subStatusExpired:
			return "订阅已过期"
		case subStatusInactive:
			return "订阅已失效"
		case subStatusDeviceOverLimit:
			return "设备超限"
		case subStatusNotFound:
			return "订阅不存在"
		default:
			return "订阅异常"
		}
	}
	expireStr := "无限期"
	if ctx.Sub != nil && !ctx.Sub.ExpireTime.IsZero() {
		expireStr = fmt.Sprintf("到期: %s", ctx.Sub.ExpireTime.Format("2006-01-02"))
	}
	// 优先显示站点名 + 到期时间，便于 Sparkle 等客户端识别
	if ctx.SiteURL != "" {
		u, err := url.Parse(ctx.SiteURL)
		if err == nil && u.Host != "" {
			return u.Host + " " + expireStr
		}
	}
	return expireStr
}

// GetSubscription serves subscription content, auto-detecting client type from User-Agent.
// For explicit Clash format, use /sub/clash/:url
// For explicit universal format, use /sub/:url
func GetSubscription(c *gin.Context) {
	ctx := buildSubscriptionContext(c)

	// If accessed via /sub/clash/ path, force clash format regardless of UA
	// Otherwise check explicit format parameter, then auto-detect from client info
	subType := c.Query("format")
	if subType == "" {
		if strings.Contains(c.Request.URL.Path, "/clash/") {
			subType = "clash"
		} else if ctx.ClientInfo != nil {
			subType = ctx.ClientInfo.SubscriptionType
		} else {
			subType = "clash"
		}
	}

	// Determine output format
	useClash := subType == "clash" || subType == "stash"
	useSurge := subType == "surge"
	useQuantumultX := subType == "quantumult" || subType == "quantumultx"

	var nodes []models.Node
	if ctx.Status != subStatusOK {
		nodes = getErrorNodes(ctx)
	} else {
		nodes = append(getInfoNodes(ctx), ctx.Nodes...)
		incrementSubscriptionCounter(ctx.Sub, subType)
	}

	subscriptionName := generateSubscriptionName(ctx)

	if useClash {
		yamlContent := services.GenerateClashYAMLWithDomain(nodes, ctx.SiteURL, subscriptionName)
		fileName := subscriptionName
		if strings.HasPrefix(subscriptionName, "到期: ") {
			fileName = "到期时间" + strings.TrimPrefix(subscriptionName, "到期: ")
		}
		encodedName := url.QueryEscape(fileName)
		c.Header("Content-Type", "text/yaml; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s.yaml", encodedName))
		c.Header("Subscription-Title", subscriptionName)
		c.Header("Profile-Title", subscriptionName)
		setSubscriptionHeaders(c, ctx)
		c.Data(http.StatusOK, "text/yaml; charset=utf-8", []byte(yamlContent))
	} else if useSurge {
		surgeContent := services.GenerateSurgeConfig(nodes, ctx.SiteURL)
		encodedName := url.QueryEscape(subscriptionName)
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s.conf", encodedName))
		c.Header("Subscription-Title", subscriptionName)
		c.Header("Profile-Title", subscriptionName)
		setSubscriptionHeaders(c, ctx)
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(surgeContent))
	} else if useQuantumultX {
		qxContent := services.GenerateQuantumultXConfig(nodes)
		encodedName := url.QueryEscape(subscriptionName)
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s.conf", encodedName))
		c.Header("Subscription-Title", subscriptionName)
		c.Header("Profile-Title", subscriptionName)
		setSubscriptionHeaders(c, ctx)
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(qxContent))
	} else {
		encoded := services.GenerateUniversalBase64(nodes)
		encodedName := url.QueryEscape(subscriptionName)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
		c.Header("Subscription-Title", subscriptionName)
		c.Header("Profile-Title", subscriptionName)
		setSubscriptionHeaders(c, ctx)
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(encoded))
	}
}

// GetUniversalSubscription serves base64-encoded protocol links (explicit).
func GetUniversalSubscription(c *gin.Context) {
	ctx := buildSubscriptionContext(c)

	var nodes []models.Node
	if ctx.Status != subStatusOK {
		nodes = getErrorNodes(ctx)
	} else {
		nodes = append(getInfoNodes(ctx), ctx.Nodes...)
		incrementSubscriptionCounter(ctx.Sub, "v2ray")
	}

	subscriptionName := generateSubscriptionName(ctx)

	encoded := services.GenerateUniversalBase64(nodes)
	setSubscriptionHeaders(c, ctx)

	// Set profile name headers for Shadowrocket and other clients
	encodedName := url.QueryEscape(subscriptionName)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
	c.Header("Subscription-Title", subscriptionName)
	c.Header("Profile-Title", subscriptionName)

	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(encoded))
}

// setSubscriptionHeaders sets common subscription response headers（Sparkle 等客户端用 Profile-Title / Profile-Update-Interval 显示名称与自动更新间隔）
func setSubscriptionHeaders(c *gin.Context, ctx *subscriptionContext) {
	if ctx.Sub != nil {
		// subscription-userinfo: upload=0; download=0; total=0; expire=<unix>
		userinfoParts := []string{"upload=0", "download=0", "total=0"}
		if !ctx.Sub.ExpireTime.IsZero() {
			userinfoParts = append(userinfoParts, fmt.Sprintf("expire=%d", ctx.Sub.ExpireTime.Unix()))
		}
		c.Header("Subscription-Userinfo", strings.Join(userinfoParts, "; "))
	}
	// 自动更新间隔：24 小时（单位小时，部分客户端据此设置定时拉取）
	c.Header("Profile-Update-Interval", "24")
	// 部分客户端用分钟
	c.Header("Subscription-Update-Interval", "1440")
}

// incrementSubscriptionCounter increments the appropriate counter based on client type
func incrementSubscriptionCounter(sub *models.Subscription, subType string) {
	db := database.GetDB()
	switch subType {
	case "clash":
		db.Model(sub).Update("clash_count", sub.ClashCount+1)
	case "surge":
		db.Model(sub).Update("surge_count", sub.SurgeCount+1)
	case "shadowrocket":
		db.Model(sub).Update("shadowrocket_count", sub.ShadowrocketCount+1)
	case "quantumult":
		db.Model(sub).Update("quanx_count", sub.QuanXCount+1)
	default:
		db.Model(sub).Update("universal_count", sub.UniversalCount+1)
	}
}

func GetSubscriptionByFormat(c *gin.Context) {
	format := c.Param("format")
	switch strings.ToLower(format) {
	case "clash":
		GetSubscription(c)
	case "v2ray", "base64", "universal":
		GetUniversalSubscription(c)
	default:
		// Default to base64 for unknown formats
		GetUniversalSubscription(c)
	}
}

// getSubscriptionBaseURL reads site_url (or domain_name as fallback) from system_configs and constructs the base URL.
func getSubscriptionBaseURL() string {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("`key` IN ?", []string{"site_url", "domain_name"}).Find(&configs)

	var domain string
	for _, c := range configs {
		if c.Key == "site_url" && c.Value != "" {
			domain = c.Value
			break
		}
		if c.Key == "domain_name" && c.Value != "" && domain == "" {
			domain = c.Value
		}
	}
	if domain == "" {
		return ""
	}
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	return strings.TrimRight(domain, "/")
}

func GetUserSubscription(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var sub models.Subscription
	if err := database.GetDB().Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.NotFound(c, "暂无订阅")
		return
	}

	baseURL := getSubscriptionBaseURL()
	var universalURL, clashURL string
	if baseURL != "" && sub.SubscriptionURL != "" {
		universalURL = fmt.Sprintf("%s/api/v1/sub/%s", baseURL, sub.SubscriptionURL)
		clashURL = fmt.Sprintf("%s/api/v1/sub/clash/%s", baseURL, sub.SubscriptionURL)
	}

	// Get package name
	var packageName string
	if sub.PackageID != nil {
		var pkg models.Package
		if err := database.GetDB().First(&pkg, *sub.PackageID).Error; err == nil {
			packageName = pkg.Name
		}
	}

	result := gin.H{
		"id":                 sub.ID,
		"user_id":            sub.UserID,
		"package_id":         sub.PackageID,
		"package_name":       packageName,
		"subscription_url":   sub.SubscriptionURL,
		"universal_url":      universalURL,
		"clash_url":          clashURL,
		"device_limit":       sub.DeviceLimit,
		"current_devices":    sub.CurrentDevices,
		"universal_count":    sub.UniversalCount,
		"clash_count":        sub.ClashCount,
		"surge_count":        sub.SurgeCount,
		"quanx_count":        sub.QuanXCount,
		"shadowrocket_count": sub.ShadowrocketCount,
		"is_active":          sub.IsActive,
		"status":             sub.Status,
		"expire_time":        sub.ExpireTime,
		"created_at":         sub.CreatedAt,
		"updated_at":         sub.UpdatedAt,
	}
	utils.Success(c, result)
}

func GetSubscriptionDevices(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var sub models.Subscription
	if err := database.GetDB().Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.Success(c, []interface{}{})
		return
	}
	var devices []models.Device
	database.GetDB().Where("subscription_id = ? AND is_active = ?", sub.ID, true).Find(&devices)
	utils.Success(c, devices)
}

func ResetSubscription(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.NotFound(c, "暂无订阅")
		return
	}
	oldURL := sub.SubscriptionURL
	newURL := utils.GenerateRandomString(32)
	devicesBefore := sub.CurrentDevices
	db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{})
	db.Model(&sub).Updates(map[string]interface{}{
		"subscription_url": newURL, "current_devices": 0,
		"clash_count": 0, "universal_count": 0,
	})
	resetBy := "user"
	db.Create(&models.SubscriptionReset{
		UserID: userID, SubscriptionID: sub.ID, ResetType: "manual",
		Reason: "用户自助重置", OldSubscriptionURL: &oldURL,
		NewSubscriptionURL: &newURL, DeviceCountBefore: devicesBefore,
		DeviceCountAfter: 0, ResetBy: &resetBy,
	})
	// 通知用户订阅已重置
	go services.NotifyUser(userID, "subscription_reset", map[string]string{"reset_by": "您自己"})
	utils.Success(c, gin.H{"new_url": newURL})
	utils.CreateSubscriptionLog(sub.ID, userID, "reset", "user", &userID, "用户自助重置订阅", nil, nil)
}

func ConvertToBalance(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	user := c.MustGet("user").(*models.User)
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ? AND status = ?", userID, "active").First(&sub).Error; err != nil {
		utils.NotFound(c, "暂无有效订阅")
		return
	}
	if sub.PackageID == nil {
		utils.BadRequest(c, "无法计算订阅价值")
		return
	}
	var pkg models.Package
	if err := db.First(&pkg, *sub.PackageID).Error; err != nil {
		utils.BadRequest(c, "套餐不存在")
		return
	}
	remaining := time.Until(sub.ExpireTime).Hours() / 24
	if remaining <= 0 {
		utils.BadRequest(c, "订阅已过期")
		return
	}
	if pkg.DurationDays <= 0 {
		utils.BadRequest(c, "套餐天数配置异常")
		return
	}
	value := math.Round(pkg.Price/float64(pkg.DurationDays)*remaining*100) / 100
	now := time.Now()
	db.Model(user).Update("balance", user.Balance+value)
	utils.CreateBalanceLogEntry(userID, "refund", value, user.Balance, user.Balance+value, nil, fmt.Sprintf("订阅转余额 (剩余%d天)", int(math.Ceil(remaining))), c)
	db.Model(&sub).Updates(map[string]interface{}{
		"status":       "disabled",
		"is_active":    false,
		"expire_time":  now,
		"package_id":   nil,
	})
	utils.CreateSubscriptionLog(sub.ID, userID, "deactivate", "user", &userID, fmt.Sprintf("订阅转余额: %.2f元", value), nil, nil)
	utils.Success(c, gin.H{
		"converted_amount": value,
		"new_balance":      user.Balance + value,
		"remaining_days":   int(math.Ceil(remaining)),
	})
}

func SendSubscriptionEmail(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	db := database.GetDB()

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var sub models.Subscription
	if err := db.Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.NotFound(c, "暂无订阅")
		return
	}

	baseURL := getSubscriptionBaseURL()
	if baseURL == "" {
		utils.BadRequest(c, "系统未配置域名，无法生成订阅链接")
		return
	}

	universalURL := fmt.Sprintf("%s/api/v1/sub/%s", baseURL, sub.SubscriptionURL)
	clashURL := fmt.Sprintf("%s/api/v1/sub/clash/%s", baseURL, sub.SubscriptionURL)

	subject, body := services.RenderEmail("subscription", map[string]string{
		"clash_url":    clashURL,
		"universal_url": universalURL,
		"expire_time":  sub.ExpireTime.Format("2006-01-02 15:04"),
	})

	go services.QueueEmail(user.Email, subject, body, "subscription")
	utils.SuccessMessage(c, "订阅信息已发送至邮箱")
}

func DeleteSubscriptionDevice(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	deviceID := c.Param("id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", userID).First(&sub).Error; err != nil {
		utils.NotFound(c, "暂无订阅")
		return
	}
	var device models.Device
	if err := db.Where("id = ? AND subscription_id = ? AND is_active = ?", deviceID, sub.ID, true).First(&device).Error; err != nil {
		utils.NotFound(c, "设备不存在")
		return
	}
	// Soft-deactivate
	db.Model(&device).Update("is_active", false)
	db.Model(&sub).UpdateColumn("current_devices", gorm.Expr("CASE WHEN current_devices > 0 THEN current_devices - 1 ELSE 0 END"))
	utils.SuccessMessage(c, "设备已删除")
}
