package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/oschwald/maxminddb-golang"
)

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
	ipLocationCache = make(map[string]ipLocationCacheEntry)
	ipLocationMu    sync.RWMutex
	ipLocationTTL   = 30 * time.Minute
	mmdbReader      *maxminddb.Reader
	mmdbOnce        sync.Once
	// ip2region 的 v4/v6 是两个独立的库，必须分别加载并各自用对应 Version，
	// 查询时再按 IP 类型选择（详见 loadIP2RegionSearchers）。
	ip4Searcher   *xdb.Searcher
	ip6Searcher   *xdb.Searcher
	ip2regionOnce sync.Once
	geoIPReloadMu sync.Mutex
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

// loadIP2RegionSearchers 分别加载 IPv4 / IPv6 库（各只加载一次）。
//
// 关键：加载时**必须**传入与实际库匹配的 Version —— SDK 的 Search() 会用
// `len(ip) != version.Bytes` 做长度校验，而 IPvx 是空 Version（Bytes=0），
// 传它会导致**任何** IP 都报 `invalid ip address( expected)`，地理位置全部退化成
// “未知”（线上实测：1.198.223.178 这类普通 IPv4 也照样失败）。
//
// 另外 v4/v6 是两套库，不能只加载其中一个就返回，否则另一族地址永远查不到。
func loadIP2RegionSearchers() (*xdb.Searcher, *xdb.Searcher) {
	ip2regionOnce.Do(func() {
		load := func(version *xdb.Version, name string) *xdb.Searcher {
			path := filepath.Join("uploads", "config", name)
			if _, err := os.Stat(path); err != nil {
				fmt.Printf("[IP2Region] 文件不存在: %s\n", path)
				return nil
			}
			searcher, err := xdb.NewWithFileOnly(version, path)
			if err != nil {
				fmt.Printf("[IP2Region] 创建失败 %s: %v\n", path, err)
				return nil
			}
			fmt.Printf("[IP2Region] 成功加载: %s (%s)\n", path, version.Name)
			return searcher
		}
		ip4Searcher = load(xdb.IPv4, "ip2region_v4.xdb")
		ip6Searcher = load(xdb.IPv6, "ip2region_v6.xdb")
		if ip4Searcher == nil && ip6Searcher == nil {
			fmt.Printf("[IP2Region] 所有数据库文件加载失败\n")
		}
	})
	return ip4Searcher, ip6Searcher
}

func lookupLocationFromIP2Region(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	v4Searcher, v6Searcher := loadIP2RegionSearchers()
	// 按地址族选择对应库：IPv4 用 v4 库，IPv6 用 v6 库
	searcher := v6Searcher
	if parsed.To4() != nil {
		searcher = v4Searcher
	}
	if searcher == nil {
		return ""
	}
	region, err := searcher.SearchByStr(ip)
	if err != nil {
		fmt.Printf("[IP2Region] 查询错误 %s: %v\n", ip, err)
		return ""
	}
	// ip2region 格式: 国家|区域|省份|城市|ISP
	// 返回 国家 省份 城市（识别不到的自动省略）
	parts := strings.Split(region, "|")
	if len(parts) < 2 {
		return ""
	}
	result := []string{}
	// 国家
	if parts[0] != "0" && parts[0] != "" {
		result = append(result, parts[0])
	}
	// 省份
	if len(parts) > 2 && parts[2] != "0" && parts[2] != "" && parts[2] != parts[0] {
		result = append(result, parts[2])
	}
	// 城市
	if len(parts) > 3 && parts[3] != "0" && parts[3] != "" && parts[3] != parts[2] {
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

	// 只返回 国家 + 城市（跳过省份/州）
	parts := make([]string, 0, 2)
	if country := record.Country.Names["zh-CN"]; country != "" {
		parts = append(parts, country)
	} else if country := record.Country.Names["en"]; country != "" {
		parts = append(parts, country)
	}
	// 跳过省份（Subdivisions），直接取城市
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
// Uses only local offline databases (ip2region + MMDB) for zero network latency.
func GetIPLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "本地"
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ""
	}

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

	// Try ip2region first (faster and more accurate for CN/Asia)
	location := lookupLocationFromIP2Region(ip)
	if location == "" {
		// Fallback to MMDB
		location = lookupLocationFromMMDB(ip)
	}

	if location == "" {
		location = "未知"
	}

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

// ReloadGeoIP 重新加载地理位置库（GeoIP 数据库更新后调用）。
//
// 为什么必须重载：ip2region/mmdb 的句柄由 sync.Once 缓存，进程启动后只加载一次；
// 只替换磁盘文件而不重载，更新会“看起来成功但完全不生效”。同时清空 IP 缓存，
// 避免旧结果（例如历史上失败留下的“未知”）继续被复用。
func ReloadGeoIP() {
	geoIPReloadMu.Lock()
	defer geoIPReloadMu.Unlock()

	// 重置懒加载状态
	ip2regionOnce = sync.Once{}
	ip4Searcher = nil
	ip6Searcher = nil
	if mmdbReader != nil {
		_ = mmdbReader.Close()
	}
	mmdbReader = nil
	mmdbOnce = sync.Once{}

	// 清空 IP → 地区 缓存
	ipLocationMu.Lock()
	ipLocationCache = make(map[string]ipLocationCacheEntry)
	ipLocationMu.Unlock()

	// 立即触发加载，让调用方随后就能用上新库
	v4, v6 := loadIP2RegionSearchers()
	loadMMDBReader()
	fmt.Printf("[GeoIP] 已重载地理位置库: ip2region_v4=%v ip2region_v6=%v\n", v4 != nil, v6 != nil)
}
