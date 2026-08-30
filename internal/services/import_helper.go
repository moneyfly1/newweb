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
