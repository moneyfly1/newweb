package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
)

type LogEntry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Level   string `json:"level"` // info, error, success
}

type ConfigUpdateConfig struct {
	URLs     []string `json:"urls"`
	Keywords []string `json:"keywords"`
	Enabled  bool     `json:"enabled"`
	Interval int      `json:"interval"` // minutes
}

type ConfigUpdateService struct {
	mu        sync.Mutex
	running   bool
	logs      []LogEntry
	stopCh    chan struct{}
	ticker    *time.Ticker
	scheduled bool
}

var (
	configUpdateInstance *ConfigUpdateService
	configUpdateOnce     sync.Once
)

func GetConfigUpdateService() *ConfigUpdateService {
	configUpdateOnce.Do(func() {
		configUpdateInstance = &ConfigUpdateService{
			logs: make([]LogEntry, 0),
		}
	})
	return configUpdateInstance
}

func (s *ConfigUpdateService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *ConfigUpdateService) IsScheduled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scheduled
}

func (s *ConfigUpdateService) GetLogs() []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]LogEntry, len(s.logs))
	copy(copied, s.logs)
	return copied
}

func (s *ConfigUpdateService) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make([]LogEntry, 0)
}

func (s *ConfigUpdateService) addLog(level, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := LogEntry{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Message: message,
		Level:   level,
	}
	s.logs = append(s.logs, entry)
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
	log.Printf("[ConfigUpdate][%s] %s", level, message)
}

// Start triggers a manual update run.
func (s *ConfigUpdateService) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("更新任务正在运行中")
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		s.runUpdate()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	return nil
}

// Stop signals a running update to stop (best-effort).
func (s *ConfigUpdateService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
		default:
			close(s.stopCh)
		}
	}
	s.running = false
	s.addLogUnlocked("info", "手动停止更新任务")
}

func (s *ConfigUpdateService) addLogUnlocked(level, message string) {
	entry := LogEntry{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Message: message,
		Level:   level,
	}
	s.logs = append(s.logs, entry)
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
}

// StartSchedule starts the background scheduled updater.
func (s *ConfigUpdateService) StartSchedule() {
	cfg, err := s.LoadConfig()
	if err != nil || !cfg.Enabled {
		return
	}
	s.mu.Lock()
	if s.scheduled {
		s.mu.Unlock()
		return
	}
	interval := cfg.Interval
	if interval < 1 {
		interval = 60
	}
	s.scheduled = true
	s.stopCh = make(chan struct{})
	s.ticker = time.NewTicker(time.Duration(interval) * time.Minute)
	s.mu.Unlock()

	s.addLog("info", fmt.Sprintf("定时更新已启动，间隔 %d 分钟", interval))

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.mu.Lock()
				if s.running {
					s.mu.Unlock()
					continue
				}
				s.running = true
				s.mu.Unlock()
				s.runUpdate()
				s.mu.Lock()
				s.running = false
				s.mu.Unlock()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// StopSchedule stops the background scheduled updater.
func (s *ConfigUpdateService) StopSchedule() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
		default:
			close(s.stopCh)
		}
	}
	s.scheduled = false
	s.addLogUnlocked("info", "定时更新已停止")
}

