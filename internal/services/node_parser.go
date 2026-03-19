package services

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/models"

	"gopkg.in/yaml.v3"
)

const (
	maxResponseSize = 10 * 1024 * 1024 // 10MB limit for subscription content
)

// FetchSubscriptionContent fetches and base64-decodes subscription content from a URL.
func FetchSubscriptionContent(urlStr string) (string, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes to prevent SSRF
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// Prevent access to private IP ranges
	if parsedURL.Hostname() != "" {
		if ips, err := net.LookupIP(parsedURL.Hostname()); err == nil {
			for _, ip := range ips {
				if isPrivateIP(ip) {
					return "", fmt.Errorf("access to private IP addresses is not allowed")
				}
			}
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ClashForAndroid/2.5.12")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Limit response size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	content := string(body)
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content))
	if err == nil {
		content = string(decoded)
	}
	return content, nil
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	// Additional checks for special ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8, 169.254.0.0/16, 224.0.0.0/4
		return ip4[0] == 0 || (ip4[0] == 169 && ip4[1] == 254) || ip4[0] >= 224
	}
	return false
}

// ParseNodeLinks parses multi-line node links into Node models.
func ParseNodeLinks(content string) ([]models.Node, error) {
	lines := strings.Split(content, "\n")
	var nodes []models.Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var node *models.Node
		var err error

		if strings.HasPrefix(line, "vmess://") {
			node, err = ParseVmessLink(line)
		} else if strings.HasPrefix(line, "vless://") {
			node, err = ParseVlessLink(line)
		} else if strings.HasPrefix(line, "trojan://") {
			node, err = ParseTrojanLink(line)
		} else if strings.HasPrefix(line, "ssr://") {
			node, err = ParseSSRLink(line)
		} else if strings.HasPrefix(line, "ss://") {
			node, err = ParseShadowsocksLink(line)
		} else if strings.HasPrefix(line, "hysteria2://") || strings.HasPrefix(line, "hy2://") {
			node, err = ParseHysteria2Link(line)
		} else if strings.HasPrefix(line, "hysteria://") {
			node, err = ParseHysteriaLink(line)
		} else if strings.HasPrefix(line, "tuic://") {
			node, err = ParseTUICLink(line)
		} else if strings.HasPrefix(line, "naive+https://") || strings.HasPrefix(line, "naive://") {
			node, err = ParseNaiveLink(line)
		} else if strings.HasPrefix(line, "anytls://") {
			node, err = ParseAnytlsLink(line)
		} else if strings.HasPrefix(line, "socks5://") || strings.HasPrefix(line, "socks://") {
			node, err = ParseSOCKSLink(line)
		}

		if err == nil && node != nil {
			nodes = append(nodes, *node)
		}
	}

	return nodes, nil
}

