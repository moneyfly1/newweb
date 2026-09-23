package handlers

import (
	"strconv"

	"cboard/v2/internal/cache"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// 后台「域名设置 + 一键应用并体检」。
//
// 解决的场景：域名在某些地区被屏蔽后客户打不开网站、拉不到订阅，
// 需要把订阅地址切到备用域名（同时老域名继续可用）。
// 这里把「改设置 → 生效 → 验证」合成一步，并把每个域名的体检结果直接列出来，
// 免去登服务器逐个 curl。

// AdminGetDomainSettings 读取域名设置并顺带返回体检结果
func AdminGetDomainSettings(c *gin.Context) {
	setting := services.GetSubscriptionDomainSetting()
	subscriptionBase := services.SubscriptionBaseURL()
	utils.Success(c, gin.H{
		"site_url":              setting.SiteURL,
		"subscription_domain":   setting.SubscriptionDomain,
		"subscription_mirrors":  setting.Mirrors,
		"backup_site_url":       setting.BackupSiteURL,
		"effective_sub_base":    subscriptionBase,
		"subscribe_path_alias":  []string{"/s/{token}", "/sub/{token}"},
		"checks":                services.CheckAllDomains(),
		"payment_callback_note": "支付回调地址不受这里影响（仍使用 pay_*_notify_url / payment_public_base_url）",
	})
}

// AdminApplyDomainSettings 一键应用域名设置：
// 保存 → 清缓存（公共配置 + 订阅内容）→ 逐域名体检 → 返回结果与提示。
func AdminApplyDomainSettings(c *gin.Context) {
	var req struct {
		SiteURL            *string  `json:"site_url"`
		SubscriptionDomain *string  `json:"subscription_domain"`
		SubscriptionMirror []string `json:"subscription_mirrors"`
		BackupSiteURL      *string  `json:"backup_site_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	save := func(key, value string) {
		result := db.Model(&models.SystemConfig{}).Where("`key` = ?", key).Updates(map[string]interface{}{"value": value})
		if result.Error == nil && result.RowsAffected == 0 {
			db.Create(&models.SystemConfig{Key: key, Value: value, Category: ""})
		}
	}

	// 站点域名与备用站点域名也允许在这里维护；支付回调相关设置一律不碰
	if req.SiteURL != nil {
		save("site_url", services.NormalizeDomainInput(*req.SiteURL))
	}
	if req.BackupSiteURL != nil {
		save("backup_site_url", services.NormalizeDomainInput(*req.BackupSiteURL))
	}
	if req.SubscriptionDomain != nil {
		save("subscription_domain", services.NormalizeDomainInput(*req.SubscriptionDomain))
	}
	if req.SubscriptionMirror != nil {
		mirrors := services.ParseMirrorDomains(joinOrJSON(req.SubscriptionMirror))
		save("subscription_mirrors", services.MirrorsToSetting(mirrors))
	}

	utils.InvalidateSettingsCache()
	utils.InvalidatePublicCache("public_config")
	cache.ClearAllSubscriptionCache()

	setting := services.GetSubscriptionDomainSetting()
	checks := services.CheckAllDomains()
	failing := 0
	for _, chk := range checks {
		if len(chk.Problems) > 0 {
			failing++
		}
	}

	summary := "域名设置已应用"
	if failing > 0 {
		summary = "域名设置已应用，但有域名未通过体检，请按提示处理"
	}
	utils.CreateAuditLog(c, "apply_domain_settings", "settings", 0,
		"应用域名设置: 订阅域名="+setting.SubscriptionDomain+" 备用="+joinOrJSON(setting.Mirrors)+" 体检未通过="+strconv.Itoa(failing))

	utils.Success(c, gin.H{
		"site_url":             setting.SiteURL,
		"subscription_domain":  setting.SubscriptionDomain,
		"subscription_mirrors": setting.Mirrors,
		"backup_site_url":      setting.BackupSiteURL,
		"effective_sub_base":   services.SubscriptionBaseURL(),
		"checks":               checks,
		"failing":              failing,
		"message":              summary,
	})
}

func joinOrJSON(items []string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += ","
		}
		out += it
	}
	return out
}
