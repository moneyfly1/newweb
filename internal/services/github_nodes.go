package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/config"
	"cboard/v2/internal/services/git"
	"cboard/v2/internal/utils"
)

const (
	// GithubNodesPublicPrefix 公开访问前缀：{site_url}/nodes/{filename}
	GithubNodesPublicPrefix = "/nodes/"
	// GithubNodesDirName 本地保存目录名（位于 uploads 目录下）
	GithubNodesDirName = "nodes"
)

// GithubNodesConfig 从系统设置中读取的同步配置
type GithubNodesConfig struct {
	Enabled  bool
	Token    string
	Repo     string // 仓库地址：owner/repo 或完整 URL
	Branch   string
	Path     string // 仓库内要下载的目录
	Interval int    // 同步间隔（分钟）
}

// ParseGithubRepoAddress 解析仓库地址（支持 owner/repo、github.com/owner/repo、完整 URL）
func ParseGithubRepoAddress(addr string) (owner, repo string, err error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", "", fmt.Errorf("仓库地址为空")
	}
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "github.com/")
	addr = strings.TrimSuffix(addr, ".git")
	addr = strings.Trim(addr, "/")
	parts := strings.Split(addr, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("仓库地址格式无效，应为 owner/repo（如 moneyfly006/nodes）")
	}
	return parts[0], parts[1], nil
}

// GithubNodesFile 已同步到本地的节点文件
type GithubNodesFile struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updated_at"`
}

// GithubNodesStatus 同步状态
type GithubNodesStatus struct {
	Running       bool              `json:"running"`
	Scheduled     bool              `json:"scheduled"`
	Enabled       bool              `json:"enabled"`
	IntervalMin   int               `json:"interval_minutes"`
	LastSyncAt    string            `json:"last_sync_at"`
	LastResult    string            `json:"last_result"`
	LastError     string            `json:"last_error"`
	FileCount     int               `json:"file_count"`
	Files         []GithubNodesFile `json:"files"`
	PublicPrefix  string            `json:"public_prefix"`
	LastFileCount int               `json:"last_file_count"`
}

// GithubNodesService 定时从 GitHub 仓库下载节点文件到 uploads/nodes 并提供公开外链
type GithubNodesService struct {
	mu             sync.Mutex
	running        bool
	scheduled      bool
	logs           []LogEntry
	lastSyncAt     time.Time
	lastResult     string // "" / "success" / "error"
	lastError      string
	lastFileCount  int
	ticker         *time.Ticker
	scheduleStopCh chan struct{}
}

var (
	githubNodesInstance *GithubNodesService
	githubNodesOnce     sync.Once
)

// GetGithubNodesService 获取单例
func GetGithubNodesService() *GithubNodesService {
	githubNodesOnce.Do(func() {
		githubNodesInstance = &GithubNodesService{
			logs: make([]LogEntry, 0),
		}
	})
	return githubNodesInstance
}

func (s *GithubNodesService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *GithubNodesService) IsScheduled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scheduled
}

func (s *GithubNodesService) GetLogs() []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]LogEntry, len(s.logs))
	copy(copied, s.logs)
	return copied
}

func (s *GithubNodesService) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make([]LogEntry, 0)
}

func (s *GithubNodesService) addLog(level, message string) {
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
	log.Printf("[GithubNodes][%s] %s", level, message)
}

func (s *GithubNodesService) setError(msg string) {
	s.mu.Lock()
	s.lastResult = "error"
	s.lastError = msg
	// 失败也记录时间，避免定时器每分钟重复尝试
	s.lastSyncAt = time.Now()
	s.mu.Unlock()
	s.addLog("error", msg)
}

