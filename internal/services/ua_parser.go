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
	reDarwin        = regexp.MustCompile(`(?i)darwin/(\d+)\.(\d+)`)
	reVersionPrefix = regexp.MustCompile(`^([\d.]+)`)
)

// ClientInfo holds parsed User-Agent information
type ClientInfo struct {
	SoftwareName     string
	SoftwareVersion  string
	OSName           string
	OSVersion        string
	DeviceModel      string
	DeviceBrand      string
	DeviceType       string // mobile, desktop, tablet, unknown
	IsBrowser        bool
	SubscriptionType string // clash, surge, shadowrocket, quantumult, v2ray
}

var proxyKeywords = []string{
	"clash", "clashandroid", "clashx", "clash-verge", "clash for windows",
	"clash for android", "clash for mac", "hiddify", "loon",
	"quantumult", "quantumult x", "qv2ray", "shadowrocket", "shadowsocks",
	"shadowsocksr", "ssr", "surfboard", "surge", "v2ray", "v2rayn",
	"v2rayng", "v2rayu", "v2rayx", "stash", "anx", "anxray", "kitsunebi",
	"pharos", "potatso", "karing", "neko", "nekoray", "nekobox", "sing-box",
	"mbox",
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

	// Dalvik = Android 的运行时。原生 App 用 HttpClient 发请求时默认就带
	// "Dalvik/2.1.0 (Linux; U; Android 14; …)"，里面没有任何客户端名。
	// 这是**真实设备**，只是软件名未知：平台能确定 → 识别成 Android，
	// 而不是丢进 Unknown（线上大量"未知设备"就是这种 UA）。
	// 注意：Dalvik/x.y.z 的数字是 Android 运行时版本，不是客户端版本，
	// 不能写进 SoftwareVersion（否则「客户端版本」维度凭空多出 2.1.0）。
	trimmed := strings.TrimSpace(lower)
	if strings.HasPrefix(trimmed, "dalvik/") || trimmed == "android" {
		info.OSName = "Android"
		if m := reAndroidVer.FindStringSubmatch(ua); len(m) > 1 {
			info.OSVersion = m[1]
		}
		info.DeviceType = "mobile"
		info.SubscriptionType = determineSubscriptionType(info)
		return info
	}

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
		// ⚠️ 顺序即优先级：自有客户端的 UA 里同时带内核标识
		// （`Mclash/0.0.7 platform/macos mihomo/1.19.31`），必须排在
		// mihomo/clash 之前，否则设备会被识别成 "Mihomo"（内核名）——
		// 网站就认不出这是哪个客户端了（实测库里确实出现了 Mihomo 行）。
		// 具体优先（对齐 myweb，减少 Unknown 与内核名误认）
		{"clashx meta", "ClashX Meta"},
		{"v2box", "V2Box"},
		{"happ/", "Happ"},
		{"nexvpn", "NexVPN"},
		{"meta/nextin", "Nextin"},
		{"tomoonclash", "ToMoon Clash"},
		{"axxray", "ANXRay"},
		{"cmfandroid", "Clash Meta"},
		{"clash.meta", "Clash Meta"},
		{"clash-meta", "Clash Meta"},
		{"clashmetaforandroid", "Clash Meta"},
		{"clash meta for android", "Clash Meta"},
		{"clash meta", "Clash Meta"},
		{"v2guard", "V2Guard"},
		{"passwall", "PassWall"},
		{"hiddifynext", "Hiddify"},
		{"hiddify-next", "Hiddify"},
		{"hiddify next", "Hiddify"},
		{"mclash", "Mclash"},
		{"moneyfly", "MoneyFly"},
		// MBox（sing-box 内核的自有客户端）：UA 形如
		// `MBox/0.0.1 (Macintosh; Mac OS X 26_6)`。必须排在内核标识之前 ——
		// 它的 UA 里可能带 sing-box/mihomo 之类的内核串，排在后面就会被
		// 识别成内核名（和当年 MoneyFly 被认成 Mihomo 是同一个坑）。
		{"mbox", "MBox"},
		{"clashmi", "ClashMi"},
		{"shadowrocket", "Shadowrocket"},
		{"quantumult%20x", "Quantumult X"},
		{"quantumult x", "Quantumult X"},
		{"quantumult", "Quantumult"},
		{"surge", "Surge"},
		{"loon", "Loon"},
		{"stash", "Stash"},
		{"flclash", "FlClash"},
		{"clashparty", "Clash Party"},
		{"clash.party", "Clash Party"},
		{"clash party", "Clash Party"},
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
		{"xray", "Xray"},
		{"sing-box", "sing-box"},
		{"nekoray", "NekoRay"},
		{"nekobox", "NekoBox"},
		{"karing", "Karing"},
		{"surfboard", "Surfboard"},
		{"pharos", "Pharos"},
		{"potatso", "Potatso"},
		{"kitsunebi", "Kitsunebi"},
		{"shadowsocks", "Shadowsocks"},
		{"v2rayu", "V2RayU"},
		{"qv2ray", "Qv2ray"},
		{"anxray", "AnXray"},
		{"matsuri", "Matsuri"},
		{"sagernet", "SagerNet"},
	}
	for _, r := range rules {
		if strings.Contains(lower, r.keyword) {
			info.SoftwareName = r.name
			info.SoftwareVersion = extractVersion(ua, r.keyword)
			return
		}
	}
	// iOS proxy app
	if (strings.Contains(lower, "cfnetwork") || strings.Contains(lower, "darwin")) &&
		(strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad")) &&
		!strings.Contains(lower, "mozilla") {
		info.SoftwareName = "iOS 代理客户端"
		return
	}
	// Browser detection
	if strings.Contains(lower, "mozilla") || strings.Contains(lower, "chrome") ||
		strings.Contains(lower, "safari") || strings.Contains(lower, "firefox") {
		if strings.Contains(lower, "edg/") {
			info.SoftwareName = "Edge 浏览器"
		} else if strings.Contains(lower, "chrome") && !strings.Contains(lower, "edg") {
			info.SoftwareName = "Chrome 浏览器"
		} else if strings.Contains(lower, "firefox") {
			info.SoftwareName = "Firefox 浏览器"
		} else if strings.Contains(lower, "safari") && !strings.Contains(lower, "chrome") {
			info.SoftwareName = "Safari 浏览器"
		} else {
			info.SoftwareName = "浏览器"
		}
		return
	}
}

