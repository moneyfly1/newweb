package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services/ghrelease"
	"cboard/v2/internal/services/software_sync"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// GitHub Releases 国内镜像直链（方案A，移植自 myweb）
//
// 用户点击下载 → /download/gh?key=<配置键> → VPS 查 GitHub 最新 Release（30分钟缓存）
// → 匹配安装包资产 → 302 跳转到国内加速镜像（5分钟探测缓存轮换）。
// 用户直接连国内镜像下载，速度快，且不占用 VPS 带宽。
//
// /download/resolve?target=<URL>：通用解析，探测可用的加速镜像后 302。
// ---------------------------------------------------------------------------

const (
	ghReleaseCacheTTL = 30 * time.Minute
	mirrorProbeTTL    = 5 * time.Minute
)

// 公共解析接口全局限流：防止滥用（每次请求会消耗 GitHub API 额度）
var resolveGate = struct {
	mu   sync.Mutex
	last time.Time
}{}

func resolveThrottled() bool {
	resolveGate.mu.Lock()
	defer resolveGate.mu.Unlock()
	if time.Since(resolveGate.last) < 50*time.Millisecond {
		return false
	}
	resolveGate.last = time.Now()
	return true
}

// defaultDownloadProxyPrefixes 国内 GitHub 加速镜像（2026-08 实测可用）
var defaultDownloadProxyPrefixes = []string{
	"https://ghfast.top/{url}",
	"https://gh-proxy.com/{url}",
	"https://gh.llkk.cc/{url}",
	"https://gh.ddlc.top/{url}",
	"{url}",
}

// loadDownloadProxyPrefixes 读取自定义镜像前缀（settings 中 download_proxy_prefixes）
func loadDownloadProxyPrefixes() []string {
	db := database.GetDB()
	if db == nil {
		return defaultDownloadProxyPrefixes
	}
	var conf models.SystemConfig
	if err := db.Where("`key` = ? AND category = ?", "download_proxy_prefixes", "software").First(&conf).Error; err != nil {
		return defaultDownloadProxyPrefixes
	}
	custom := parseProxyPrefixes(conf.Value)
	if len(custom) == 0 {
		return defaultDownloadProxyPrefixes
	}
	return custom
}

func parseProxyPrefixes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parsed []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			return normalizeProxyPrefixes(parsed)
		}
	}
	lines := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	return normalizeProxyPrefixes(lines)
}

func normalizeProxyPrefixes(items []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(items)+1)
	for _, item := range items {
		p := strings.TrimSpace(item)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	hasDirect := false
	for _, p := range out {
		if p == "{url}" || p == "direct" || p == "DIRECT" {
			hasDirect = true
			break
		}
	}
	if !hasDirect {
		out = append(out, "{url}")
	}
	return out
}

// GitHubResolve 根据软件配置 key 解析 GitHub 最新 Release 资产并 302 到国内镜像直链
func GitHubResolve(c *gin.Context) {
	if !resolveThrottled() {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": 1, "message": "请求过于频繁，请稍后再试"})
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "缺少 key 参数"})
		return
	}
	sw := software_sync.FindSoftwareByConfigKey(key)
	if sw == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "未知的软件配置键"})
		return
	}
	t := software_sync.FindTarget(key)
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "未知的软件配置键"})
		return
	}

	prefixes := loadDownloadProxyPrefixes()
	release, err := cachedLatestRelease(sw.Repo, prefixes, utils.GetSetting("gh_nodes_token"))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "message": "获取 GitHub 版本失败: " + err.Error()})
		return
	}
	asset, aerr := FindAssetFor(release, t)
	if aerr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": aerr.Error()})
		return
	}

	dlURL := pickMirrorURL(prefixes, asset.BrowserDownloadURL)
	c.Redirect(http.StatusFound, dlURL)
}

