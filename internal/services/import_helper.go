package services

import (
	"errors"

	"cboard/v2/internal/models"
	"gorm.io/gorm"
)

// 导入相关错误哨兵
var (
	ErrEmptySubscriptionURL = errors.New("订阅URL不能为空")
	ErrEmptyNodeLinks       = errors.New("节点链接不能为空")
	ErrUnsupportedImportType = errors.New("不支持的导入类型")
)

// ImportSource 统一导入请求来源（节点/专线共用）
type ImportSource struct {
	Type  string // "subscription" | "links"
	URL   string // 订阅地址（Type=subscription 时）
	Links string // 节点链接文本（Type=links 时）
}

// FetchAndParseImport 统一获取并解析导入内容：
// - subscription: 抓取订阅 URL 内容后解析（Clash YAML / JSON / 传统链接）
// - links: 直接解析多行节点链接
// 返回解析出的节点列表；内容仅在 subscription 时用于记录来源
func FetchAndParseImport(src ImportSource) ([]models.Node, error) {
	var content string
	switch src.Type {
	case "subscription":
		if src.URL == "" {
			return nil, ErrEmptySubscriptionURL
		}
		var err error
		content, err = FetchSubscriptionContent(src.URL)
		if err != nil {
			return nil, err
		}
	case "links":
		if src.Links == "" {
			return nil, ErrEmptyNodeLinks
		}
		content = src.Links
	default:
		return nil, ErrUnsupportedImportType
	}
	return ParseSubscriptionContent(content)
}

// FilterExistingNodesByName 按节点名过滤已存在的节点（节点导入去重用）
func FilterExistingNodesByName(db *gorm.DB, nodes []models.Node, isManual bool, sourceURL string) []models.Node {
	if len(nodes) == 0 {
		return nodes
	}
	names := make([]string, 0, len(nodes))
	for _, n := range nodes {
		names = append(names, n.Name)
	}
	var existingNames []string
	db.Where("name IN ?", names).Pluck("name", &existingNames)
	existingSet := make(map[string]bool, len(existingNames))
	for _, nm := range existingNames {
		existingSet[nm] = true
	}
	toInsert := make([]models.Node, 0, len(nodes))
	for _, node := range nodes {
		if existingSet[node.Name] {
			continue
		}
		node.IsManual = isManual
		if sourceURL != "" {
			node.SourceURL = sourceURL
		}
		toInsert = append(toInsert, node)
	}
	return toInsert
}

// BuildCustomNodesFromNodes 将解析出的节点转换为专线节点模型
// （提取域名/端口，保留原始 config）
func BuildCustomNodesFromNodes(nodes []models.Node) []models.CustomNode {
	customNodes := make([]models.CustomNode, 0, len(nodes))
	for _, node := range nodes {
		domain := ""
		port := 443
		if node.Config != nil && *node.Config != "" {
			if d, p, err := ExtractDomainPortFromNodeLink(*node.Config); err == nil {
				domain = d
				if p > 0 {
					port = p
				}
			}
		}
		cn := models.CustomNode{
			Name:        node.Name,
			DisplayName: node.Name,
			Protocol:    node.Type,
			Domain:      domain,
			Port:        port,
			Config:      "",
			IsActive:    true,
		}
		if node.Config != nil {
			cn.Config = *node.Config
		}
		customNodes = append(customNodes, cn)
	}
	return customNodes
}

// CustomNodeSyncResult 专线订阅同步结果
type CustomNodeSyncResult struct {
	Total     int `json:"total"`      // 订阅中解析出的节点数
	Inserted  int `json:"inserted"`   // 新增节点数
	Updated   int `json:"updated"`    // 更新节点数
	Deactivated int `json:"deactivated"` // 原订阅中消失被停用的节点数
	Kept      int `json:"kept"`       // 保留未变的节点数
}

// SyncCustomNodesFromSubscription 同步订阅到专线节点：
// 1. 同一 SourceURL 的旧节点按名称匹配更新（域名/端口/配置/显示名），保留 ID 与分配关系
// 2. 新名称的节点插入（标记 SourceURL）
// 3. 旧订阅中存在但新订阅已消失的节点 → 停用（IsActive=false），不删除，保留分配
// 返回各统计。
func SyncCustomNodesFromSubscription(db *gorm.DB, nodes []models.Node, sourceURL string) (CustomNodeSyncResult, error) {
	result := CustomNodeSyncResult{Total: len(nodes)}
	if len(nodes) == 0 {
		return result, nil
	}

	// 该订阅源下已有的专线节点
	var existing []models.CustomNode
	if err := db.Where("source_url = ?", sourceURL).Find(&existing).Error; err != nil {
		return result, err
	}
	existingByName := make(map[string]*models.CustomNode, len(existing))
	for i := range existing {
		existingByName[existing[i].Name] = &existing[i]
	}

	incomingNames := make(map[string]bool, len(nodes))
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, node := range nodes {
			incomingNames[node.Name] = true
			domain, port := extractCustomNodeDomainPort(node)
			configStr := ""
			if node.Config != nil {
				configStr = *node.Config
			}

			if old, ok := existingByName[node.Name]; ok {
				// 更新旧节点（保留 ID → 分配关系不动）
				updates := map[string]interface{}{
					"display_name": node.Name,
					"protocol":     node.Type,
					"domain":       domain,
					"port":         port,
					"config":       configStr,
					"is_active":    true,
				}
				if err := tx.Model(&models.CustomNode{}).Where("id = ?", old.ID).Updates(updates).Error; err != nil {
					return err
				}
				result.Updated++
			} else {
				// 新节点
				cn := models.CustomNode{
					Name: node.Name, DisplayName: node.Name, Protocol: node.Type,
					Domain: domain, Port: port, Config: configStr,
					IsActive: true, SourceURL: sourceURL,
				}
				if err := tx.Create(&cn).Error; err != nil {
					return err
				}
				result.Inserted++
			}
		}

		// 旧订阅中消失的节点 → 停用（保留分配，不删除）
		for name, old := range existingByName {
			if !incomingNames[name] && old.IsActive {
				if err := tx.Model(&models.CustomNode{}).Where("id = ?", old.ID).
					Update("is_active", false).Error; err != nil {
					return err
				}
				result.Deactivated++
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}

	// 统计保留未变的（已更新 + 已停用之外的存活节点）
	result.Kept = len(existing) - result.Updated - result.Deactivated
	if result.Kept < 0 {
		result.Kept = 0
	}
	return result, nil
}

// extractCustomNodeDomainPort 从节点链接提取域名/端口（供专线更新用）
func extractCustomNodeDomainPort(node models.Node) (string, int) {
	domain, port := "", 443
	if node.Config != nil && *node.Config != "" {
		if d, p, err := ExtractDomainPortFromNodeLink(*node.Config); err == nil {
			domain = d
			if p > 0 {
				port = p
			}
		}
	}
	return domain, port
}
