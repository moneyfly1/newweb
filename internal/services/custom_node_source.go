package services

import (
	"fmt"
	"strings"
	"time"

	"cboard/v2/internal/cache"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"gorm.io/gorm"
)

// 专线节点「订阅来源」的同步逻辑。
//
// 背景：专线节点以前只能一次性导入，订阅内容变化后不会跟着更新，节点会逐渐失效；
// 后台也看不到当初导入的是哪个订阅链接。这里把「来源」作为一等实体管理：
// 记录链接、间隔、上次同步结果，并按需自动重新拉取（订阅一变节点就跟着变）。

// ListCustomNodeSources 列出全部订阅来源（顺带把历史上用 source_url 导入过、
// 但还没有来源记录的情况补齐，避免老数据在界面上看不到）。
func ListCustomNodeSources(db *gorm.DB) ([]models.CustomNodeSource, error) {
	var sources []models.CustomNodeSource
	if err := db.Order("id ASC").Find(&sources).Error; err != nil {
		return nil, err
	}

	known := make(map[string]bool, len(sources))
	for _, s := range sources {
		known[s.URL] = true
	}

	// 回填：custom_nodes 里已有 source_url 但来源表还没有的
	var urls []string
	db.Model(&models.CustomNode{}).
		Where("source_url IS NOT NULL AND source_url != ''").
		Distinct().Pluck("source_url", &urls)
	for _, u := range urls {
		if known[u] {
			continue
		}
		name := u
		if i := strings.Index(name, "://"); i > 0 {
			name = name[i+3:]
		}
		if len(name) > 60 {
			name = name[:60]
		}
		src := models.CustomNodeSource{Name: name, URL: u, Enabled: true, IntervalHours: 6}
		if err := db.Create(&src).Error; err == nil {
			sources = append(sources, src)
			known[u] = true
		}
	}
	return sources, nil
}

// SyncCustomNodeSource 立即同步一条来源：拉取 → 解析 → 增/改/停用节点 → 记录结果。
func SyncCustomNodeSource(db *gorm.DB, src *models.CustomNodeSource) (CustomNodeSyncResult, error) {
	var result CustomNodeSyncResult

	nodes, err := FetchAndParseImport(ImportSource{Type: "subscription", URL: src.URL})
	if err != nil {
		markSourceResult(db, src, "error", "拉取失败: "+err.Error(), 0)
		return result, fmt.Errorf("拉取订阅失败: %w", err)
	}
	if len(nodes) == 0 {
		markSourceResult(db, src, "error", "订阅里没有解析到任何节点", 0)
		return result, fmt.Errorf("订阅里没有解析到任何节点")
	}

	result, err = SyncCustomNodesFromSubscription(db, nodes, src.URL)
	if err != nil {
		markSourceResult(db, src, "error", "写入失败: "+err.Error(), 0)
		return result, err
	}

	msg := fmt.Sprintf("新增 %d，更新 %d，停用 %d", result.Inserted, result.Updated, result.Deactivated)
	markSourceResult(db, src, "ok", msg, result.Total)
	cache.ClearAllSubscriptionCache()
	return result, nil
}

func markSourceResult(db *gorm.DB, src *models.CustomNodeSource, status, message string, total int) {
	now := time.Now()
	updates := map[string]interface{}{
		"last_sync_at": &now,
		"last_status":  status,
		"last_message": message,
	}
	if total > 0 {
		updates["node_total"] = total
	}
	if err := db.Model(&models.CustomNodeSource{}).Where("id = ?", src.ID).Updates(updates).Error; err != nil {
		utils.SysError("custom_node", fmt.Sprintf("更新订阅来源状态失败: id=%d err=%v", src.ID, err))
	}
	src.LastSyncAt = &now
	src.LastStatus = status
	src.LastMessage = message
	if total > 0 {
		src.NodeTotal = total
	}
}

