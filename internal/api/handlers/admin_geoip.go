package handlers

import (
	"github.com/gin-gonic/gin"

	"cboard/v2/internal/utils"
)

// GetGeoIPStatus 管理端：GeoIP 库状态（是否加载、路径、文件大小、缓存条目数）。
func GetGeoIPStatus(c *gin.Context) {
	utils.Success(c, utils.MMDBStatus())
}

// ReloadGeoIP 管理端：重新加载 GeoIP 库并清空地区缓存。
//
// 上传/替换 MMDB 文件后调用，无需重启服务。
func ReloadGeoIP(c *gin.Context) {
	if err := utils.ReloadMMDB(); err != nil {
		utils.InternalError(c, "重新加载 GeoIP 库失败: "+err.Error())
		return
	}
	utils.CreateAuditLog(c, "reload_geoip", "settings", 0, "重新加载 GeoIP 库")
	utils.Success(c, utils.MMDBStatus())
}
