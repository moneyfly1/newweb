package handlers

import (
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

// ── Public config ──

func GetPublicConfig(c *gin.Context) {
	// 60s 内存缓存 + CDN 边缘缓存（Cache-Control）：公共配置高频访问且低频变更，
	// 让 Cloudflare 等 CDN 在边缘直接命中，避免每次经回源链路
	c.Header("Cache-Control", "public, max-age=60")
	// 60s 内存缓存：公共配置高访问，避免每次查库
	if cached := utils.GetPublicCache("public_config"); cached != nil {
		utils.Success(c, cached)
		return
	}
	db := database.GetDB()
	var configs []models.SystemConfig
	publicKeys := []string{
		"site_name", "site_description", "site_url", "site_icon",
		"support_email", "support_qq", "support_telegram",
		"register_enabled", "register_email_verify", "register_invite_required",
		// Client download URLs
		"client_clash_windows_url", "client_v2rayn_url", "client_clashparty_windows_url",
		"client_hiddify_windows_url", "client_flclash_windows_url",
		"client_clash_android_url", "client_v2rayng_url", "client_hiddify_android_url",
		"client_flclash_macos_url", "client_clashparty_macos_url",
		"client_shadowrocket_url", "client_stash_url", "client_singbox_url",
		"client_clash_linux_url",
		// GitHub 自动下载扩展键（各平台/架构）
		"client_v2rayn_macos_url", "client_v2rayn_macos_arm_url",
		"client_hiddify_macos_url", "client_hiddify_macos_arm_url",
		"client_clashverge_windows_url", "client_clashverge_macos_url", "client_clashverge_macos_arm_url",
		"client_clashparty_macos_arm_url", "client_flclash_macos_arm_url",
		"client_flclash_android_url", "client_hiddify_linux_url", "client_hiddify_linux_arm_url",
		"client_flclash_linux_url", "client_flclash_linux_arm_url",
		"client_clashverge_linux_url", "client_clashverge_linux_arm_url",
		"download_proxy_prefixes",
		// Telegram login
		"telegram_login_enabled", "telegram_bot_username",
		// Custom package
		"custom_package_enabled", "custom_package_price_per_device_year",
		"custom_package_min_devices", "custom_package_max_devices",
		"custom_package_min_months", "custom_package_duration_discounts",
	}
	db.Select("`key`, value").Where("is_public = ? OR `key` IN ?", true, publicKeys).Find(&configs)
	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}
	utils.SetPublicCache("public_config", result)
	utils.Success(c, result)
}

func ListPackages(c *gin.Context) {
	// 60s 内存缓存 + CDN 边缘缓存
	c.Header("Cache-Control", "public, max-age=60")
	// 60s 内存缓存
	if cached := utils.GetPublicCache("public_packages"); cached != nil {
		utils.Success(c, cached)
		return
	}
	var packages []models.Package
	database.GetDB().Where("is_active = ?", true).Order("sort_order ASC").Find(&packages)
	utils.SetPublicCache("public_packages", packages)
	utils.Success(c, packages)
}

func GetPackage(c *gin.Context) {
	id := c.Param("id")
	var pkg models.Package
	if err := database.GetDB().First(&pkg, id).Error; err != nil {
		utils.NotFound(c, "套餐不存在")
		return
	}
	utils.Success(c, pkg)
}

// ── Config update service (admin) ──

func AdminConfigUpdateStatus(c *gin.Context) {
	svc := services.GetConfigUpdateService()
	utils.Success(c, gin.H{
		"running":   svc.IsRunning(),
		"scheduled": svc.IsScheduled(),
	})
}

func AdminGetConfigUpdateConfig(c *gin.Context) {
	svc := services.GetConfigUpdateService()
	cfg, err := svc.LoadConfig()
	if err != nil {
		utils.InternalError(c, "加载配置失败: "+err.Error())
		return
	}
	utils.Success(c, cfg)
}

func AdminSaveConfigUpdateConfig(c *gin.Context) {
	var cfg services.ConfigUpdateConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	svc := services.GetConfigUpdateService()
	if svc.IsScheduled() {
		svc.StopSchedule()
	}
	if err := svc.SaveConfig(&cfg); err != nil {
		utils.InternalError(c, "保存配置失败: "+err.Error())
		return
	}
	if cfg.Enabled {
		svc.StartSchedule()
	}
	utils.SuccessMessage(c, "配置已保存")
}

func AdminStartConfigUpdate(c *gin.Context) {
	svc := services.GetConfigUpdateService()
	if err := svc.Start(); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.SuccessMessage(c, "更新任务已启动")
}

func AdminStopConfigUpdate(c *gin.Context) {
	svc := services.GetConfigUpdateService()
	svc.Stop()
	utils.SuccessMessage(c, "更新任务已停止")
}

func AdminGetConfigUpdateLogs(c *gin.Context) {
	svc := services.GetConfigUpdateService()
	utils.Success(c, svc.GetLogs())
}

func AdminClearConfigUpdateLogs(c *gin.Context) {
	svc := services.GetConfigUpdateService()
	svc.ClearLogs()
	utils.SuccessMessage(c, "日志已清空")
}
