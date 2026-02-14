package utils

import (
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

// validSortField ensures sort field is a safe column name (prevents SQL injection).
var validSortField = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

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
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
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

func (p Pagination) OrderClause() string {
	return p.Sort + " " + p.Order
}