func ParseVmessLink(link string) (*models.Node, error) {
	encoded := strings.TrimPrefix(link, "vmess://")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
	}

	var vmessConfig map[string]interface{}
	if err := json.Unmarshal(decoded, &vmessConfig); err != nil {
		return nil, err
	}

	name := ""
	if ps, ok := vmessConfig["ps"].(string); ok {
		name = ps
	}
	if name == "" {
		name = "VMess Node"
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "vmess",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

func ParseVlessLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	name := u.Fragment
	if name == "" {
		name = "VLESS Node"
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "vless",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

func ParseTrojanLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	name := u.Fragment
	if name == "" {
		name = "Trojan Node"
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "trojan",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

func ParseShadowsocksLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	name := u.Fragment
	if name == "" {
		name = "Shadowsocks Node"
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "ss",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

// ParseSSRLink parses an ssr:// link into a Node model.
func ParseSSRLink(link string) (*models.Node, error) {
	encoded := strings.TrimPrefix(link, "ssr://")
	// SSR links are base64 encoded
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
	}

	// Format: host:port:protocol:method:obfs:base64(password)/?params
	mainAndParams := strings.SplitN(string(decoded), "/?", 2)
	parts := strings.SplitN(mainAndParams[0], ":", 6)
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid ssr link format")
	}

	name := "SSR Node"
	if len(mainAndParams) > 1 {
		params, _ := url.ParseQuery(mainAndParams[1])
		if remarks := params.Get("remarks"); remarks != "" {
			remarksDecoded, err := base64.RawURLEncoding.DecodeString(remarks)
			if err == nil {
				name = string(remarksDecoded)
			}
		}
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "ssr",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

// ParseHysteriaLink parses a hysteria:// link into a Node model.
func ParseHysteriaLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	name := u.Fragment
	if name == "" {
		name = "Hysteria Node"
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "hysteria",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

// ParseHysteria2Link parses a hysteria2:// or hy2:// link into a Node model.
func ParseHysteria2Link(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	name := u.Fragment
	if name == "" {
		name = "Hysteria2 Node"
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "hysteria2",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

// ParseTUICLink parses a tuic:// link into a Node model.
func ParseTUICLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	name := u.Fragment
	if name == "" {
		name = "TUIC Node"
	}

	region := DetectRegion(name)
	config := link

	return &models.Node{
		Name:     name,
		Region:   region,
		Type:     "tuic",
		Status:   "online",
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}, nil
}

// ParseNaiveLink parses naive:// or naive+https:// links
func ParseNaiveLink(link string) (*models.Node, error) {
	// Normalize to https:// for URL parsing
	normalized := link
	for _, prefix := range []string{"naive+https://", "naive://"} {
		if strings.HasPrefix(normalized, prefix) {
			normalized = "https://" + strings.TrimPrefix(normalized, prefix)
			break
		}
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return nil, err
	}
	name := ""
	if u.Fragment != "" {
		name, _ = url.QueryUnescape(u.Fragment)
	}
	if name == "" {
		name = u.Hostname()
	}
	region := DetectRegion(name)
	config := link
	return &models.Node{
		Name:     name,
		Type:     "naive",
		Region:   region,
		Status:   "online",
		Config:   &config,
		IsActive: true,
	}, nil
}

// ParseAnytlsLink parses anytls:// links
func ParseAnytlsLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	name := ""
	if u.Fragment != "" {
		name, _ = url.QueryUnescape(u.Fragment)
	}
	if name == "" {
		name = u.Hostname()
	}
	region := DetectRegion(name)
	config := link
	return &models.Node{
		Name:     name,
		Type:     "anytls",
		Region:   region,
		Status:   "online",
		Config:   &config,
		IsActive: true,
	}, nil
}

// ParseSOCKSLink parses socks5:// and socks:// links
func ParseSOCKSLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	name := ""
	if u.Fragment != "" {
		name, _ = url.QueryUnescape(u.Fragment)
	}
	if name == "" {
		name = u.Hostname()
	}
	nodeType := "socks5"
	if strings.HasPrefix(link, "socks://") {
		nodeType = "socks"
	}
	region := DetectRegion(name)
	config := link
	return &models.Node{
		Name:     name,
		Type:     nodeType,
		Region:   region,
		Status:   "online",
		Config:   &config,
		IsActive: true,
	}, nil
}

// ParseHTTPLink parses http:// and https:// proxy links
func ParseHTTPLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	name := ""
	if u.Fragment != "" {
		name, _ = url.QueryUnescape(u.Fragment)
	}
	if name == "" {
		name = u.Hostname()
	}
	region := DetectRegion(name)
	config := link
	return &models.Node{
		Name:     name,
		Type:     "http",
		Region:   region,
		Status:   "online",
		Config:   &config,
		IsActive: true,
	}, nil
}

func portToInt(port string) int {
	p, err := strconv.Atoi(port)
	if err != nil {
		return 0
	}
	return p
}

func DetectRegion(name string) string {
	lower := strings.ToLower(name)

	// Use word boundary matching to avoid false positives
	// Check for exact matches or matches with word boundaries
	regionMap := map[string][]string{
		"香港":    {"香港", " hk ", " hk-", "-hk-", "-hk ", "hong kong", "hongkong", "🇭🇰"},
		"美国":    {"美国", " us ", " us-", "-us-", "-us ", "usa", "united states", "america", "🇺🇸"},
		"日本":    {"日本", " jp ", " jp-", "-jp-", "-jp ", "japan", "tokyo", "🇯🇵"},
		"新加坡":   {"新加坡", " sg ", " sg-", "-sg-", "-sg ", "singapore", "🇸🇬"},
		"台湾":    {"台湾", " tw ", " tw-", "-tw-", "-tw ", "taiwan", "🇹🇼"},
		"韩国":    {"韩国", " kr ", " kr-", "-kr-", "-kr ", "korea", "seoul", "🇰🇷"},
		"英国":    {"英国", " uk ", " uk-", "-uk-", "-uk ", "united kingdom", "london", "🇬🇧"},
		"德国":    {"德国", " de ", " de-", "-de-", "-de ", "germany", "🇩🇪"},
		"法国":    {"法国", " fr ", " fr-", "-fr-", "-fr ", "france", "🇫🇷"},
		"加拿大":   {"加拿大", " ca ", " ca-", "-ca-", "-ca ", "canada", "🇨🇦"},
		"澳大利亚":  {"澳大利亚", "澳", " au ", " au-", "-au-", "-au ", "australia", "🇦🇺"},
		"俄罗斯":   {"俄罗斯", " ru ", " ru-", "-ru-", "-ru ", "russia", "🇷🇺"},
		"印度":    {"印度", " in ", " in-", "-in-", "-in ", "india", "🇮🇳"},
		"马来西亚":  {"马来西亚", "大马", " my ", " my-", "-my-", "-my ", "malaysia", "🇲🇾"},
		"菲律宾":   {"菲律宾", " ph ", " ph-", "-ph-", "-ph ", "philippines", "🇵🇭"},
		"柬埔寨":   {"柬埔寨", " kh ", " kh-", "-kh-", "-kh ", "cambodia", "🇰🇭"},
		"越南":    {"越南", " vn ", " vn-", "-vn-", "-vn ", "vietnam", "🇻🇳"},
		"泰国":    {"泰国", " th ", " th-", "-th-", "-th ", "thailand", "🇹🇭"},
		"印度尼西亚": {"印度尼西亚", "印尼", " id ", " id-", "-id-", "-id ", "indonesia", "🇮🇩"},
		"土耳其":   {"土耳其", " tr ", " tr-", "-tr-", "-tr ", "turkey", "🇹🇷"},
		"巴西":    {"巴西", " br ", " br-", "-br-", "-br ", "brazil", "🇧🇷"},
		"荷兰":    {"荷兰", " nl ", " nl-", "-nl-", "-nl ", "netherlands", "🇳🇱"},
		"意大利":   {"意大利", " it ", " it-", "-it-", "-it ", "italy", "🇮🇹"},
		"西班牙":   {"西班牙", " es ", " es-", "-es-", "-es ", "spain", "🇪🇸"},
		"瑞士":    {"瑞士", " ch ", " ch-", "-ch-", "-ch ", "switzerland", "🇨🇭"},
		"瑞典":    {"瑞典", " se ", " se-", "-se-", "-se ", "sweden", "🇸🇪"},
		"波兰":    {"波兰", " pl ", " pl-", "-pl-", "-pl ", "poland", "🇵🇱"},
		"阿联酋":   {"阿联酋", " ae ", " ae-", "-ae-", "-ae ", "uae", "🇦🇪"},
		"新西兰":   {"新西兰", " nz ", " nz-", "-nz-", "-nz ", "new zealand", "🇳🇿"},
		"南非":    {"南非", " za ", " za-", "-za-", "-za ", "south africa", "🇿🇦"},
		"爱尔兰":   {"爱尔兰", " ie ", " ie-", "-ie-", "-ie ", "ireland", "🇮🇪"},
		"墨西哥":   {"墨西哥", " mx ", " mx-", "-mx-", "-mx ", "mexico", "🇲🇽"},
		"阿根廷":   {"阿根廷", " ar ", " ar-", "-ar-", "-ar ", "argentina", "🇦🇷"},
		"哥伦比亚":  {"哥伦比亚", " co ", " co-", "-co-", "-co ", "colombia", "🇨🇴"},
		"智利":    {"智利", " cl ", " cl-", "-cl-", "-cl ", "chile", "🇨🇱"},
		"埃及":    {"埃及", " eg ", " eg-", "-eg-", "-eg ", "egypt", "🇪🇬"},
		"以色列":   {"以色列", " il ", " il-", "-il-", "-il ", "israel", "🇮🇱"},
		"乌克兰":   {"乌克兰", " ua ", " ua-", "-ua-", "-ua ", "ukraine", "🇺🇦"},
		"罗马尼亚":  {"罗马尼亚", " ro ", " ro-", "-ro-", "-ro ", "romania", "🇷🇴"},
		"匈牙利":   {"匈牙利", " hu ", " hu-", "-hu-", "-hu ", "hungary", "🇭🇺"},
		"捷克":    {"捷克", " cz ", " cz-", "-cz-", "-cz ", "czech", "🇨🇿"},
		"希腊":    {"希腊", " gr ", " gr-", "-gr-", "-gr ", "greece", "🇬🇷"},
		"葡萄牙":   {"葡萄牙", " pt ", " pt-", "-pt-", "-pt ", "portugal", "🇵🇹"},
		"芬兰":    {"芬兰", " fi ", " fi-", "-fi-", "-fi ", "finland", "🇫🇮"},
		"挪威":    {"挪威", " no ", " no-", "-no-", "-no ", "norway", "🇳🇴"},
		"丹麦":    {"丹麦", " dk ", " dk-", "-dk-", "-dk ", "denmark", "🇩🇰"},
		"奥地利":   {"奥地利", " at ", " at-", "-at-", "-at ", "austria", "🇦🇹"},
		"比利时":   {"比利时", " be ", " be-", "-be-", "-be ", "belgium", "🇧🇪"},
		"缅甸":    {"缅甸", " mm ", " mm-", "-mm-", "-mm ", "myanmar", "🇲🇲"},
		"老挝":    {"老挝", " la ", " la-", "-la-", "-la ", "laos", "🇱🇦"},
		"巴基斯坦":  {"巴基斯坦", " pk ", " pk-", "-pk-", "-pk ", "pakistan", "🇵🇰"},
		"孟加拉":   {"孟加拉", " bd ", " bd-", "-bd-", "-bd ", "bangladesh", "🇧🇩"},
		"蒙古":    {"蒙古", " mn ", " mn-", "-mn-", "-mn ", "mongolia", "🇲🇳"},
		"哈萨克斯坦": {"哈萨克斯坦", " kz ", " kz-", "-kz-", "-kz ", "kazakhstan", "🇰🇿"},
	}

	// Add spaces around the name for boundary matching
	paddedLower := " " + lower + " "

	for region, keywords := range regionMap {
		for _, keyword := range keywords {
			// For keywords with spaces (word boundaries), check in padded string
			if strings.HasPrefix(keyword, " ") || strings.HasSuffix(keyword, " ") {
				if strings.Contains(paddedLower, keyword) {
					return region
				}
			} else {
				// For other keywords, use regular contains
				if strings.Contains(lower, keyword) {
					return region
				}
			}
		}
	}

	return "其他"
}

// VmessLinkToClashMap parses a vmess:// link into a Clash-compatible proxy map.
func VmessLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	encoded := strings.TrimPrefix(link, "vmess://")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		return nil, err
	}

	m := map[string]interface{}{
		"name":    name,
		"type":    "vmess",
		"server":  fmt.Sprintf("%v", cfg["add"]),
		"port":    toInt(cfg["port"]),
		"uuid":    fmt.Sprintf("%v", cfg["id"]),
		"alterId": toInt(cfg["aid"]),
		"cipher":  "auto",
	}
	if net, ok := cfg["net"].(string); ok && net != "" {
		obfsType, _ := cfg["type"].(string)
		if net == "tcp" && obfsType == "http" {
			m["network"] = "http"
			httpOpts := map[string]interface{}{
				"method": "GET",
			}
			if path, ok := cfg["path"].(string); ok && path != "" {
				httpOpts["path"] = []string{path}
			} else {
				httpOpts["path"] = []string{"/"}
			}
			if host, ok := cfg["host"].(string); ok && host != "" {
				httpOpts["headers"] = map[string]interface{}{
					"Host": []string{host},
				}
			}
			m["http-opts"] = httpOpts
		} else {
			m["network"] = net
			if net == "ws" {
				wsOpts := map[string]interface{}{}
				if path, ok := cfg["path"].(string); ok && path != "" {
					wsOpts["path"] = path
				}
				if host, ok := cfg["host"].(string); ok && host != "" {
					wsOpts["headers"] = map[string]interface{}{"Host": host}
				}
				if len(wsOpts) > 0 {
					m["ws-opts"] = wsOpts
				}
			} else if net == "grpc" {
				if path, ok := cfg["path"].(string); ok && path != "" {
					m["grpc-opts"] = map[string]interface{}{"grpc-service-name": path}
				}
			} else if net == "h2" {
				h2Opts := map[string]interface{}{}
				if path, ok := cfg["path"].(string); ok && path != "" {
					h2Opts["path"] = path
				}
				if host, ok := cfg["host"].(string); ok && host != "" {
					h2Opts["host"] = []string{host}
				}
				if len(h2Opts) > 0 {
					m["h2-opts"] = h2Opts
				}
			}
		}
	}
	if tls, ok := cfg["tls"].(string); ok && tls == "tls" {
		m["tls"] = true
		if sni, ok := cfg["sni"].(string); ok && sni != "" {
			m["servername"] = sni
		} else if host, ok := cfg["host"].(string); ok && host != "" {
			m["servername"] = host
		}
	}
	return m, nil
}

// VlessLinkToClashMap parses a vless:// link into a Clash-compatible proxy map.
func VlessLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	m := map[string]interface{}{
		"name":   name,
		"type":   "vless",
		"server": host,
		"port":   port,
		"uuid":   u.User.Username(),
	}
	if t := q.Get("type"); t != "" {
		m["network"] = t
		if t == "ws" {
			wsOpts := map[string]interface{}{}
			if p := q.Get("path"); p != "" {
				wsOpts["path"] = p
			}
			if h := q.Get("host"); h != "" {
				wsOpts["headers"] = map[string]interface{}{"Host": h}
			}
			if len(wsOpts) > 0 {
				m["ws-opts"] = wsOpts
			}
		} else if t == "grpc" {
			if sn := q.Get("serviceName"); sn != "" {
				m["grpc-opts"] = map[string]interface{}{"grpc-service-name": sn}
			}
		}
	}
	sec := q.Get("security")
	if sec == "tls" || sec == "reality" {
		m["tls"] = true
		if sni := q.Get("sni"); sni != "" {
			m["servername"] = sni
		}
		if sec == "reality" {
			m["reality-opts"] = map[string]interface{}{
				"public-key": q.Get("pbk"),
				"short-id":   q.Get("sid"),
			}
			if fp := q.Get("fp"); fp != "" {
				m["client-fingerprint"] = fp
			}
		}
	}
	if flow := q.Get("flow"); flow != "" {
		m["flow"] = flow
	}
	return m, nil
}

