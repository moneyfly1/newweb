package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"cboard/v2/internal/models"
	"gopkg.in/yaml.v3"
)

func ParseNodeLinks(content string) ([]models.Node, error) {
	lines := strings.Split(content, "\n")
	var nodes []models.Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		node, err := parseNodeFromLine(line)
		if err == nil && node != nil {
			nodes = append(nodes, *node)
		}
	}

	return nodes, nil
}


type clashSubscription struct {
	Proxies []map[string]interface{} `yaml:"proxies"`
}

// ParseSubscriptionContent parses either Clash YAML subscriptions, JSON node lists, or traditional node links.

func ParseSubscriptionContent(content string) ([]models.Node, error) {
	content = normalizeSubscriptionContent(content)
	if content == "" {
		return nil, nil
	}

	// Try JSON extraction (e.g. {"data":[{"vmessLink":"..."}]})
	if len(content) > 0 && (content[0] == '{' || content[0] == '[') {
		if nodes, ok := parseJSONNodeList(content); ok {
			return nodes, nil
		}
	}

	if nodes, ok, err := parseClashSubscription(content); ok {
		return nodes, err
	}

	return ParseNodeLinks(content)
}

// parseJSONNodeList recursively walks any JSON structure and collects proxy links.

func parseJSONNodeList(content string) ([]models.Node, bool) {
	var raw interface{}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, false
	}
	links := extractProxyLinksFromJSON(raw)
	if len(links) == 0 {
		return nil, false
	}
	var nodes []models.Node
	seen := make(map[string]bool)
	for _, link := range links {
		if seen[link] {
			continue
		}
		seen[link] = true
		node, err := parseNodeFromLine(link)
		if err == nil && node != nil {
			nodes = append(nodes, *node)
		}
	}
	return nodes, len(nodes) > 0
}


var proxyLinkPrefixes = []string{
	"vmess://", "vless://", "trojan://", "ss://", "ssr://",
	"hysteria://", "hysteria2://", "tuic://", "naive+", "anytls://", "wireguard://",
}


func extractProxyLinksFromJSON(v interface{}) []string {
	var links []string
	switch val := v.(type) {
	case string:
		s := strings.TrimSpace(val)
		for _, prefix := range proxyLinkPrefixes {
			if strings.HasPrefix(s, prefix) {
				links = append(links, s)
				break
			}
		}
	case []interface{}:
		for _, item := range val {
			links = append(links, extractProxyLinksFromJSON(item)...)
		}
	case map[string]interface{}:
		for _, item := range val {
			links = append(links, extractProxyLinksFromJSON(item)...)
		}
	}
	return links
}


func ExtractDomainPortFromNodeLink(link string) (string, int, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", 0, fmt.Errorf("empty link")
	}

	switch {
	case strings.HasPrefix(link, "vmess://"):
		return extractDomainPortFromVmessLink(link)
	case strings.HasPrefix(link, "ssr://"):
		proxy, err := SSRLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "ss://"):
		proxy, err := ShadowsocksLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "vless://"):
		proxy, err := VlessLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "trojan://"):
		proxy, err := TrojanLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "hysteria2://"), strings.HasPrefix(link, "hy2://"):
		proxy, err := Hysteria2LinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "hysteria://"):
		proxy, err := HysteriaLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "tuic://"):
		proxy, err := TUICLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "naive+https://"), strings.HasPrefix(link, "naive://"):
		proxy, err := HTTPLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "socks5://"), strings.HasPrefix(link, "socks://"):
		proxy, err := SOCKSLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "http://"), strings.HasPrefix(link, "https://"):
		proxy, err := HTTPLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "wg://"):
		proxy, err := WireGuardLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	case strings.HasPrefix(link, "anytls://"):
		proxy, err := AnytlsLinkToClashMap(link, "")
		if err != nil {
			return "", 0, err
		}
		return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
	default:
		u, err := url.Parse(link)
		if err != nil {
			return "", 0, err
		}
		host, portStr := splitHostPort(u.Host)
		port := parsePortWithDefault(portStr, 0)
		if host == "" || port <= 0 {
			return "", 0, fmt.Errorf("unable to extract domain or port")
		}
		return host, port, nil
	}
}


func extractDomainPortFromVmessLink(link string) (string, int, error) {
	encoded := strings.TrimPrefix(link, "vmess://")
	encodedBase64 := encoded
	if idx := strings.Index(encodedBase64, "?"); idx != -1 {
		encodedBase64 = encodedBase64[:idx]
	}
	if idx := strings.Index(encodedBase64, "#"); idx != -1 {
		encodedBase64 = encodedBase64[:idx]
	}

	decoded, err := base64.StdEncoding.DecodeString(encodedBase64)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encodedBase64)
		if err != nil {
			decoded, err = base64.RawURLEncoding.DecodeString(encodedBase64)
			if err != nil {
				return "", 0, err
			}
		}
	}

	var vmessConfig map[string]interface{}
	if err := json.Unmarshal(decoded, &vmessConfig); err == nil {
		host := stringFromMap(vmessConfig, "add")
		port := intFromMap(vmessConfig, "port", 0)
		if host == "" || port <= 0 {
			return "", 0, fmt.Errorf("invalid vmess config")
		}
		return host, port, nil
	}

	proxy, err := VmessLinkToClashMap(link, "")
	if err != nil {
		return "", 0, err
	}
	return stringFromMap(proxy, "server"), intFromMap(proxy, "port", 0), nil
}