// FindAssetFor 在 Release 资产中按目标匹配规则挑选最合适的文件
func FindAssetFor(release *ghrelease.Release, t *software_sync.Target) (*ghrelease.Asset, error) {
	if release == nil {
		return nil, fmt.Errorf("无 Release 数据")
	}
	// 先尝试 Preferred 规则
	if len(t.Preferred) > 0 {
		for _, asset := range release.Assets {
			for _, re := range t.Preferred {
				if re.MatchString(asset.Name) {
					return &asset, nil
				}
			}
		}
	}
	// 再尝试 Patterns
	for _, asset := range release.Assets {
		for _, re := range t.Patterns {
			if re.MatchString(asset.Name) {
				return &asset, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到匹配的下载文件（平台: %s, 架构: %s）", t.OS, t.Arch)
}

// ---------------------------------------------------------------------------
// Release 缓存（30 分钟，避免每个用户点击都打 GitHub API）
// ---------------------------------------------------------------------------

type ghReleaseCacheEntry struct {
	Release *ghrelease.Release
	Expire  time.Time
}

var ghReleaseCache sync.Map // repo → ghReleaseCacheEntry

func cachedLatestRelease(repo string, prefixes []string, token string) (*ghrelease.Release, error) {
	if v, ok := ghReleaseCache.Load(repo); ok {
		e := v.(ghReleaseCacheEntry)
		if time.Now().Before(e.Expire) {
			return e.Release, nil
		}
		ghReleaseCache.Delete(repo)
	}
	rel, err := ghrelease.Latest(repo, prefixes, token)
	if err != nil {
		return nil, err
	}
	ghReleaseCache.Store(repo, ghReleaseCacheEntry{Release: rel, Expire: time.Now().Add(ghReleaseCacheTTL)})
	return rel, nil
}

// ---------------------------------------------------------------------------
// 镜像选择：按配置前缀构造候选地址，逐个探测可用性（5 分钟缓存），全挂则直连
// ---------------------------------------------------------------------------

type mirrorProbeEntry struct {
	OK     bool
	Expire time.Time
}

var mirrorProbeCache sync.Map // host → mirrorProbeEntry

func pickMirrorURL(prefixes []string, rawURL string) string {
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var candidate string
		switch {
		case p == "{url}" || strings.EqualFold(p, "direct"):
			candidate = rawURL
		case strings.Contains(p, "{url}"):
			candidate = strings.ReplaceAll(p, "{url}", rawURL)
		default:
			candidate = strings.TrimRight(p, "/") + "/" + rawURL
		}
		if probeMirrorOK(candidate) {
			return candidate
		}
	}
	return rawURL // 兜底：直连 GitHub
}

func probeMirrorOK(rawURL string) bool {
	host := urlHost(rawURL)
	if host != "" {
		if v, ok := mirrorProbeCache.Load(host); ok {
			e := v.(mirrorProbeEntry)
			if time.Now().Before(e.Expire) {
				return e.OK
			}
		}
	}
	ok := probeHead(rawURL)
	if host != "" {
		mirrorProbeCache.Store(host, mirrorProbeEntry{OK: ok, Expire: time.Now().Add(mirrorProbeTTL)})
	}
	return ok
}

// probeHead 用 Range 请求探测镜像是否可下载（206/200 且非 HTML 视为可用）
func probeHead(rawURL string) bool {
	client := &http.Client{Timeout: 6 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Range", "bytes=0-0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return false
	}
	ct := strings.ToLower(resp.Header.Get("content-type"))
	if strings.Contains(ct, "text/html") {
		return false
	}
	return true
}

func urlHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// ---------------------------------------------------------------------------
// 通用下载解析：/download/resolve?target=<URL>（探测加速镜像后 302）
// ---------------------------------------------------------------------------

func ResolveDownload(c *gin.Context) {
	target := strings.TrimSpace(c.Query("target"))
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "缺少 target 参数"})
		return
	}
	if !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "http://") {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的下载链接"})
		return
	}
	if err := validateDownloadURL(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的下载链接"})
		return
	}

	candidates := buildDownloadCandidates(target, loadDownloadProxyPrefixes())

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resultCh := make(chan string, len(candidates))
	for _, candidate := range candidates {
		go func(u string) {
			if isDownloadURLReachable(u) {
				resultCh <- u
			}
		}(candidate)
	}

	select {
	case u := <-resultCh:
		c.Redirect(http.StatusFound, u)
		return
	case <-ctx.Done():
	}
	c.Redirect(http.StatusFound, target) // 所有代理不可用时回退原始地址
}

// validateDownloadURL 校验下载目标 URL：协议白名单 + 解析后主机非内网/回环地址（防 SSRF）
func validateDownloadURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("不支持的协议: %s", parsed.Scheme)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL缺少主机名")
	}
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return fmt.Errorf("禁止访问本地地址")
	}
	ips, err := net.LookupIP(hostname)
	if err == nil {
		for _, ip := range ips {
			if ip.IsLoopback() || ip.IsPrivate() {
				return fmt.Errorf("禁止访问内网地址: %s", ip.String())
			}
		}
	}
	return nil
}

func buildDownloadCandidates(target string, prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	seen := make(map[string]struct{})
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var candidate string
		switch {
		case p == "{url}" || strings.EqualFold(p, "direct"):
			candidate = target
		case strings.Contains(p, "{url}"):
			candidate = strings.ReplaceAll(p, "{url}", target)
		default:
			candidate = strings.TrimRight(p, "/") + "/" + target
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	if _, ok := seen[target]; !ok {
		out = append(out, target)
	}
	return out
}

func isDownloadURLReachable(u string) bool {
	client := &http.Client{
		Timeout: 2500 * time.Millisecond,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true
		}
	}
	req, err = http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err = client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
