package services

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Pre-compiled regexps (avoid recompiling on every ParseUserAgent call)
var (
	reIPhoneOS      = regexp.MustCompile(`(?i)iPhone\s*OS\s+([\d_]+)`)
	reAndroidVer    = regexp.MustCompile(`(?i)Android\s+([\d.]+)`)
	reIPhoneModel   = regexp.MustCompile(`(?i)iPhone(\d+,\d+)`)
	reIPadModel     = regexp.MustCompile(`(?i)iPad(\d+,\d+)`)
	reIPhoneProMax  = regexp.MustCompile(`iPhone\s+(\d+)\s+Pro\s+Max`)
	reIPhonePro     = regexp.MustCompile(`iPhone\s+(\d+)\s+Pro`)
	reIPhoneMini    = regexp.MustCompile(`iPhone\s+(\d+)\s+mini`)
	reIPhoneNum     = regexp.MustCompile(`iPhone\s+(\d+)`)
	reAndroidBuild  = regexp.MustCompile(`;\s*([^;]+)\s*[Bb]uild`)
	reVersionPrefix = regexp.MustCompile(`^([\d.]+)`)
)

// ClientInfo holds parsed User-Agent information
type ClientInfo struct {
	SoftwareName    string
	SoftwareVersion string
	OSName          string
	OSVersion       string
	DeviceModel     string
	DeviceBrand     string
	DeviceType      string // mobile, desktop, tablet, unknown
	IsBrowser       bool
	SubscriptionType string // clash, surge, shadowrocket, quantumult, v2ray
}

var proxyKeywords = []string{
	"clash", "clashandroid", "clashx", "clash-verge", "clash for windows",
	"clash for android", "clash for mac", "hiddify", "loon",
	"quantumult", "quantumult x", "qv2ray", "shadowrocket", "shadowsocks",
	"shadowsocksr", "ssr", "surfboard", "surge", "v2ray", "v2rayn",
	"v2rayng", "v2rayu", "v2rayx", "stash", "anx", "anxray", "kitsunebi",
	"pharos", "potatso", "karing", "neko", "nekoray", "nekobox", "sing-box",
}

var browserKeywords = []string{
	"mozilla", "chrome", "safari", "firefox", "edge", "opera",
	"msie", "trident", "webkit", "gecko", "browser",
}

// ParseUserAgent parses a User-Agent string to identify the client software, OS, and device
func ParseUserAgent(ua string) *ClientInfo {
	info := &ClientInfo{
		SoftwareName:     "Unknown",
		OSName:           "Unknown",
		DeviceType:       "unknown",
		SubscriptionType: "v2ray",
	}
	if ua == "" {
		return info
	}

	lower := strings.ToLower(ua)
	detectSoftware(lower, ua, info)
	detectOS(lower, ua, info)
	if info.OSName == "Unknown" {
		inferOSFromSoftware(info)
	}
	detectDevice(lower, ua, info)
	info.DeviceType = determineDeviceType(lower, info)
	info.SubscriptionType = determineSubscriptionType(info)
	info.IsBrowser = isBrowserRequest(lower)
	return info
}

