package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ListNodes returns paginated active nodes.
// If the user has assigned custom/dedicated nodes, they are included in the list.
// Stats in the response reflect the global totals, not just the current page.
func ListNodes(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)
	userID := c.GetUint("user_id")

	// --- Fetch public nodes ---
	var totalPublic int64
	pubQuery := db.Model(&models.Node{}).Where("is_active = ?", true)
	if region := c.Query("region"); region != "" {
		pubQuery = pubQuery.Where("region = ?", region)
	}
	if status := c.Query("status"); status != "" {
		pubQuery = pubQuery.Where("status = ?", status)
	}
	pubQuery.Count(&totalPublic)

	// --- Fetch user custom nodes (if authenticated and has subscription) ---
	var customNodes []models.Node
	var hasActiveSub bool
	var isDedicatedOnly bool
	if userID > 0 {
		var activeSub int64
		db.Model(&models.Subscription{}).Where("user_id = ? AND status = ?", userID, "active").Count(&activeSub)
		hasActiveSub = activeSub > 0
		if hasActiveSub {
			var sub models.Subscription
			if err := db.Where("user_id = ? AND status = ?", userID, "active").First(&sub).Error; err == nil {
				customNodes, isDedicatedOnly = fetchUserCustomNodes(db, userID, sub.ExpireTime)
			}
		}
	}

	// --- Determine full node list and stats ---
	var allNodes []models.Node
	if isDedicatedOnly {
		// Dedicated-only mode: only custom nodes, no public nodes
		allNodes = customNodes
	} else {
		// Normal mode: custom nodes first, then public nodes
		var allPublic []models.Node
		db.Model(&models.Node{}).Where("is_active = ?", true).Order("order_index ASC").Find(&allPublic)
		allNodes = append(customNodes, allPublic...)
	}

	// Compute global stats from full node list
	totalAll := int64(len(allNodes))
	var onlineCount int64
	regionSet := make(map[string]struct{})
	for _, n := range allNodes {
		if n.Status == "online" {
			onlineCount++
		}
		if n.Region != "" {
			regionSet[n.Region] = struct{}{}
		}
	}

	// --- Paginate the combined list ---
	offset := p.Offset()
	limit := p.PageSize
	if limit <= 0 {
		limit = 20
	}
	if offset >= len(allNodes) {
		allNodes = nil
	} else {
		end := offset + limit
		if end > len(allNodes) {
			end = len(allNodes)
		}
		allNodes = allNodes[offset:end]
	}

	// --- Strip configs from unauthenticated users ---
	if !hasActiveSub {
		for i := range allNodes {
			allNodes[i].Config = nil
		}
	}

	// Map Type to protocol for frontend compatibility
	type NodeResponse struct {
		models.Node
		Protocol string `json:"protocol"`
	}
	result := make([]NodeResponse, len(allNodes))
	for i, n := range allNodes {
		result[i] = NodeResponse{Node: n, Protocol: n.Type}
	}

	utils.Success(c, gin.H{
		"items":     result,
		"total":     totalAll,
		"page":      p.Page,
		"page_size": p.PageSize,
		"stats": gin.H{
			"total":   totalAll,
			"online":  onlineCount,
			"regions": int64(len(regionSet)),
		},
	})
}

// GetNodeStats returns node counts grouped by status and region.
// Includes both public and user-specific custom nodes.
func GetNodeStats(c *gin.Context) {
	db := database.GetDB()

	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	type RegionCount struct {
		Region string `json:"region"`
		Count  int64  `json:"count"`
	}

	// Gather all nodes (public + user custom)
	var allNodes []models.Node
	db.Model(&models.Node{}).Where("is_active = ?", true).Find(&allNodes)

	userID := c.GetUint("user_id")
	if userID > 0 {
		var sub models.Subscription
		if err := db.Where("user_id = ? AND status = ?", userID, "active").First(&sub).Error; err == nil {
			customNodes, _ := fetchUserCustomNodes(db, userID, sub.ExpireTime)
			allNodes = append(allNodes, customNodes...)
		}
	}

	// Aggregate by status
	statusMap := make(map[string]int64)
	for _, n := range allNodes {
		statusMap[n.Status]++
	}
	byStatus := make([]StatusCount, 0, len(statusMap))
	for s, c := range statusMap {
		byStatus = append(byStatus, StatusCount{Status: s, Count: c})
	}

	// Aggregate by region
	regionMap := make(map[string]int64)
	for _, n := range allNodes {
		if n.Region != "" {
			regionMap[n.Region]++
		}
	}
	byRegion := make([]RegionCount, 0, len(regionMap))
	for r, c := range regionMap {
		byRegion = append(byRegion, RegionCount{Region: r, Count: c})
	}

	utils.Success(c, gin.H{"by_status": byStatus, "by_region": byRegion})
}

// GetNode returns a single node by ID.
func GetNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的节点ID")
		return
	}

	db := database.GetDB()
	var node models.Node
	if err := db.Where("id = ? AND is_active = ?", id, true).First(&node).Error; err != nil {
		utils.NotFound(c, "节点不存在")
		return
	}

	// Strip config for unauthenticated requests
	if c.GetUint("user_id") == 0 {
		node.Config = nil
	}

	utils.Success(c, node)
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

func extractNodeAddressForTest(config string) string {
	addr, err := extractHostPort(config)
	if err != nil {
		return ""
	}
	return addr
}

// testNodeConnectivity performs a TCP dial to the node and returns latency.
func testNodeConnectivity(config string) (latencyMs int, reachable bool) {
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

// TestNode performs a connectivity test on a single node.
func TestNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的节点ID")
		return
	}
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "节点不存在")
		return
	}
	if node.Config == nil || *node.Config == "" {
		utils.BadRequest(c, "节点无配置信息")
		return
	}

	latency, reachable := testNodeConnectivity(*node.Config)
	now := time.Now()
	status := "offline"
	if reachable {
		status = "online"
	}
	if err := db.Model(&node).Updates(map[string]interface{}{
		"status": status, "latency": latency, "last_test": &now,
	}).Error; err != nil {
		utils.InternalError(c, "更新节点测试结果失败")
		return
	}

	utils.Success(c, gin.H{
		"node_id":   node.ID,
		"name":      node.Name,
		"status":    status,
		"latency":   latency,
		"reachable": reachable,
	})
}

// BatchTestNodes tests multiple nodes at once.
func BatchTestNodes(c *gin.Context) {
	db := database.GetDB()
	var nodes []models.Node
	db.Where("is_active = ? AND config IS NOT NULL AND config != ''", true).Find(&nodes)

	type Result struct {
		NodeID    uint   `json:"node_id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		Latency   int    `json:"latency"`
		Reachable bool   `json:"reachable"`
	}

	var (
		results []Result
		mu      sync.Mutex
		wg      sync.WaitGroup
	)
	now := time.Now()

	// 限制并发测试数量，避免大量 goroutine 耗尽资源
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
			latency, reachable := testNodeConnectivity(*n.Config)
			status := "offline"
			if reachable {
				status = "online"
			}
			if err := db.Model(&n).Updates(map[string]interface{}{
				"status": status, "latency": latency, "last_test": &now,
			}).Error; err != nil {
				utils.SysError("node", fmt.Sprintf("批量更新节点测试结果失败: node=%d err=%v", n.ID, err))
			}
			mu.Lock()
			results = append(results, Result{
				NodeID: n.ID, Name: n.Name,
				Status: status, Latency: latency, Reachable: reachable,
			})
			mu.Unlock()
		}(node)
	}
	wg.Wait()

	utils.Success(c, gin.H{"tested": len(results), "results": results})
}
