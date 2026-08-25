package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func VmessLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	encoded := strings.TrimPrefix(link, "vmess://")

	// Split off query/fragment before base64 decode
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
				return nil, err
			}
		}
	}

	// Try standard JSON vmess format
	var cfg map[string]interface{}
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		// Non-standard format: Base64(auto:uuid@server:port)?query_params
		return convertNonStandardToClashMap(link, name, "vmess")
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
			} else if net == "httpupgrade" {
				huOpts := map[string]interface{}{}
				if path, ok := cfg["path"].(string); ok && path != "" {
					huOpts["path"] = path
				}
				if host, ok := cfg["host"].(string); ok && host != "" {
					huOpts["host"] = host
				}
				if len(huOpts) > 0 {
					m["httpupgrade-opts"] = huOpts
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

	// Detect non-standard Base64 format: vless://Base64(auto:uuid@server:port)?params
	if u.User == nil || u.User.Username() == "" || u.Hostname() == "" {
		return convertNonStandardToClashMap(link, name, "vless")
	}

	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port := parsePortWithDefault(portStr, 0)

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
		} else if t == "h2" {
			h2Opts := map[string]interface{}{}
			if p := q.Get("path"); p != "" {
				h2Opts["path"] = p
			}
			if h := q.Get("host"); h != "" {
				h2Opts["host"] = []string{h}
			}
			if len(h2Opts) > 0 {
				m["h2-opts"] = h2Opts
			}
		} else if t == "httpupgrade" {
			huOpts := map[string]interface{}{}
			if p := q.Get("path"); p != "" {
				huOpts["path"] = p
			}
			if h := q.Get("host"); h != "" {
				huOpts["host"] = h
			}
			if len(huOpts) > 0 {
				m["httpupgrade-opts"] = huOpts
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
			// short-id 为空或非法时省略字段（Clash Meta 空 short-id 合法，
			// 但 "short-id: ''" 空字符串会报 invalid REALITY short ID）
			opts := map[string]interface{}{"public-key": q.Get("pbk")}
			if sid := sanitizeRealityShortID(q.Get("sid")); sid != "" {
				opts["short-id"] = sid
			}
			m["reality-opts"] = opts
		}
		if fp := q.Get("fp"); fp != "" {
			m["client-fingerprint"] = fp
		}
	}
	if flow := q.Get("flow"); flow != "" {
		m["flow"] = flow
	}
	if alpn := q.Get("alpn"); alpn != "" {
		m["alpn"] = strings.Split(alpn, ",")
	}
	if q.Get("allowInsecure") == "1" || q.Get("insecure") == "1" {
		m["skip-cert-verify"] = true
	}
	if enc := q.Get("encryption"); enc != "" && enc != "none" {
		m["encryption"] = enc
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
	port := parsePortWithDefault(portStr, 0)

	password := ""
	if u.User != nil {
		password = u.User.Username()
	}

	m := map[string]interface{}{
		"name":     name,
		"type":     "trojan",
		"server":   host,
		"port":     port,
		"password": password,
	}
	if sni := q.Get("sni"); sni != "" {
		m["sni"] = sni
	} else if peer := q.Get("peer"); peer != "" {
		m["sni"] = peer
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
		} else if t == "grpc" {
			if sn := q.Get("serviceName"); sn != "" {
				m["grpc-opts"] = map[string]interface{}{"grpc-service-name": sn}
			}
		}
	}
	if q.Get("allowInsecure") == "1" || q.Get("insecure") == "1" {
		m["skip-cert-verify"] = true
	}
	if alpn := q.Get("alpn"); alpn != "" {
		m["alpn"] = strings.Split(alpn, ",")
	}
	if fp := q.Get("fp"); fp != "" {
		m["client-fingerprint"] = fp
	}
	// Plugin support (obfs-local / v2ray-plugin etc.)
	if pluginStr := q.Get("plugin"); pluginStr != "" {
		parts := strings.Split(pluginStr, ";")
		if len(parts) > 0 {
			pluginName := strings.TrimSpace(parts[0])
			switch pluginName {
			case "simple-obfs", "obfs-local":
				pluginName = "obfs"
			}
			m["plugin"] = pluginName
			pluginOpts := map[string]interface{}{}
			for _, part := range parts[1:] {
				kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					switch key {
					case "obfs":
						pluginOpts["mode"] = val
					case "obfs-host":
						pluginOpts["host"] = val
					case "obfs-uri":
						pluginOpts["path"] = val
					default:
						pluginOpts[key] = val
					}
				}
			}
			if len(pluginOpts) > 0 {
				m["plugin-opts"] = pluginOpts
			}
		}
	}
	return m, nil
}

