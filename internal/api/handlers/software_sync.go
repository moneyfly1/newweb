package handlers

import (
	"sort"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services/ghrelease"
	"cboard/v2/internal/services/software_sync"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// 软件同步管理接口（版本自动检测）
// ---------------------------------------------------------------------------

// SoftwareVersions 返回所有已检出软件版本（供前台/管理端显示"最新版"）
func SoftwareVersions(c *gin.Context) {
	utils.Success(c, gin.H{"list": software_sync.ListVersions()})
}

// SoftwareSyncStatus 同步状态
func SoftwareSyncStatus(c *gin.Context) {
	utils.Success(c, software_sync.GetStatus())
}

// SoftwareSyncRun 手动触发立即同步
func SoftwareSyncRun(c *gin.Context) {
	if !software_sync.TriggerAsync() {
		utils.BadRequest(c, "同步任务正在进行中")
		return
	}
	utils.SuccessMessage(c, "同步任务已启动")
}

// SoftwareSyncConfigGet 获取同步配置
func SoftwareSyncConfigGet(c *gin.Context) {
	cfg := software_sync.LoadSyncConfig()
	utils.Success(c, gin.H{
		"enabled":          cfg.Enabled,
		"interval_hours":   int(cfg.Interval.Hours()),
		"proxy_prefixes":   software_sync.LoadProxyPrefixes(),
		"github_token_set": software_sync.LoadGitHubToken() != "",
	})
}

// SoftwareSyncConfigSave 保存同步配置
func SoftwareSyncConfigSave(c *gin.Context) {
	var req struct {
		Enabled        bool   `json:"enabled"`
		IntervalHours  int    `json:"interval_hours"`
		ProxyPrefixes  string `json:"proxy_prefixes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if err := software_sync.SaveSyncConfig(req.Enabled, req.IntervalHours); err != nil {
		utils.InternalError(c, "保存同步配置失败")
		return
	}
	if req.ProxyPrefixes != "" {
		if err := saveProxyPrefixes(req.ProxyPrefixes); err != nil {
			utils.InternalError(c, "保存加速镜像前缀失败")
			return
		}
	}
	utils.SuccessMessage(c, "同步配置已保存")
}

func saveProxyPrefixes(raw string) error {
	db := database.GetDB()
	var count int64
	db.Model(&models.SystemConfig{}).Where("`key` = ? AND category = ?", "download_proxy_prefixes", "software").Count(&count)
	if count > 0 {
		return db.Model(&models.SystemConfig{}).Where("`key` = ? AND category = ?", "download_proxy_prefixes", "software").
			Updates(map[string]interface{}{"value": raw}).Error
	}
	return db.Create(&models.SystemConfig{Key: "download_proxy_prefixes", Value: raw, Category: "software"}).Error
}

// SoftwareVersionCheck 版本对照表：GitHub 最新版 vs 已检出版本
func SoftwareVersionCheck(c *gin.Context) {
	type row struct {
		Key          string `json:"key"`
		Name         string `json:"name"`
		Label        string `json:"label"`
		OS           string `json:"os"`
		Arch         string `json:"arch"`
		GitHubVer    string `json:"github_version"`
		CheckedVer   string `json:"checked_version"`
		FileName     string `json:"file_name"`
		Custom       bool   `json:"custom"`
		UpToDate     bool   `json:"up_to_date"`
		Configured   bool   `json:"configured"`
	}
	fileIDMap, err := software_sync.LoadFileIDMap()
	if err != nil {
		utils.InternalError(c, "读取版本记录失败")
		return
	}
	prefixes := software_sync.LoadProxyPrefixes()
	token := software_sync.LoadGitHubToken()

	releaseCache := map[string]*ghrelease.Release{}
	rows := make([]row, 0)
	for _, sw := range software_sync.Catalog {
		release, ok := releaseCache[sw.Repo]
		if !ok {
			release, err = ghrelease.Latest(sw.Repo, prefixes, token)
			if err != nil {
				release = nil
			}
			releaseCache[sw.Repo] = release
		}
		ghVer := ""
		if release != nil {
			ghVer = software_sync.ReleaseVersion(release)
		}
		for _, t := range sw.Targets {
			entry, hasEntry := fileIDMap[t.ConfigKey]
			r := row{
				Key: t.ConfigKey, Name: sw.Name, Label: t.Label, OS: t.OS, Arch: t.Arch,
				GitHubVer: ghVer,
			}
			if hasEntry {
				r.CheckedVer = entry.Version
				r.FileName = entry.FileName
				r.UpToDate = ghVer != "" && entry.Version == ghVer
			}
			r.Custom = !hasEntry
			rows = append(rows, r)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Label < rows[j].Label
	})
	utils.Success(c, gin.H{"list": rows})
}