// TrojanLinkToClashMap parses a trojan:// link into a Clash-compatible proxy map.
func TrojanLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	m := map[string]interface{}{
		"name":     name,
		"type":     "trojan",
		"server":   host,
		"port":     port,
		"password": u.User.Username(),
	}
	if sni := q.Get("sni"); sni != "" {
		m["sni"] = sni
	}
	if t := q.Get("type"); t != "" && t != "tcp" {
		m["network"] = t
		if t == "ws" {
			wsOpts := map[string]interface{}{}
			if p := q.Get("path"); p != "" {
				wsOpts["path"] = p
			}
			if h := q.Get("host"); h != "" {
				wsOpts["headers"] = map[string]interface{}{"Host": h}
			}
			if len(wsOpts) > 0 {
				m["ws-opts"] = wsOpts
			}
		}
	}
	if q.Get("allowInsecure") == "1" || q.Get("insecure") == "1" {
		m["skip-cert-verify"] = true
	}
	return m, nil
}

// ShadowsocksLinkToClashMap parses an ss:// link into a Clash-compatible proxy map.
func ShadowsocksLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	var cipher, password string
	userInfo := u.User.Username()
	// ss:// can encode method:password in base64 as the userinfo
	decoded, err := base64.StdEncoding.DecodeString(userInfo)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(userInfo)
	}
	if err == nil && strings.Contains(string(decoded), ":") {
		parts := strings.SplitN(string(decoded), ":", 2)
		cipher = parts[0]
		password = parts[1]
	} else {
		cipher = userInfo
		if p, ok := u.User.Password(); ok {
			password = p
		}
	}
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	m := map[string]interface{}{
		"name":     name,
		"type":     "ss",
		"server":   host,
		"port":     port,
		"cipher":   cipher,
		"password": password,
	}
	return m, nil
}

