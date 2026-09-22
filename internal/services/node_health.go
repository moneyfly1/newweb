package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// 节点状态自动刷新（对标 Xboard 节点主动上报：CBoard 节点是订阅源，无法上报，
// 由调度器定期探测并回写，让节点页/订阅列表始终显示新鲜状态，无需手动点击测试）。

// isUDPBasedProtocol 判断是否 UDP 系协议（QUIC / WireGuard）。
//
// 这些协议只监听 UDP：用 TCP 去连它的端口必然失败（端口上根本没有 TCP 监听），
// 于是线上 114 条 hysteria2 节点被全线判为离线、延迟 0。
func isUDPBasedProtocol(config string) bool {
	switch detectNodeTypeFromLink(strings.TrimSpace(config)) {
	case "hysteria", "hysteria2", "tuic", "wireguard":
		return true
	}
	return false
}

// TestNodeConnectivity 探测节点可达性并返回延迟。
//
// TCP 系协议（vmess/vless/trojan/ss/ssr/socks5/http/anytls）：TCP 连接即可判断。
// UDP 系协议（hysteria/hysteria2/tuic/wireguard）：TCP 探测毫无意义——
// 只做「域名可解析 + UDP 可发」的弱校验，可达性不做断言（第二个返回值为 false
// 但也不代表离线，调用方据 isUDPBasedProtocol 决定是否回写状态）。
func TestNodeConnectivity(config string) (latencyMs int, reachable bool) {
	addr, err := extractHostPort(config)
	if err != nil {
		return 0, false
	}

	if isUDPBasedProtocol(config) {
		start := time.Now()
		conn, err := net.DialTimeout("udp", addr, 5*time.Second)
		if err != nil {
			return 0, false
		}
		_ = conn.Close()
		// UDP 无握手，「连上」不代表服务端在跑；这里只回报探测本身可用
		return int(time.Since(start).Milliseconds()), true
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return 0, false
	}
	_ = conn.Close()
	return int(time.Since(start).Milliseconds()), true
}

// ExtractNodeAddressForTest returns host:port for a node config (empty if invalid).
func ExtractNodeAddressForTest(config string) string {
	addr, err := extractHostPort(config)
	if err != nil {
		return ""
	}
	return addr
}

// detectNodeTypeFromLink 按链接 scheme 推断节点类型（与订阅生成用的类型名一致）。
func detectNodeTypeFromLink(link string) string {
	scheme := link
	if i := strings.Index(link, "://"); i > 0 {
		scheme = strings.ToLower(link[:i])
	}
	switch scheme {
	case "vmess":
		return "vmess"
	case "vless":
		return "vless"
	case "trojan":
		return "trojan"
	case "ss":
		return "ss"
	case "ssr":
		return "ssr"
	case "hysteria":
		return "hysteria"
	case "hysteria2", "hy2":
		return "hysteria2"
	case "tuic":
		return "tuic"
	case "socks", "socks5":
		return "socks5"
	case "http", "https", "naive", "naive+https":
		return "http"
	case "anytls":
		return "anytls"
	case "wg", "wireguard":
		return "wireguard"
	}
	return ""
}

