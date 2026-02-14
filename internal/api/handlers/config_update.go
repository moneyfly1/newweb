package handlers

import (
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

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