func detectOS(lower, ua string, info *ClientInfo) {
	// 自有客户端（Mclash / MoneyFly）用 `platform/<os>` 明确标注平台，
	// 标准 UA 关键词（windows / mac os x / android…）里一个都没有，
	// 不先认这个标记就会得到 OSName=Unknown → 设备类型 unknown、
	// 特征不足导致指纹退化成整个 UA 的哈希（版本一变就多出一台"新设备"）。
	switch {
	case strings.Contains(lower, "platform/windows"):
		info.OSName = "Windows"
		return
	case strings.Contains(lower, "platform/macos") || strings.Contains(lower, "platform/mac"):
		info.OSName = "macOS"
		return
	case strings.Contains(lower, "platform/android"):
		info.OSName = "Android"
		return
	case strings.Contains(lower, "platform/ios"):
		info.OSName = "iOS"
		return
	case strings.Contains(lower, "platform/linux"):
		info.OSName = "Linux"
		return
	}

	switch {
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		info.OSName = "iOS"
		if m := reIPhoneOS.FindStringSubmatch(ua); len(m) > 1 {
			info.OSVersion = strings.ReplaceAll(m[1], "_", ".")
		}
		// iOS 客户端 UA 常只带 "CFNetwork/… Darwin/27.0.0"（没有 iPhone OS 段），
		// 于是「系统版本」列长期空白。Darwin 与 iOS 版本号并不一一对应
		// （Darwin 24 = iOS 18、Darwin 25 = iOS 26），如实标注成 "Darwin 27.0"，
		// 让这一列不再是空的，也不误导。
		if info.OSVersion == "" {
			if m := reDarwin.FindStringSubmatch(ua); len(m) > 2 {
				info.OSVersion = "Darwin " + m[1] + "." + m[2]
			}
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
	macOnlyApps := map[string]bool{
		"ClashX": true, "ClashX Pro": true, "Clash for Mac": true, "V2RayU": true, "V2RayX": true,
	}
	winOnlyApps := map[string]bool{
		"v2rayN": true,
	}
	androidApps := map[string]bool{
		"Clash for Android": true, "v2rayNG": true, "Surfboard": true,
	}
	// 多平台客户端：默认 Windows，但会被 browser_os 参数覆盖
	multiPlatformApps := map[string]bool{
		"Clash for Windows": true, "Clash Verge": true, "FlClash": true,
		"Mihomo Party": true, "Clash Party": true, "Hiddify": true,
		"Clash": true, "Mihomo": true,
	}

	switch {
	case iosApps[info.SoftwareName]:
		info.OSName = "iOS"
	case winOnlyApps[info.SoftwareName]:
		info.OSName = "Windows"
	case macOnlyApps[info.SoftwareName]:
		info.OSName = "macOS"
	case androidApps[info.SoftwareName]:
		info.OSName = "Android"
	case multiPlatformApps[info.SoftwareName]:
		// 默认 Windows，会被 browser_os 参数覆盖
		info.OSName = "Windows"
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
	// 社交/IM 的链接预览爬虫：用户把订阅链接发进聊天窗口，由服务端来抓一次。
	// 它们不是设备（没有 OS、没有硬件指纹），若登记会白占名额，
	// 所以归入「浏览器访问」——v2 的订阅入口对 IsBrowser 不登记设备。
	previewKeywords := []string{
		"telegrambot", "whatsapp/", "链接预览", "facebookexternalhit",
		"slackbot", "discordbot", "preview",
	}
	for _, kw := range previewKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
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

	// 只使用**稳定特征**：客户端名 + 系统 + 硬件型号。
	// 刻意不含 os_version：系统升级（macOS 15 → 26.6）不代表换了设备，
	// 否则同一台机器会凭空多出一台。客户端版本号同理不参与。
	var features []string
	if info.SoftwareName != "Unknown" {
		features = append(features, "software:"+info.SoftwareName)
	}
	if info.OSName != "Unknown" {
		features = append(features, "os:"+info.OSName)
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
