package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/cache"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// stringField 从更新 map 里取字符串字段
func stringField(updates map[string]interface{}, key string) (string, bool) {
	v, ok := updates[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// toInt 把 JSON 解出来的数字（float64 / int / json.Number 字符串）统一成 int
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func AdminListCustomNodes(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)
	query := db.Model(&models.CustomNode{})

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		var matchedNodeIDs []uint
		db.Model(&models.UserCustomNode{}).
			Joins("JOIN users ON users.id = user_custom_nodes.user_id").
			Where("users.email LIKE ? OR users.username LIKE ?", like, like).
			Distinct().
			Pluck("user_custom_nodes.custom_node_id", &matchedNodeIDs)

		query = query.Where(
			db.Where("custom_nodes.name LIKE ? OR custom_nodes.display_name LIKE ? OR custom_nodes.domain LIKE ? OR custom_nodes.protocol LIKE ? OR CAST(custom_nodes.port AS CHAR) LIKE ?", like, like, like, like, like).
				Or("custom_nodes.id IN ?", matchedNodeIDs),
		)
	}

	// 协议 / 状态筛选（与节点管理页风格一致）
	if protocol := strings.TrimSpace(c.Query("protocol")); protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var nodes []models.CustomNode
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&nodes)

	utils.SuccessPage(c, nodes, total, p.Page, p.PageSize)
}

