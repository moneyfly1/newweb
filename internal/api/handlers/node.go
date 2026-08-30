package handlers

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
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
		now := time.Now()
		var activeSub int64
		// 有效订阅：status=active 且 is_active=1 且未过期（与订阅接口判定一致，防止免费/过期账号读取节点真实配置）
		db.Model(&models.Subscription{}).Where("user_id = ? AND status = ? AND is_active = ? AND expire_time > ?", userID, models.SubStatusActive, true, now).Count(&activeSub)
		hasActiveSub = activeSub > 0
		if hasActiveSub {
			var sub models.Subscription
			if err := db.Where("user_id = ? AND status = ? AND is_active = ? AND expire_time > ?", userID, models.SubStatusActive, true, now).First(&sub).Error; err == nil {
				customNodes, isDedicatedOnly, _ = fetchUserCustomNodes(db, userID, sub.ExpireTime)
			}
		}
	}

	// --- Determine full node list and stats ---
	// 过滤条件（region/status/protocol）应用到完整列表，保证分页与统计一致
	region := c.Query("region")
	statusFilter := c.Query("status")
	protocol := c.Query("protocol")
	applyFilters := func(ns []models.Node) []models.Node {
		out := make([]models.Node, 0, len(ns))
		for _, n := range ns {
			if region != "" && n.Region != region {
				continue
			}
			if statusFilter != "" && n.Status != statusFilter {
				continue
			}
			if protocol != "" && n.Type != protocol {
				continue
			}
			out = append(out, n)
		}
		return out
	}

	var allNodes []models.Node
	if isDedicatedOnly {
		// Dedicated-only mode: only custom nodes, no public nodes
		allNodes = applyFilters(customNodes)
	} else {
		// Normal mode: custom nodes first, then public nodes
		var allPublic []models.Node
		pubListQuery := db.Model(&models.Node{}).Where("is_active = ?", true)
		if region != "" {
			pubListQuery = pubListQuery.Where("region = ?", region)
		}
		if statusFilter != "" {
			pubListQuery = pubListQuery.Where("status = ?", statusFilter)
		}
		if protocol != "" {
			pubListQuery = pubListQuery.Where("type = ?", protocol)
		}
		pubListQuery.Order("order_index ASC").Find(&allPublic)
		allNodes = append(applyFilters(customNodes), allPublic...)
	}

	// Compute global stats from full node list
	totalAll := int64(len(allNodes))
	var onlineCount int64
	regionSet := make(map[string]struct{})
	for _, n := range allNodes {
		if n.Status == models.NodeStatusOnline {
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
		if err := db.Where("user_id = ? AND status = ? AND is_active = ? AND expire_time > ?", userID, models.SubStatusActive, true, time.Now()).First(&sub).Error; err == nil {
			customNodes, _, _ := fetchUserCustomNodes(db, userID, sub.ExpireTime)
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

	// 与 ListNodes 一致：仅对具有有效订阅的用户返回完整节点配置，
	// 防止无订阅/免费账号枚举节点真实服务器地址（核心资产泄露）。
	userID := c.GetUint("user_id")
	hasActiveSub := false
	if userID > 0 {
		var activeSub int64
		db.Model(&models.Subscription{}).Where("user_id = ? AND status = ? AND is_active = ? AND expire_time > ?", userID, models.SubStatusActive, true, time.Now()).Count(&activeSub)
		hasActiveSub = activeSub > 0
	}
	if !hasActiveSub {
		node.Config = nil
	}

	utils.Success(c, node)
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

	latency, reachable := services.TestNodeConnectivity(*node.Config)
	now := time.Now()
	status := models.NodeStatusOffline
	if reachable {
		status = models.NodeStatusOnline
	}
	// 仅管理员测试结果回写全局节点状态；普通用户测试只读不写库，
	// 防止任意登录用户批量探测/篡改共享节点状态（越权写面）。
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok && u.IsAdmin {
			if err := db.Model(&node).Updates(map[string]interface{}{
				"status": status, "latency": latency, "last_test": &now,
			}).Error; err != nil {
				utils.InternalError(c, "更新节点测试结果失败")
				return
			}
		}
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
			latency, reachable := services.TestNodeConnectivity(*n.Config)
			status := models.NodeStatusOffline
			if reachable {
				status = models.NodeStatusOnline
			}
			// 普通用户批量测速不写库（防止任意登录用户篡改全局节点状态），仅管理员可回写
			if user, exists := c.Get("user"); exists {
				if u, ok := user.(*models.User); ok && u.IsAdmin {
					if err := db.Model(&n).Updates(map[string]interface{}{
						"status": status, "latency": latency, "last_test": &now,
					}).Error; err != nil {
						utils.SysError("node", fmt.Sprintf("批量更新节点测试结果失败: node=%d err=%v", n.ID, err))
					}
				}
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
