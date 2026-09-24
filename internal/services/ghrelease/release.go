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

// TagInfo 一个 tag（用于仓库还没有 Release 资产时回退取版本号）
type TagInfo struct {
	Name string `json:"name"`
}

// LatestTag 取仓库最新 tag 名（如 v0.0.1）。
//
// 为什么需要：自研客户端（如 moneyfly004/Mclash）可能只打了 tag、还没发布
// Release 资产。此时按 Release 解析会直接失败，链接就变成"点了没反应"。
// 有了它至少能识别出版本号，并把下载入口指向仓库的 Releases 页面。
func LatestTag(repo string, prefixes []string, token string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=1", repo)
	if len(prefixes) == 0 {
		prefixes = DefaultProxyPrefixes
	}
	var lastErr error
	for _, cand := range buildCandidateURLs(apiURL, prefixes) {
		tags, err := fetchTags(cand, token)
		if err != nil {
			lastErr = err
			continue
		}
		if len(tags) > 0 {
			return tags[0].Name, nil
		}
		return "", fmt.Errorf("仓库没有任何 tag")
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("无可用候选地址")
	}
	return "", lastErr
}

// LatestListed 从 Releases 列表里取最新一条。
// /releases/latest 在「只有预发布版本」时会 404，但列表接口仍能取到。
func LatestListed(repo string, prefixes []string, token string) (*Release, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=1", repo)
	if len(prefixes) == 0 {
		prefixes = DefaultProxyPrefixes
	}
	var lastErr error
	for _, cand := range buildCandidateURLs(apiURL, prefixes) {
		list, err := fetchReleaseList(cand, token)
		if err != nil {
			lastErr = err
			continue
		}
		if len(list) > 0 {
			return &list[0], nil
		}
		return nil, fmt.Errorf("仓库没有任何 Release")
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("无可用候选地址")
	}
	return nil, lastErr
}

func fetchTags(url, token string) ([]TagInfo, error) {
	body, err := getJSON(url, token)
	if err != nil {
		return nil, err
	}
	var tags []TagInfo
	if err := json.Unmarshal(body, &tags); err != nil {
		return nil, fmt.Errorf("解析 tags 失败: %w", err)
	}
	return tags, nil
}

func fetchReleaseList(url, token string) ([]Release, error) {
	body, err := getJSON(url, token)
	if err != nil {
		return nil, err
	}
	var list []Release
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("解析 releases 失败: %w", err)
	}
	return list, nil
}

func getJSON(url, token string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cboard-download")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, strings.TrimSpace(string(body[:min(len(body), 120)])))
	}
	return body, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