// LoadConfig 从系统设置读取配置（每次运行都重新读取，设置修改即时生效）
func (s *GithubNodesService) LoadConfig() *GithubNodesConfig {
	cfg := &GithubNodesConfig{
		Enabled:  utils.IsBoolSetting("gh_nodes_enabled"),
		Token:    strings.TrimSpace(utils.GetSetting("gh_nodes_token")),
		Repo:     strings.TrimSpace(utils.GetSetting("gh_nodes_repo")),
		Branch:   strings.TrimSpace(utils.GetSetting("gh_nodes_branch")),
		Path:     strings.TrimSpace(utils.GetSetting("gh_nodes_path")),
		Interval: utils.GetIntSetting("gh_nodes_interval", 10),
	}
	if cfg.Repo == "" {
		cfg.Repo = "moneyfly006/nodes"
	}
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if cfg.Path == "" {
		cfg.Path = "nodes"
	}
	if cfg.Interval < 1 {
		cfg.Interval = 10
	}
	if cfg.Interval > 1440 {
		cfg.Interval = 1440
	}
	return cfg
}

// localRoot 本地保存目录：uploads/nodes
func (s *GithubNodesService) localRoot() string {
	return filepath.Join(config.AppConfig.UploadDir, GithubNodesDirName)
}

// Start 手动触发一次同步
func (s *GithubNodesService) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("同步任务正在运行中")
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		s.runSync()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	return nil
}

// StartSchedule 启动定时调度：每分钟检查一次，满足间隔且启用时自动同步。
// 间隔/开关在设置页修改后无需重启服务，最多 1 分钟内生效。
func (s *GithubNodesService) StartSchedule() {
	s.mu.Lock()
	if s.scheduled {
		s.mu.Unlock()
		return
	}
	s.scheduled = true
	s.scheduleStopCh = make(chan struct{})
	s.ticker = time.NewTicker(time.Minute)
	s.mu.Unlock()

	go func() {
		for {
			select {
			case <-s.ticker.C:
				cfg := s.LoadConfig()
				if !cfg.Enabled {
					continue
				}
				s.mu.Lock()
				due := s.lastSyncAt.IsZero() || time.Since(s.lastSyncAt) >= time.Duration(cfg.Interval)*time.Minute
				s.mu.Unlock()
				if due {
					_ = s.Start() // 正在运行时 Start 返回错误，直接跳过
				}
			case <-s.scheduleStopCh:
				return
			}
		}
	}()
}

// StopSchedule 停止定时调度
func (s *GithubNodesService) StopSchedule() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.scheduleStopCh != nil {
		select {
		case <-s.scheduleStopCh:
		default:
			close(s.scheduleStopCh)
		}
	}
	s.scheduled = false
}

// Status 当前状态（含本地已同步文件列表）
func (s *GithubNodesService) Status() GithubNodesStatus {
	cfg := s.LoadConfig()
	s.mu.Lock()
	st := GithubNodesStatus{
		Running:       s.running,
		Scheduled:     s.scheduled,
		Enabled:       cfg.Enabled,
		IntervalMin:   cfg.Interval,
		LastResult:    s.lastResult,
		LastError:     s.lastError,
		LastFileCount: s.lastFileCount,
		PublicPrefix:  GithubNodesPublicPrefix,
	}
	if !s.lastSyncAt.IsZero() {
		st.LastSyncAt = s.lastSyncAt.Format("2006-01-02 15:04:05")
	}
	s.mu.Unlock()

	root := s.localRoot()
	files := make([]GithubNodesFile, 0)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || strings.HasSuffix(info.Name(), ".part") {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		files = append(files, GithubNodesFile{
			Path:      filepath.ToSlash(rel),
			Size:      info.Size(),
			UpdatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		return nil
	})
	st.Files = files
	st.FileCount = len(files)
	return st
}

type remoteNodeFile struct {
	RelPath     string
	DownloadURL string
}

// listRemoteFiles 递归列出仓库目录下的所有文件，返回相对路径与下载地址。
// 列表失败（token 无效 401、仓库/目录不存在 404 等）时直接返回错误，本地文件不做任何改动。
func (s *GithubNodesService) listRemoteFiles(client *git.GitClient, cfg *GithubNodesConfig) ([]remoteNodeFile, error) {
	prefix := strings.Trim(cfg.Path, "/")
	if prefix == "" {
		prefix = "nodes"
	}

	var result []remoteNodeFile
	var walk func(dir string, depth int) error
	walk = func(dir string, depth int) error {
		if depth > 10 {
			return fmt.Errorf("目录层级过深（超过 10 层）")
		}
		entries, err := client.ListDirWithRef(dir, cfg.Branch)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if len(result) >= 2000 {
				return fmt.Errorf("文件数量超过上限 2000")
			}
			rel := strings.TrimPrefix(e.Path, prefix)
			rel = strings.Trim(rel, "/")
			switch e.Type {
			case "dir":
				if err := walk(e.Path, depth+1); err != nil {
					return err
				}
			case "file":
				if err := validateRelPath(rel); err != nil {
					s.addLog("error", fmt.Sprintf("跳过非法路径 [%s]: %s", e.Path, err.Error()))
					continue
				}
				result = append(result, remoteNodeFile{RelPath: rel, DownloadURL: e.DownloadURL})
			}
		}
		return nil
	}

	if err := walk(prefix, 0); err != nil {
		return nil, err
	}
	return result, nil
}

