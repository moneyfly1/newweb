package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/config"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 设备与客户端使用（对齐 myweb 的 /admin/analytics/device-usage/users）：
//
// 为什么单独做这个接口：用户列表只能看到「设备数 + 前两个客户端」，而客服/运营
// 真正要回答的问题是「这个客户到底用了几台设备、哪些客户端、哪个在用」以及
// 「用某个客户端的用户有哪些」。myweb 的「设备与客户端使用 → 按用户」就是
// 这个视图：clients 返回**全部**客户端（含版本），并支持按客户端/机型/系统/
// 用户名/邮箱搜索。

// officialClientNames 官方自研客户端（与订阅访问白名单保持一致）
var officialClientNames = []string{"moneyfly", "mbox"}

func isOfficialClientName(name string) bool {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return false
	}
	for _, n := range officialClientNames {
		if key == n {
			return true
		}
	}
	return false
}

func lowerOfficialClientNames() []string {
	out := make([]string, 0, len(officialClientNames))
	for _, n := range officialClientNames {
		out = append(out, strings.ToLower(n))
	}
	return out
}

func deviceStr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

// DeviceUsageUserRow 按用户聚合的设备/客户端使用情况
type DeviceUsageUserRow struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	UserActive  bool     `json:"user_active"`
	Devices     int64    `json:"devices"`
	OnlineCount int64    `json:"online_count"`
	Clients     []string `json:"clients"` // 全部客户端（名称+版本，去重）
	Brands      []string `json:"brands"`  // 机型/系统摘要（最多 3 个）
	OfficialUse bool     `json:"official_use"`
	OnlyThird   bool     `json:"only_third"`
	LastAccess  string   `json:"last_access"`
	SubStatus   string   `json:"sub_status"`
	SubExpire   string   `json:"sub_expire"`
}

// deviceUsageDeviceQuery 设备维度的通用筛选（q / days / official）
func deviceUsageDeviceQuery(db *gorm.DB, q, official string, days int) *gorm.DB {
	tx := db.Model(&models.Device{}).
		Where("is_active = ?", true).
		Where("user_id IS NOT NULL")

	if s := strings.TrimSpace(q); s != "" {
		like := "%" + s + "%"
		userSub := db.Model(&models.User{}).Select("id").
			Where("username LIKE ? OR email LIKE ?", like, like)
		tx = tx.Where(`(software_name LIKE ? OR software_version LIKE ?
			OR os_name LIKE ? OR os_version LIKE ?
			OR device_model LIKE ? OR device_brand LIKE ?
			OR user_id IN (?))`,
			like, like, like, like, like, like, userSub)
	}
	if days > 0 {
		tx = tx.Where("last_access >= ?", time.Now().AddDate(0, 0, -days))
	}
	switch strings.ToLower(strings.TrimSpace(official)) {
	case "official":
		tx = tx.Where("LOWER(TRIM(software_name)) IN ?", lowerOfficialClientNames())
	case "third_party", "unofficial":
		tx = tx.Where("LOWER(TRIM(software_name)) NOT IN ?", lowerOfficialClientNames())
	}
	return tx
}

