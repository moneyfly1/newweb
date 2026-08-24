package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"cboard/v2/internal/models"
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
