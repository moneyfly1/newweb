package handlers

import (
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func GetPublicConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	// Return is_public configs + registration/site settings needed by frontend
	publicKeys := []string{
		"site_name", "site_description", "site_url",
		"support_email", "support_qq", "support_telegram",
		"register_enabled", "register_email_verify", "register_invite_required",
		// Client download URLs
		"client_clash_windows_url", "client_v2rayn_url", "client_mihomo_windows_url",
		"client_hiddify_windows_url", "client_flclash_windows_url",
		"client_clash_android_url", "client_v2rayng_url", "client_hiddify_android_url",
		"client_flclash_macos_url", "client_mihomo_macos_url",
		"client_shadowrocket_url", "client_stash_url", "client_singbox_url",
		"client_clash_linux_url",
	}
	db.Where("is_public = ? OR `key` IN ?", true, publicKeys).Find(&configs)
	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}
	utils.Success(c, result)
}

func ListPackages(c *gin.Context) {
	var packages []models.Package
	database.GetDB().Where("is_active = ?", true).Order("sort_order ASC").Find(&packages)
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
