package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func decodeBase64Flexible(s string) (string, error) {
	// Try standard base64
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return string(decoded), nil
	}
	// Try raw standard (no padding)
	if decoded, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return string(decoded), nil
	}
	// Try URL-safe base64
	if decoded, err := base64.URLEncoding.DecodeString(s); err == nil {
		return string(decoded), nil
	}
	// Try raw URL-safe
	if decoded, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return string(decoded), nil
	}
	return "", fmt.Errorf("not base64")
}

// normalizeNonStandardLink converts non-standard Base64 encoded links to standard format.
// Non-standard format: protocol://Base64(user:pass@server:port)?query_params#fragment
// or: protocol://Base64(method:pass@server:port)#fragment
// Returns the normalized link and whether it was converted.

func normalizeNonStandardLink(link string) (string, bool) {
	// Find the scheme
	schemeEnd := strings.Index(link, "://")
	if schemeEnd < 0 {
		return link, false
	}
	scheme := link[:schemeEnd]
	rest := link[schemeEnd+3:]

	// Split off query and fragment
	base64Part := rest
	queryPart := ""
	fragmentPart := ""

	if idx := strings.Index(base64Part, "?"); idx != -1 {
		queryPart = base64Part[idx:]
		base64Part = base64Part[:idx]
		// Fragment might be in query part
		if fidx := strings.Index(queryPart, "#"); fidx != -1 {
			fragmentPart = queryPart[fidx:]
			queryPart = queryPart[:fidx]
		}
	} else if idx := strings.Index(base64Part, "#"); idx != -1 {
		fragmentPart = base64Part[idx:]
		base64Part = base64Part[:idx]
	}

	// Try to decode the base64 part
	decoded, err := decodeBase64Flexible(base64Part)
	if err != nil {
		return link, false
	}

	// Check if decoded looks like user:pass@server:port or server:port
	if !strings.Contains(decoded, "@") && !strings.Contains(decoded, ":") {
		return link, false
	}

	// URL-decode the decoded string (some have %3A etc)
	if unescaped, err := url.QueryUnescape(decoded); err == nil {
		decoded = unescaped
	}

	// Reconstruct as standard link
	normalized := scheme + "://" + decoded + queryPart + fragmentPart

	// For query-based params, convert remarks to fragment if no fragment
	if fragmentPart == "" && queryPart != "" {
		parsed, err := url.Parse(normalized)
		if err == nil {
			remarks := parsed.Query().Get("remarks")
			if remarks != "" {
				q := parsed.Query()
				q.Del("remarks")
				parsed.RawQuery = q.Encode()
				parsed.Fragment = remarks
				normalized = parsed.String()
			}
		}
	}

	return normalized, true
}

// convertNonStandardToClashMap handles non-standard Base64 links with query-based params
// and converts them to Clash proxy maps directly.