func detectSoftware(lower, ua string, info *ClientInfo) {
	rules := []struct {
		keyword string
		name    string
	}{
		{"shadowrocket", "Shadowrocket"},
		{"quantumult%20x", "Quantumult X"},
		{"quantumult x", "Quantumult X"},
		{"quantumult", "Quantumult"},
		{"surge", "Surge"},
		{"loon", "Loon"},
		{"stash", "Stash"},
		{"flclash", "FlClash"},
		{"clash.party", "Clash Party"},
		{"clash-verge", "Clash Verge"},
		{"clash for windows", "Clash for Windows"},
		{"clash for android", "Clash for Android"},
		{"clashandroid", "Clash for Android"},
		{"clashx pro", "ClashX Pro"},
		{"clashx", "ClashX"},
		{"clash for mac", "Clash for Mac"},
		{"mihomo.party", "Mihomo Party"},
		{"mihomo/", "Mihomo"},
		{"mihomo", "Mihomo"},
		{"clash", "Clash"},
		{"hiddify", "Hiddify"},
		{"v2rayn", "v2rayN"},
		{"v2rayng", "v2rayNG"},
		{"v2rayu", "V2RayU"},
		{"v2rayx", "V2RayX"},
		{"v2ray", "V2Ray"},
		{"sing-box", "sing-box"},
		{"nekoray", "NekoRay"},
		{"nekobox", "NekoBox"},
		{"karing", "Karing"},
		{"surfboard", "Surfboard"},
		{"pharos", "Pharos"},
		{"potatso", "Potatso"},
		{"kitsunebi", "Kitsunebi"},
	}
	for _, r := range rules {
		if strings.Contains(lower, r.keyword) {
			info.SoftwareName = r.name
			info.SoftwareVersion = extractVersion(ua, r.keyword)
			return
		}
	}
	// iOS proxy app: CFNetwork + Darwin + iPhone/iPad without Mozilla
	if (strings.Contains(lower, "cfnetwork") || strings.Contains(lower, "darwin")) &&
		(strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad")) &&
		!strings.Contains(lower, "mozilla") {
		info.SoftwareName = "Shadowrocket"
	}
}

func detectOS(lower, ua string, info *ClientInfo) {
	switch {
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		info.OSName = "iOS"
		if m := reIPhoneOS.FindStringSubmatch(ua); len(m) > 1 {
			info.OSVersion = strings.ReplaceAll(m[1], "_", ".")
		}
	case strings.Contains(lower, "android"):
		info.OSName = "Android"
		if m := reAndroidVer.FindStringSubmatch(ua); len(m) > 1 {
			info.OSVersion = m[1]
		}
	case strings.Contains(lower, "windows"):
		info.OSName = "Windows"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os") || strings.Contains(lower, "darwin"):
		info.OSName = "macOS"
	case strings.Contains(lower, "linux"):
		info.OSName = "Linux"
	}
}

func inferOSFromSoftware(info *ClientInfo) {
	iosApps := map[string]bool{
		"Shadowrocket": true, "Quantumult": true, "Quantumult X": true,
		"Surge": true, "Loon": true, "Stash": true, "Pharos": true,
		"Potatso": true, "Kitsunebi": true, "Karing": true,
	}
	winApps := map[string]bool{
		"Clash for Windows": true, "Clash Verge": true, "v2rayN": true,
	}
	macApps := map[string]bool{
		"ClashX": true, "ClashX Pro": true, "Clash for Mac": true, "V2RayU": true, "V2RayX": true,
	}
	androidApps := map[string]bool{
		"Clash for Android": true, "v2rayNG": true, "Surfboard": true,
	}
	switch {
	case iosApps[info.SoftwareName]:
		info.OSName = "iOS"
	case winApps[info.SoftwareName]:
		info.OSName = "Windows"
	case macApps[info.SoftwareName]:
		info.OSName = "macOS"
	case androidApps[info.SoftwareName]:
		info.OSName = "Android"
	}
}

// iphoneModelMap maps internal iPhone identifiers to friendly model names
var iphoneModelMap = map[string]string{
	// iPhone 12 series
	"iPhone13,1": "iPhone 12 mini",
	"iPhone13,2": "iPhone 12",
	"iPhone13,3": "iPhone 12 Pro",
	"iPhone13,4": "iPhone 12 Pro Max",
	// iPhone 13 series
	"iPhone14,2": "iPhone 13 Pro",
	"iPhone14,3": "iPhone 13 Pro Max",
	"iPhone14,4": "iPhone 13 mini",
	"iPhone14,5": "iPhone 13",
	// iPhone SE 3rd gen
	"iPhone14,6": "iPhone SE (3rd)",
	// iPhone 14 series
	"iPhone14,7": "iPhone 14",
	"iPhone14,8": "iPhone 14 Plus",
	"iPhone15,2": "iPhone 14 Pro",
	"iPhone15,3": "iPhone 14 Pro Max",
	"iPhone15,4": "iPhone 15",
	"iPhone15,5": "iPhone 15 Plus",
	// iPhone 15 series
	"iPhone16,1": "iPhone 15 Pro",
	"iPhone16,2": "iPhone 15 Pro Max",
	// iPhone 16 series
	"iPhone17,1": "iPhone 16 Pro",
	"iPhone17,2": "iPhone 16 Pro Max",
	"iPhone17,3": "iPhone 16",
	"iPhone17,4": "iPhone 16 Plus",
	"iPhone17,5": "iPhone 16e",
}

func detectDevice(lower, ua string, info *ClientInfo) {
	// iPhone model
	if m := reIPhoneModel.FindStringSubmatch(ua); len(m) > 1 {
		modelID := "iPhone" + m[1]
		if name, ok := iphoneModelMap[modelID]; ok {
			info.DeviceModel = name
		} else {
			info.DeviceModel = "iPhone " + strings.Replace(m[1], ",", ".", -1)
		}
		info.DeviceBrand = "Apple"
	} else if m := reIPadModel.FindStringSubmatch(ua); len(m) > 1 {
		info.DeviceModel = "iPad " + strings.Replace(m[1], ",", ".", -1)
		info.DeviceBrand = "Apple"
	} else if strings.Contains(lower, "iphone") {
		if m := reIPhoneProMax.FindStringSubmatch(ua); len(m) > 1 {
			info.DeviceModel = "iPhone " + m[1] + " Pro Max"
		} else if m := reIPhonePro.FindStringSubmatch(ua); len(m) > 1 {
			info.DeviceModel = "iPhone " + m[1] + " Pro"
		} else if m := reIPhoneMini.FindStringSubmatch(ua); len(m) > 1 {
			info.DeviceModel = "iPhone " + m[1] + " mini"
		} else if m := reIPhoneNum.FindStringSubmatch(ua); len(m) > 1 {
			info.DeviceModel = "iPhone " + m[1]
		} else {
			info.DeviceModel = "iPhone"
		}
		info.DeviceBrand = "Apple"
	} else if strings.Contains(lower, "ipad") {
		info.DeviceModel = "iPad"
		info.DeviceBrand = "Apple"
	}

	// Android model: "Build/..." pattern + brand detection
	if info.DeviceModel == "" {
		if m := reAndroidBuild.FindStringSubmatch(ua); len(m) > 1 {
			info.DeviceModel = strings.TrimSpace(m[1])
			nameLower := strings.ToLower(info.DeviceModel)
			brands := []struct {
				brand    string
				keywords []string
			}{
				{"Samsung", []string{"samsung", "galaxy", "sm-"}},
				{"Huawei", []string{"huawei", "honor"}},
				{"Xiaomi", []string{"xiaomi", "redmi", "mi ", "poco"}},
				{"OPPO", []string{"oppo", "oneplus", "realme"}},
				{"vivo", []string{"vivo", "iqoo"}},
				{"Google", []string{"pixel"}},
				{"Sony", []string{"sony", "xperia"}},
				{"LG", []string{"lg-", "lg "}},
				{"Motorola", []string{"motorola", "moto"}},
			}
			for _, b := range brands {
				for _, kw := range b.keywords {
					if strings.Contains(nameLower, kw) {
						info.DeviceBrand = b.brand
						return
					}
				}
			}
			info.DeviceBrand = "Android"
		}
	}
}

func determineDeviceType(lower string, info *ClientInfo) string {
	switch {
	case strings.Contains(lower, "ipad") || strings.Contains(lower, "tablet"):
		return "tablet"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "android") ||
		info.OSName == "iOS" || info.OSName == "Android":
		return "mobile"
	case info.OSName == "Windows" || info.OSName == "macOS" || info.OSName == "Linux":
		return "desktop"
	default:
		return "unknown"
	}
}