// AdminDeviceUsageUsers 按用户列出「谁在用哪些客户端/设备」
// GET /api/v1/admin/analytics/device-usage/users?q=&official=all|official|third_party&days=0&page=1&page_size=20
func AdminDeviceUsageUsers(c *gin.Context) {
	db := database.GetDB()
	q := c.Query("q")
	official := c.DefaultQuery("official", "all")
	days, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("days", "0")))
	p := utils.GetPagination(c)

	dim := resolveDeviceUsageDim(c.DefaultQuery("dim", ""))
	key := strings.TrimSpace(c.Query("key"))

	base := deviceUsageDeviceQuery(db, q, official, days)
	// 分组下钻：点「按客户端分布」里的某一行 → 只看这个客户端/平台的用户
	if key != "" {
		base = base.Where(dim.Column+" = ?", key)
	}
	if dim.IOSOnly {
		base = base.Where("LOWER(COALESCE(os_name, '')) LIKE ?", "%ios%")
	}

	var totalUsers int64
	base.Session(&gorm.Session{}).Distinct("user_id").Count(&totalUsers)

	type userAgg struct {
		UserID     int64
		Devices    int64
		LastAccess *string
	}
	var aggs []userAgg
	if err := base.Session(&gorm.Session{}).
		Select("user_id, COUNT(*) AS devices, MAX(last_access) AS last_access").
		Group("user_id").
		Order("last_access DESC").
		Offset(p.Offset()).Limit(p.PageSize).
		Scan(&aggs).Error; err != nil {
		utils.InternalError(c, "按用户聚合设备失败: "+err.Error())
		return
	}

	userIDs := make([]uint, 0, len(aggs))
	for _, a := range aggs {
		userIDs = append(userIDs, uint(a.UserID))
	}
	if len(userIDs) == 0 {
		utils.SuccessPage(c, []DeviceUsageUserRow{}, 0, p.Page, p.PageSize)
		return
	}

	// 当前页用户的设备行（同一套筛选条件），在内存里聚合成「全部客户端」
	var devices []models.Device
	base.Session(&gorm.Session{}).Where("user_id IN ?", userIDs).
		Order("last_access DESC").Find(&devices)

	type acc struct {
		clients     []string
		brands      []string
		online      int64
		officialUse bool
		third       bool
	}
	accs := make(map[uint]*acc, len(userIDs))
	for i := range devices {
		d := &devices[i]
		if d.UserID == nil {
			continue
		}
		uid := uint(*d.UserID)
		a := accs[uid]
		if a == nil {
			a = &acc{clients: make([]string, 0, 4), brands: make([]string, 0, 3)}
			accs[uid] = a
		}
		if d.ComputeOnline() {
			a.online++
		}
		sw := deviceStr(d.SoftwareName)
		if sw != "" {
			label := sw
			if ver := deviceStr(d.SoftwareVersion); ver != "" {
				label = sw + " " + ver
			}
			dup := false
			for _, cl := range a.clients {
				if cl == label {
					dup = true
					break
				}
			}
			if !dup {
				a.clients = append(a.clients, label)
			}
			if isOfficialClientName(sw) {
				a.officialUse = true
			} else {
				a.third = true
			}
		}
		brand := deviceStr(d.DeviceModel)
		if brand == "" {
			brand = deviceStr(d.OSName)
		}
		if brand != "" {
			dup := false
			for _, b := range a.brands {
				if b == brand {
					dup = true
					break
				}
			}
			if !dup && len(a.brands) < 3 {
				a.brands = append(a.brands, brand)
			}
		}
	}

	users := make([]models.User, 0, len(userIDs))
	db.Select("id", "username", "email", "is_active").Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[uint]models.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}
	subMap := make(map[uint]models.Subscription, len(userIDs))
	var subs []models.Subscription
	db.Where("user_id IN ?", userIDs).Order("created_at DESC").Find(&subs)
	for i := range subs {
		if _, ok := subMap[subs[i].UserID]; !ok {
			subMap[subs[i].UserID] = subs[i]
		}
	}

	rows := make([]DeviceUsageUserRow, 0, len(aggs))
	for _, a := range aggs {
		uid := uint(a.UserID)
		u := userMap[uid]
		row := DeviceUsageUserRow{
			UserID:     uid,
			Username:   u.Username,
			Email:      u.Email,
			UserActive: u.IsActive,
			Devices:    a.Devices,
			LastAccess: formatAggTimeStr(a.LastAccess),
		}
		if ac := accs[uid]; ac != nil {
			row.Clients = ac.clients
			row.Brands = ac.brands
			row.OnlineCount = ac.online
			row.OfficialUse = ac.officialUse
			row.OnlyThird = ac.third && !ac.officialUse
		}
		if s, ok := subMap[uid]; ok {
			row.SubStatus = s.Status
			if !s.ExpireTime.IsZero() {
				row.SubExpire = s.ExpireTime.Format("2006-01-02")
			}
		}
		rows = append(rows, row)
	}

	utils.SuccessPage(c, rows, totalUsers, p.Page, p.PageSize)
}

// formatAggTimeStr 聚合结果里的时间字符串（SQLite 返回字符串，MySQL 返回 time）
func formatAggTimeStr(v *string) string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return ""
	}
	t := strings.TrimSpace(*v)
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, l := range layouts {
		if parsed, err := time.Parse(l, t); err == nil {
			return parsed.Format(utils.LayoutDateTime)
		}
	}
	if len(t) >= 16 {
		return t[:16]
	}
	return t
}

