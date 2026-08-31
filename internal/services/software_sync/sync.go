package software_sync

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services/ghrelease"
	"cboard/v2/internal/utils"
)

// cfgCategory 软件同步配置存储分类
const cfgCategory = "software"

// FileEntry 版本记录条目（记录各目标已检出的最新版本与资产名）
type FileEntry struct {
	FileName  string `json:"fileName"`
	Size      int64  `json:"size"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}

// ReportItem 单目标同步结果
type ReportItem struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Label    string `json:"label"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	FileName string `json:"file_name,omitempty"`
	Status   string `json:"status"` // ok / skip / error
	Message  string `json:"message,omitempty"`
}

// SyncStatus 同步状态（供管理接口/前端轮询）
type SyncStatus struct {
	Running       bool         `json:"running"`
	Enabled       bool         `json:"enabled"`
	IntervalHours int          `json:"interval_hours"`
	LastRun       string       `json:"last_run"`
	LastReport    []ReportItem `json:"last_report"`
}

var (
	statusMu    sync.Mutex
	syncRunning bool
	lastRun     string
	lastReport  []ReportItem
)

// ReleaseVersion 从 tag_name 提取版本号（去 v 前缀）
func ReleaseVersion(r *ghrelease.Release) string {
	if r == nil {
		return ""
	}
	return strings.TrimPrefix(r.TagName, "v")
}

// LoadFileIDMap 读取版本记录（存于 system_configs: category=software, key=file_id_map）
func LoadFileIDMap() (map[string]FileEntry, error) {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("`key` = ? AND category = ?", "file_id_map", cfgCategory).First(&conf).Error; err != nil {
		return map[string]FileEntry{}, nil
	}
	var m map[string]FileEntry
	if err := json.Unmarshal([]byte(conf.Value), &m); err != nil {
		return map[string]FileEntry{}, err
	}
	if m == nil {
		m = map[string]FileEntry{}
	}
	return m, nil
}

func saveFileIDMap(m map[string]FileEntry) error {
	data, _ := json.Marshal(m)
	return saveSystemConfig("file_id_map", string(data), cfgCategory)
}

// loadSoftwareValue 读取某个软件下载配置键的值
func loadSoftwareValue(key string) string {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("`key` = ?", key).First(&conf).Error; err != nil {
		return ""
	}
	return conf.Value
}

// saveSoftwareValue 写入软件下载配置键（pan:// 或自定义链接）
func saveSoftwareValue(key, value string) error {
	return saveSystemConfig(key, value, "")
}

// 实际写入：直接操作 system_configs 表
func saveSystemConfig(key, value, category string) error {
	db := database.GetDB()
	var count int64
	db.Model(&models.SystemConfig{}).Where("`key` = ?", key).Count(&count)
	if count > 0 {
		return db.Model(&models.SystemConfig{}).Where("`key` = ?", key).
			Updates(map[string]interface{}{"value": value}).Error
	}
	return db.Create(&models.SystemConfig{Key: key, Value: value, Category: category}).Error
}

// LoadGitHubToken 复用 GitHub 节点同步的 token（提高 API 限额）
func LoadGitHubToken() string {
	return strings.TrimSpace(utils.GetSetting("gh_nodes_token"))
}

// LoadProxyPrefixes 加载加速镜像前缀
func LoadProxyPrefixes() []string {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("`key` = ? AND category = ?", "download_proxy_prefixes", cfgCategory).First(&conf).Error; err != nil {
		return ghrelease.DefaultProxyPrefixes
	}
	parsed := parsePrefixes(conf.Value)
	if len(parsed) == 0 {
		return ghrelease.DefaultProxyPrefixes
	}
	return parsed
}

func parsePrefixes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	if strings.HasPrefix(raw, "[") {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	if len(out) == 0 {
		parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == ',' || r == ';' })
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	// 确保包含直连兜底
	hasDirect := false
	for _, p := range out {
		if p == "{url}" || strings.EqualFold(p, "direct") {
			hasDirect = true
			break
		}
	}
	if !hasDirect {
		out = append(out, "{url}")
	}
	return out
}

// ---------------------------------------------------------------------------
// 同步主流程：GitHub 最新版检测 → 更新版本记录 + 写 pan:// 配置
// ---------------------------------------------------------------------------

// IsRunning 是否正在同步
func IsRunning() bool {
	statusMu.Lock()
	defer statusMu.Unlock()
	return syncRunning
}

// TriggerAsync 后台触发一次同步（已在运行则忽略）
func TriggerAsync() bool {
	statusMu.Lock()
	if syncRunning {
		statusMu.Unlock()
		return false
	}
	syncRunning = true
	statusMu.Unlock()
	go func() {
		report := runSync()
		statusMu.Lock()
		syncRunning = false
		lastRun = time.Now().Format(time.RFC3339)
		lastReport = report
		statusMu.Unlock()
	}()
	return true
}

// GetStatus 当前同步状态
func GetStatus() SyncStatus {
	statusMu.Lock()
	defer statusMu.Unlock()
	cfg := LoadSyncConfig()
	return SyncStatus{
		Running:       syncRunning,
		Enabled:       cfg.Enabled,
		IntervalHours: int(cfg.Interval.Hours()),
		LastRun:       lastRun,
		LastReport:    lastReport,
	}
}

