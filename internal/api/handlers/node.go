package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ListNodes returns paginated active nodes.
// If the request carries a valid auth token (user_id > 0), extra fields are included.
func ListNodes(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	var total int64
	query := db.Model(&models.Node{}).Where("is_active = ?", true)

	// optional filters
	if region := c.Query("region"); region != "" {
		query = query.Where("region = ?", region)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	var nodes []models.Node
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&nodes)

	// If unauthenticated, strip sensitive config
	userID := c.GetUint("user_id")
	if userID == 0 {
		for i := range nodes {
			nodes[i].Config = nil
		}
	}

	utils.SuccessPage(c, nodes, total, p.Page, p.PageSize)
}

// GetNodeStats returns node counts grouped by status and region.
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

	var byStatus []StatusCount
	db.Model(&models.Node{}).Where("is_active = ?", true).
		Select("status, count(*) as count").Group("status").Scan(&byStatus)

	var byRegion []RegionCount
	db.Model(&models.Node{}).Where("is_active = ?", true).
		Select("region, count(*) as count").Group("region").Scan(&byRegion)

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
	conn.Close()
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
	db.Model(&node).Updates(map[string]interface{}{
		"status": status, "latency": latency, "last_test": &now,
	})

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

	var results []Result
	now := time.Now()
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		latency, reachable := testNodeConnectivity(*node.Config)
		status := "offline"
		if reachable {
			status = "online"
		}
		db.Model(&node).Updates(map[string]interface{}{
			"status": status, "latency": latency, "last_test": &now,
		})
		results = append(results, Result{
			NodeID: node.ID, Name: node.Name,
			Status: status, Latency: latency, Reachable: reachable,
		})
	}

	utils.Success(c, gin.H{"tested": len(results), "results": results})
}