// SSRLinkToClashMap parses an ssr:// link into a Clash-compatible proxy map.
func SSRLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	encoded := strings.TrimPrefix(link, "ssr://")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
	}

	mainAndParams := strings.SplitN(string(decoded), "/?", 2)
	parts := strings.SplitN(mainAndParams[0], ":", 6)
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid ssr link format")
	}

	host := parts[0]
	port, _ := strconv.Atoi(parts[1])
	protocol := parts[2]
	method := parts[3]
	obfs := parts[4]
	passwordB64 := parts[5]

	passwordBytes, err := base64.RawURLEncoding.DecodeString(passwordB64)
	if err != nil {
		passwordBytes, _ = base64.StdEncoding.DecodeString(passwordB64)
	}
	password := string(passwordBytes)

	m := map[string]interface{}{
		"name":     name,
		"type":     "ssr",
		"server":   host,
		"port":     port,
		"cipher":   method,
		"password": password,
		"protocol": protocol,
		"obfs":     obfs,
	}

	if len(mainAndParams) > 1 {
		params, _ := url.ParseQuery(mainAndParams[1])
		if pp := params.Get("protoparam"); pp != "" {
			ppDecoded, err := base64.RawURLEncoding.DecodeString(pp)
			if err == nil {
				m["protocol-param"] = string(ppDecoded)
			}
		}
		if op := params.Get("obfsparam"); op != "" {
			opDecoded, err := base64.RawURLEncoding.DecodeString(op)
			if err == nil {
				m["obfs-param"] = string(opDecoded)
			}
		}
	}

	return m, nil
}

// HysteriaLinkToClashMap parses a hysteria:// link into a Clash-compatible proxy map.
func HysteriaLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	m := map[string]interface{}{
		"name":   name,
		"type":   "hysteria",
		"server": host,
		"port":   port,
	}
	if auth := q.Get("auth"); auth != "" {
		m["auth-str"] = auth
	}
	if peer := q.Get("peer"); peer != "" {
		m["sni"] = peer
	}
	if insecure := q.Get("insecure"); insecure == "1" {
		m["skip-cert-verify"] = true
	}
	if up := q.Get("upmbps"); up != "" {
		m["up"] = up
	}
	if down := q.Get("downmbps"); down != "" {
		m["down"] = down
	}
	if proto := q.Get("protocol"); proto != "" {
		m["protocol"] = proto
	}
	return m, nil
}

// Hysteria2LinkToClashMap parses a hysteria2:// or hy2:// link into a Clash-compatible proxy map.
func Hysteria2LinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	password := ""
	if u.User != nil {
		password = u.User.Username()
	}

	m := map[string]interface{}{
		"name":     name,
		"type":     "hysteria2",
		"server":   host,
		"port":     port,
		"password": password,
	}
	if sni := q.Get("sni"); sni != "" {
		m["sni"] = sni
	}
	if insecure := q.Get("insecure"); insecure == "1" {
		m["skip-cert-verify"] = true
	}
	return m, nil
}