func AdminCreateCustomNode(c *gin.Context) {
	var req struct {
		Name             string     `json:"name" binding:"required"`
		DisplayName      string     `json:"display_name"`
		Protocol         string     `json:"protocol"`
		Domain           string     `json:"domain"`
		Port             int        `json:"port"`
		Config           string     `json:"config"`
		Status           string     `json:"status"`
		ExpireTime       *time.Time `json:"expire_time"`
		FollowUserExpire bool       `json:"follow_user_expire"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// 与编辑保持同一套规则：下发以配置链接为准，域名/端口只是派生字段。
	// 创建时若填了链接 → 按链接反推；若只填了域名/端口 → 把它们写进链接，避免创建完就「不生效」。
	domain, port, protocol, config := req.Domain, req.Port, req.Protocol, req.Config
	warning := ""
	if strings.TrimSpace(config) != "" {
		if h, p, err := services.NodeLinkHostPort(config); err == nil {
			domain = h
			if p > 0 {
				port = p
			}
			if typ := services.DetectNodeTypeFromLink(config); typ != "" {
				protocol = typ
			}
		} else {
			warning = "配置链接无法解析（" + err.Error() + "），节点可能无法下发，请检查链接格式"
		}
	} else if domain != "" && port > 0 {
		warning = "只填了域名/端口但没有配置链接，用户拿不到这个节点，请在「配置信息」里填入链接"
	}

	node := models.CustomNode{
		Name:             req.Name,
		DisplayName:      req.DisplayName,
		Domain:           domain,
		Port:             port,
		Protocol:         protocol,
		Status:           req.Status,
		Config:           config,
		ExpireTime:       req.ExpireTime,
		FollowUserExpire: req.FollowUserExpire,
	}
	if err := database.GetDB().Create(&node).Error; err != nil {
		utils.InternalError(c, "创建专线节点失败")
		return
	}
	utils.CreateAuditLog(c, "create_custom_node", "custom_node", node.ID, fmt.Sprintf("创建专线节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
	if warning != "" {
		utils.Success(c, gin.H{"node": node, "warning": warning})
		return
	}
	utils.Success(c, node)
}

func AdminUpdateCustomNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "专线节点不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	allowed := map[string]bool{
		"name": true, "display_name": true, "protocol": true, "domain": true, "port": true,
		"config": true, "status": true, "is_active": true, "expire_time": true,
		"follow_user_expire": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if !allowed[k] {
			continue
		}
		if v == nil {
			continue // null 剔除，防 GORM Updates(map) 写 SQL NULL 清空字段
		}
		updates[k] = v
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	// 下发完全以「配置链接」为准，domain/port 只是派生字段。
	// 因此这里要把两者的关系理清，否则会出现「列表改了、用户拿到的还是旧地址」的假生效：
	//   · 改了配置链接 → 按链接反推 domain/port/协议（列表显示真相）
	//   · 只改了域名/端口 → 把配置链接里的 host:port 一起改写（下发真正生效）
	//   · 链接形态认不出来 → 保留 domain/port 的展示更新，但明确告诉管理员下发仍以原链接为准
	warning := ""
	newConfig, hasConfig := stringField(updates, "config")
	oldConfig := node.Config
	if hasConfig && newConfig != "" && newConfig != oldConfig {
		if host, port, err := services.NodeLinkHostPort(newConfig); err == nil {
			updates["domain"] = host
			if port > 0 {
				updates["port"] = port
			}
			if typ := services.DetectNodeTypeFromLink(newConfig); typ != "" {
				updates["protocol"] = typ
			}
		} else {
			warning = "配置链接无法解析（" + err.Error() + "），节点可能无法下发，请检查链接格式"
		}
	} else if _, hasDomain := updates["domain"]; hasDomain || updates["port"] != nil {
		targetHost, targetPort := node.Domain, node.Port
		if v, ok := stringField(updates, "domain"); ok && v != "" {
			targetHost = v
		}
		if v, ok := updates["port"]; ok {
			if p, ok2 := toInt(v); ok2 && p > 0 {
				targetPort = p
			}
		}
		cfg := oldConfig
		if cfg == "" {
			warning = "该节点还没有配置链接，域名/端口无法同步到下发内容，请在「配置」里填入链接"
		} else if rewritten, err := services.RewriteNodeLinkHostPort(cfg, targetHost, targetPort); err != nil {
			warning = "配置链接无法自动改写（" + err.Error() + "），列表会更新，但用户拿到的仍是原链接里的地址，请直接编辑「配置」字段"
		} else {
			updates["config"] = rewritten
			if typ := services.DetectNodeTypeFromLink(rewritten); typ != "" {
				updates["protocol"] = typ
			}
		}
	}

	if err := db.Model(&node).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新专线节点失败")
		return
	}
	utils.CreateAuditLog(c, "update_custom_node", "custom_node", node.ID, fmt.Sprintf("更新专线节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
	if warning != "" {
		utils.Success(c, gin.H{"node": node, "warning": warning})
		return
	}
	utils.Success(c, node)
}

func AdminDeleteCustomNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	// Remove user assignments first
	if err := db.Where("custom_node_id = ?", id).Delete(&models.UserCustomNode{}).Error; err != nil {
		utils.InternalError(c, "删除专线节点分配关系失败")
		return
	}
	if err := db.Delete(&models.CustomNode{}, id).Error; err != nil {
		utils.InternalError(c, "删除专线节点失败")
		return
	}
	utils.CreateAuditLog(c, "delete_custom_node", "custom_node", uint(id), fmt.Sprintf("删除专线节点 ID: %d", id))
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "专线节点已删除")
}

func AdminAssignCustomNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	var req struct {
		UserIDs       []uint     `json:"user_ids" binding:"required"`
		ExpiresAt     *time.Time `json:"expires_at"`
		DedicatedOnly bool       `json:"dedicated_only"`
		LimitDevices  bool       `json:"limit_devices"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.UserIDs) == 0 {
		utils.BadRequest(c, "请选择要分配的用户")
		return
	}
	if err := replaceCustomNodeAssignments(database.GetDB(), uint(id), req.UserIDs, req.ExpiresAt, req.DedicatedOnly, req.LimitDevices); err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.CreateAuditLog(c, "assign_custom_node", "custom_node", uint(id), fmt.Sprintf("分配专线节点给 %d 个用户", len(req.UserIDs)))
	cache.ClearAllSubscriptionCache()
	utils.SuccessMessage(c, "分配成功")
}

