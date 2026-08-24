package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"cboard/v2/internal/models"
)

func ParseVmessLink(link string) (*models.Node, error) {
	encoded := strings.TrimPrefix(link, "vmess://")

	// Split off query string before base64 decode
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

	// Try standard JSON vmess format first
	var vmessConfig map[string]interface{}
	if err := json.Unmarshal(decoded, &vmessConfig); err == nil {
		name := ""
		if ps, ok := vmessConfig["ps"].(string); ok {
			name = ps
		}
		return buildNode(name, "VMess Node", "vmess", link), nil
	}

	// Non-standard format: Base64(auto:uuid@server:port)?query_params
	decodedStr := string(decoded)
	if strings.Contains(decodedStr, "@") {
		// Extract name from query remarks or fragment
		name := "VMess Node"
		if idx := strings.Index(encoded, "?"); idx != -1 {
			queryStr := encoded[idx+1:]
			if fidx := strings.Index(queryStr, "#"); fidx != -1 {
				queryStr = queryStr[:fidx]
			}
			q := parseQuerySafe(queryStr)
			if r := q.Get("remarks"); r != "" {
				name = decodeFragment(r)
			}
		}
		if idx := strings.Index(encoded, "#"); idx != -1 {
			if n := decodeFragment(encoded[idx+1:]); n != "" {
				name = n
			}
		}
		return buildNode(name, "VMess Node", "vmess", link), nil
	}

	return nil, fmt.Errorf("invalid vmess link format")
}


func ParseVlessLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	name := u.Fragment
	// Check for non-standard format: vless://Base64(...)
	if u.User == nil || u.User.Username() == "" {
		// Try non-standard Base64 format
		encoded := strings.TrimPrefix(link, "vless://")
		base64Part := encoded
		if idx := strings.Index(base64Part, "?"); idx != -1 {
			base64Part = base64Part[:idx]
		}
		if idx := strings.Index(base64Part, "#"); idx != -1 {
			base64Part = base64Part[:idx]
		}
		if decoded, err := decodeBase64Flexible(base64Part); err == nil && strings.Contains(decoded, "@") {
			q := u.Query()
			if r := q.Get("remarks"); r != "" {
				name = r
			}
		}
	}

	return buildNode(name, "VLESS Node", "vless", link), nil
}


func ParseTrojanLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	return buildNode(u.Fragment, "Trojan Node", "trojan", link), nil
}


func ParseShadowsocksLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	return buildNode(u.Fragment, "Shadowsocks Node", "ss", link), nil
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
		params := parseQuerySafe(mainAndParams[1])
		if remarks := params.Get("remarks"); remarks != "" {
			remarksDecoded, err := base64.RawURLEncoding.DecodeString(remarks)
			if err == nil {
				name = string(remarksDecoded)
			}
		}
	}

	return buildNode(name, "SSR Node", "ssr", link), nil
}

// ParseHysteriaLink parses a hysteria:// link into a Node model.

func ParseHysteriaLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	return buildNode(u.Fragment, "Hysteria Node", "hysteria", link), nil
}

// ParseHysteria2Link parses a hysteria2:// or hy2:// link into a Node model.

func ParseHysteria2Link(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	return buildNode(u.Fragment, "Hysteria2 Node", "hysteria2", link), nil
}

// ParseTUICLink parses a tuic:// link into a Node model.

func ParseTUICLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	return buildNode(u.Fragment, "TUIC Node", "tuic", link), nil
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
	name := decodeFragment(u.Fragment)
	if name == "" {
		name = u.Hostname()
	}
	return buildNode(name, "Naive Node", "naive", link), nil
}

// ParseAnytlsLink parses anytls:// links

func ParseAnytlsLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	name := decodeFragment(u.Fragment)
	if name == "" {
		name = u.Hostname()
	}
	return buildNode(name, "AnyTLS Node", "anytls", link), nil
}

// ParseSOCKSLink parses socks5:// and socks:// links

func ParseSOCKSLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	name := decodeFragment(u.Fragment)
	// Check for remarks in query (GOST format)
	if name == "" {
		if r := u.Query().Get("remarks"); r != "" {
			name = decodeFragment(r)
		}
	}

	// Handle Base64 encoded socks links: socks://Base64(user:pass@host:port)?params
	// or socks://Base64(host:port)?params
	if u.Hostname() == "" || (u.User == nil && name != "") {
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
			if unescaped, e := url.QueryUnescape(decoded); e == nil {
				decoded = unescaped
			}
			// decoded could be "user:pass@host:port" or "host:port"
			if name == "" {
				if strings.Contains(decoded, "@") {
					parts := strings.SplitN(decoded, "@", 2)
					name = parts[1]
				} else {
					name = decoded
				}
			}
		}
	}

	if name == "" {
		name = u.Hostname()
	}
	if name == "" {
		name = "SOCKS Node"
	}
	return buildNode(name, "SOCKS Node", "socks5", link), nil
}

// ParseHTTPLink parses http:// and https:// proxy links

func ParseHTTPLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	name := decodeFragment(u.Fragment)
	// Check remarks in query
	if name == "" {
		if r := u.Query().Get("remarks"); r != "" {
			name = decodeFragment(r)
		}
	}
	if name == "" {
		name = u.Hostname()
	}
	if name == "" {
		name = "HTTP Node"
	}
	return buildNode(name, "HTTP Node", "http", link), nil
}

// ParseWireGuardLink parses a wg:// link into a Node model.

func ParseWireGuardLink(link string) (*models.Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	name := decodeFragment(u.Fragment)
	if name == "" {
		name = "WireGuard " + u.Hostname()
	}
	return buildNode(name, "WireGuard Node", "wireguard", link), nil
}

// decodeBase64Flexible tries multiple base64 encodings