// TUICLinkToClashMap parses a tuic:// link into a Clash-compatible proxy map.
func TUICLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	uuid := ""
	password := ""
	if u.User != nil {
		uuid = u.User.Username()
		if p, ok := u.User.Password(); ok {
			password = p
		}
	}

	m := map[string]interface{}{
		"name":     name,
		"type":     "tuic",
		"server":   host,
		"port":     port,
		"uuid":     uuid,
		"password": password,
	}
	if cc := q.Get("congestion_control"); cc != "" {
		m["congestion-controller"] = cc
	}
	if alpn := q.Get("alpn"); alpn != "" {
		m["alpn"] = strings.Split(alpn, ",")
	}
	if sni := q.Get("sni"); sni != "" {
		m["sni"] = sni
	}
	return m, nil
}

// SOCKSLinkToClashMap parses a socks5:// or socks:// link into a Clash-compatible proxy map.
func SOCKSLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 1080
	}
	m := map[string]interface{}{
		"name":   name,
		"type":   "socks5",
		"server": host,
		"port":   port,
		"udp":    true,
	}
	if u.User != nil {
		m["username"] = u.User.Username()
		if pw, ok := u.User.Password(); ok {
			m["password"] = pw
		}
	}
	return m, nil
}

// HTTPLinkToClashMap parses an http:// or https:// proxy link into a Clash-compatible proxy map.
func HTTPLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		if strings.HasPrefix(link, "https://") {
			port = 443
		} else {
			port = 80
		}
	}
	m := map[string]interface{}{
		"name":   name,
		"type":   "http",
		"server": host,
		"port":   port,
	}
	if strings.HasPrefix(link, "https://") {
		m["tls"] = true
	}
	if u.User != nil {
		m["username"] = u.User.Username()
		if pw, ok := u.User.Password(); ok {
			m["password"] = pw
		}
	}
	return m, nil
}

// AnytlsLinkToClashMap parses an anytls:// link into a Clash-compatible proxy map.
func AnytlsLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	host, portStr := splitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 443
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
		if pw, ok := u.User.Password(); ok && pw != "" {
			password = pw
		}
	}
	sni := u.Query().Get("sni")
	if sni == "" {
		sni = host
	}
	m := map[string]interface{}{
		"name":               name,
		"type":               "anytls",
		"server":             host,
		"port":               port,
		"password":           password,
		"udp":                true,
		"client-fingerprint": "chrome",
		"sni":                sni,
	}
	return m, nil
}

// NodeConfigToClashMap converts a node's Config link to a Clash proxy map.
func NodeConfigToClashMap(nodeType string, configLink string, nodeName string) (map[string]interface{}, error) {
	switch nodeType {
	case "vmess":
		return VmessLinkToClashMap(configLink, nodeName)
	case "vless":
		return VlessLinkToClashMap(configLink, nodeName)
	case "trojan":
		return TrojanLinkToClashMap(configLink, nodeName)
	case "ss":
		return ShadowsocksLinkToClashMap(configLink, nodeName)
	case "ssr":
		return SSRLinkToClashMap(configLink, nodeName)
	case "hysteria":
		return HysteriaLinkToClashMap(configLink, nodeName)
	case "hysteria2":
		return Hysteria2LinkToClashMap(configLink, nodeName)
	case "tuic":
		return TUICLinkToClashMap(configLink, nodeName)
	case "socks5", "socks":
		return SOCKSLinkToClashMap(configLink, nodeName)
	case "http":
		return HTTPLinkToClashMap(configLink, nodeName)
	case "anytls":
		return AnytlsLinkToClashMap(configLink, nodeName)
	default:
		return nil, fmt.Errorf("unsupported type: %s", nodeType)
	}
}

// GenerateClashYAML generates a proper Clash YAML config from nodes.
func GenerateClashYAML(nodes []models.Node) string {
	return GenerateClashYAMLWithDomain(nodes, "", "")
}

// GenerateClashYAMLWithDomain generates Clash YAML using the template file (uploads/config/temp.yaml).
// subscriptionName is used for the YAML `name` field (e.g. "到期: 2026-03-15").
func GenerateClashYAMLWithDomain(nodes []models.Node, siteDomain string, subscriptionName string) string {
	var proxies []map[string]interface{}
	var proxyNames []string
	var infoNames []string
	usedNames := make(map[string]bool)

	for _, n := range nodes {
		if n.Config == nil || *n.Config == "" {
			continue
		}
		name := n.Name
		origName := name
		counter := 1
		for usedNames[name] {
			name = fmt.Sprintf("%s_%d", origName, counter)
			counter++
		}
		usedNames[name] = true

		m, err := NodeConfigToClashMap(n.Type, *n.Config, name)
		if err != nil {
			continue
		}
		proxies = append(proxies, m)
		proxyNames = append(proxyNames, name)

		if server, ok := m["server"].(string); ok && server == "baidu.com" {
			infoNames = append(infoNames, name)
		}
	}

	// Real proxy names (exclude info nodes) for auto-select groups
	infoSet := make(map[string]bool)
	for _, n := range infoNames {
		infoSet[n] = true
	}
	var realNames []string
	for _, n := range proxyNames {
		if !infoSet[n] {
			realNames = append(realNames, n)
		}
	}

	// Try template-based generation
	if result := generateFromTemplate(proxies, proxyNames, realNames, subscriptionName); result != "" {
		return result
	}

	// Fallback: generate default YAML
	return generateDefaultClashYAML(proxies, proxyNames, realNames, siteDomain, subscriptionName)
}