// ---------------------------------------------------------------------------
// 分组分布（对齐 myweb 的 /admin/analytics/device-usage）
// ---------------------------------------------------------------------------

// DeviceUsageDim 分组维度（Column 只来自白名单，杜绝注入）
type DeviceUsageDim struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Column  string `json:"-"`
	IOSOnly bool   `json:"ios_only"`
}

var deviceUsageDims = []DeviceUsageDim{
	{Key: "client", Label: "客户端软件", Column: "software_name"},
	{Key: "client_version", Label: "客户端版本", Column: "software_version"},
	{Key: "platform", Label: "平台（系统）", Column: "os_name"},
	{Key: "os_version", Label: "系统版本", Column: "os_version"},
	{Key: "ios_model", Label: "iOS 设备型号", Column: "device_model", IOSOnly: true},
	{Key: "device_type", Label: "设备类型", Column: "device_type"},
	{Key: "brand", Label: "设备品牌", Column: "device_brand"},
}

func resolveDeviceUsageDim(key string) DeviceUsageDim {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, d := range deviceUsageDims {
		if d.Key == k {
			return d
		}
	}
	return deviceUsageDims[0]
}

// DeviceUsageGroup 一条分组统计
type DeviceUsageGroup struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Devices    int64   `json:"devices"`
	Users      int64   `json:"users"`
	Share      float64 `json:"share"`
	LastAccess string  `json:"last_access"`
	Official   bool    `json:"official"`
}

// AdminDeviceUsageGroups 设备/客户端分组分布
// GET /api/v1/admin/analytics/device-usage?dim=client&q=&official=all|official|third_party&days=0&page=1&page_size=20
func AdminDeviceUsageGroups(c *gin.Context) {
	db := database.GetDB()
	dim := resolveDeviceUsageDim(c.DefaultQuery("dim", "client"))
	q := c.Query("q")
	official := c.DefaultQuery("official", "all")
	days, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("days", "0")))
	p := utils.GetPagination(c)

	base := deviceUsageDeviceQuery(db, q, official, days)
	if dim.IOSOnly {
		base = base.Where("LOWER(COALESCE(os_name, '')) LIKE ?", "%ios%")
	}

	var totalDevices int64
	if err := base.Session(&gorm.Session{}).Count(&totalDevices).Error; err != nil {
		utils.InternalError(c, "统计设备失败: "+err.Error())
		return
	}
	var totalUsers int64
	base.Session(&gorm.Session{}).Distinct("user_id").Count(&totalUsers)

	var officialDevices int64
	deviceUsageDeviceQuery(db, q, "official", days).
		Where(func() string {
			if dim.IOSOnly {
				return "LOWER(COALESCE(os_name, '')) LIKE '%ios%'"
			}
			return "1 = 1"
		}()).
		Count(&officialDevices)
	thirdPartyDevices := totalDevices - officialDevices
	if thirdPartyDevices < 0 {
		thirdPartyDevices = 0
	}

	var groupTotal int64
	base.Session(&gorm.Session{}).Distinct(dim.Column).Count(&groupTotal)

	var rows []struct {
		Key        string
		Devices    int64
		Users      int64
		LastAccess *string
	}
	if err := base.Session(&gorm.Session{}).
		Select(dim.Column + " AS key, COUNT(*) AS devices, COUNT(DISTINCT user_id) AS users, MAX(last_access) AS last_access").
		Group(dim.Column).
		Order("devices DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Scan(&rows).Error; err != nil {
		utils.InternalError(c, "分组统计失败: "+err.Error())
		return
	}

	groups := make([]DeviceUsageGroup, 0, len(rows))
	for _, r := range rows {
		key := strings.TrimSpace(r.Key)
		label := key
		if label == "" {
			label = "未知"
		}
		share := 0.0
		if totalDevices > 0 {
			share = float64(r.Devices) * 100 / float64(totalDevices)
		}
		groups = append(groups, DeviceUsageGroup{
			Key: key, Label: label,
			Devices: r.Devices, Users: r.Users,
			Share:      float64(int(share*100+0.5)) / 100,
			LastAccess: formatAggTimeStr(r.LastAccess),
			Official:   isOfficialClientName(key),
		})
	}

	utils.Success(c, gin.H{
		"dim":         dim.Key,
		"dim_label":   dim.Label,
		"dims":        deviceUsageDims,
		"page":        p.Page,
		"limit":       p.PageSize,
		"group_total": groupTotal,
		"summary": gin.H{
			"devices":             totalDevices,
			"users":               totalUsers,
			"official_devices":    officialDevices,
			"third_party_devices": thirdPartyDevices,
		},
		"groups": groups,
	})
}

