package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// AdminDeviceStats 设备分析（对齐 myweb 的按设备统计）。
//
// 返回：总量/在线、客户端分布、系统分布、设备类型分布，以及可筛选的明细列表。
// 在线判定与 Device.ComputeOnline 一致：3 分钟内有心跳，或 24 小时内拉过订阅。
func AdminDeviceStats(c *gin.Context) {
	db := database.GetDB()

	now := time.Now()
	hbCutoff := now.Add(-models.HeartbeatOnlineWindow)
	subCutoff := now.Add(-models.SubscriptionOnlineWindow)

	var total, online int64
	db.Model(&models.Device{}).Count(&total)
	db.Model(&models.Device{}).
		Where("(last_heartbeat IS NOT NULL AND last_heartbeat >= ?) OR last_access >= ?", hbCutoff, subCutoff).
		Count(&online)

	type groupCount struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}
	var bySoftware, byOS, byType []groupCount
	db.Model(&models.Device{}).
		Select("COALESCE(NULLIF(TRIM(software_name), ''), '未知') as name, COUNT(*) as count").
		Group("name").Order("count DESC").Limit(20).Scan(&bySoftware)
	db.Model(&models.Device{}).
		Select("COALESCE(NULLIF(TRIM(os_name), ''), '未知') as name, COUNT(*) as count").
		Group("name").Order("count DESC").Limit(20).Scan(&byOS)
	db.Model(&models.Device{}).
		Select("COALESCE(NULLIF(TRIM(device_type), ''), '未知') as name, COUNT(*) as count").
		Group("name").Order("count DESC").Scan(&byType)

	// 明细：支持按客户端/系统/在线状态筛选 + 关键字（IP / 型号 / 用户名）
	q := db.Model(&models.Device{})
	if sw := strings.TrimSpace(c.Query("software")); sw != "" {
		q = q.Where("software_name = ?", sw)
	}
	if osName := strings.TrimSpace(c.Query("os")); osName != "" {
		q = q.Where("os_name = ?", osName)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("ip_address LIKE ? OR device_model LIKE ? OR device_name LIKE ? OR user_agent LIKE ?", like, like, like, like)
	}
	if onlineOnly := c.Query("online"); onlineOnly == "true" {
		q = q.Where("(last_heartbeat IS NOT NULL AND last_heartbeat >= ?) OR last_access >= ?", hbCutoff, subCutoff)
	}

	var totalFiltered int64
	q.Count(&totalFiltered)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}

	var devices []models.Device
	q.Order("last_access DESC").Offset((page - 1) * size).Limit(size).Find(&devices)

	// 明细补上在线标记 + 用户名（前端直接展示，不用再查一遍）
	type deviceRow struct {
		models.Device
		Online   bool   `json:"online"`
		Username string `json:"username"`
	}
	rows := make([]deviceRow, 0, len(devices))
	userIDs := make([]uint, 0, len(devices))
	for _, d := range devices {
		if d.UserID != nil {
			userIDs = append(userIDs, uint(*d.UserID))
		}
	}
	nameMap := map[uint]string{}
	if len(userIDs) > 0 {
		var users []models.User
		db.Select("id", "username").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			nameMap[u.ID] = u.Username
		}
	}
	for _, d := range devices {
		row := deviceRow{Device: d, Online: d.ComputeOnline()}
		if d.UserID != nil {
			row.Username = nameMap[uint(*d.UserID)]
		}
		rows = append(rows, row)
	}

	utils.Success(c, gin.H{
		"total":        total,
		"online":       online,
		"offline":      total - online,
		"by_software":  bySoftware,
		"by_os":        byOS,
		"by_type":      byType,
		"items":        rows,
		"total_filter": totalFiltered,
		"page":         page,
		"page_size":    size,
	})
}