func AdminBatchAssignCustomNodes(c *gin.Context) {
	var req struct {
		IDs           []uint     `json:"ids" binding:"required"`
		UserIDs       []uint     `json:"user_ids" binding:"required"`
		ExpiresAt     *time.Time `json:"expires_at"`
		DedicatedOnly bool       `json:"dedicated_only"`
		LimitDevices  bool       `json:"limit_devices"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要分配的专线节点")
		return
	}
	if len(req.UserIDs) == 0 {
		utils.BadRequest(c, "请选择要分配的用户")
		return
	}

	db := database.GetDB()
	uniqueNodeIDs := uniqueUintSlice(req.IDs)
	uniqueUserIDs := uniqueUintSlice(req.UserIDs)
	successCount := 0

	for _, nodeID := range uniqueNodeIDs {
		if err := replaceCustomNodeAssignments(db, nodeID, uniqueUserIDs, req.ExpiresAt, req.DedicatedOnly, req.LimitDevices); err == nil {
			successCount++
		}
	}

	if successCount == 0 {
		utils.InternalError(c, "批量分配失败")
		return
	}

	utils.CreateAuditLog(c, "batch_assign_custom_node", "custom_node", 0, fmt.Sprintf("批量分配 %d 个专线节点给 %d 个用户", len(uniqueNodeIDs), len(uniqueUserIDs)))
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"success": successCount,
		"total":   len(uniqueNodeIDs),
		"message": "批量分配成功",
	})
}

func replaceCustomNodeAssignments(db *gorm.DB, nodeID uint, userIDs []uint, expiresAt *time.Time, dedicatedOnly bool, limitDevices bool) error {
	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("专线节点不存在")
	}

	uniqueUserIDs := uniqueUintSlice(userIDs)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("custom_node_id = ?", nodeID).Delete(&models.UserCustomNode{}).Error; err != nil {
			return fmt.Errorf("清理分配关系失败")
		}
		if len(uniqueUserIDs) == 0 {
			return nil
		}
		assignments := make([]models.UserCustomNode, 0, len(uniqueUserIDs))
		for _, uid := range uniqueUserIDs {
			assignments = append(assignments, models.UserCustomNode{
				UserID: uid, CustomNodeID: nodeID,
				ExpiresAt: expiresAt, DedicatedOnly: dedicatedOnly, LimitDevices: limitDevices,
			})
		}
		if err := tx.CreateInBatches(assignments, 100).Error; err != nil {
			return fmt.Errorf("分配专线节点失败")
		}
		return nil
	})
}

func uniqueUintSlice(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func AdminImportCustomNodeLinks(c *gin.Context) {
	var req struct {
		Links string `json:"links" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 统一导入流程（与节点导入共用 FetchAndParseImport/BuildCustomNodesFromNodes）
	nodes, err := services.FetchAndParseImport(services.ImportSource{Type: "links", Links: req.Links})
	if err != nil {
		utils.BadRequest(c, "导入失败: "+err.Error())
		return
	}
	if len(nodes) == 0 {
		utils.BadRequest(c, "未找到有效的节点")
		return
	}

	db := database.GetDB()
	customNodes := services.BuildCustomNodesFromNodes(nodes)
	result := db.CreateInBatches(customNodes, 100)
	successCount := int(result.RowsAffected)

	utils.CreateAuditLog(c, "import_custom_node_links", "custom_node", 0, "导入专线节点链接")
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"total":   len(nodes),
		"success": successCount,
		"message": "导入完成",
	})
}