func parseClashSubscription(content string) ([]models.Node, bool, error) {
	var sub clashSubscription
	if err := yaml.Unmarshal([]byte(content), &sub); err != nil {
		return nil, false, nil
	}
	if len(sub.Proxies) == 0 {
		return nil, false, nil
	}

	var nodes []models.Node
	for _, proxy := range sub.Proxies {
		node, err := clashProxyToNode(proxy)
		if err != nil || node == nil {
			continue
		}
		nodes = append(nodes, *node)
	}
	return nodes, true, nil
}


func clashProxyToNode(proxy map[string]interface{}) (*models.Node, error) {
	nodeType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", proxy["type"])))
	name := strings.TrimSpace(fmt.Sprintf("%v", proxy["name"]))
	if nodeType == "" {
		return nil, fmt.Errorf("missing proxy type")
	}

	link, normalizedType, err := clashProxyToLink(proxy, nodeType, name)
	if err != nil {
		return nil, err
	}
	return buildNode(name, strings.ToUpper(normalizedType)+" Node", normalizedType, link), nil
}


type nodeLinkRule struct {
	prefixes  []string
	parser    func(string) (*models.Node, error)
	condition func(string) bool
}


var nodeLinkRules = []nodeLinkRule{
	{prefixes: []string{"vmess://"}, parser: ParseVmessLink},
	{prefixes: []string{"vless://"}, parser: ParseVlessLink},
	{prefixes: []string{"trojan://"}, parser: ParseTrojanLink},
	{prefixes: []string{"ssr://"}, parser: ParseSSRLink},
	{prefixes: []string{"ss://"}, parser: ParseShadowsocksLink},
	{prefixes: []string{"hysteria2://", "hy2://"}, parser: ParseHysteria2Link},
	{prefixes: []string{"hysteria://"}, parser: ParseHysteriaLink},
	{prefixes: []string{"tuic://"}, parser: ParseTUICLink},
	{prefixes: []string{"naive+https://", "naive://"}, parser: ParseNaiveLink},
	{prefixes: []string{"anytls://"}, parser: ParseAnytlsLink},
	{prefixes: []string{"socks5://", "socks://"}, parser: ParseSOCKSLink},
	{prefixes: []string{"wg://"}, parser: ParseWireGuardLink},
	{
		prefixes:  []string{"http://", "https://"},
		parser:    ParseHTTPLink,
		condition: isLikelyHTTPProxyLink,
	},
}


func parseNodeFromLine(line string) (*models.Node, error) {
	for _, rule := range nodeLinkRules {
		if !hasAnyPrefix(line, rule.prefixes) {
			continue
		}
		if rule.condition != nil && !rule.condition(line) {
			return nil, nil
		}
		return rule.parser(line)
	}
	return nil, nil
}


func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}


func isLikelyHTTPProxyLink(link string) bool {
	return strings.Contains(link, "method=") || strings.Contains(link, "remarks=") || strings.Contains(link, "#")
}


func buildNode(name string, defaultName string, nodeType string, link string) *models.Node {
	resolvedName := sanitizeNodeName(name)
	if resolvedName == "" {
		resolvedName = defaultName
	}
	region := DetectRegion(resolvedName)
	config := link
	return &models.Node{
		Name:     resolvedName,
		Region:   region,
		Type:     nodeType,
		Status:   models.NodeStatusOnline,
		Config:   &config,
		IsActive: true,
		IsManual: false,
	}
}


func sanitizeNodeName(name string) string {
	cleaned := strings.TrimSpace(name)
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}


func decodeFragment(fragment string) string {
	if fragment == "" {
		return ""
	}
	if decoded, err := url.QueryUnescape(fragment); err == nil {
		return decoded
	}
	return fragment
}


func parseQuerySafe(raw string) url.Values {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return url.Values{}
	}
	return values
}


func parseIntOrDefault(raw string, defaultVal int) int {
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return n
}


func parsePortWithDefault(portStr string, defaultPort int) int {
	port := parseIntOrDefault(portStr, 0)
	if port <= 0 {
		return defaultPort
	}
	return port
}


func boolFromMap(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	switch value := v.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true") || value == "1"
	case int:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	default:
		return false
	}
}


func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return ""
}


func intFromMap(m map[string]interface{}, key string, defaultVal int) int {
	v, ok := m[key]
	if !ok {
		return defaultVal
	}
	switch value := v.(type) {
	case int:
		if value > 0 {
			return value
		}
	case int64:
		if value > 0 {
			return int(value)
		}
	case float64:
		if value > 0 {
			return int(value)
		}
	case string:
		if n, err := strconv.Atoi(value); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}


func stringSliceFromValue(v interface{}) []string {
	switch value := v.(type) {
	case []string:
		return value
	case []interface{}:
		res := make([]string, 0, len(value))
		for _, item := range value {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				res = append(res, s)
			}
		}
		return res
	case string:
		if value == "" {
			return nil
		}
		return []string{value}
	default:
		return nil
	}
}


func encodeNameFragment(name string) string {
	if name == "" {
		return ""
	}
	return "#" + url.QueryEscape(name)
}


func encodeHostPortQuery(host string, path string) string {
	if host == "" && path == "" {
		return ""
	}
	q := url.Values{}
	if host != "" {
		q.Set("host", host)
	}
	if path != "" {
		q.Set("path", path)
	}
	return q.Encode()
}

