// Package ghrelease 从 GitHub Releases API 获取最新版本信息（带加速镜像候选重试）。
package ghrelease

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Asset 一个发布资产
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Release 最新版本
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// DefaultProxyPrefixes 国内 GitHub 加速镜像（2026-08 实测可用）
var DefaultProxyPrefixes = []string{
	"https://ghfast.top/{url}",
	"https://gh-proxy.com/{url}",
	"https://gh.llkk.cc/{url}",
	"https://gh.ddlc.top/{url}",
	"{url}",
}

// Latest 获取指定仓库的最新 Release。
// prefixes 为加速前缀列表（含 {url} 占位符），逐个候选请求，成功即返回。
// token 可选：提供则附带 Authorization 头（提高 API 限额）。
func Latest(repo string, prefixes []string, token string) (*Release, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	if len(prefixes) == 0 {
		prefixes = DefaultProxyPrefixes
	}
	candidates := buildCandidateURLs(apiURL, prefixes)

	var lastErr error
	for _, cand := range candidates {
		rel, err := fetchOne(cand, token)
		if err == nil {
			return rel, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("无可用候选地址")
	}
	return nil, lastErr
}

func fetchOne(url, token string) (*Release, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "CBoard/2.0")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func buildCandidateURLs(rawURL string, prefixes []string) []string {
	out := make([]string, 0, len(prefixes)+1)
	seen := make(map[string]struct{})
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var cand string
		switch {
		case p == "{url}" || strings.EqualFold(p, "direct"):
			cand = rawURL
		case strings.Contains(p, "{url}"):
			cand = strings.ReplaceAll(p, "{url}", rawURL)
		default:
			cand = strings.TrimRight(p, "/") + "/" + rawURL
		}
		if _, ok := seen[cand]; ok {
			continue
		}
		seen[cand] = struct{}{}
		out = append(out, cand)
	}
	if _, ok := seen[rawURL]; !ok {
		out = append(out, rawURL)
	}
	return out
}