// ---------------------------------------------------------------------------
// 「引导第三方客户端用户升级到官方客户端」批量发信
// 对齐 myweb 的 /admin/analytics/device-usage/notify-upgrade：
//   - dry_run 默认 true：先返回渲染好的标题/正文预览与目标人数，确认后再发；
//   - 单次最多 maxNotifyBatch 人，超出截断并返回 truncated；
//   - 只发活跃账号 + 有邮箱 + 未退订的用户，跳过原因逐类计数；
//   - 同一用户多台第三方设备只发一封（按用户去重）。
// ---------------------------------------------------------------------------

const maxNotifyBatchUsers = 1000

const defaultUpgradeSubject = "建议升级到官方客户端 MoneyFly / MBox"

type notifyUpgradeRequest struct {
	Dim      string   `json:"dim"`
	Key      string   `json:"key"`
	Official string   `json:"official"`
	Days     string   `json:"days"`
	Q        string   `json:"q"`
	UserIDs  []uint   `json:"user_ids"`
	Subject  string   `json:"subject"`
	Content  string   `json:"content"`
	DryRun   *bool    `json:"dry_run"`
	Note     string   `json:"note"`
}

// deviceUsageTargetUserIDs 按筛选条件取需要提醒的用户（去重）
func deviceUsageTargetUserIDs(db *gorm.DB, req notifyUpgradeRequest) []uint {
	if len(req.UserIDs) > 0 {
		out := make([]uint, 0, len(req.UserIDs))
		seen := map[uint]bool{}
		for _, id := range req.UserIDs {
			if id > 0 && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		return out
	}
	official := strings.TrimSpace(req.Official)
	if official == "" {
		official = "third_party" // 默认只看非官方客户端：这正是要劝升级的人
	}
	days, _ := strconv.Atoi(strings.TrimSpace(req.Days))
	dim := resolveDeviceUsageDim(req.Dim)
	base := deviceUsageDeviceQuery(db, req.Q, official, days)
	if key := strings.TrimSpace(req.Key); key != "" {
		base = base.Where(dim.Column+" = ?", key)
	}
	var ids []uint
	base.Session(&gorm.Session{}).Where("user_id IS NOT NULL").Distinct("user_id").Pluck("user_id", &ids)
	return ids
}

// AdminNotifyClientUpgrade 批量发「升级到官方客户端」邮件
// POST /api/v1/admin/analytics/device-usage/notify-upgrade
func AdminNotifyClientUpgrade(c *gin.Context) {
	var req notifyUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误")
		return
	}
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	db := database.GetDB()
	ids := deviceUsageTargetUserIDs(db, req)
	if len(ids) == 0 {
		utils.Success(c, gin.H{
			"targets": 0, "dry_run": dryRun,
			"message": "当前筛选条件下没有可提醒的用户",
		})
		return
	}
	truncated := 0
	if len(ids) > maxNotifyBatchUsers {
		truncated = len(ids) - maxNotifyBatchUsers
		ids = ids[:maxNotifyBatchUsers]
	}

	var users []models.User
	db.Where("id IN ?", ids).Find(&users)

	// 每个用户最近一次第三方客户端（邮件里点名更有说服力）
	clientOf := make(map[uint]string, len(ids))
	type row struct {
		UserID       int64
		SoftwareName string
	}
	var rows []row
	db.Model(&models.Device{}).
		Select("user_id, COALESCE(software_name, '') AS software_name").
		Where("user_id IN ? AND is_active = ?", ids, true).
		Order("last_access DESC").
		Scan(&rows)
	for _, r := range rows {
		uid := uint(r.UserID)
		if _, ok := clientOf[uid]; ok {
			continue
		}
		name := strings.TrimSpace(r.SoftwareName)
		if name == "" || isOfficialClientName(name) {
			continue
		}
		clientOf[uid] = name
	}

	baseURL := strings.TrimRight(configBaseURL(), "/")
	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = defaultUpgradeSubject
	}

	skippedNoEmail, skippedInactive, skippedUnsubscribed := 0, 0, 0
	targets := make([]models.User, 0, len(users))
	for _, u := range users {
		if strings.TrimSpace(u.Email) == "" {
			skippedNoEmail++
			continue
		}
		if !u.IsActive {
			skippedInactive++
			continue
		}
		if !u.EmailNotifications {
			skippedUnsubscribed++
			continue
		}
		targets = append(targets, u)
	}

	queued, failed := 0, 0
	failReasons := make([]string, 0, 5)
	var preview gin.H
	for i := range targets {
		u := targets[i]
		body := strings.TrimSpace(req.Content)
		if body == "" {
			body = buildUpgradeNoticeHTML(displayName(u), clientOf[u.ID], baseURL, req.Note)
		}
		if preview == nil {
			preview = gin.H{"user_id": u.ID, "email": u.Email, "subject": subject, "content": body}
		}
		if dryRun {
			continue
		}
		// QueueEmail 只负责入队（内部 recover 兜底，无返回值）；
		// 真正投递由邮件队列 worker 处理。
		services.QueueEmail(u.Email, subject, body, "client_upgrade")
		queued++
	}

	smtpConfigured := false
	smtpHost, smtpUser := "", ""
	if cfg, err := services.GetSMTPConfig(); err == nil && cfg != nil {
		smtpConfigured = strings.TrimSpace(cfg.Host) != "" && strings.TrimSpace(cfg.Username) != ""
		smtpHost = cfg.Host
		smtpUser = cfg.Username
	}

	out := gin.H{
		"dry_run":              dryRun,
		"targets":              len(targets),
		"truncated":            truncated,
		"queued":               queued,
		"failed":               failed,
		"skipped_no_email":     skippedNoEmail,
		"skipped_inactive":     skippedInactive,
		"skipped_unsubscribed": skippedUnsubscribed,
		"smtp_configured":      smtpConfigured,
		"smtp_host":            smtpHost,
		"smtp_user":            smtpUser,
		"max_batch":            maxNotifyBatchUsers,
	}
	if preview != nil {
		out["preview"] = preview
	}
	if len(failReasons) > 0 {
		out["fail_reasons"] = failReasons
	}
	utils.Success(c, out)
}

