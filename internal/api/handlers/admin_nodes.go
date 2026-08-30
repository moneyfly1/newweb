package handlers

import (
	"fmt"
	"strconv"
	"time"
	"cboard/v2/internal/cache"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func AdminListNodes(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.Node{})
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if region := c.Query("region"); region != "" {
		query = query.Where("region = ?", region)
	}
	if sourceIndex := c.Query("source_index"); sourceIndex != "" {
		if si, err := strconv.Atoi(sourceIndex); err == nil {
			query = query.Where("source_index = ?", si)
		}
	}
	if sourceURL := c.Query("source_url"); sourceURL != "" {
		query = query.Where("source_url = ?", sourceURL)
	}
	if isManual := c.Query("is_manual"); isManual != "" {
		query = query.Where("is_manual = ?", isManual == "1" || isManual == "true")
	}
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR region LIKE ? OR type LIKE ? OR description LIKE ? OR config LIKE ?", like, like, like, like, like)
	}

	var total int64
	query.Count(&total)

	var nodes []models.Node
	query.Select("id, name, region, type, status, load, speed, uptime, latency, description, is_recommended, is_active, is_manual, source_index, source_url, order_index, last_test, created_at, updated_at").
		Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&nodes)

	utils.SuccessPage(c, nodes, total, p.Page, p.PageSize)
}

func AdminCreateNode(c *gin.Context) {
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Region        string  `json:"region"`
		Type          string  `json:"type"`
		Status        string  `json:"status"`
		Description   *string `json:"description"`
		Config        *string `json:"config"`
		IsRecommended bool    `json:"is_recommended"`
		OrderIndex    int     `json:"order_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	node := models.Node{
		Name: req.Name, Region: req.Region, Type: req.Type, Status: req.Status,
		Description: req.Description, Config: req.Config, IsRecommended: req.IsRecommended,
		OrderIndex: req.OrderIndex, IsManual: true,
	}
	if err := database.GetDB().Create(&node).Error; err != nil {
		utils.InternalError(c, "创建节点失败")
		return
	}
	utils.CreateAuditLog(c, "create_node", "node", node.ID, fmt.Sprintf("创建节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, node)
}

func AdminUpdateNode(c *gin.Context) {
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

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	allowed := map[string]bool{
		"name": true, "region": true, "type": true, "status": true, "description": true,
		"config": true, "is_recommended": true, "is_active": true, "is_manual": true,
		"order_index": true, "source_index": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&node).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新节点失败")
		return
	}
	utils.CreateAuditLog(c, "update_node", "node", uint(id), fmt.Sprintf("更新节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, node)
}

func AdminGetNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的节点ID")
		return
	}
	var node models.Node
	if err := database.GetDB().First(&node, id).Error; err != nil {
		utils.NotFound(c, "节点不存在")
		return
	}
	utils.Success(c, node)
}

func AdminDeleteNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的节点ID")
		return
	}
	if err := database.GetDB().Delete(&models.Node{}, id).Error; err != nil {
		utils.InternalError(c, "删除节点失败")
		return
	}
	utils.CreateAuditLog(c, "delete_node", "node", uint(id), "删除节点")
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "节点已删除")
}

func AdminImportNodes(c *gin.Context) {
	var req struct {
		Type  string `json:"type" binding:"required"`
		URL   string `json:"url"`
		Links string `json:"links"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 统一导入流程（与专线导入共用 FetchAndParseImport/FilterExistingNodesByName）
	nodes, err := services.FetchAndParseImport(services.ImportSource{Type: req.Type, URL: req.URL, Links: req.Links})
	if err != nil {
		utils.BadRequest(c, "导入失败: "+err.Error())
		return
	}
	if len(nodes) == 0 {
		utils.BadRequest(c, "未找到有效的节点")
		return
	}

	db := database.GetDB()
	sourceURL := ""
	if req.Type == "subscription" {
		sourceURL = req.URL
	}
	// 预筛已存在节点（按 name 判重），剩余批量写入（单事务）
	toInsert := services.FilterExistingNodesByName(db, nodes, true, sourceURL)
	successCount := 0
	if len(toInsert) > 0 {
		if err := db.CreateInBatches(toInsert, 200).Error; err == nil {
			successCount = len(toInsert)
		}
	}

	cache.ClearAllSubscriptionCache()
	utils.CreateAuditLog(c, "import_nodes", "node", 0, "从链接导入节点")
	utils.Success(c, gin.H{
		"total":   len(nodes),
		"success": successCount,
		"message": "导入完成",
	})
}

func AdminTestNode(c *gin.Context) {
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
	status := models.NodeStatusOffline
	if reachable {
		status = models.NodeStatusOnline
	}
	if err := db.Model(&node).Updates(map[string]interface{}{
		"status": status, "latency": latency, "last_test": &now,
	}).Error; err != nil {
		utils.InternalError(c, "更新节点测试结果失败")
		return
	}

	utils.CreateAuditLog(c, "test_node", "node", node.ID, "测试节点延迟")
	utils.Success(c, gin.H{
		"node_id":   node.ID,
		"name":      node.Name,
		"status":    status,
		"latency":   latency,
		"reachable": reachable,
		"address":   extractNodeAddressForTest(*node.Config),
	})
}

// ==================== Custom Node Management ====================

func AdminBatchNodeAction(c *gin.Context) {
	var req struct {
		IDs    []uint `json:"ids" binding:"required"`
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择节点")
		return
	}

	db := database.GetDB()
	var affected int64

	switch req.Action {
	case "enable":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("is_active", true)
		affected = result.RowsAffected
	case "disable":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("is_active", false)
		affected = result.RowsAffected
	case "online":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("status", models.NodeStatusOnline)
		affected = result.RowsAffected
	case "offline":
		result := db.Model(&models.Node{}).Where("id IN ?", req.IDs).Update("status", models.NodeStatusOffline)
		affected = result.RowsAffected
	case "delete":
		result := db.Where("id IN ?", req.IDs).Delete(&models.Node{})
		affected = result.RowsAffected
	default:
		utils.BadRequest(c, "不支持的操作: "+req.Action)
		return
	}

	utils.CreateAuditLog(c, "batch_node_action", "node", 0, fmt.Sprintf("批量操作节点: %s, 影响 %d 个节点", req.Action, affected))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{"affected": affected, "action": req.Action})
}

// ==================== Check-In Stats ====================

var defaultProtocolFilter = map[string][]string{
	"clash_protocols":     {"vmess", "vless", "trojan", "ss", "ssr", "hysteria", "hysteria2", "tuic", "anytls", "socks5", "http", "wireguard"},
	"universal_protocols": {"vmess", "vless", "trojan", "ss", "ssr", "hysteria", "hysteria2", "tuic", "anytls", "socks", "socks5", "http", "wireguard"},
}