// extractHostPort tries to extract host:port from a node config link.
//
// 关键：必须与「生成订阅」走同一套解析（NodeConfigToClashMap）。
// 此前这里只认 vmess/vless/trojan/ss，其余协议一律返回 "unsupported protocol" 被判离线——
// 线上 hysteria2 节点 114 条全部显示离线、延迟 0，客户导入 SOCKS/Hysteria2 节点后
// 看到的就是「一直超时」，而订阅里其实是能用的。
func extractHostPort(config string) (string, error) {
	config = strings.TrimSpace(config)
	if config == "" {
		return "", fmt.Errorf("empty node config")
	}

	// 1) 先用与订阅生成一致的解析器取 server/port
	if typ := detectNodeTypeFromLink(config); typ != "" {
		if proxy, err := NodeConfigToClashMap(typ, config, "probe"); err == nil {
			host := stringFromMap(proxy, "server")
			port := intFromMap(proxy, "port", 0)
			if host != "" && port > 0 {
				return net.JoinHostPort(host, strconv.Itoa(port)), nil
			}
		}
	}

	// 2) 兜底：下面的老逻辑（处理个别非标准写法）

	// vmess:// is base64-encoded JSON
	if strings.HasPrefix(config, "vmess://") {
		raw := strings.TrimPrefix(config, "vmess://")
		raw = strings.SplitN(raw, "#", 2)[0]
		decoded, err := base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(raw)
		}
		if err != nil {
			return "", fmt.Errorf("vmess base64 decode failed")
		}
		var obj map[string]interface{}
		if err := json.Unmarshal(decoded, &obj); err != nil {
			return "", err
		}
		host, _ := obj["add"].(string)
		port := fmt.Sprintf("%v", obj["port"])
		if host == "" {
			return "", fmt.Errorf("vmess: no host")
		}
		return net.JoinHostPort(host, port), nil
	}

	// vless://, trojan://, ss:// — standard URI format
	for _, prefix := range []string{"vless://", "trojan://", "ss://"} {
		if strings.HasPrefix(config, prefix) {
			// ss:// may have base64-encoded userinfo
			if prefix == "ss://" {
				raw := strings.TrimPrefix(config, "ss://")
				// Remove fragment
				raw = strings.SplitN(raw, "#", 2)[0]
				// Try to find @ separator
				if idx := strings.LastIndex(raw, "@"); idx >= 0 {
					hostPort := raw[idx+1:]
					hostPort = strings.SplitN(hostPort, "?", 2)[0]
					hostPort = strings.SplitN(hostPort, "/", 2)[0]
					if _, _, err := net.SplitHostPort(hostPort); err == nil {
						return hostPort, nil
					}
				}
			}
			u, err := url.Parse(config)
			if err != nil {
				return "", err
			}
			host := u.Hostname()
			port := u.Port()
			if port == "" {
				port = "443"
			}
			if host == "" {
				return "", fmt.Errorf("no host in URL")
			}
			return net.JoinHostPort(host, port), nil
		}
	}

	return "", fmt.Errorf("unsupported protocol")
}

// AutoTestActiveNodes probes all active nodes and writes back status/latency/last_test.
// Called by the scheduler so node states stay fresh without manual clicks.
// Returns (tested, online).
func AutoTestActiveNodes() (int, int) {
	db := database.GetDB()
	var nodes []models.Node
	// 固定在线的节点不做探测：它们的可达性与服务端所在地区无关（例如仅中国境内可达），
	// 探测必然失败并把状态改写回 offline，等于把管理员的设置白设了。
	db.Where("is_active = ? AND pinned_online = ? AND config IS NOT NULL AND config != ''", true, false).Find(&nodes)

	if len(nodes) == 0 {
		return 0, 0
	}

	now := time.Now()
	var (
		mu      sync.Mutex
		results []models.Node
		wg      sync.WaitGroup
	)
	// 限制并发探测数量，避免大量 goroutine 耗尽资源
	sem := make(chan struct{}, 20)

	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(n models.Node) {
			defer wg.Done()
			defer func() { <-sem }()
			latency, reachable := TestNodeConnectivity(*n.Config)
			status := models.NodeStatusOffline
			if reachable {
				status = models.NodeStatusOnline
			}
			if _, err := extractHostPort(*n.Config); err != nil {
				// 解析不出 host:port（未知/新增协议）时保持原状态：
				// 探测能力不足不等于节点挂了，硬判离线会让客户以为节点全废。
				mu.Lock()
				results = append(results, n)
				mu.Unlock()
				return
			}
			// 只回写发生变化的状态，减少无谓的 DB 写入
			if n.Status != status || n.Latency != latency {
				if err := db.Model(&n).Updates(map[string]interface{}{
					"status": status, "latency": latency, "last_test": &now,
				}).Error; err != nil {
					utils.SysError("node", fmt.Sprintf("自动刷新节点状态失败: node=%d err=%v", n.ID, err))
				}
			}
			mu.Lock()
			results = append(results, n)
			mu.Unlock()
		}(node)
	}
	wg.Wait()

	online := 0
	for _, n := range results {
		if n.Status == models.NodeStatusOnline {
			online++
		}
	}
	if len(results) > 0 {
		log.Printf("[NodeHealth] 自动刷新完成: 已测 %d 个节点, 在线 %d 个", len(results), online)
	}
	return len(results), online
}