// AdminImportCustomNodes 专线节点导入（支持订阅 URL / 节点链接）
// 订阅导入采用"同步更新"模式：
//   - 同一订阅地址的旧节点按名称更新（保留 ID 与分配关系）
//   - 新节点插入；原订阅中消失的节点停用（不删除，保留分配）
func AdminImportCustomNodes(c *gin.Context) {
	var req struct {
		Type  string `json:"type" binding:"required"` // "subscription" | "links"
		URL   string `json:"url"`
		Links string `json:"links"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

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

	// 订阅模式：同步更新（保留分配关系）
	if req.Type == "subscription" {
		syncResult, err := services.SyncCustomNodesFromSubscription(db, nodes, req.URL)
		if err != nil {
			utils.InternalError(c, "同步订阅失败: "+err.Error())
			return
		}
		utils.CreateAuditLog(c, "import_custom_nodes_sub", "custom_node", 0,
			fmt.Sprintf("订阅导入专线节点: 新增 %d 更新 %d 停用 %d", syncResult.Inserted, syncResult.Updated, syncResult.Deactivated))
		cache.ClearAllSubscriptionCache()
		utils.Success(c, gin.H{
			"total":       syncResult.Total,
			"inserted":    syncResult.Inserted,
			"updated":     syncResult.Updated,
			"deactivated": syncResult.Deactivated,
			"message":     fmt.Sprintf("同步完成: 新增 %d, 更新 %d, 停用 %d", syncResult.Inserted, syncResult.Updated, syncResult.Deactivated),
		})
		return
	}

	// 链接模式：仅新增（跳过同名，不更新已有，避免误覆盖手工节点）
	customNodes := services.BuildCustomNodesFromNodes(nodes)
	toInsert := make([]models.CustomNode, 0, len(customNodes))
	var existingNames []string
	var existingCount int64
	for _, cn := range customNodes {
		db.Model(&models.CustomNode{}).Where("name = ?", cn.Name).Count(&existingCount)
		if existingCount > 0 {
			continue
		}
		_ = existingNames
		toInsert = append(toInsert, cn)
	}
	successCount := 0
	if len(toInsert) > 0 {
		if err := db.CreateInBatches(toInsert, 100).Error; err == nil {
			successCount = len(toInsert)
		}
	}
	utils.CreateAuditLog(c, "import_custom_nodes_links", "custom_node", 0, "链接导入专线节点")
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"total":   len(nodes),
		"success": successCount,
		"message": fmt.Sprintf("导入完成: 成功 %d/%d（同名已存在则跳过）", successCount, len(nodes)),
	})
}

func AdminBatchDeleteCustomNodes(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要删除的节点")
		return
	}

	db := database.GetDB()
	if err := db.Where("custom_node_id IN ?", req.IDs).Delete(&models.UserCustomNode{}).Error; err != nil {
		utils.InternalError(c, "批量删除分配关系失败")
		return
	}
	result := db.Where("id IN ?", req.IDs).Delete(&models.CustomNode{})
	utils.CreateAuditLog(c, "batch_delete_custom_nodes", "custom_node", 0, "批量删除专线节点")
	cache.ClearAllSubscriptionCache()
	utils.Success(c, gin.H{
		"deleted": result.RowsAffected,
		"message": "批量删除完成",
	})
}

func AdminGetCustomNodeLink(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, id).Error; err != nil {
		utils.NotFound(c, "专线节点不存在")
		return
	}
	utils.Success(c, gin.H{
		"link":     node.Config,
		"name":     node.DisplayName,
		"protocol": node.Protocol,
	})
}

func AdminGetCustomNodeUsers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的专线节点ID")
		return
	}
	db := database.GetDB()
	var assignments []models.UserCustomNode
	db.Where("custom_node_id = ?", id).Limit(1000).Find(&assignments)

	var userIDs []uint
	for _, a := range assignments {
		userIDs = append(userIDs, a.UserID)
	}

	var users []models.User
	if len(userIDs) > 0 {
		db.Where("id IN ?", userIDs).Select("id, username, email").Find(&users)
	}

	utils.Success(c, gin.H{
		"user_ids": userIDs,
		"users":    users,
	})
}

// ==================== 专线节点「订阅来源」管理 ====================
//
// 让后台可以看到专线节点是从哪个订阅链接导入的、随时改链接/立即更新/删除整条来源，
// 并由调度器按间隔自动重新拉取（订阅内容变化 → 节点跟着变化，不会长期不更新而失效）。

// AdminListCustomNodeSources 列出全部订阅来源（含节点数统计与分配情况）
func AdminListCustomNodeSources(c *gin.Context) {
	db := database.GetDB()
	sources, err := services.ListCustomNodeSources(db)
	if err != nil {
		utils.InternalError(c, "读取订阅来源失败: "+err.Error())
		return
	}

	type row struct {
		models.CustomNodeSource
		ActiveNodes   int64 `json:"active_nodes"`
		InactiveNodes int64 `json:"inactive_nodes"`
		AssignedUsers int64 `json:"assigned_users"`
	}
	out := make([]row, 0, len(sources))
	for _, src := range sources {
		var active, inactive, assigned int64
		db.Model(&models.CustomNode{}).Where("source_url = ? AND is_active = ?", src.URL, true).Count(&active)
		db.Model(&models.CustomNode{}).Where("source_url = ? AND is_active = ?", src.URL, false).Count(&inactive)
		db.Model(&models.UserCustomNode{}).
			Where("custom_node_id IN (?)", db.Model(&models.CustomNode{}).Select("id").Where("source_url = ?", src.URL)).
			Count(&assigned)
		out = append(out, row{CustomNodeSource: src, ActiveNodes: active, InactiveNodes: inactive, AssignedUsers: assigned})
	}
	utils.Success(c, gin.H{"list": out, "total": len(out)})
}

// AdminCreateCustomNodeSource 新增订阅来源并立即同步一次
func AdminCreateCustomNodeSource(c *gin.Context) {
	var req struct {
		URL           string `json:"url" binding:"required"`
		Name          string `json:"name"`
		IntervalHours *int   `json:"interval_hours"`
		Enabled       *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		utils.BadRequest(c, "订阅链接必须以 http:// 或 https:// 开头")
		return
	}

	db := database.GetDB()
	var exists int64
	db.Model(&models.CustomNodeSource{}).Where("url = ?", req.URL).Count(&exists)
	if exists > 0 {
		utils.BadRequest(c, "该订阅链接已存在")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = req.URL
		if i := strings.Index(name, "://"); i > 0 {
			name = name[i+3:]
		}
		if len(name) > 60 {
			name = name[:60]
		}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	// 没传间隔 → 默认 6 小时；显式传 0 → 只手动更新（不参与自动同步）
	interval := 6
	if req.IntervalHours != nil {
		interval = *req.IntervalHours
		if interval < 0 {
			interval = 0
		}
	}

	src := models.CustomNodeSource{Name: name, URL: req.URL, Enabled: enabled, IntervalHours: interval}
	if err := db.Create(&src).Error; err != nil {
		utils.InternalError(c, "创建订阅来源失败: "+err.Error())
		return
	}

	result, syncErr := services.SyncCustomNodeSource(db, &src)
	if syncErr != nil {
		// 来源已创建，但首次同步失败：把失败原因返回给管理员，便于改链接重试
		utils.Success(c, gin.H{
			"id": src.ID, "url": src.URL,
			"sync_error": syncErr.Error(),
			"message":    "订阅来源已保存，但首次同步失败：" + syncErr.Error(),
		})
		return
	}
	utils.CreateAuditLog(c, "create_custom_node_source", "custom_node", src.ID,
		"新增专线订阅来源: 新增 "+strconv.Itoa(result.Inserted)+" 更新 "+strconv.Itoa(result.Updated))
	utils.Success(c, gin.H{
		"id": src.ID, "url": src.URL,
		"inserted": result.Inserted, "updated": result.Updated, "total": result.Total,
		"message": fmt.Sprintf("同步完成：新增 %d，更新 %d", result.Inserted, result.Updated),
	})
}

// AdminUpdateCustomNodeSource 修改订阅来源：换链接 / 改名称 / 开关自动同步 / 改间隔
func AdminUpdateCustomNodeSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	var src models.CustomNodeSource
	if err := db.First(&src, uint(id)).Error; err != nil {
		utils.NotFound(c, "订阅来源不存在")
		return
	}

	var req struct {
		URL           *string `json:"url"`
		Name          *string `json:"name"`
		Enabled       *bool   `json:"enabled"`
		IntervalHours *int    `json:"interval_hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.IntervalHours != nil {
		v := *req.IntervalHours
		if v < 0 {
			v = 0
		}
		updates["interval_hours"] = v
	}
	if len(updates) > 0 {
		if err := db.Model(&models.CustomNodeSource{}).Where("id = ?", src.ID).Updates(updates).Error; err != nil {
			utils.InternalError(c, "更新失败: "+err.Error())
			return
		}
	}

	// 换了链接：把节点归属搬到新链接并按新链接重新同步
	if req.URL != nil && strings.TrimSpace(*req.URL) != "" && strings.TrimSpace(*req.URL) != src.URL {
		newURL := strings.TrimSpace(*req.URL)
		if !strings.HasPrefix(newURL, "http://") && !strings.HasPrefix(newURL, "https://") {
			utils.BadRequest(c, "订阅链接必须以 http:// 或 https:// 开头")
			return
		}
		var dup int64
		db.Model(&models.CustomNodeSource{}).Where("url = ? AND id != ?", newURL, src.ID).Count(&dup)
		if dup > 0 {
			utils.BadRequest(c, "该订阅链接已被其它来源使用")
			return
		}
		result, err := services.ChangeCustomNodeSourceURL(db, &src, newURL)
		if err != nil {
			utils.Success(c, gin.H{"id": src.ID, "url": newURL, "sync_error": err.Error(),
				"message": "链接已更新，但同步失败：" + err.Error()})
			return
		}
		utils.CreateAuditLog(c, "update_custom_node_source", "custom_node", src.ID, "更换专线订阅链接")
		utils.Success(c, gin.H{"id": src.ID, "url": newURL, "inserted": result.Inserted,
			"updated": result.Updated, "deactivated": result.Deactivated,
			"message": fmt.Sprintf("链接已更新并同步：新增 %d，更新 %d，停用 %d", result.Inserted, result.Updated, result.Deactivated)})
		return
	}

	db.First(&src, uint(id))
	utils.Success(c, src)
}

