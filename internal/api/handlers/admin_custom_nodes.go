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
			db.Where("custom_nodes.name LIKE ? OR custom_nodes.display_name LIKE ? OR custom_nodes.domain LIKE ? OR CAST(custom_nodes.port AS CHAR) LIKE ?", like, like, like, like).
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
	node := models.CustomNode{
		Name:             req.Name,
		DisplayName:      req.DisplayName,
		Domain:           req.Domain,
		Port:             req.Port,
		Protocol:         req.Protocol,
		Status:           req.Status,
		Config:           req.Config,
		ExpireTime:       req.ExpireTime,
		FollowUserExpire: req.FollowUserExpire,
	}
	if err := database.GetDB().Create(&node).Error; err != nil {
		utils.InternalError(c, "创建专线节点失败")
		return
	}
	utils.CreateAuditLog(c, "create_custom_node", "custom_node", node.ID, fmt.Sprintf("创建专线节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
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
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&node).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新专线节点失败")
		return
	}
	utils.CreateAuditLog(c, "update_custom_node", "custom_node", node.ID, fmt.Sprintf("更新专线节点: %s", node.Name))
	cache.ClearAllSubscriptionCache()
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

	nodes, err := services.ParseSubscriptionContent(req.Links)
	if err != nil {
		utils.BadRequest(c, "解析节点失败: "+err.Error())
		return
	}
	if len(nodes) == 0 {
		utils.BadRequest(c, "未找到有效的节点")
		return
	}

	db := database.GetDB()
	customNodes := make([]models.CustomNode, 0, len(nodes))
	for _, node := range nodes {
		domain := ""
		port := 443
		if node.Config != nil && *node.Config != "" {
			if extractedDomain, extractedPort, extractErr := services.ExtractDomainPortFromNodeLink(*node.Config); extractErr == nil {
				domain = extractedDomain
				if extractedPort > 0 {
					port = extractedPort
				}
			}
		}
		customNode := models.CustomNode{
			Name:        node.Name,
			DisplayName: node.Name,
			Protocol:    node.Type,
			Domain:      domain,
			Port:        port,
			Config:      "",
			IsActive:    true,
		}
		if node.Config != nil {
			customNode.Config = *node.Config
		}
		customNodes = append(customNodes, customNode)
	}

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

// ==================== Subscription Management ====================