// generateFromTemplate loads uploads/config/temp.yaml and injects proxies + updates proxy-groups.
func generateFromTemplate(proxies []map[string]interface{}, allNames, realNames []string, subscriptionName string) string {
	data, err := os.ReadFile("uploads/config/temp.yaml")
	if err != nil {
		return ""
	}

	var templateConfig yaml.Node
	if err := yaml.Unmarshal(data, &templateConfig); err != nil {
		return ""
	}

	// templateConfig is a Document node; the actual mapping is its first child
	if templateConfig.Kind != yaml.DocumentNode || len(templateConfig.Content) == 0 {
		return ""
	}
	root := templateConfig.Content[0]
	if root.Kind != yaml.MappingNode {
		return ""
	}

	// Build proxies YAML using our ordered writer for deterministic output
	var proxiesSB strings.Builder
	for _, p := range proxies {
		writeClashProxy(&proxiesSB, p)
	}
	var proxiesNode yaml.Node
	if err := yaml.Unmarshal([]byte("proxies:\n"+proxiesSB.String()), &proxiesNode); err != nil {
		return ""
	}

	// Inject subscription name as YAML "name" field (used by Clash clients as profile display name)
	if subscriptionName != "" {
		nameFound := false
		for i := 0; i < len(root.Content)-1; i += 2 {
			if root.Content[i].Value == "name" {
				root.Content[i+1].Value = subscriptionName
				nameFound = true
				break
			}
		}
		if !nameFound {
			// Prepend name field to the root mapping
			root.Content = append([]*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: subscriptionName, Tag: "!!str"},
			}, root.Content...)
		}
	}

	// Walk the root mapping and update proxies + proxy-groups
	for i := 0; i < len(root.Content)-1; i += 2 {
		keyNode := root.Content[i]
		valNode := root.Content[i+1]

		if keyNode.Value == "proxies" {
			// Replace proxies value with our generated proxies
			if proxiesNode.Kind == yaml.DocumentNode && len(proxiesNode.Content) > 0 {
				mappingNode := proxiesNode.Content[0]
				if mappingNode.Kind == yaml.MappingNode && len(mappingNode.Content) >= 2 {
					*valNode = *mappingNode.Content[1]
				}
			}
		}

		if keyNode.Value == "proxy-groups" && valNode.Kind == yaml.SequenceNode {
			updateProxyGroupsYAML(valNode, allNames, realNames)
		}

		// 为 Sparkle 等客户端：模板中的 profile 增加自动更新间隔（小时）
		if keyNode.Value == "profile" && valNode.Kind == yaml.MappingNode {
			injectProfileUpdateInterval(valNode, 24)
		}
	}

	output, err := yaml.Marshal(&templateConfig)
	if err != nil {
		return ""
	}
	return unescapeUnicode(string(output))
}

// injectProfileUpdateInterval sets profile.update-interval (hours) for Clash/Sparkle 自动更新.
func injectProfileUpdateInterval(profileNode *yaml.Node, hours int) {
	if profileNode.Kind != yaml.MappingNode {
		return
	}
	val := strconv.Itoa(hours)
	for j := 0; j < len(profileNode.Content)-1; j += 2 {
		if profileNode.Content[j].Value == "update-interval" {
			profileNode.Content[j+1].Value = val
			return
		}
	}
	profileNode.Content = append(profileNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "update-interval", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"},
	)
}

// updateProxyGroupsYAML updates proxy-groups in the YAML node tree.
func updateProxyGroupsYAML(groupsNode *yaml.Node, allNames, realNames []string) {
	// Collect group names
	groupNames := make(map[string]bool)
	for _, g := range groupsNode.Content {
		if g.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j < len(g.Content)-1; j += 2 {
			if g.Content[j].Value == "name" {
				groupNames[g.Content[j+1].Value] = true
			}
		}
	}

	for _, g := range groupsNode.Content {
		if g.Kind != yaml.MappingNode {
			continue
		}
		var gType string
		var proxiesIdx int = -1
		for j := 0; j < len(g.Content)-1; j += 2 {
			if g.Content[j].Value == "type" {
				gType = g.Content[j+1].Value
			}
			if g.Content[j].Value == "proxies" {
				proxiesIdx = j + 1
			}
		}
		if proxiesIdx < 0 || (gType != "select" && gType != "url-test" && gType != "fallback" && gType != "load-balance") {
			continue
		}

		// Collect special entries (DIRECT, REJECT, group references)
		var specials []string
		oldVal := g.Content[proxiesIdx]
		if oldVal.Kind == yaml.SequenceNode {
			for _, item := range oldVal.Content {
				if item.Kind == yaml.ScalarNode {
					if item.Value == "DIRECT" || item.Value == "REJECT" || groupNames[item.Value] {
						specials = append(specials, item.Value)
					}
				}
			}
		}

		// Build new proxies list
		var newItems []*yaml.Node
		for _, s := range specials {
			newItems = append(newItems, &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"})
		}
		names := allNames
		if gType != "select" && len(realNames) > 0 {
			names = realNames
		}
		for _, n := range names {
			newItems = append(newItems, &yaml.Node{Kind: yaml.ScalarNode, Value: n, Tag: "!!str"})
		}

		g.Content[proxiesIdx] = &yaml.Node{
			Kind:    yaml.SequenceNode,
			Tag:     "!!seq",
			Content: newItems,
		}
	}
}

// unescapeUnicode converts \UXXXXXXXX and \uXXXX escape sequences back to actual Unicode characters.
func unescapeUnicode(s string) string {
	result := s
	// Handle \UXXXXXXXX (8-digit)
	for {
		idx := strings.Index(result, "\\U")
		if idx < 0 || idx+10 > len(result) {
			break
		}
		hexStr := result[idx+2 : idx+10]
		codePoint, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			// Not a valid escape, skip
			result = result[:idx] + "U" + result[idx+2:]
			continue
		}
		result = result[:idx] + string(rune(codePoint)) + result[idx+10:]
	}
	// Handle \uXXXX (4-digit)
	for {
		idx := strings.Index(result, "\\u")
		if idx < 0 || idx+6 > len(result) {
			break
		}
		hexStr := result[idx+2 : idx+6]
		codePoint, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			result = result[:idx] + "u" + result[idx+2:]
			continue
		}
		result = result[:idx] + string(rune(codePoint)) + result[idx+6:]
	}
	return result
}

