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

func loadMMDBReader() *maxminddb.Reader {
	mmdbOnce.Do(func() {
		candidates := []string{
			filepath.Join("uploads", "config", "geoip.metadb"),
			filepath.Join("uploads", "config", "Country.mmdb"),
			filepath.Join("uploads", "config", "GeoLite2-City.mmdb"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err != nil {
				continue
			}
			reader, err := maxminddb.Open(candidate)
			if err == nil {
				mmdbReader = reader
				return
			}
		}
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
		return ""
	}
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

	// Try MMDB first
	if location := lookupLocationFromMMDB(ip); location != "" {
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
	location := queryIPAPI(ip)
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
