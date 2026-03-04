package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// QueryBuilder 数据库查询构建器
type QueryBuilder struct {
	db    *gorm.DB
	model interface{}
	err   error
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(db *gorm.DB) *QueryBuilder {
	return &QueryBuilder{db: db}
}

// Model 设置查询模型
func (qb *QueryBuilder) Model(model interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.model = model
	qb.db = qb.db.Model(model)
	return qb
}

// Paginate 分页查询
func (qb *QueryBuilder) Paginate(page, pageSize int) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	qb.db = qb.db.Offset(offset).Limit(pageSize)
	return qb
}

// Filter 添加过滤条件
func (qb *QueryBuilder) Filter(field, op string, value interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if value == nil || value == "" {
		return qb
	}

	switch op {
	case "=", "!=", ">", ">=", "<", "<=":
		qb.db = qb.db.Where(fmt.Sprintf("%s %s ?", field, op), value)
	case "like":
		qb.db = qb.db.Where(fmt.Sprintf("%s LIKE ?", field), "%"+fmt.Sprint(value)+"%")
	case "in":
		qb.db = qb.db.Where(fmt.Sprintf("%s IN ?", field), value)
	case "not in":
		qb.db = qb.db.Where(fmt.Sprintf("%s NOT IN ?", field), value)
	case "between":
		if vals, ok := value.([]interface{}); ok && len(vals) == 2 {
			qb.db = qb.db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), vals[0], vals[1])
		}
	case "is null":
		qb.db = qb.db.Where(fmt.Sprintf("%s IS NULL", field))
	case "is not null":
		qb.db = qb.db.Where(fmt.Sprintf("%s IS NOT NULL", field))
	default:
		qb.err = fmt.Errorf("unsupported operator: %s", op)
	}
	return qb
}

// Search 多字段搜索
func (qb *QueryBuilder) Search(fields []string, keyword string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if keyword == "" || len(fields) == 0 {
		return qb
	}

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return qb
	}

	var conditions []string
	var args []interface{}
	for _, field := range fields {
		conditions = append(conditions, fmt.Sprintf("%s LIKE ?", field))
		args = append(args, "%"+keyword+"%")
	}

	query := strings.Join(conditions, " OR ")
	qb.db = qb.db.Where(query, args...)
	return qb
}

// Sort 排序
func (qb *QueryBuilder) Sort(field, order string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if field == "" {
		return qb
	}

	order = strings.ToUpper(strings.TrimSpace(order))
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	qb.db = qb.db.Order(fmt.Sprintf("%s %s", field, order))
	return qb
}

// Where 添加自定义 WHERE 条件
func (qb *QueryBuilder) Where(query interface{}, args ...interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.db = qb.db.Where(query, args...)
	return qb
}

// Select 选择字段
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if len(fields) > 0 {
		qb.db = qb.db.Select(fields)
	}
	return qb
}

// Preload 预加载关联
func (qb *QueryBuilder) Preload(query string, args ...interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.db = qb.db.Preload(query, args...)
	return qb
}

// Group 分组
func (qb *QueryBuilder) Group(field string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if field != "" {
		qb.db = qb.db.Group(field)
	}
	return qb
}

// Having 添加 HAVING 条件
func (qb *QueryBuilder) Having(query interface{}, args ...interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.db = qb.db.Having(query, args...)
	return qb
}

// Join 连接查询
func (qb *QueryBuilder) Join(query string, args ...interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.db = qb.db.Joins(query, args...)
	return qb
}

// Execute 执行查询
func (qb *QueryBuilder) Execute(dest interface{}) error {
	if qb.err != nil {
		return qb.err
	}
	return qb.db.Find(dest).Error
}

// First 查询第一条记录
func (qb *QueryBuilder) First(dest interface{}) error {
	if qb.err != nil {
		return qb.err
	}
	return qb.db.First(dest).Error
}

// Count 统计数量
func (qb *QueryBuilder) Count(count *int64) error {
	if qb.err != nil {
		return qb.err
	}
	return qb.db.Count(count).Error
}

// Exists 检查是否存在
func (qb *QueryBuilder) Exists() (bool, error) {
	if qb.err != nil {
		return false, qb.err
	}
	var count int64
	err := qb.db.Count(&count).Error
	return count > 0, err
}

// Delete 删除记录
func (qb *QueryBuilder) Delete() error {
	if qb.err != nil {
		return qb.err
	}
	return qb.db.Delete(qb.model).Error
}

// Update 更新记录
func (qb *QueryBuilder) Update(column string, value interface{}) error {
	if qb.err != nil {
		return qb.err
	}
	return qb.db.Update(column, value).Error
}

// Updates 批量更新
func (qb *QueryBuilder) Updates(values interface{}) error {
	if qb.err != nil {
		return qb.err
	}
	return qb.db.Updates(values).Error
}

// GetDB 获取底层 DB 对象（用于复杂查询）
func (qb *QueryBuilder) GetDB() *gorm.DB {
	return qb.db
}

// Error 获取错误
func (qb *QueryBuilder) Error() error {
	return qb.err
}

// DateRange 日期范围过滤
func (qb *QueryBuilder) DateRange(field, startDate, endDate string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if startDate != "" {
		qb.db = qb.db.Where(fmt.Sprintf("%s >= ?", field), startDate)
	}
	if endDate != "" {
		qb.db = qb.db.Where(fmt.Sprintf("%s <= ?", field), endDate)
	}
	return qb
}

// FilterMap 批量添加过滤条件
func (qb *QueryBuilder) FilterMap(filters map[string]interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	for field, value := range filters {
		if value != nil && value != "" {
			qb.db = qb.db.Where(fmt.Sprintf("%s = ?", field), value)
		}
	}
	return qb
}
