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

// 地区字符串的固定取值。
//
// 以前「查不到」这件事有三种写法：循环地址回 "本地"、内网回 "本地网络"、
// 两个库都查不到回 "未知"，各处判断又各自手写字符串比较——
// 少写一个分支就会把「未知」当成真实地区统计进去（历史回填就踩过这个坑）。
const (
	// LocationLocal 本机回环地址
	LocationLocal = "本地"
	// LocationPrivate 内网/保留地址
	LocationPrivate = "本地网络"
	// LocationUnknown 两个离线库都查不到
	LocationUnknown = "未知"
)

// IsUnknownLocation 判断地区值是否为「没有真实地区信息」（空 / 未知 / 本机）。
func IsUnknownLocation(location string) bool {
	switch strings.TrimSpace(location) {
	case "", LocationUnknown, LocationLocal, LocationPrivate:
		return true
	}
	return false
}

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

// joinLocationParts 拼装「国家 省份 城市」形式的地区字符串。
//
// 两个离线库此前各拼各的：ip2region 输出「国家 省份 城市」，MMDB 输出「国家 城市」
// （注释里写着「跳过省份」）。同一个 IP 走不同库得到不同粒度的地区，
// 后台的地区统计（按 国家/省份/城市 聚合）就会把同一批用户拆成两组。
// 现在两个库都按同一形状输出，查不到的部分自动省略、相邻重复段去重。
func joinLocationParts(parts ...string) string {
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "0" {
			continue
		}
		if len(result) > 0 && result[len(result)-1] == p {
			continue
		}
		result = append(result, p)
	}
	return strings.Join(result, " ")
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
	// 注意字段布局：不能写死下标（见 parseIP2RegionFields）
	return joinLocationParts(parseIP2RegionFields(strings.Split(region, "|")))
}

// parseIP2RegionFields 解析 ip2region 的返回串，返回 国家/省份/城市。
//
// 为什么需要这个函数：xdb 文件有两种字段布局，而代码此前一律按
// 「国家|区域|省份|城市|ISP」取下标 —— 线上实际的库是新版布局
// 「国家|省份|城市|ISP|国家码」，于是取到的是
//
//	省份 ← 城市（郑州市）、城市 ← ISP（电信）
//
// 结果存进数据库的地区是「中国 郑州市 电信」「United States Google LLC」：
// 省份列里放的是城市、城市列里放的是运营商，后台按 国家/省份/城市 聚合统计自然错位。
// 现在按内容判断布局（新版末位是两位国家码），两种库都能得到 国家 省份 城市。
func parseIP2RegionFields(parts []string) (country, province, city string) {
	if len(parts) < 2 {
		return "", "", ""
	}
	country = cleanRegionPart(parts[0])
	// 新版布局的第 2 段是省份（真实地名），旧版布局的第 2 段是「区域」，
	// 实际数据里恒为 "0" 或空 —— 用它来区分两种布局最可靠。
	if len(parts) >= 5 && !isEmptyRegionPart(parts[1]) && isCountryCode(parts[4]) {
		// 新版 ip2region_v4.xdb / ip2region_v6.xdb: 国家|省份|城市|ISP|国家码
		return country, cleanRegionPart(parts[1]), cleanRegionPart(parts[2])
	}
	// 旧版 ip2region.db 时代: 国家|区域|省份|城市|ISP
	if len(parts) > 3 {
		return country, cleanRegionPart(parts[2]), cleanRegionPart(parts[3])
	}
	return country, "", ""
}

// cleanRegionPart 把 ip2region 用来表示「无数据」的 "0" 归一成空串。
func cleanRegionPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "0" {
		return ""
	}
	return s
}

func isEmptyRegionPart(s string) bool {
	return cleanRegionPart(s) == ""
}

// isCountryCode 判断字符串是否为两位国家码（新版 xdb 用 "0" 表示无数据）。
func isCountryCode(s string) bool {
	if s == "0" {
		return true
	}
	if len(s) != 2 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
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

	// 与 ip2region 输出同一形状：国家 省份 城市
	// （此前这里只取「国家 城市」，同一个 IP 走 MMDB 回退路径就少了省份，
	//   后台按 国家/省份/城市 聚合时会把同一批用户拆成两组）
	return formatMMDBLocation(record)
}

// formatMMDBLocation 把 MMDB 查询结果格式化成与 ip2region 一致的地区串。
// 单独抽出来是为了能直接测：不必真的加载 60MB 的库文件就能验证输出形状。
func formatMMDBLocation(record mmdbCityRecord) string {
	country := pickLocalizedName(record.Country.Names)
	province := ""
	if len(record.Subdivisions) > 0 {
		province = pickLocalizedName(record.Subdivisions[0].Names)
	}
	city := pickLocalizedName(record.City.Names)
	return joinLocationParts(country, province, city)
}

// pickLocalizedName 优先取中文名，其次英文名（两个离线库都按此规则）。
func pickLocalizedName(names map[string]string) string {
	if names == nil {
		return ""
	}
	if v := names["zh-CN"]; v != "" {
		return v
	}
	return names["en"]
}

// GetIPLocation returns a location string for the given IP address.
// Uses only local offline databases (ip2region + MMDB) for zero network latency.
func GetIPLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return LocationLocal
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ""
	}

	if isPrivateIP(ip) {
		return LocationPrivate
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
		location = LocationUnknown
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