func configBaseURL() string {
	if config.AppConfig != nil && strings.TrimSpace(config.AppConfig.BaseURL) != "" {
		return config.AppConfig.BaseURL
	}
	return "https://new.moneyfly.top"
}

func displayName(u models.User) string {
	if u.Nickname != nil && strings.TrimSpace(*u.Nickname) != "" {
		return strings.TrimSpace(*u.Nickname)
	}
	if strings.TrimSpace(u.Username) != "" {
		return u.Username
	}
	return u.Email
}

// buildUpgradeNoticeHTML 默认升级提醒正文（纯 HTML，邮件客户端兼容）
func buildUpgradeNoticeHTML(name, client, baseURL, note string) string {
	if client == "" {
		client = "第三方客户端"
	}
	extra := ""
	if strings.TrimSpace(note) != "" {
		extra = "<p style=\"color:#666\">" + strings.TrimSpace(note) + "</p>"
	}
	return fmt.Sprintf(`<h2>建议升级到官方客户端 MoneyFly / MBox</h2>
<p>%s，你好：</p>
<p>系统检测到你正在使用 <b>%s</b>。官方客户端在稳定性、线路质量与设备管理上都更好，建议升级：</p>
<ul>
  <li>更稳定的线路与自动故障切换</li>
  <li>多设备统一管理，订阅地址自动同步</li>
  <li>Windows / macOS / Android / iOS 全平台支持</li>
</ul>
%s
<p>下载与使用说明：<a href="%s/">%s</a></p>
<p style="color:#999;font-size:12px">如果这不是你本人操作，请忽略本邮件或联系客服。</p>`,
		name, client, extra, baseURL, baseURL)
}