// AdminSyncCustomNodeSource 立即更新一条来源
func AdminSyncCustomNodeSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db := database.GetDB()
	var src models.CustomNodeSource
	if err := db.First(&src, uint(id)).Error; err != nil {
		utils.NotFound(c, "订阅来源不存在")
		return
	}
	result, err := services.SyncCustomNodeSource(db, &src)
	if err != nil {
		utils.BadRequest(c, "同步失败: "+err.Error())
		return
	}
	utils.CreateAuditLog(c, "sync_custom_node_source", "custom_node", src.ID, "手动更新专线订阅来源")
	utils.Success(c, gin.H{
		"inserted": result.Inserted, "updated": result.Updated,
		"deactivated": result.Deactivated, "total": result.Total,
		"message": fmt.Sprintf("同步完成：新增 %d，更新 %d，停用 %d", result.Inserted, result.Updated, result.Deactivated),
	})
}

// AdminSyncAllCustomNodeSources 一键更新全部来源
func AdminSyncAllCustomNodeSources(c *gin.Context) {
	db := database.GetDB()
	sources, err := services.ListCustomNodeSources(db)
	if err != nil {
		utils.InternalError(c, "读取订阅来源失败: "+err.Error())
		return
	}
	totalInserted, totalUpdated, failed, skipped := 0, 0, 0, 0
	messages := []string{}
	for i := range sources {
		src := &sources[i]
		// 关掉自动同步的来源不参与「全部更新」（单条「立即更新」仍然可以手动刷新）
		if !src.Enabled {
			skipped++
			continue
		}
		result, err := services.SyncCustomNodeSource(db, src)
		if err != nil {
			failed++
			messages = append(messages, src.Name+": "+err.Error())
			continue
		}
		totalInserted += result.Inserted
		totalUpdated += result.Updated
	}
	utils.CreateAuditLog(c, "sync_all_custom_node_sources", "custom_node", 0, "一键更新全部专线订阅来源")
	msg := fmt.Sprintf("已更新 %d 个来源：新增 %d，更新 %d，失败 %d", len(sources)-skipped, totalInserted, totalUpdated, failed)
	if skipped > 0 {
		msg += fmt.Sprintf("（已跳过 %d 个关闭自动同步的来源）", skipped)
	}
	utils.Success(c, gin.H{
		"sources": len(sources), "synced": len(sources) - skipped, "skipped": skipped,
		"inserted": totalInserted, "updated": totalUpdated, "failed": failed,
		"messages": messages, "message": msg,
	})
}

// AdminDeleteCustomNodeSource 删除订阅来源（默认连它导入的节点一起删）
func AdminDeleteCustomNodeSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	// delete_nodes=false 时只解除来源关系，把节点保留成手工节点
	deleteNodes := c.DefaultQuery("delete_nodes", "true") != "false"

	db := database.GetDB()
	var src models.CustomNodeSource
	if err := db.First(&src, uint(id)).Error; err != nil {
		utils.NotFound(c, "订阅来源不存在")
		return
	}
	deleted, err := services.DeleteCustomNodeSource(db, &src, deleteNodes)
	if err != nil {
		utils.InternalError(c, "删除失败: "+err.Error())
		return
	}
	utils.CreateAuditLog(c, "delete_custom_node_source", "custom_node", src.ID,
		fmt.Sprintf("删除专线订阅来源（删除节点: %v, 影响 %d 个）", deleteNodes, deleted))
	msg := fmt.Sprintf("已删除来源，并删除其导入的 %d 个节点", deleted)
	if !deleteNodes {
		msg = fmt.Sprintf("已删除来源，%d 个节点已转为手工节点保留", deleted)
	}
	utils.Success(c, gin.H{"deleted_nodes": deleted, "message": msg})
}