func (s *ConfigUpdateService) runUpdate() {
	s.mu.Lock()
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.addLog("info", "开始更新节点...")

	cfg, err := s.LoadConfig()
	if err != nil {
		s.addLog("error", "加载配置失败: "+err.Error())
		return
	}

	if len(cfg.URLs) == 0 {
		s.addLog("error", "没有配置订阅URL")
		return
	}

	var allNodes []models.Node

	for _, u := range cfg.URLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		// Check stop signal
		select {
		case <-s.stopCh:
			s.addLog("info", "更新已被中断")
			return
		default:
		}

		s.addLog("info", fmt.Sprintf("正在获取订阅: %s", u))
		content, err := FetchSubscriptionContent(u)
		if err != nil {
			s.addLog("error", fmt.Sprintf("获取订阅失败 [%s]: %s", u, err.Error()))
			continue
		}

		nodes, err := ParseNodeLinks(content)
		if err != nil {
			s.addLog("error", fmt.Sprintf("解析节点失败 [%s]: %s", u, err.Error()))
			continue
		}

		s.addLog("info", fmt.Sprintf("从 %s 解析到 %d 个节点", u, len(nodes)))
		allNodes = append(allNodes, nodes...)
	}

	// Filter by keywords
	if len(cfg.Keywords) > 0 {
		allNodes = s.filterNodes(allNodes, cfg.Keywords)
		s.addLog("info", fmt.Sprintf("关键词过滤后剩余 %d 个节点", len(allNodes)))
	}

	if len(allNodes) == 0 {
		s.addLog("info", "没有找到有效节点，跳过更新")
		return
	}

	db := database.GetDB()

	// Delete old auto-imported nodes and reset auto-increment
	result := db.Where("is_manual = ?", false).Delete(&models.Node{})
	s.addLog("info", fmt.Sprintf("已删除 %d 个旧的自动导入节点", result.RowsAffected))

	// Check if there are any manual nodes left
	var manualCount int64
	db.Model(&models.Node{}).Where("is_manual = ?", true).Count(&manualCount)
	if manualCount == 0 {
		// No nodes left at all, reset the auto-increment sequence
		db.Exec("DELETE FROM sqlite_sequence WHERE name = 'nodes'")
		s.addLog("info", "已重置节点ID序列")
	}

	// Insert new nodes
	successCount := 0
	for i, node := range allNodes {
		node.IsManual = false
		node.OrderIndex = i
		if err := db.Create(&node).Error; err == nil {
			successCount++
		}
	}

	s.addLog("success", fmt.Sprintf("更新完成: 共 %d 个节点，成功导入 %d 个", len(allNodes), successCount))
}

func (s *ConfigUpdateService) filterNodes(nodes []models.Node, keywords []string) []models.Node {
	if len(keywords) == 0 {
		return nodes
	}
	var filtered []models.Node
	for _, node := range nodes {
		match := false
		nameLower := strings.ToLower(node.Name)
		regionLower := strings.ToLower(node.Region)
		configLower := ""
		if node.Config != nil {
			configLower = strings.ToLower(*node.Config)
		}
		for _, kw := range keywords {
			kw = strings.TrimSpace(strings.ToLower(kw))
			if kw == "" {
				continue
			}
			// Also check region aliases (e.g. "hk" matches "香港")
			if strings.Contains(nameLower, kw) || strings.Contains(configLower, kw) || strings.Contains(regionLower, kw) || matchRegionAlias(kw, nameLower, regionLower) {
				match = true
				break
			}
		}
		if match {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

// matchRegionAlias checks if a keyword is a common region alias
func matchRegionAlias(keyword, name, region string) bool {
	aliases := map[string][]string{
		"hk": {"香港", "hong kong"},
		"us": {"美国", "united states", "usa"},
		"jp": {"日本", "japan"},
		"sg": {"新加坡", "singapore"},
		"tw": {"台湾", "taiwan"},
		"kr": {"韩国", "korea"},
		"uk": {"英国", "united kingdom"},
		"de": {"德国", "germany"},
		"fr": {"法国", "france"},
		"au": {"澳大利亚", "澳洲", "australia"},
		"ru": {"俄罗斯", "russia"},
		"in": {"印度", "india"},
		"ca": {"加拿大", "canada"},
	}
	if regions, ok := aliases[keyword]; ok {
		for _, r := range regions {
			if strings.Contains(name, r) || strings.Contains(region, r) {
				return true
			}
		}
	}
	// Reverse: if keyword is Chinese, check if name/region contains it
	return false
}

func (s *ConfigUpdateService) LoadConfig() (*ConfigUpdateConfig, error) {
	db := database.GetDB()
	var cfg models.SystemConfig
	result := db.Where("key = ? AND category = ?", "config_update", "node").First(&cfg)
	if result.Error != nil {
		return &ConfigUpdateConfig{
			URLs:     []string{},
			Keywords: []string{},
			Enabled:  false,
			Interval: 60,
		}, nil
	}
	var config ConfigUpdateConfig
	if err := json.Unmarshal([]byte(cfg.Value), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *ConfigUpdateService) SaveConfig(config *ConfigUpdateConfig) error {
	db := database.GetDB()
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	var existing models.SystemConfig
	result := db.Where("key = ? AND category = ?", "config_update", "node").First(&existing)
	if result.Error != nil {
		return db.Create(&models.SystemConfig{
			Key:         "config_update",
			Value:       string(data),
			Type:        "json",
			Category:    "node",
			DisplayName: "节点自动更新配置",
			Description: "订阅节点自动采集配置",
		}).Error
	}
	return db.Model(&existing).Update("value", string(data)).Error
}
