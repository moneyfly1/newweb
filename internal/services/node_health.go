package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// 节点状态自动刷新（对标 Xboard 节点主动上报：CBoard 节点是订阅源，无法上报，
// 由调度器定期探测并回写，让节点页/订阅列表始终显示新鲜状态，无需手动点击测试）。

// TestNodeConnectivity performs a TCP dial to the node and returns latency.
func TestNodeConnectivity(config string) (latencyMs int, reachable bool) {
	addr, err := extractHostPort(config)
	if err != nil {
		return 0, false
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

// extractHostPort tries to extract host:port from a node config link.
func extractHostPort(config string) (string, error) {
	config = strings.TrimSpace(config)

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
	db.Where("is_active = ? AND config IS NOT NULL AND config != ''", true).Find(&nodes)

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