// SyncConfig 同步配置
type SyncConfig struct {
	Enabled  bool
	Interval time.Duration
}

// LoadSyncConfig 读取同步配置
func LoadSyncConfig() SyncConfig {
	cfg := SyncConfig{Enabled: true, Interval: 12 * time.Hour}
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", cfgCategory).Find(&configs)
	for _, c := range configs {
		switch c.Key {
		case "sync_enabled":
			cfg.Enabled = c.Value == "" || c.Value == "true" || c.Value == "1"
		case "sync_interval_hours":
			var h int
			if n, _ := fmt.Sscanf(c.Value, "%d", &h); n == 1 && h >= 1 {
				cfg.Interval = time.Duration(h) * time.Hour
			}
		}
	}
	return cfg
}

// SaveSyncConfig 保存同步配置
func SaveSyncConfig(enabled bool, intervalHours int) error {
	if intervalHours < 1 {
		intervalHours = 12
	}
	if intervalHours > 168 {
		intervalHours = 168
	}
	if err := saveSystemConfig("sync_enabled", fmt.Sprintf("%v", enabled), cfgCategory); err != nil {
		return err
	}
	return saveSystemConfig("sync_interval_hours", fmt.Sprintf("%d", intervalHours), cfgCategory)
}

// runSync 同步主流程
func runSync() []ReportItem {
	report := make([]ReportItem, 0)
	prefixes := LoadProxyPrefixes()
	ghToken := LoadGitHubToken()
	releaseCache := map[string]*ghrelease.Release{}

	fileIDMap, err := LoadFileIDMap()
	if err != nil {
		report = append(report, ReportItem{Status: "error", Message: "读取版本记录失败: " + err.Error()})
		return report
	}

	newVersions := 0
	for _, sw := range Catalog {
		release, ok := releaseCache[sw.Repo]
		if !ok {
			release, err = ghrelease.Latest(sw.Repo, prefixes, ghToken)
			if err != nil {
				for _, t := range sw.Targets {
					report = append(report, ReportItem{Key: t.ConfigKey, Name: sw.Name, Label: t.Label, OS: t.OS, Arch: t.Arch, Status: "error", Message: "获取 GitHub 版本失败: " + err.Error()})
				}
				continue
			}
			releaseCache[sw.Repo] = release
		}
		version := ReleaseVersion(release)

		for _, t := range sw.Targets {
			item := ReportItem{Key: t.ConfigKey, Name: sw.Name, Label: t.Label, OS: t.OS, Arch: t.Arch, Version: version}

			// 该入口配置了手工自定义外部链接（非 pan://）→ 跳过自动分发
			if current := loadSoftwareValue(t.ConfigKey); current != "" && !strings.HasPrefix(current, "pan://") {
				item.Status = "skip"
				item.Message = "该入口使用自定义链接，跳过自动分发"
				report = append(report, item)
				continue
			}

			asset, aerr := FindAssetFor(release, &t)
			if aerr != nil {
				item.Status = "error"
				item.Message = aerr.Error()
				report = append(report, item)
				continue
			}

			// 版本与资产名一致 → 已是最新，跳过
			entry, hasEntry := fileIDMap[t.ConfigKey]
			if hasEntry && entry.Version == version && entry.FileName == asset.Name {
				item.Status = "skip"
				item.FileName = asset.Name
				item.Message = "已是最新版本"
				report = append(report, item)
				continue
			}

			// 有新版本（或首次检出）：更新版本记录 + 写 pan:// 配置
			fileIDMap[t.ConfigKey] = FileEntry{
				FileName:  asset.Name,
				Size:      asset.Size,
				Version:   version,
				UpdatedAt: time.Now().Format(time.RFC3339),
			}
			if err := saveSoftwareValue(t.ConfigKey, "pan://"+t.ConfigKey); err != nil {
				item.Status = "error"
				item.Message = "写入配置失败: " + err.Error()
				report = append(report, item)
				continue
			}
			newVersions++
			item.Status = "ok"
			item.FileName = asset.Name
			item.Message = fmt.Sprintf("检测到新版本 v%s", version)
			report = append(report, item)
		}
	}

	if err := saveFileIDMap(fileIDMap); err != nil {
		report = append(report, ReportItem{Status: "error", Message: "保存版本记录失败: " + err.Error()})
	}
	if newVersions > 0 {
		log.Printf("[SoftwareSync] 检测到 %d 个软件有新版本", newVersions)
	}
	return report
}

// VersionEntry 版本接口输出项
type VersionEntry struct {
	Key       string `json:"key"`
	Version   string `json:"version"`
	FileName  string `json:"file_name"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updated_at"`
}

// ListVersions 返回所有已检出版本的列表（供前台显示版本号）
func ListVersions() []VersionEntry {
	fileIDMap, err := LoadFileIDMap()
	if err != nil {
		return nil
	}
	out := make([]VersionEntry, 0)
	for key, entry := range fileIDMap {
		if entry.Version == "" {
			continue
		}
		out = append(out, VersionEntry{
			Key: key, Version: entry.Version, FileName: entry.FileName,
			Size: entry.Size, UpdatedAt: entry.UpdatedAt,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