func determineSubscriptionType(info *ClientInfo) string {
	clashApps := map[string]bool{
		"Clash": true, "Clash for Windows": true, "Clash for Android": true,
		"Clash for Mac": true, "ClashX": true, "ClashX Pro": true,
		"Clash Verge": true, "Stash": true, "Hiddify": true,
		"FlClash": true, "Clash Party": true, "Mihomo": true, "Mihomo Party": true,
	}
	if clashApps[info.SoftwareName] {
		return "clash"
	}
	switch info.SoftwareName {
	case "Surge":
		return "surge"
	case "Shadowrocket":
		return "shadowrocket"
	case "Quantumult", "Quantumult X":
		return "quantumult"
	case "Loon":
		return "loon"
	case "Surfboard":
		return "clash" // Surfboard supports Clash format
	default:
		return "v2ray"
	}
}

func isBrowserRequest(lower string) bool {
	// Check proxy keywords first — proxy clients take priority
	for _, kw := range proxyKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}
	// iOS proxy app pattern
	if (strings.Contains(lower, "cfnetwork") || strings.Contains(lower, "darwin")) &&
		(strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad")) &&
		!strings.Contains(lower, "mozilla") {
		return false
	}
	// Check browser keywords
	for _, kw := range browserKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func extractVersion(ua, keyword string) string {
	lower := strings.ToLower(ua)
	idx := strings.Index(lower, keyword)
	if idx < 0 {
		return ""
	}
	rest := ua[idx+len(keyword):]
	rest = strings.TrimLeft(rest, "/ ")
	if m := reVersionPrefix.FindString(rest); m != "" {
		return m
	}
	return ""
}

// GenerateDeviceFingerprint creates a stable device fingerprint based on parsed UA features
// This is more stable than SHA256(UA+IP) because it ignores version changes
func GenerateDeviceFingerprint(ua, ip string) string {
	info := ParseUserAgent(ua)

	var features []string
	if info.SoftwareName != "Unknown" {
		features = append(features, "software:"+info.SoftwareName)
	}
	if info.OSName != "Unknown" {
		features = append(features, "os:"+info.OSName)
	}
	if info.OSVersion != "" {
		features = append(features, "os_version:"+info.OSVersion)
	}
	if info.DeviceModel != "" {
		features = append(features, "model:"+info.DeviceModel)
	}
	if info.DeviceBrand != "" {
		features = append(features, "brand:"+info.DeviceBrand)
	}

	// If too few features, fall back to UA-based hash
	if len(features) < 2 {
		return fmt.Sprintf("%x", sha256.Sum256([]byte(ua)))
	}

	sort.Strings(features)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(features, "|"))))
}
