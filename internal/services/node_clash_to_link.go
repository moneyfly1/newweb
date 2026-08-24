package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func clashProxyToLink(proxy map[string]interface{}, nodeType string, name string) (string, string, error) {
	switch nodeType {
	case "vmess":
		return clashVMessProxyToLink(proxy, name)
	case "vless":
		return clashVLESSProxyToLink(proxy, name)
	case "trojan":
		return clashTrojanProxyToLink(proxy, name)
	case "ss":
		return clashShadowsocksProxyToLink(proxy, name)
	case "ssr":
		return clashSSRProxyToLink(proxy, name)
	case "hysteria":
		return clashHysteriaProxyToLink(proxy, name)
	case "hysteria2":
		return clashHysteria2ProxyToLink(proxy, name)
	case "tuic":
		return clashTUICProxyToLink(proxy, name)
	case "socks5", "socks":
		return clashSOCKSProxyToLink(proxy, name)
	case "http":
		return clashHTTPProxyToLink(proxy, name)
	case "wireguard":
		return clashWireGuardProxyToLink(proxy, name)
	case "anytls":
		return clashAnyTLSProxyToLink(proxy, name)
	default:
		return "", "", fmt.Errorf("unsupported clash proxy type: %s", nodeType)
	}
}


func clashVMessProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	cfg := map[string]interface{}{
		"v":    "2",
		"ps":   name,
		"add":  stringFromMap(proxy, "server"),
		"port": strconv.Itoa(intFromMap(proxy, "port", 0)),
		"id":   stringFromMap(proxy, "uuid"),
		"aid":  strconv.Itoa(intFromMap(proxy, "alterId", 0)),
		"scy":  stringFromMap(proxy, "cipher"),
		"net":  stringFromMap(proxy, "network"),
		"type": "none",
		"host": "",
		"path": "",
		"tls":  "",
		"sni":  stringFromMap(proxy, "servername"),
	}
	if cfg["scy"] == "" {
		cfg["scy"] = "auto"
	}
	if cfg["net"] == "" {
		cfg["net"] = "tcp"
	}
	if boolFromMap(proxy, "tls") {
		cfg["tls"] = "tls"
	}
	if httpOpts, ok := proxy["http-opts"].(map[string]interface{}); ok {
		cfg["net"] = "tcp"
		cfg["type"] = "http"
		if paths := stringSliceFromValue(httpOpts["path"]); len(paths) > 0 {
			cfg["path"] = paths[0]
		}
		if headers, ok := httpOpts["headers"].(map[string]interface{}); ok {
			if hosts := stringSliceFromValue(headers["Host"]); len(hosts) > 0 {
				cfg["host"] = hosts[0]
			}
		}
	}
	if wsOpts, ok := proxy["ws-opts"].(map[string]interface{}); ok {
		cfg["net"] = "ws"
		cfg["path"] = stringFromMap(wsOpts, "path")
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			cfg["host"] = stringFromMap(headers, "Host")
		}
	}
	if grpcOpts, ok := proxy["grpc-opts"].(map[string]interface{}); ok {
		cfg["net"] = "grpc"
		cfg["path"] = stringFromMap(grpcOpts, "grpc-service-name")
	}
	if h2Opts, ok := proxy["h2-opts"].(map[string]interface{}); ok {
		cfg["net"] = "h2"
		cfg["path"] = stringFromMap(h2Opts, "path")
		if hosts := stringSliceFromValue(h2Opts["host"]); len(hosts) > 0 {
			cfg["host"] = hosts[0]
		}
	}
	if huOpts, ok := proxy["httpupgrade-opts"].(map[string]interface{}); ok {
		cfg["net"] = "httpupgrade"
		cfg["path"] = stringFromMap(huOpts, "path")
		cfg["host"] = stringFromMap(huOpts, "host")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", "", err
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(data), "vmess", nil
}


func clashVLESSProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 0)
	uuid := stringFromMap(proxy, "uuid")
	if host == "" || port <= 0 || uuid == "" {
		return "", "", fmt.Errorf("invalid vless proxy")
	}
	q := url.Values{}
	network := stringFromMap(proxy, "network")
	if network != "" {
		q.Set("type", network)
	}
	if wsOpts, ok := proxy["ws-opts"].(map[string]interface{}); ok {
		q.Set("type", "ws")
		if path := stringFromMap(wsOpts, "path"); path != "" {
			q.Set("path", path)
		}
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			if hostHeader := stringFromMap(headers, "Host"); hostHeader != "" {
				q.Set("host", hostHeader)
			}
		}
	}
	if grpcOpts, ok := proxy["grpc-opts"].(map[string]interface{}); ok {
		q.Set("type", "grpc")
		if serviceName := stringFromMap(grpcOpts, "grpc-service-name"); serviceName != "" {
			q.Set("serviceName", serviceName)
		}
	}
	if h2Opts, ok := proxy["h2-opts"].(map[string]interface{}); ok {
		q.Set("type", "h2")
		if path := stringFromMap(h2Opts, "path"); path != "" {
			q.Set("path", path)
		}
		if hosts := stringSliceFromValue(h2Opts["host"]); len(hosts) > 0 {
			q.Set("host", hosts[0])
		}
	}
	if huOpts, ok := proxy["httpupgrade-opts"].(map[string]interface{}); ok {
		q.Set("type", "httpupgrade")
		if path := stringFromMap(huOpts, "path"); path != "" {
			q.Set("path", path)
		}
		if hostHeader := stringFromMap(huOpts, "host"); hostHeader != "" {
			q.Set("host", hostHeader)
		}
	}
	if boolFromMap(proxy, "tls") {
		q.Set("security", "tls")
	}
	if sni := stringFromMap(proxy, "servername"); sni != "" {
		q.Set("sni", sni)
	}
	if fp := stringFromMap(proxy, "client-fingerprint"); fp != "" {
		q.Set("fp", fp)
	}
	if flow := stringFromMap(proxy, "flow"); flow != "" {
		q.Set("flow", flow)
	}
	if alpn := stringSliceFromValue(proxy["alpn"]); len(alpn) > 0 {
		q.Set("alpn", strings.Join(alpn, ","))
	}
	if boolFromMap(proxy, "skip-cert-verify") {
		q.Set("allowInsecure", "1")
	}
	link := fmt.Sprintf("vless://%s@%s:%d", uuid, host, port)
	if encoded := q.Encode(); encoded != "" {
		link += "?" + encoded
	}
	return link + encodeNameFragment(name), "vless", nil
}


func clashTrojanProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 0)
	password := stringFromMap(proxy, "password")
	if host == "" || port <= 0 || password == "" {
		return "", "", fmt.Errorf("invalid trojan proxy")
	}
	q := url.Values{}
	if sni := stringFromMap(proxy, "sni"); sni != "" {
		q.Set("sni", sni)
	}
	if network := stringFromMap(proxy, "network"); network != "" {
		q.Set("type", network)
	}
	if wsOpts, ok := proxy["ws-opts"].(map[string]interface{}); ok {
		q.Set("type", "ws")
		if path := stringFromMap(wsOpts, "path"); path != "" {
			q.Set("path", path)
		}
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			if hostHeader := stringFromMap(headers, "Host"); hostHeader != "" {
				q.Set("host", hostHeader)
			}
		}
	}
	if grpcOpts, ok := proxy["grpc-opts"].(map[string]interface{}); ok {
		q.Set("type", "grpc")
		if serviceName := stringFromMap(grpcOpts, "grpc-service-name"); serviceName != "" {
			q.Set("serviceName", serviceName)
		}
	}
	if boolFromMap(proxy, "skip-cert-verify") {
		q.Set("allowInsecure", "1")
	}
	if alpn := stringSliceFromValue(proxy["alpn"]); len(alpn) > 0 {
		q.Set("alpn", strings.Join(alpn, ","))
	}
	if fp := stringFromMap(proxy, "client-fingerprint"); fp != "" {
		q.Set("fp", fp)
	}
	link := fmt.Sprintf("trojan://%s@%s:%d", url.QueryEscape(password), host, port)
	if encoded := q.Encode(); encoded != "" {
		link += "?" + encoded
	}
	return link + encodeNameFragment(name), "trojan", nil
}


func clashShadowsocksProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 0)
	cipher := stringFromMap(proxy, "cipher")
	password := stringFromMap(proxy, "password")
	if host == "" || port <= 0 || cipher == "" {
		return "", "", fmt.Errorf("invalid shadowsocks proxy")
	}
	userinfo := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cipher, password)))
	link := fmt.Sprintf("ss://%s@%s:%d", userinfo, host, port)
	if plugin := stringFromMap(proxy, "plugin"); plugin != "" {
		parts := []string{plugin}
		if opts, ok := proxy["plugin-opts"].(map[string]interface{}); ok {
			for k, v := range opts {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
		}
		q := url.Values{}
		q.Set("plugin", strings.Join(parts, ";"))
		link += "?" + q.Encode()
	}
	return link + encodeNameFragment(name), "ss", nil
}


func clashSSRProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 0)
	cipher := stringFromMap(proxy, "cipher")
	password := stringFromMap(proxy, "password")
	protocol := stringFromMap(proxy, "protocol")
	obfs := stringFromMap(proxy, "obfs")
	if host == "" || port <= 0 || cipher == "" || password == "" {
		return "", "", fmt.Errorf("invalid ssr proxy")
	}
	if protocol == "" {
		protocol = "origin"
	}
	if obfs == "" {
		obfs = "plain"
	}
	base := fmt.Sprintf("%s:%d:%s:%s:%s:%s", host, port, protocol, cipher, obfs, base64.RawURLEncoding.EncodeToString([]byte(password)))
	params := url.Values{}
	params.Set("remarks", base64.RawURLEncoding.EncodeToString([]byte(name)))
	if protocolParam := stringFromMap(proxy, "protocol-param"); protocolParam != "" {
		params.Set("protoparam", base64.RawURLEncoding.EncodeToString([]byte(protocolParam)))
	}
	if obfsParam := stringFromMap(proxy, "obfs-param"); obfsParam != "" {
		params.Set("obfsparam", base64.RawURLEncoding.EncodeToString([]byte(obfsParam)))
	}
	full := base + "/?" + params.Encode()
	return "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(full)), "ssr", nil
}


func clashHysteriaProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 0)
	if host == "" || port <= 0 {
		return "", "", fmt.Errorf("invalid hysteria proxy")
	}
	q := url.Values{}
	if auth := stringFromMap(proxy, "auth-str"); auth != "" {
		q.Set("auth", auth)
	}
	if peer := stringFromMap(proxy, "sni"); peer != "" {
		q.Set("peer", peer)
	}
	if boolFromMap(proxy, "skip-cert-verify") {
		q.Set("insecure", "1")
	}
	if up := stringFromMap(proxy, "up"); up != "" {
		q.Set("upmbps", up)
	}
	if down := stringFromMap(proxy, "down"); down != "" {
		q.Set("downmbps", down)
	}
	if protocol := stringFromMap(proxy, "protocol"); protocol != "" {
		q.Set("protocol", protocol)
	}
	if alpn := stringSliceFromValue(proxy["alpn"]); len(alpn) > 0 {
		q.Set("alpn", strings.Join(alpn, ","))
	}
	if obfs := stringFromMap(proxy, "obfs"); obfs != "" {
		q.Set("obfs", obfs)
	}
	link := fmt.Sprintf("hysteria://%s:%d", host, port)
	if encoded := q.Encode(); encoded != "" {
		link += "?" + encoded
	}
	return link + encodeNameFragment(name), "hysteria", nil
}


func clashHysteria2ProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 0)
	password := stringFromMap(proxy, "password")
	if host == "" || port <= 0 {
		return "", "", fmt.Errorf("invalid hysteria2 proxy")
	}
	q := url.Values{}
	if sni := stringFromMap(proxy, "sni"); sni != "" {
		q.Set("sni", sni)
	}
	if boolFromMap(proxy, "skip-cert-verify") {
		q.Set("insecure", "1")
	}
	if alpn := stringSliceFromValue(proxy["alpn"]); len(alpn) > 0 {
		q.Set("alpn", strings.Join(alpn, ","))
	}
	if fp := stringFromMap(proxy, "client-fingerprint"); fp != "" {
		q.Set("fp", fp)
	}
	if obfs := stringFromMap(proxy, "obfs"); obfs != "" {
		q.Set("obfs", obfs)
	}
	if obfsPassword := stringFromMap(proxy, "obfs-password"); obfsPassword != "" {
		q.Set("obfs-password", obfsPassword)
	}
	link := fmt.Sprintf("hysteria2://%s@%s:%d", url.QueryEscape(password), host, port)
	if encoded := q.Encode(); encoded != "" {
		link += "?" + encoded
	}
	return link + encodeNameFragment(name), "hysteria2", nil
}


func clashTUICProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 0)
	uuid := stringFromMap(proxy, "uuid")
	password := stringFromMap(proxy, "password")
	if host == "" || port <= 0 || uuid == "" {
		return "", "", fmt.Errorf("invalid tuic proxy")
	}
	q := url.Values{}
	if cc := stringFromMap(proxy, "congestion-controller"); cc != "" {
		q.Set("congestion_control", cc)
	}
	if alpn := stringSliceFromValue(proxy["alpn"]); len(alpn) > 0 {
		q.Set("alpn", strings.Join(alpn, ","))
	}
	if sni := stringFromMap(proxy, "sni"); sni != "" {
		q.Set("sni", sni)
	}
	userInfo := uuid
	if password != "" {
		userInfo += ":" + password
	}
	link := fmt.Sprintf("tuic://%s@%s:%d", userInfo, host, port)
	if encoded := q.Encode(); encoded != "" {
		link += "?" + encoded
	}
	return link + encodeNameFragment(name), "tuic", nil
}


func clashSOCKSProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 1080)
	if host == "" {
		return "", "", fmt.Errorf("invalid socks proxy")
	}
	username := stringFromMap(proxy, "username")
	password := stringFromMap(proxy, "password")
	link := "socks5://"
	if username != "" {
		link += url.QueryEscape(username)
		if password != "" {
			link += ":" + url.QueryEscape(password)
		}
		link += "@"
	}
	link += fmt.Sprintf("%s:%d", host, port)
	return link + encodeNameFragment(name), "socks5", nil
}


func clashHTTPProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 80)
	if host == "" {
		return "", "", fmt.Errorf("invalid http proxy")
	}
	scheme := "http://"
	if boolFromMap(proxy, "tls") {
		scheme = "https://"
	}
	username := stringFromMap(proxy, "username")
	password := stringFromMap(proxy, "password")
	link := scheme
	if username != "" {
		link += url.QueryEscape(username)
		if password != "" {
			link += ":" + url.QueryEscape(password)
		}
		link += "@"
	}
	link += fmt.Sprintf("%s:%d", host, port)
	return link + encodeNameFragment(name), "http", nil
}


func clashWireGuardProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 51820)
	privateKey := stringFromMap(proxy, "private-key")
	if host == "" || privateKey == "" {
		return "", "", fmt.Errorf("invalid wireguard proxy")
	}
	q := url.Values{}
	if publicKey := stringFromMap(proxy, "public-key"); publicKey != "" {
		q.Set("publicKey", publicKey)
	}
	if ip := stringFromMap(proxy, "ip"); ip != "" {
		if ipv6 := stringFromMap(proxy, "ipv6"); ipv6 != "" {
			q.Set("ip", ip+","+ipv6)
		} else {
			q.Set("ip", ip)
		}
	} else if ipv6 := stringFromMap(proxy, "ipv6"); ipv6 != "" {
		q.Set("ipv6", ipv6)
	}
	if mtu := stringFromMap(proxy, "mtu"); mtu != "" {
		q.Set("mtu", mtu)
	}
	if psk := stringFromMap(proxy, "preshared-key"); psk != "" {
		q.Set("presharedKey", psk)
	}
	if reserved := stringFromMap(proxy, "reserved"); reserved != "" {
		q.Set("reserved", reserved)
	}
	if keepalive := stringFromMap(proxy, "keepalive"); keepalive != "" {
		q.Set("keepalive", keepalive)
	}
	link := fmt.Sprintf("wg://%s@%s:%d", url.QueryEscape(privateKey), host, port)
	if encoded := q.Encode(); encoded != "" {
		link += "?" + encoded
	}
	return link + encodeNameFragment(name), "wireguard", nil
}


func clashAnyTLSProxyToLink(proxy map[string]interface{}, name string) (string, string, error) {
	host := stringFromMap(proxy, "server")
	port := intFromMap(proxy, "port", 443)
	password := stringFromMap(proxy, "password")
	if host == "" || password == "" {
		return "", "", fmt.Errorf("invalid anytls proxy")
	}
	q := url.Values{}
	if sni := stringFromMap(proxy, "sni"); sni != "" {
		q.Set("sni", sni)
	}
	link := fmt.Sprintf("anytls://%s@%s:%d", url.QueryEscape(password), host, port)
	if encoded := q.Encode(); encoded != "" {
		link += "?" + encoded
	}
	return link + encodeNameFragment(name), "anytls", nil
}