func convertNonStandardToClashMap(link string, name string, nodeType string) (map[string]interface{}, error) {
	schemeEnd := strings.Index(link, "://")
	if schemeEnd < 0 {
		return nil, fmt.Errorf("invalid link")
	}
	rest := link[schemeEnd+3:]

	// Split base64 part from query
	base64Part := rest
	queryStr := ""
	if idx := strings.Index(base64Part, "?"); idx != -1 {
		queryStr = base64Part[idx+1:]
		base64Part = base64Part[:idx]
	}
	if idx := strings.Index(base64Part, "#"); idx != -1 {
		base64Part = base64Part[:idx]
	}
	// Remove fragment from query
	if idx := strings.Index(queryStr, "#"); idx != -1 {
		queryStr = queryStr[:idx]
	}

	decoded, err := decodeBase64Flexible(base64Part)
	if err != nil {
		return nil, fmt.Errorf("cannot decode base64: %w", err)
	}
	if unescaped, err := url.QueryUnescape(decoded); err == nil {
		decoded = unescaped
	}

	// Parse decoded: user:pass@server:port or method:pass@server:port
	var userPart, serverPart string
	if atIdx := strings.LastIndex(decoded, "@"); atIdx != -1 {
		userPart = decoded[:atIdx]
		serverPart = decoded[atIdx+1:]
	} else {
		serverPart = decoded
	}

	host, portStr := splitHostPort(serverPart)
	port := parsePortWithDefault(portStr, 0)

	query := parseQuerySafe(queryStr)

	m := map[string]interface{}{
		"name":   name,
		"type":   nodeType,
		"server": host,
		"port":   port,
	}

	switch nodeType {
	case "vless":
		// userPart = "auto:uuid"
		uuid := userPart
		if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
			uuid = parts[1]
		}
		m["uuid"] = uuid

		// obfs/transport
		if obfs := query.Get("obfs"); obfs != "" {
			switch obfs {
			case "websocket":
				m["network"] = "ws"
				wsOpts := map[string]interface{}{}
				if p := query.Get("path"); p != "" {
					wsOpts["path"] = p
				}
				if opStr := query.Get("obfsParam"); opStr != "" {
					var obfsParam map[string]interface{}
					if json.Unmarshal([]byte(opStr), &obfsParam) == nil {
						if h, ok := obfsParam["Host"].(string); ok && h != "" {
							wsOpts["headers"] = map[string]interface{}{"Host": h}
						}
					}
				}
				if len(wsOpts) > 0 {
					m["ws-opts"] = wsOpts
				}
			case "grpc":
				m["network"] = "grpc"
				if p := query.Get("path"); p != "" {
					m["grpc-opts"] = map[string]interface{}{"grpc-service-name": p}
				}
			}
		} else if p := query.Get("path"); p != "" {
			// Default to ws if path is present
			m["network"] = "ws"
			wsOpts := map[string]interface{}{"path": p}
			if opStr := query.Get("obfsParam"); opStr != "" {
				var obfsParam map[string]interface{}
				if json.Unmarshal([]byte(opStr), &obfsParam) == nil {
					if h, ok := obfsParam["Host"].(string); ok && h != "" {
						wsOpts["headers"] = map[string]interface{}{"Host": h}
					}
				}
			}
			m["ws-opts"] = wsOpts
		}

		// TLS
		if query.Get("tls") == "1" {
			m["tls"] = true
			if peer := query.Get("peer"); peer != "" {
				m["servername"] = peer
			}
		}
		// Reality
		if pbk := query.Get("pbk"); pbk != "" {
			m["tls"] = true
			realityOpts := map[string]interface{}{"public-key": pbk}
			if sid := sanitizeRealityShortID(query.Get("sid")); sid != "" {
				realityOpts["short-id"] = sid
			}
			m["reality-opts"] = realityOpts
			if peer := query.Get("peer"); peer != "" {
				m["servername"] = peer
			}
			m["client-fingerprint"] = "chrome"
		}
		// XTLS
		if query.Get("xtls") == "2" {
			m["flow"] = "xtls-rprx-vision"
		}

	case "vmess":
		// userPart = "auto:uuid"
		uuid := userPart
		cipher := "auto"
		if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
			cipher = parts[0]
			uuid = parts[1]
		}
		m["uuid"] = uuid
		m["cipher"] = cipher
		alterId := 0
		if aid := query.Get("alterId"); aid != "" {
			alterId = parseIntOrDefault(aid, 0)
		}
		m["alterId"] = alterId

		// obfs/transport
		if obfs := query.Get("obfs"); obfs == "websocket" {
			m["network"] = "ws"
			wsOpts := map[string]interface{}{}
			if p := query.Get("path"); p != "" {
				wsOpts["path"] = p
			}
			if opStr := query.Get("obfsParam"); opStr != "" {
				var obfsParam map[string]interface{}
				if json.Unmarshal([]byte(opStr), &obfsParam) == nil {
					if h, ok := obfsParam["Host"].(string); ok && h != "" {
						wsOpts["headers"] = map[string]interface{}{"Host": h}
					} else if h, ok := obfsParam["HOST"].(string); ok && h != "" {
						wsOpts["headers"] = map[string]interface{}{"Host": h}
					}
				}
			}
			if len(wsOpts) > 0 {
				m["ws-opts"] = wsOpts
			}
		}

		// TLS
		if query.Get("tls") == "1" {
			m["tls"] = true
			if peer := query.Get("peer"); peer != "" {
				m["servername"] = peer
			}
		}
		if query.Get("allowInsecure") == "1" {
			m["skip-cert-verify"] = true
		}
		if alpn := query.Get("alpn"); alpn != "" {
			m["alpn"] = strings.Split(alpn, ",")
		}

	case "trojan":
		m["password"] = userPart

		if peer := query.Get("peer"); peer != "" {
			m["sni"] = peer
		}
		if alpn := query.Get("alpn"); alpn != "" {
			m["alpn"] = strings.Split(alpn, ",")
		}
		if query.Get("allowInsecure") == "1" {
			m["skip-cert-verify"] = true
		}

	case "ss":
		// userPart = "method:password"
		if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
			m["cipher"] = parts[0]
			m["password"] = parts[1]
		}

	case "socks5", "socks":
		m["type"] = "socks5"
		m["udp"] = true
		if userPart != "" {
			if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
				m["username"] = parts[0]
				m["password"] = parts[1]
			}
		}

	case "tuic":
		// userPart = "auto:uuid" or "uuid:password"
		uuid := userPart
		password := ""
		if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
			uuid = parts[1]
			password = parts[0]
			// If first part is "auto", use second part as uuid
			if parts[0] == "auto" {
				uuid = parts[1]
				password = ""
			}
		}
		m["uuid"] = uuid
		if password != "" {
			m["password"] = password
		}
		if cc := query.Get("congestion_control"); cc != "" {
			m["congestion-controller"] = cc
		}
		if alpn := query.Get("alpn"); alpn != "" {
			m["alpn"] = strings.Split(alpn, ",")
		}
		if sni := query.Get("sni"); sni != "" {
			m["sni"] = sni
		}
	}

	return m, nil
}

// getRawQueryParam extracts a query parameter value without decoding + as space.
// This is needed for base64 values that contain + characters.

func getRawQueryParam(rawQuery string, key string) string {
	for _, part := range strings.Split(rawQuery, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			k := decodeFragment(kv[0])
			if k == key {
				// Only percent-decode, don't convert + to space
				val := strings.ReplaceAll(kv[1], "%2B", "+")
				val = strings.ReplaceAll(val, "%2b", "+")
				if unescaped, err := url.PathUnescape(val); err == nil {
					return unescaped
				}
				return val
			}
		}
	}
	return ""
}


func portToInt(port string) int {
	p, err := strconv.Atoi(port)
	if err != nil {
		return 0
	}
	return p
}