// updateProxyGroups injects proxy names into each group, preserving special entries.
// generateDefaultClashYAML is the fallback when no template file exists.
func generateDefaultClashYAML(proxies []map[string]interface{}, allNames, realNames []string, siteDomain, subscriptionName string) string {
	var sb strings.Builder

	// When no real nodes exist, fall back to allNames to avoid empty proxy groups
	autoNames := realNames
	if len(autoNames) == 0 {
		autoNames = allNames
	}

	if subscriptionName != "" {
		sb.WriteString(fmt.Sprintf("name: %s\n", escapeYAML(subscriptionName)))
	}
	sb.WriteString("mixed-port: 7890\n")
	sb.WriteString("allow-lan: true\n")
	sb.WriteString("bind-address: '*'\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n")
	sb.WriteString("ipv6: false\n")
	sb.WriteString("external-controller: 127.0.0.1:9090\n")
	sb.WriteString("find-process-mode: always\n")
	sb.WriteString("unified-delay: true\n")
	sb.WriteString("tcp-concurrent: true\n")
	sb.WriteString("\n")
	sb.WriteString("profile:\n")
	sb.WriteString("  store-selected: true\n")
	sb.WriteString("  store-fake-ip: true\n")
	sb.WriteString("  update-interval: 24\n")
	sb.WriteString("\n")
	sb.WriteString("dns:\n")
	sb.WriteString("  enable: true\n")
	sb.WriteString("  listen: 0.0.0.0:1053\n")
	sb.WriteString("  ipv6: false\n")
	sb.WriteString("  enhanced-mode: fake-ip\n")
	sb.WriteString("  fake-ip-range: 198.18.0.1/16\n")
	sb.WriteString("  fake-ip-filter:\n")
	sb.WriteString("    - '*.lan'\n")
	sb.WriteString("    - '*.local'\n")
	sb.WriteString("    - localhost.ptlogin2.qq.com\n")
	sb.WriteString("    - '+.msftconnecttest.com'\n")
	sb.WriteString("    - '+.msftncsi.com'\n")
	sb.WriteString("  default-nameserver:\n")
	sb.WriteString("    - 223.5.5.5\n")
	sb.WriteString("    - 119.29.29.29\n")
	sb.WriteString("  nameserver:\n")
	sb.WriteString("    - https://dns.alidns.com/dns-query\n")
	sb.WriteString("    - https://doh.pub/dns-query\n")
	sb.WriteString("  fallback:\n")
	sb.WriteString("    - https://1.1.1.1/dns-query\n")
	sb.WriteString("    - https://dns.google/dns-query\n")
	sb.WriteString("  fallback-filter:\n")
	sb.WriteString("    geoip: true\n")
	sb.WriteString("    geoip-code: CN\n")
	sb.WriteString("\n")

	sb.WriteString("proxies:\n")
	for _, p := range proxies {
		writeClashProxy(&sb, p)
	}

	// 17 个代理组（与老项目 goweb 模板一致）
	grpSelect := "🚀 节点选择"
	grpAuto := "♻️ 自动选择"
	grpFallover := "🔰 故障转移"
	grpBalance := "🔮 负载均衡"
	grpDirect := "🎯 全球直连"
	grpBlock := "🛑 全球拦截"
	grpFish := "🐟 漏网之鱼"
	grpApple := "📱 苹果服务"
	grpMicrosoft := "🍎 微软服务"
	grpGoogle := "🔍 谷歌服务"
	grpTelegram := "📲 电报消息"
	grpOpenAI := "🤖 OpenAI"
	grpStreamIntl := "📺 国际流媒体"
	grpStreamCN := "📺 国内流媒体"
	grpForeign := "🌐 国外网站"
	grpChina := "🇨🇳 国内网站"
	grpLocal := "🏠 本地网络"

	sb.WriteString("\nproxy-groups:\n")

	// 1. 🚀 节点选择
	sb.WriteString("  - name: " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - " + escapeYAML(grpFallover) + "\n")
	sb.WriteString("      - " + escapeYAML(grpBalance) + "\n")
	sb.WriteString("      - DIRECT\n")
	for _, name := range allNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 2. ♻️ 自动选择
	sb.WriteString("  - name: " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("    type: url-test\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    tolerance: 50\n")
	sb.WriteString("    proxies:\n")
	for _, name := range autoNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 3. 🔰 故障转移
	sb.WriteString("  - name: " + escapeYAML(grpFallover) + "\n")
	sb.WriteString("    type: fallback\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    proxies:\n")
	for _, name := range autoNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 4. 🔮 负载均衡
	sb.WriteString("  - name: " + escapeYAML(grpBalance) + "\n")
	sb.WriteString("    type: load-balance\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    strategy: consistent-hashing\n")
	sb.WriteString("    proxies:\n")
	for _, name := range autoNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 5. 🎯 全球直连
	sb.WriteString("  - name: " + escapeYAML(grpDirect) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")

	// 6. 🛑 全球拦截
	sb.WriteString("  - name: " + escapeYAML(grpBlock) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - REJECT\n")
	sb.WriteString("      - DIRECT\n")

	// 7. 🐟 漏网之鱼
	sb.WriteString("  - name: " + escapeYAML(grpFish) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")

	// 8. 📱 苹果服务
	sb.WriteString("  - name: " + escapeYAML(grpApple) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 9. 🍎 微软服务
	sb.WriteString("  - name: " + escapeYAML(grpMicrosoft) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 10. 🔍 谷歌服务
	sb.WriteString("  - name: " + escapeYAML(grpGoogle) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 11. 📲 电报消息
	sb.WriteString("  - name: " + escapeYAML(grpTelegram) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 12. 🤖 OpenAI
	sb.WriteString("  - name: " + escapeYAML(grpOpenAI) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 13. 📺 国际流媒体
	sb.WriteString("  - name: " + escapeYAML(grpStreamIntl) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 14. 📺 国内流媒体
	sb.WriteString("  - name: " + escapeYAML(grpStreamCN) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")

	// 15. 🌐 国外网站
	sb.WriteString("  - name: " + escapeYAML(grpForeign) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 16. 🇨🇳 国内网站
	sb.WriteString("  - name: " + escapeYAML(grpChina) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")

	// 17. 🏠 本地网络
	sb.WriteString("  - name: " + escapeYAML(grpLocal) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")

	sb.WriteString("\nrules:\n")
	if siteDomain != "" {
		d := siteDomain
		for _, prefix := range []string{"https://", "http://"} {
			d = strings.TrimPrefix(d, prefix)
		}
		d = strings.TrimRight(d, "/")
		sb.WriteString("  - DOMAIN-SUFFIX," + d + "," + grpDirect + "\n")
	}
	sb.WriteString("  - DOMAIN-SUFFIX,local," + grpLocal + "\n")
	sb.WriteString("  - IP-CIDR,127.0.0.0/8," + grpLocal + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,172.16.0.0/12," + grpLocal + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,192.168.0.0/16," + grpLocal + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,10.0.0.0/8," + grpLocal + ",no-resolve\n")
	// 苹果
	sb.WriteString("  - DOMAIN-SUFFIX,apple.com," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,icloud.com," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,apple.news," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,apple.ae," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-KEYWORD,apple," + grpApple + "\n")
	// 微软
	sb.WriteString("  - DOMAIN-SUFFIX,microsoft.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,windows.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,live.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,office.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-KEYWORD,microsoft," + grpMicrosoft + "\n")
	// 谷歌
	sb.WriteString("  - DOMAIN-SUFFIX,google.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,gstatic.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,youtube.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,googleapis.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-KEYWORD,google," + grpGoogle + "\n")
	// 电报
	sb.WriteString("  - DOMAIN-SUFFIX,telegram.org," + grpTelegram + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,t.me," + grpTelegram + "\n")
	sb.WriteString("  - IP-CIDR,91.108.4.0/22," + grpTelegram + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,149.154.160.0/20," + grpTelegram + ",no-resolve\n")
	// OpenAI
	sb.WriteString("  - DOMAIN-SUFFIX,openai.com," + grpOpenAI + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,chatgpt.com," + grpOpenAI + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,ai.com," + grpOpenAI + "\n")
	// 国际流媒体
	sb.WriteString("  - DOMAIN-SUFFIX,netflix.com," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,netflix.net," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,disneyplus.com," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,hbo.com," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,spotify.com," + grpStreamIntl + "\n")
	// 国内流媒体
	sb.WriteString("  - DOMAIN-SUFFIX,iqiyi.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,bilibili.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,youku.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,tencentvideo.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,qq.com," + grpStreamCN + "\n")
	// 国内网站直连
	sb.WriteString("  - GEOIP,CN," + grpChina + "\n")
	// 国外网站
	sb.WriteString("  - GEOIP,!CN," + grpForeign + "\n")
	// 广告拦截
	sb.WriteString("  - DOMAIN-KEYWORD,adservice," + grpBlock + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,doubleclick.net," + grpBlock + "\n")
	sb.WriteString("  - MATCH," + grpFish + "\n")

	return sb.String()
}
func GenerateUniversalBase64(nodes []models.Node) string {
	var links []string
	for _, n := range nodes {
		if n.Config != nil && *n.Config != "" {
			links = append(links, strings.TrimSpace(*n.Config))
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n")))
}

func writeClashProxy(sb *strings.Builder, m map[string]interface{}) {
	sb.WriteString("  - ")
	// Write fields in a deterministic order
	orderedKeys := []string{"name", "type", "server", "port", "uuid", "alterId", "cipher", "username", "password", "flow", "network", "tls", "servername", "sni", "client-fingerprint", "skip-cert-verify", "udp", "protocol", "protocol-param", "obfs", "obfs-param", "auth-str", "up", "down", "congestion-controller", "alpn"}
	written := make(map[string]bool)

	first := true
	for _, key := range orderedKeys {
		val, ok := m[key]
		if !ok {
			continue
		}
		written[key] = true
		if first {
			sb.WriteString("{")
			first = false
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(escapeYAML(key))
		sb.WriteString(": ")
		writeYAMLInlineValue(sb, val)
	}

	// Write remaining keys sorted
	remaining := make([]string, 0)
	for k := range m {
		if !written[k] {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		if first {
			sb.WriteString("{")
			first = false
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(escapeYAML(key))
		sb.WriteString(": ")
		writeYAMLInlineValue(sb, m[key])
	}
	sb.WriteString("}\n")
}

func writeYAMLInlineValue(sb *strings.Builder, val interface{}) {
	switch v := val.(type) {
	case string:
		sb.WriteString(escapeYAML(v))
	case int:
		sb.WriteString(strconv.Itoa(v))
	case float64:
		if v == float64(int(v)) {
			sb.WriteString(strconv.Itoa(int(v)))
		} else {
			sb.WriteString(fmt.Sprintf("%g", v))
		}
	case bool:
		if v {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case map[string]interface{}:
		sb.WriteString("{")
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(escapeYAML(k))
			sb.WriteString(": ")
			writeYAMLInlineValue(sb, v[k])
		}
		sb.WriteString("}")
	case []interface{}:
		sb.WriteString("[")
		for i, item := range v {
			if i > 0 {
				sb.WriteString(", ")
			}
			writeYAMLInlineValue(sb, item)
		}
		sb.WriteString("]")
	case []string:
		sb.WriteString("[")
		for i, item := range v {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(escapeYAML(item))
		}
		sb.WriteString("]")
	default:
		sb.WriteString(fmt.Sprintf("%v", val))
	}
}

func escapeYAML(s string) string {
	if s == "" {
		return "\"\""
	}
	needsQuotes := false
	special := ":\"'#@&*?|>!%`[]{}, \n\r\t"
	for _, c := range special {
		if strings.ContainsRune(s, c) {
			needsQuotes = true
			break
		}
	}
	if needsQuotes {
		escaped := strings.ReplaceAll(s, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return "\"" + escaped + "\""
	}
	return s
}

func splitHostPort(hostport string) (string, string) {
	// Use net.SplitHostPort for proper IPv6 support
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		// If parsing fails, assume no port and return the whole string as host
		return hostport, ""
	}
	return host, port
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0
		}
		return n
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0
		}
		return int(n)
	default:
		return 0
	}
}