// ShadowsocksLinkToClashMap parses an ss:// link into a Clash-compatible proxy map.

func ShadowsocksLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	var cipher, password, host string
	var port int

	// Try non-standard format first: ss://Base64(method:pass@server:port)#name
	encoded := strings.TrimPrefix(link, "ss://")
	base64Part := encoded
	if idx := strings.Index(base64Part, "#"); idx != -1 {
		base64Part = base64Part[:idx]
	}
	if idx := strings.Index(base64Part, "?"); idx != -1 {
		base64Part = base64Part[:idx]
	}

	if decoded, err := decodeBase64Flexible(base64Part); err == nil && strings.Contains(decoded, "@") {
		// Full Base64 format: method:pass@server:port
		atIdx := strings.LastIndex(decoded, "@")
		userPart := decoded[:atIdx]
		serverPart := decoded[atIdx+1:]
		host, portStr := splitHostPort(serverPart)
		port = parsePortWithDefault(portStr, 0)
		if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
			cipher = parts[0]
			password = parts[1]
		}
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

	// Standard format: ss://Base64(method:password)@server:port or ss://method:password@server:port
	userInfo := ""
	if u.User != nil {
		userInfo = u.User.Username()
	}
	if decoded, err := decodeBase64Flexible(userInfo); err == nil && strings.Contains(decoded, ":") {
		parts := strings.SplitN(decoded, ":", 2)
		cipher = parts[0]
		password = parts[1]
	} else {
		cipher = userInfo
		if p, ok := u.User.Password(); ok {
			password = p
		}
	}
	host, portStr := splitHostPort(u.Host)
	port = parsePortWithDefault(portStr, 0)

	m := map[string]interface{}{
		"name":     name,
		"type":     "ss",
		"server":   host,
		"port":     port,
		"cipher":   cipher,
		"password": password,
	}

	// 解析 plugin 参数 (simple-obfs, v2ray-plugin 等)
	if pluginStr := u.Query().Get("plugin"); pluginStr != "" {
		parts := strings.Split(pluginStr, ";")
		if len(parts) > 0 {
			pluginName := strings.TrimSpace(parts[0])
			switch pluginName {
			case "simple-obfs", "obfs-local":
				pluginName = "obfs"
			}
			m["plugin"] = pluginName
			pluginOpts := map[string]interface{}{}
			for _, part := range parts[1:] {
				kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					switch key {
					case "obfs":
						pluginOpts["mode"] = val
					case "obfs-host":
						pluginOpts["host"] = val
					case "obfs-uri", "path":
						pluginOpts["path"] = val
					case "tls":
						pluginOpts["tls"] = true
					default:
						pluginOpts[key] = val
					}
				}
			}
			if len(pluginOpts) > 0 {
				m["plugin-opts"] = pluginOpts
			}
		}
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
	port := parsePortWithDefault(parts[1], 0)
	protocol := parts[2]
	method := parts[3]
	obfs := parts[4]
	passwordB64 := parts[5]

	passwordBytes, err := base64.RawURLEncoding.DecodeString(passwordB64)
	if err != nil {
		passwordBytes, err = base64.StdEncoding.DecodeString(passwordB64)
		if err != nil {
			return nil, fmt.Errorf("invalid ssr password encoding: %w", err)
		}
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
		params := parseQuerySafe(mainAndParams[1])
		if remarks := params.Get("remarks"); remarks != "" {
			if decodedRemarks, err := decodeBase64Flexible(remarks); err == nil && decodedRemarks != "" {
				if name == "" {
					m["name"] = sanitizeNodeName(decodedRemarks)
				}
			}
		}
		if pp := params.Get("protoparam"); pp != "" {
			ppDecoded, err := decodeBase64Flexible(pp)
			if err == nil {
				m["protocol-param"] = ppDecoded
			}
		}
		if op := params.Get("obfsparam"); op != "" {
			opDecoded, err := decodeBase64Flexible(op)
			if err == nil {
				m["obfs-param"] = opDecoded
			}
		}
		if group := params.Get("group"); group != "" {
			if decodedGroup, err := decodeBase64Flexible(group); err == nil && decodedGroup != "" {
				m["group"] = decodedGroup
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
	port := parsePortWithDefault(portStr, 0)

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
	if alpn := q.Get("alpn"); alpn != "" {
		m["alpn"] = strings.Split(alpn, ",")
	}
	if obfs := q.Get("obfs"); obfs != "" {
		m["obfs"] = obfs
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
	port := parsePortWithDefault(portStr, 0)

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
	if alpn := q.Get("alpn"); alpn != "" {
		m["alpn"] = strings.Split(alpn, ",")
	}
	if fp := q.Get("fp"); fp != "" {
		m["client-fingerprint"] = fp
	}
	if obfs := q.Get("obfs"); obfs != "" {
		m["obfs"] = obfs
		if obfsPw := q.Get("obfs-password"); obfsPw != "" {
			m["obfs-password"] = obfsPw
		}
	}
	return m, nil
}

// TUICLinkToClashMap parses a tuic:// link into a Clash-compatible proxy map.

func TUICLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	// Detect non-standard Base64 format: tuic://Base64(auto:uuid@server:port)?params
	if u.User == nil || u.User.Username() == "" || u.Hostname() == "" {
		return convertNonStandardToClashMap(link, name, "tuic")
	}

	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port := parsePortWithDefault(portStr, 0)

	uuid := ""
	password := ""
	if u.User != nil {
		userInfo := u.User.Username()
		if p, ok := u.User.Password(); ok {
			// Standard format: uuid:password in URL userinfo
			uuid = userInfo
			password = p
		} else if strings.Contains(userInfo, ":") {
			// URL-encoded colon: tuic://uuid%3Apassword@host:port
			parts := strings.SplitN(userInfo, ":", 2)
			uuid = parts[0]
			password = parts[1]
		} else {
			uuid = userInfo
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

	// Check for fully Base64 encoded format: socks://Base64(user:pass@host:port)?params
	// or socks://Base64(host:port)?params
	q := u.Query()
	host := u.Hostname()
	portStr := u.Port()

	var username, password string
	hasPassword := false

	if host == "" || (u.User == nil && (q.Get("remarks") != "" || q.Get("gost") != "")) {
		// Non-standard: entire authority is Base64
		encoded := strings.TrimPrefix(link, "socks://")
		encoded = strings.TrimPrefix(encoded, "socks5://")
		base64Part := encoded
		if idx := strings.Index(base64Part, "?"); idx != -1 {
			base64Part = base64Part[:idx]
		}
		if idx := strings.Index(base64Part, "#"); idx != -1 {
			base64Part = base64Part[:idx]
		}
		if decoded, err := decodeBase64Flexible(base64Part); err == nil {
			// Split user@server BEFORE url-decoding, so %3A in username is preserved
			if atIdx := strings.LastIndex(decoded, "@"); atIdx != -1 {
				userPart := decoded[:atIdx]
				serverPart := decoded[atIdx+1:]
				// URL-decode server part
				if unescaped, e := url.QueryUnescape(serverPart); e == nil {
					serverPart = unescaped
				}
				host, portStr = splitHostPort(serverPart)
				// Split user:pass on first unencoded colon, then URL-decode each part
				if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
					u1, _ := url.QueryUnescape(parts[0])
					u2, _ := url.QueryUnescape(parts[1])
					username = u1
					password = u2
					hasPassword = true
				} else {
					if u1, e := url.QueryUnescape(userPart); e == nil {
						username = u1
					}
				}
			} else {
				if unescaped, e := url.QueryUnescape(decoded); e == nil {
					decoded = unescaped
				}
				host, portStr = splitHostPort(decoded)
			}
		}
	} else {
		host, portStr = splitHostPort(u.Host)
		if u.User != nil {
			username = u.User.Username()
			pw, hasPw := u.User.Password()
			if hasPw {
				password = pw
				hasPassword = true
			}

			// GOST 格式: Base64 编码的 user:pass
			if decoded, err := decodeBase64Flexible(username); err == nil {
				if parts := strings.SplitN(decoded, ":", 2); len(parts) == 2 {
					username = parts[0]
					password = parts[1]
					hasPassword = true
				}
			}
		}
	}

	port := parsePortWithDefault(portStr, 1080)
	m := map[string]interface{}{
		"name":   name,
		"type":   "socks5",
		"server": host,
		"port":   port,
		"udp":    true,
	}
	if username != "" {
		m["username"] = username
	}
	if hasPassword {
		m["password"] = password
	}

	// GOST WebSocket 传输层支持 (gost param is Base64 encoded JSON)
	if gostB64 := q.Get("gost"); gostB64 != "" {
		if gostJSON, err := decodeBase64Flexible(gostB64); err == nil {
			var gostCfg map[string]interface{}
			if json.Unmarshal([]byte(gostJSON), &gostCfg) == nil {
				route, _ := gostCfg["route"].(string)
				if route == "ws" {
					m["network"] = "ws"
					wsOpts := map[string]interface{}{}
					if p, ok := gostCfg["path"].(string); ok && p != "" {
						wsOpts["path"] = p
					}
					if h, ok := gostCfg["host"].(string); ok && h != "" {
						wsOpts["headers"] = map[string]interface{}{"Host": h}
					}
					if len(wsOpts) > 0 {
						m["ws-opts"] = wsOpts
					}
				}
			}
		} else {
			// Fallback: try as URL query
			gostParams := parseQuerySafe(gostB64)
			if t := gostParams.Get("type"); t == "ws" || strings.Contains(gostB64, "ws") {
				m["network"] = "ws"
				wsOpts := map[string]interface{}{}
				if p := gostParams.Get("path"); p != "" {
					wsOpts["path"] = p
				}
				if h := gostParams.Get("host"); h != "" {
					wsOpts["headers"] = map[string]interface{}{"Host": h}
				}
				if len(wsOpts) > 0 {
					m["ws-opts"] = wsOpts
				}
			}
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

	host := u.Hostname()
	portStr := u.Port()
	var username, password string

	// Check for non-standard Base64 format: http://Base64(user:pass@server:port)?params
	if host == "" || (u.User == nil && u.Query().Get("method") != "") {
		encoded := strings.TrimPrefix(link, "http://")
		encoded = strings.TrimPrefix(encoded, "https://")
		base64Part := encoded
		if idx := strings.Index(base64Part, "?"); idx != -1 {
			base64Part = base64Part[:idx]
		}
		if idx := strings.Index(base64Part, "#"); idx != -1 {
			base64Part = base64Part[:idx]
		}
		if decoded, err := decodeBase64Flexible(base64Part); err == nil {
			if unescaped, e := url.QueryUnescape(decoded); e == nil {
				decoded = unescaped
			}
			if atIdx := strings.LastIndex(decoded, "@"); atIdx != -1 {
				userPart := decoded[:atIdx]
				serverPart := decoded[atIdx+1:]
				host, portStr = splitHostPort(serverPart)
				if parts := strings.SplitN(userPart, ":", 2); len(parts) == 2 {
					username = parts[0]
					password = parts[1]
				}
			} else {
				host, portStr = splitHostPort(decoded)
			}
		}
	} else {
		host, portStr = splitHostPort(u.Host)
		if u.User != nil {
			username = u.User.Username()
			if pw, ok := u.User.Password(); ok {
				password = pw
			}
		}
	}

	defaultPort := 80
	if strings.HasPrefix(link, "https://") {
		defaultPort = 443
	}
	port := parsePortWithDefault(portStr, defaultPort)
	m := map[string]interface{}{
		"name":   name,
		"type":   "http",
		"server": host,
		"port":   port,
	}
	if strings.HasPrefix(link, "https://") {
		m["tls"] = true
	}
	if username != "" {
		m["username"] = username
	}
	if password != "" {
		m["password"] = password
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
	port := parsePortWithDefault(portStr, 443)
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

// WireGuardLinkToClashMap parses a wg:// link into a Clash-compatible proxy map.

func WireGuardLinkToClashMap(link string, name string) (map[string]interface{}, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	host, portStr := splitHostPort(u.Host)
	port := parsePortWithDefault(portStr, 51820)

	privateKey := ""
	if u.User != nil {
		privateKey = u.User.Username()
	}
	// privateKey may also be in query params
	// Use RawQuery to avoid + being decoded as space
	if pk := getRawQueryParam(u.RawQuery, "privateKey"); pk != "" {
		privateKey = pk
	}

	m := map[string]interface{}{
		"name":        name,
		"type":        "wireguard",
		"server":      host,
		"port":        port,
		"private-key": privateKey,
		"udp":         true,
	}

	if pk := getRawQueryParam(u.RawQuery, "publicKey"); pk != "" {
		m["public-key"] = pk
	}
	if ip := q.Get("ip"); ip != "" {
		// ip may contain both ipv4 and ipv6 separated by comma
		ips := strings.Split(ip, ",")
		m["ip"] = strings.TrimSpace(ips[0])
		if len(ips) > 1 {
			m["ipv6"] = strings.TrimSpace(ips[1])
		}
	}
	if ipv6 := q.Get("ipv6"); ipv6 != "" {
		m["ipv6"] = ipv6
	}
	if mtu := q.Get("mtu"); mtu != "" {
		if mtuInt, err := strconv.Atoi(mtu); err == nil {
			m["mtu"] = mtuInt
		}
	}
	if psk := getRawQueryParam(u.RawQuery, "presharedKey"); psk != "" {
		m["preshared-key"] = psk
	}
	if reserved := q.Get("reserved"); reserved != "" {
		m["reserved"] = reserved
	}
	if keepalive := q.Get("keepalive"); keepalive != "" {
		if ka, err := strconv.Atoi(keepalive); err == nil {
			m["keepalive"] = ka
		}
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
	case "wireguard":
		return WireGuardLinkToClashMap(configLink, nodeName)
	default:
		return nil, fmt.Errorf("unsupported type: %s", nodeType)
	}
}

// GenerateClashYAML generates a proper Clash YAML config from nodes.

// sanitizeRealityShortID 校验并规范化 REALITY short-id：
// Clash Meta 要求十六进制字符串（1-16 字节，通常 8 位）。
// 空值或非法格式返回 ""，调用方应省略该字段（空 short-id 在 Clash 中合法，
// 但 "short-id: ''" 空字符串会触发 "invalid REALITY short ID" 错误）。
func sanitizeRealityShortID(sid string) string {
	sid = strings.TrimSpace(sid)
	if sid == "" {
		return ""
	}
	// 必须为十六进制字符（大小写均可），且长度不超过 32（16 字节）
	if len(sid) > 32 {
		return ""
	}
	for _, ch := range sid {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return ""
		}
	}
	return sid
}
