package utils

import (
	"net"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// trustedProxies 可信代理 IP/CIDR 列表（由 SetTrustedProxies 配置）
var (
	trustedProxyCIDRs []*net.IPNet
	trustedProxyMu    sync.RWMutex
)

// SetTrustedProxies 配置可信代理列表，启动时调用一次即可。
// 传入 CIDR 或单 IP，例如 ["127.0.0.1/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "::1/128"]
// 以及 Cloudflare IP 段等。
func SetTrustedProxies(cidrs []string) {
	trustedProxyMu.Lock()
	defer trustedProxyMu.Unlock()
	trustedProxyCIDRs = nil
	for _, cidr := range cidrs {
		if !strings.Contains(cidr, "/") {
			// 单个 IP，自动加上掩码
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		if _, network, err := net.ParseCIDR(cidr); err == nil {
			trustedProxyCIDRs = append(trustedProxyCIDRs, network)
		}
	}
}

// isTrustedProxy 检查给定 IP 是否在可信代理列表中
func isTrustedProxy(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	trustedProxyMu.RLock()
	defer trustedProxyMu.RUnlock()
	for _, cidr := range trustedProxyCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// GetRealClientIP extracts the real client IP from proxy headers.
// 仅当直连 IP 属于可信代理时才信任转发头，否则使用直连 IP。
func GetRealClientIP(c *gin.Context) string {
	// 获取 TCP 层直连 IP（Gin 的 RemoteIP，不受 header 影响）
	directIP := c.RemoteIP()

	// 仅当直连 IP 是可信代理时，才读取转发头
	if isTrustedProxy(directIP) {
		if ip := firstValidIP(c.GetHeader("CF-Connecting-IP")); ip != "" {
			return ip
		}
		if ip := firstValidIP(c.GetHeader("X-Real-IP")); ip != "" {
			return ip
		}
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			parts := strings.SplitN(xff, ",", 2)
			if ip := firstValidIP(parts[0]); ip != "" {
				return ip
			}
		}
	}

	if ip := firstValidIP(directIP); ip != "" {
		return ip
	}
	return directIP
}

func firstValidIP(raw string) string {
	ip := strings.TrimSpace(raw)
	if ip == "" || net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

// IsInternalIP 判断地址是否属于回环 / 私有网段 / 链路本地（即「服务器内部」来源）。
//
// 用途：拦截内部请求（运维脚本、批量巡检、健康检查、本机 curl 127.0.0.1）
// 被登记成「用户设备」。真实用户经 nginx 或 Cloudflare 访问时，
// GetRealClientIP 会从 X-Forwarded-For / CF-Connecting-IP 取到公网 IP，
// 因此这里返回 true 的只可能是内部来源。
func IsInternalIP(ip string) bool {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast()
}
