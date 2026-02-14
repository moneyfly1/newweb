package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"cboard/v2/internal/models"
)

// FetchSubscriptionContent fetches and base64-decodes subscription content from a URL.
func FetchSubscriptionContent(urlStr string) (string, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
		} else if strings.HasPrefix(line, "ss://") {
			node, err = ParseShadowsocksLink(line)
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

func DetectRegion(name string) string {
	lower := strings.ToLower(name)

	regionMap := map[string][]string{
		"香港":  {"香港", "hk", "hong kong", "🇭🇰"},
		"美国":  {"美国", "us", "usa", "united states", "🇺🇸"},
		"日本":  {"日本", "jp", "japan", "🇯🇵"},
		"新加坡": {"新加坡", "sg", "singapore", "🇸🇬"},
		"台湾":  {"台湾", "tw", "taiwan", "🇹🇼"},
		"韩国":  {"韩国", "kr", "korea", "🇰🇷"},
		"英国":  {"英国", "uk", "united kingdom", "🇬🇧"},
		"德国":  {"德国", "de", "germany", "🇩🇪"},
		"法国":  {"法国", "fr", "france", "🇫🇷"},
		"加拿大": {"加拿大", "ca", "canada", "🇨🇦"},
		"澳大利亚": {"澳大利亚", "au", "australia", "🇦🇺"},
		"俄罗斯": {"俄罗斯", "ru", "russia", "🇷🇺"},
		"印度":  {"印度", "in", "india", "🇮🇳"},
	}

	for region, keywords := range regionMap {
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				return region
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
	default:
		return nil, fmt.Errorf("unsupported type: %s", nodeType)
	}
}

// GenerateClashYAML generates a proper Clash YAML config from nodes.
func GenerateClashYAML(nodes []models.Node) string {
	return GenerateClashYAMLWithDomain(nodes, "")
}

// GenerateClashYAMLWithDomain generates Clash YAML with full proxy-groups and rules.
func GenerateClashYAMLWithDomain(nodes []models.Node, siteDomain string) string {
	var proxies []map[string]interface{}
	var proxyNames []string
	// Separate info nodes (fake SS) from real proxy nodes
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

		// Info nodes use baidu.com:1234 as server
		if server, ok := m["server"].(string); ok && server == "baidu.com" {
			infoNames = append(infoNames, name)
		}
	}

	// Real proxy names (exclude info nodes) for auto-select groups
	var realNames []string
	infoSet := make(map[string]bool)
	for _, n := range infoNames {
		infoSet[n] = true
	}
	for _, n := range proxyNames {
		if !infoSet[n] {
			realNames = append(realNames, n)
		}
	}

	var sb strings.Builder

	// Head: basic config + DNS
	sb.WriteString("port: 7890\n")
	sb.WriteString("socks-port: 7891\n")
	sb.WriteString("allow-lan: true\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n")
	sb.WriteString("external-controller: :9090\n\n")
	sb.WriteString("dns:\n")
	sb.WriteString("  enable: true\n")
	sb.WriteString("  nameserver:\n")
	sb.WriteString("    - 119.29.29.29\n")
	sb.WriteString("    - 223.5.5.5\n")
	sb.WriteString("  fallback:\n")
	sb.WriteString("    - 8.8.8.8\n")
	sb.WriteString("    - 8.8.4.4\n\n")

	// Proxies
	sb.WriteString("proxies:\n")
	for _, p := range proxies {
		writeClashProxy(&sb, p)
	}

	// Proxy groups
	sb.WriteString("\nproxy-groups:\n")

	grpSelect := "\U0001F680 节点选择"
	grpAuto := "\u267B\uFE0F 自动选择"
	grpDirect := "\U0001F3AF 全球直连"
	grpBlock := "\U0001F6D1 全球拦截"
	grpFallback := "\U0001F41F 漏网之鱼"

	// 🚀 节点选择 - manual select, includes auto-select + DIRECT + all real nodes
	sb.WriteString("  - name: " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")
	for _, name := range realNames {
		sb.WriteString("      - ")
		sb.WriteString(escapeYAML(name))
		sb.WriteString("\n")
	}

	// ♻️ 自动选择 - url-test with all real nodes
	sb.WriteString("  - name: " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("    type: url-test\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    tolerance: 50\n")
	sb.WriteString("    proxies:\n")
	for _, name := range realNames {
		sb.WriteString("      - ")
		sb.WriteString(escapeYAML(name))
		sb.WriteString("\n")
	}

	// 🎯 全球直连
	sb.WriteString("  - name: " + escapeYAML(grpDirect) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")

	// 🛑 全球拦截
	sb.WriteString("  - name: " + escapeYAML(grpBlock) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - REJECT\n")
	sb.WriteString("      - DIRECT\n")

	// 🐟 漏网之鱼
	sb.WriteString("  - name: " + escapeYAML(grpFallback) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpDirect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")

	// Rules
	sb.WriteString("\nrules:\n")
	if siteDomain != "" {
		// Strip protocol prefix for domain rule
		d := siteDomain
		for _, prefix := range []string{"https://", "http://"} {
			d = strings.TrimPrefix(d, prefix)
		}
		d = strings.TrimRight(d, "/")
		sb.WriteString("  - DOMAIN-SUFFIX," + d + "," + grpDirect + "\n")
	}
	sb.WriteString("  - IP-CIDR,127.0.0.0/8," + grpDirect + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,172.16.0.0/12," + grpDirect + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,192.168.0.0/16," + grpDirect + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,10.0.0.0/8," + grpDirect + ",no-resolve\n")
	sb.WriteString("  - GEOIP,CN," + grpDirect + "\n")
	sb.WriteString("  - MATCH," + grpFallback + "\n")

	return sb.String()
}

// GenerateUniversalBase64 generates base64-encoded links for all nodes.
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
	orderedKeys := []string{"name", "type", "server", "port", "uuid", "alterId", "cipher", "password", "flow", "network", "tls", "servername", "sni", "client-fingerprint", "skip-cert-verify", "udp"}
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
	idx := strings.LastIndex(hostport, ":")
	if idx < 0 {
		return hostport, ""
	}
	return hostport[:idx], hostport[idx+1:]
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		n, _ := strconv.Atoi(val)
		return n
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}
