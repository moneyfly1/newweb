package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/oschwald/maxminddb-golang"
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

type mmdbCityRecord struct {
	Country struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
}

const ipCacheMaxSize = 2048

var (
	ipLocationClient = &http.Client{Timeout: 3 * time.Second}
	ipLocationCache  = make(map[string]ipLocationCacheEntry)
	ipLocationMu     sync.RWMutex
	ipLocationTTL    = 30 * time.Minute
	mmdbReader       *maxminddb.Reader
	mmdbOnce         sync.Once
	ip2regionSearcher *xdb.Searcher
	ip2regionOnce     sync.Once
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

func loadIP2RegionSearcher() *xdb.Searcher {
	ip2regionOnce.Do(func() {
		xdbPath := filepath.Join("uploads", "config", "ip2region.xdb")
		if _, err := os.Stat(xdbPath); err != nil {
			fmt.Printf("[IP2Region] 文件不存在: %s\n", xdbPath)
			return
		}
		cBuff, err := xdb.LoadContentFromFile(xdbPath)
		if err != nil {
			fmt.Printf("[IP2Region] 加载失败: %v\n", err)
			return
		}
		searcher, err := xdb.NewWithBuffer(nil, cBuff)
		if err != nil {
			fmt.Printf("[IP2Region] 创建失败: %v\n", err)
			return
		}
		ip2regionSearcher = searcher
		fmt.Printf("[IP2Region] 成功加载: %s\n", xdbPath)
	})
	return ip2regionSearcher
}

func lookupLocationFromIP2Region(ip string) string {
	searcher := loadIP2RegionSearcher()
	if searcher == nil {
		return ""
	}
	region, err := searcher.SearchByStr(ip)
	if err != nil {
		return ""
	}
	// ip2region 格式: 国家|区域|省份|城市|ISP
	parts := strings.Split(region, "|")
	if len(parts) < 4 {
		return ""
	}
	result := []string{}
	if parts[0] != "0" && parts[0] != "" {
		result = append(result, parts[0])
	}
	if parts[2] != "0" && parts[2] != "" && parts[2] != parts[0] {
		result = append(result, parts[2])
	}
	if parts[3] != "0" && parts[3] != "" && parts[3] != parts[2] {
		result = append(result, parts[3])
	}
	return strings.Join(result, " ")
}

func loadMMDBReader() *maxminddb.Reader {
	mmdbOnce.Do(func() {
		candidates := []string{
			filepath.Join("uploads", "config", "GeoLite2-City.mmdb"),
			filepath.Join("uploads", "config", "geoip.metadb"),
			filepath.Join("uploads", "config", "Country.mmdb"),
		}
		for _, candidate := range candidates {
			fmt.Printf("[MMDB] 尝试加载: %s\n", candidate)
			if _, err := os.Stat(candidate); err != nil {
				fmt.Printf("[MMDB] 文件不存在: %v\n", err)
				continue
			}
			reader, err := maxminddb.Open(candidate)
			if err == nil {
				mmdbReader = reader
				fmt.Printf("[MMDB] 成功加载: %s\n", candidate)
				return
			}
			fmt.Printf("[MMDB] 加载失败: %v\n", err)
		}
		fmt.Printf("[MMDB] 所有数据库文件加载失败\n")
	})
	return mmdbReader
}

func lookupLocationFromMMDB(ip string) string {
	reader := loadMMDBReader()
	if reader == nil {
		return ""
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ""
	}
	var record mmdbCityRecord
	if err := reader.Lookup(parsedIP, &record); err != nil {
		fmt.Printf("[MMDB] 查询失败: %v\n", err)
		return ""
	}

	fmt.Printf("[MMDB] 原始数据 - Country: %v, Subdivisions: %v, City: %v\n",
		record.Country.Names, record.Subdivisions, record.City.Names)

	parts := make([]string, 0, 3)
	if country := record.Country.Names["zh-CN"]; country != "" {
		parts = append(parts, country)
	} else if country := record.Country.Names["en"]; country != "" {
		parts = append(parts, country)
	}
	if len(record.Subdivisions) > 0 {
		if region := record.Subdivisions[0].Names["zh-CN"]; region != "" {
			parts = append(parts, region)
		} else if region := record.Subdivisions[0].Names["en"]; region != "" {
			parts = append(parts, region)
		}
	}
	if city := record.City.Names["zh-CN"]; city != "" {
		parts = append(parts, city)
	} else if city := record.City.Names["en"]; city != "" {
		parts = append(parts, city)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

// GetIPLocation returns a location string for the given IP address.
// Prefers local MMDB lookup and falls back to the free ip-api.com service.
func GetIPLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "本地"
	}

	// Validate IP format
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		fmt.Printf("[IP] 无效的 IP 格式: %s\n", ip)
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

	// Try ip2region first (best for Chinese IPs)
	location := lookupLocationFromIP2Region(ip)
	fmt.Printf("[IP] IP2Region 查询 %s => %s\n", ip, location)
	if location != "" && strings.Contains(location, " ") {
		ipLocationMu.Lock()
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

	// Try MMDB second
	if location == "" {
		location = lookupLocationFromMMDB(ip)
		fmt.Printf("[IP] MMDB 查询 %s => %s\n", ip, location)
	}

	// 如果 MMDB 只返回国家（没有省份/城市），尝试 API 获取更详细信息
	if location != "" && !strings.Contains(location, " ") {
		apiLocation := queryIPAPI(ip)
		fmt.Printf("[IP] API 补充查询 %s => %s\n", ip, apiLocation)
		if apiLocation != "" && strings.Contains(apiLocation, " ") {
			location = apiLocation
		}
	}

	if location != "" {
		ipLocationMu.Lock()
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

	// Fallback to ip-api.com (supports both IPv4 and IPv6)
	location = queryIPAPI(ip)
	fmt.Printf("[IP] API 查询 %s => %s\n", ip, location)
	if location != "" {
		ipLocationMu.Lock()
		if len(ipLocationCache) >= ipCacheMaxSize {
			ipLocationCache = make(map[string]ipLocationCacheEntry)
		}
		ipLocationCache[ip] = ipLocationCacheEntry{
			location: location,
			expireAt: now.Add(ipLocationTTL),
		}
		ipLocationMu.Unlock()
	}
	return location
}

func queryIPAPI(ip string) string {
	urls := []string{
		fmt.Sprintf("https://ip-api.com/json/%s?lang=zh-CN&fields=status,country,regionName,city,query", ip),
		fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN&fields=status,country,regionName,city,query", ip),
	}
	var resp *http.Response
	var err error
	for _, apiURL := range urls {
		fmt.Printf("[API] 请求: %s\n", apiURL)
		resp, err = ipLocationClient.Get(apiURL) // #nosec G107 -- ip already validated by net.ParseIP
		if err == nil {
			break
		}
		fmt.Printf("[API] 请求失败: %v\n", err)
	}
	if err != nil || resp == nil {
		fmt.Printf("[API] 所有请求失败\n")
		return ""
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		fmt.Printf("[API] 解析失败: %v\n", err)
		return ""
	}
	fmt.Printf("[API] 响应: status=%s, country=%s, region=%s, city=%s\n", info.Status, info.Country, info.Region, info.City)
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