// SyncDueCustomNodeSources 同步所有「到点该更新」的来源（供调度器调用）。
// 返回同步条数与失败条数。
func SyncDueCustomNodeSources() (int, int) {
	db := database.GetDB()
	if db == nil {
		return 0, 0
	}
	sources, err := ListCustomNodeSources(db)
	if err != nil {
		utils.SysError("custom_node", "读取专线订阅来源失败: "+err.Error())
		return 0, 0
	}

	synced, failed := 0, 0
	now := time.Now()
	for i := range sources {
		src := &sources[i]
		if !src.Enabled || src.IntervalHours <= 0 {
			continue
		}
		if src.LastSyncAt != nil && now.Sub(*src.LastSyncAt) < time.Duration(src.IntervalHours)*time.Hour {
			continue
		}
		if _, err := SyncCustomNodeSource(db, src); err != nil {
			failed++
			utils.SysError("custom_node", fmt.Sprintf("专线订阅自动同步失败: %s → %v", maskSourceURL(src.URL), err))
			continue
		}
		synced++
	}
	if synced > 0 || failed > 0 {
		utils.SysInfo("custom_node", fmt.Sprintf("专线订阅自动同步完成: 成功 %d，失败 %d", synced, failed))
	}
	return synced, failed
}

// DeleteCustomNodeSource 删除一条来源及其导入的节点（同时清理用户分配关系）。
// 手工创建的专线节点（source_url 为空）不受影响。
func DeleteCustomNodeSource(db *gorm.DB, src *models.CustomNodeSource, deleteNodes bool) (int64, error) {
	var deleted int64
	err := db.Transaction(func(tx *gorm.DB) error {
		if deleteNodes {
			var ids []uint
			if err := tx.Model(&models.CustomNode{}).
				Where("source_url = ?", src.URL).Pluck("id", &ids).Error; err != nil {
				return err
			}
			if len(ids) > 0 {
				// 先清分配关系：否则用户侧会残留指向已删除节点的分配记录
				if err := tx.Where("custom_node_id IN ?", ids).Delete(&models.UserCustomNode{}).Error; err != nil {
					return err
				}
				r := tx.Where("id IN ?", ids).Delete(&models.CustomNode{})
				if r.Error != nil {
					return r.Error
				}
				deleted = r.RowsAffected
			}
		} else {
			// 只解除来源关系：节点保留为手工节点
			r := tx.Model(&models.CustomNode{}).Where("source_url = ?", src.URL).Update("source_url", "")
			if r.Error != nil {
				return r.Error
			}
			deleted = r.RowsAffected
		}
		return tx.Delete(&models.CustomNodeSource{}, src.ID).Error
	})
	if err == nil {
		cache.ClearAllSubscriptionCache()
	}
	return deleted, err
}

// ChangeCustomNodeSourceURL 更换订阅链接：把该来源下的节点归属改到新链接，再按新链接同步。
func ChangeCustomNodeSourceURL(db *gorm.DB, src *models.CustomNodeSource, newURL string) (CustomNodeSyncResult, error) {
	newURL = strings.TrimSpace(newURL)
	if newURL == "" {
		return CustomNodeSyncResult{}, fmt.Errorf("订阅链接不能为空")
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CustomNode{}).Where("source_url = ?", src.URL).
			Update("source_url", newURL).Error; err != nil {
			return err
		}
		name := newURL
		if i := strings.Index(name, "://"); i > 0 {
			name = name[i+3:]
		}
		if len(name) > 60 {
			name = name[:60]
		}
		updates := map[string]interface{}{"url": newURL}
		if src.Name == "" || src.Name == src.URL || strings.HasPrefix(src.Name, name[:minInt(len(name), 10)]) {
			updates["name"] = name
		}
		return tx.Model(&models.CustomNodeSource{}).Where("id = ?", src.ID).Updates(updates).Error
	})
	if err != nil {
		return CustomNodeSyncResult{}, err
	}
	src.URL = newURL
	return SyncCustomNodeSource(db, src)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maskSourceURL 订阅链接脱敏（日志里不出现完整 token）
func maskSourceURL(u string) string {
	if len(u) <= 30 {
		return u
	}
	return u[:24] + "…" + u[len(u)-6:]
}
