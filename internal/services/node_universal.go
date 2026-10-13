package services

import (
	"cboard/v2/internal/models"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

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
	m = normalizeClashProxyMap(m)
	sb.WriteString("  - ")
	// Write fields in a deterministic order
	orderedKeys := []string{"name", "server", "port", "type", "uuid", "alterId", "cipher", "username", "password", "tls", "tfo", "flow", "skip-cert-verify", "servername", "sni", "client-fingerprint", "network", "udp", "protocol", "protocol-param", "obfs", "obfs-param", "auth-str", "up", "down", "congestion-controller", "alpn"}
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
		writeYAMLInlineValueForKey(sb, key, val)
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
		writeYAMLInlineValueForKey(sb, key, m[key])
	}
	sb.WriteString("}\n")
}

func normalizeClashProxyMap(m map[string]interface{}) map[string]interface{} {
	needsSkipCertVerify := shouldClashSkipCertVerify(m)
	needsTFO := shouldClashDisableTFO(m)
	if !needsSkipCertVerify && !needsTFO {
		return m
	}
	clone := make(map[string]interface{}, len(m)+1)
	for k, v := range m {
		clone[k] = v
	}
	if needsSkipCertVerify {
		clone["skip-cert-verify"] = true
	}
	if needsTFO {
		clone["tfo"] = false
	}
	return clone
}

func shouldClashSkipCertVerify(m map[string]interface{}) bool {
	typ, _ := m["type"].(string)
	switch typ {
	case "vless", "vmess", "trojan", "hysteria", "hysteria2":
		tls, _ := m["tls"].(bool)
		return tls || typ == "trojan" || typ == "hysteria" || typ == "hysteria2"
	case "tuic", "anytls":
		return true
	default:
		return false
	}
}

func shouldClashDisableTFO(m map[string]interface{}) bool {
	tls, _ := m["tls"].(bool)
	if !tls {
		return false
	}
	typ, _ := m["type"].(string)
	switch typ {
	case "vless", "vmess", "trojan":
		return true
	default:
		return false
	}
}

func writeYAMLInlineValue(sb *strings.Builder, val interface{}) {
	writeYAMLInlineValueForKey(sb, "", val)
}

// writeYAMLInlineValueForKey 与 writeYAMLInlineValue 相同，但知道当前字段名，
// 用于把 short-id / public-key / uuid / password 等字段强制按字符串输出。
func writeYAMLInlineValueForKey(sb *strings.Builder, key string, val interface{}) {
	switch v := val.(type) {
	case string:
		if forceQuoteYAMLKey(key) {
			sb.WriteString(quoteYAMLString(v))
		} else {
			sb.WriteString(escapeYAML(v))
		}
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
			writeYAMLInlineValueForKey(sb, k, v[k])
		}
		sb.WriteString("}")
	case []interface{}:
		sb.WriteString("[")
		for i, item := range v {
			if i > 0 {
				sb.WriteString(", ")
			}
			writeYAMLInlineValueForKey(sb, key, item)
		}
		sb.WriteString("]")
	case []string:
		sb.WriteString("[")
		for i, item := range v {
			if i > 0 {
				sb.WriteString(", ")
			}
			if forceQuoteYAMLKey(key) {
				sb.WriteString(quoteYAMLString(item))
			} else {
				sb.WriteString(escapeYAML(item))
			}
		}
		sb.WriteString("]")
	default:
		sb.WriteString(fmt.Sprintf("%v", val))
	}
}

// yamlPlainIsNonString 判断一个不加引号的纯量会被 YAML 隐式解析成"非字符串"
// （整数 / 浮点 / 科学计数法 / 布尔 / null / 时间戳）。
//
// 线上真实故障：REALITY 的 short-id "450259e7" 是合法十六进制，但 YAML 会把它
// 当成 450259 x 10^7 的浮点数，客户端把订阅回写成内核配置后变成
// 4502590000000.0，mihomo 直接 fatal: invalid REALITY short ID。
// 因此这类值必须加引号输出。
func yamlPlainIsNonString(s string) bool {
	switch s {
	case "":
		return false
	case "~", "null", "Null", "NULL":
		return true
	case "true", "True", "TRUE", "false", "False", "FALSE":
		return true
	case "y", "Y", "yes", "Yes", "YES", "n", "N", "no", "No", "NO",
		"on", "On", "ON", "off", "Off", "OFF":
		return true
	case ".inf", ".Inf", ".INF", "+.inf", "+.Inf", "+.INF",
		"-.inf", "-.Inf", "-.INF", ".nan", ".NaN", ".NAN":
		return true
	}
	plain := strings.ReplaceAll(s, "_", "")
	if _, err := strconv.ParseInt(plain, 0, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseUint(plain, 0, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(plain, 64); err == nil {
		return true
	}
	// YAML 1.1 的日期 / 时间戳（如 2026-10-27）同样会被解析成非字符串
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' &&
		isASCIIDigits(s[0:4]) && isASCIIDigits(s[5:7]) && isASCIIDigits(s[8:10]) {
		return true
	}
	return false
}

func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// quoteYAMLString 始终以双引号输出，并转义反斜杠、双引号与控制字符。
func quoteYAMLString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// forceQuoteYAMLKeys 这些字段一律按字符串（带引号）输出：它们的值是十六进制串
// 或口令，一旦被 YAML 当数值读走，客户端回写订阅时就会变形。
var forceQuoteYAMLKeys = map[string]bool{
	"short-id":      true,
	"public-key":    true,
	"private-key":   true,
	"uuid":          true,
	"password":      true,
	"servername":    true,
	"sni":           true,
	"psk":           true,
	"auth-str":      true,
	"obfs-password": true,
}

func forceQuoteYAMLKey(key string) bool {
	return forceQuoteYAMLKeys[strings.ToLower(strings.TrimSpace(key))]
}

func escapeYAML(s string) string {
	if s == "" {
		return "\"\""
	}
	if yamlPlainIsNonString(s) {
		return quoteYAMLString(s)
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
		return quoteYAMLString(s)
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
