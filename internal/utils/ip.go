package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type IPInfo struct {
	Country string `json:"country"`
	Region  string `json:"regionName"`
	City    string `json:"city"`
	Query   string `json:"query"`
	Status  string `json:"status"`
}

type ipLocationCacheEntry struct {
	location string
	expireAt time.Time
}

const ipCacheMaxSize = 2048

var (
	ipLocationClient = &http.Client{Timeout: 3 * time.Second}
	ipLocationCache  = make(map[string]ipLocationCacheEntry)
	ipLocationMu     sync.RWMutex
	ipLocationTTL    = 30 * time.Minute
)

func init() {
	go ipCacheCleaner()
}

// ipCacheCleaner periodically removes expired entries from the IP location cache.
func ipCacheCleaner() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		ipLocationMu.Lock()
		for k, v := range ipLocationCache {
			if now.After(v.expireAt) {
				delete(ipLocationCache, k)
			}
		}
		ipLocationMu.Unlock()
	}
}

// GetIPLocation returns a location string for the given IP address.
// Uses the free ip-api.com service. Returns empty string on failure.
func GetIPLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "本地"
	}

	// Validate IP format
	if net.ParseIP(ip) == nil {
		return ""
	}

	// Check for private IP ranges
	if isPrivateIP(ip) {
		return "本地网络"
	}

	now := time.Now()
	ipLocationMu.RLock()
	if cached, ok := ipLocationCache[ip]; ok && now.Before(cached.expireAt) {
		ipLocationMu.RUnlock()
		return cached.location
	}
	ipLocationMu.RUnlock()

	urls := []string{
		fmt.Sprintf("https://ip-api.com/json/%s?lang=zh-CN&fields=status,country,regionName,city,query", ip),
		fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN&fields=status,country,regionName,city,query", ip),
	}
	var resp *http.Response
	var err error
	for _, apiURL := range urls {
		resp, err = ipLocationClient.Get(apiURL) // #nosec G107 -- ip already validated by net.ParseIP
		if err == nil {
			break
		}
	}
	if err != nil || resp == nil {
		return ""
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return ""
	}
	if info.Status != "success" {
		return ""
	}

	location := info.Country
	if info.Region != "" && info.Region != info.Country {
		location += " " + info.Region
	}
	if info.City != "" && info.City != info.Region {
		location += " " + info.City
	}

	ipLocationMu.Lock()
	// Evict all if cache grows too large
	if len(ipLocationCache) >= ipCacheMaxSize {
		ipLocationCache = make(map[string]ipLocationCacheEntry)
	}
	ipLocationCache[ip] = ipLocationCacheEntry{
		location: location,
		expireAt: now.Add(ipLocationTTL),
	}
	ipLocationMu.Unlock()

	return location
}

// isPrivateIP checks if an IP address is in a private/reserved range
func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(parsed) {
			return true
		}
	}
	return false
}
