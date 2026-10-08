package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"cboard/v2/internal/database"

	"github.com/gin-gonic/gin"
)

// validSortField ensures sort field is a safe column name (prevents SQL injection).
var validSortField = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

const (
	MaxPageSize   = 100
	MaxPageNumber = 10000 // 防止超大 offset 导致性能问题
)

type Pagination struct {
	Page     int
	PageSize int
	Sort     string
	Order    string
}

func GetPagination(c *gin.Context) Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	// 添加页码上限，防止超大 offset 攻击
	if page > MaxPageNumber {
		page = MaxPageNumber
	}

	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	sort := c.DefaultQuery("sort", "id")
	if !validSortField.MatchString(sort) {
		sort = "id"
	}

	order := c.DefaultQuery("order", "desc")
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	return Pagination{Page: page, PageSize: pageSize, Sort: sort, Order: order}
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// dateTimeColumns 是各表里的时间列。
//
// 为什么要单独处理：SQLite 没有真正的 datetime 类型，GORM 写进去的是字符串，
// 而历史数据里格式并不统一（`2026-09-16 00:09:47+08:00`、`2022-12-09T11:38:13`、
// `2026-09-16 14:06:57` 三种混用）。按文本排序时，分隔符 'T'(0x54) 与空格(0x20)
// 参与比较，带时区与不带时区也会被当成同一时间轴 —— 同一天里不同格式的行就会排错。
// 用 SQLite 的 datetime() 先归一化到 UTC 再比较，不管存的是哪种格式都得到正确顺序。
// MySQL / PostgreSQL 的时间列本身就是 datetime 类型，不需要也不应该包函数。
var dateTimeColumns = map[string]bool{
	"created_at":    true,
	"updated_at":    true,
	"last_login":    true,
	"expire_time":   true,
	"expires_at":    true,
	"registered_at": true,
	"last_sync_at":  true,
	"used_at":       true,
	"paid_at":       true,
}

func isSQLite() bool {
	db := database.GetDB()
	if db == nil {
		return false
	}
	return strings.HasPrefix(strings.ToLower(db.Dialector.Name()), "sqlite")
}

func (p Pagination) OrderClause() string {
	if dateTimeColumns[p.Sort] && isSQLite() {
		// COALESCE：datetime() 解析不了的值会返回 NULL，NULL 在 SQLite 里排最前，
		// 这里退回原文，避免脏数据把整批正常记录挤到后面
		return fmt.Sprintf("COALESCE(datetime(%s), %s) %s", p.Sort, p.Sort, p.Order)
	}
	return p.Sort + " " + p.Order
}