func validateRelPath(rel string) error {
	clean := filepath.Clean(rel)
	if clean == "." || clean == "" || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
		return fmt.Errorf("非法相对路径")
	}
	return nil
}

// runSync 执行一次完整同步：下载远端文件 -> 清理本地多余文件。
func (s *GithubNodesService) runSync() {
	s.addLog("info", "开始同步 GitHub 节点文件...")

	cfg := s.LoadConfig()
	if cfg.Token == "" {
		s.setError("未配置 GitHub Token，请在系统设置中填写")
		return
	}
	owner, repo, err := ParseGithubRepoAddress(cfg.Repo)
	if err != nil {
		s.setError(err.Error())
		return
	}

	client := git.NewClient(git.PlatformGitHub, cfg.Token, owner, repo)
	files, err := s.listRemoteFiles(client, cfg)
	if err != nil {
		s.setError("获取仓库文件列表失败: " + err.Error())
		return
	}

	root := s.localRoot()
	expected := make(map[string]bool, len(files))
	successCount := 0
	for _, f := range files {
		expected[f.RelPath] = true
		local := filepath.Join(root, f.RelPath)
		part := local + ".part"
		// 私有仓库的 download_url 需要带 Authorization 头，DownloadFile 已处理；
		// 若实测 404，可改用 contents API + Accept: application/vnd.github.raw 作为回退。
		if err := client.DownloadFile(f.DownloadURL, part); err != nil {
			os.Remove(part)
			s.addLog("error", fmt.Sprintf("下载失败 [%s]: %s", f.RelPath, err.Error()))
			continue
		}
		// 先写 .part 再改名，避免公开外链暴露下载不完整的文件
		if err := os.Rename(part, local); err != nil {
			os.Remove(part)
			s.addLog("error", fmt.Sprintf("保存失败 [%s]: %s", f.RelPath, err.Error()))
			continue
		}
		successCount++
	}

	// 仅当远端列表获取成功后清理本地多余文件，保证仓库与本地目录一致
	pruned := s.pruneLocal(root, expected)

	s.mu.Lock()
	s.lastSyncAt = time.Now()
	s.lastResult = "success"
	s.lastError = ""
	s.lastFileCount = successCount
	s.mu.Unlock()
	s.addLog("success", fmt.Sprintf("同步完成: 成功 %d/%d 个文件，清理 %d 个本地多余文件", successCount, len(files), pruned))
}

// pruneLocal 删除本地存在但远端已不存在的文件（含残留的 .part 临时文件），返回删除数量。
func (s *GithubNodesService) pruneLocal(root string, expected map[string]bool) int {
	pruned := 0
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || path == root || info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, ".part") || !expected[rel] {
			if os.Remove(path) == nil {
				pruned++
			}
		}
		return nil
	})
	// 清理空目录（多轮，处理嵌套空目录）
	for i := 0; i < 10; i++ {
		removed := false
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || path == root || !info.IsDir() {
				return nil
			}
			entries, _ := os.ReadDir(path)
			if len(entries) == 0 && os.Remove(path) == nil {
				removed = true
			}
			return nil
		})
		if !removed {
			break
		}
	}
	return pruned
}
