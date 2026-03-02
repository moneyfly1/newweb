# 日志、统计、逻辑优化任务清单

## 创建时间
2026-03-02

## 任务概述
全面检查和修复：日志系统、订单统计、数据统计、业务逻辑、代码结构、列表状态筛选等问题。

---

## 📋 待检查和修复的问题

### 1. 日志系统问题

#### 1.1 日志文件管理
- ⚠️ backend.log 已达 6MB，无日志轮转
- ⚠️ 缺少日志级别控制（DEBUG/INFO/WARN/ERROR）
- ⚠️ 缺少结构化日志

**修复方案：**
```go
// 使用 lumberjack 实现日志轮转
import "gopkg.in/natefinch/lumberjack.v2"

logger := &lumberjack.Logger{
    Filename:   "./logs/backend.log",
    MaxSize:    10, // MB
    MaxBackups: 3,
    MaxAge:     28, // days
    Compress:   true,
}
```

#### 1.2 日志查询功能
- ✅ 已有审计日志、登录日志、余额日志等
- ⚠️ 需检查日志查询是否有性能问题
- ⚠️ 需检查日志是否有分页限制

**待检查文件：**
- `internal/api/handlers/admin.go` - AdminAuditLogs, AdminLoginLogs 等函数

### 2. 订单统计问题

#### 2.1 统计准确性
- ⚠️ 需检查订单金额统计是否正确（是否使用 final_amount）
- ⚠️ 需检查退款订单是否正确排除
- ⚠️ 需检查时区问题

**待检查：**
```go
// AdminDashboard 中的收入统计
db.Model(&models.Order{}).
    Where("status = ? AND DATE(payment_time) = ?", "paid", today).
    Select("COALESCE(SUM(amount), 0)").Scan(&revenueToday)
```

**问题：** 使用 `amount` 而不是 `final_amount`（实付金额）

**修复：**
```go
Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&revenueToday)
```

#### 2.2 订单状态筛选
- ⚠️ 需检查前端订单列表的状态筛选是否完整
- ⚠️ 需检查是否支持多状态筛选
- ⚠️ 需检查状态枚举是否一致

**待检查文件：**
- `frontend/src/views/order/Index.vue`
- `frontend/src/views/admin/orders/Index.vue`

### 3. 数据统计问题

#### 3.1 Dashboard 统计
**当前统计项：**
- 总用户数
- 总订单数
- 活跃订阅数
- 今日收入
- 本月收入
- 待处理订单
- 待处理工单
- 收入趋势（30天）
- 用户增长（30天）

**潜在问题：**
1. ⚠️ 收入统计使用 `amount` 而非 `final_amount`
2. ⚠️ 退款订单未排除
3. ⚠️ 并发查询可能有性能问题
4. ⚠️ 缺少缓存机制

#### 3.2 财务报表
- ⚠️ 需检查财务报表的准确性
- ⚠️ 需检查是否包含所有收入来源（订单、充值）
- ⚠️ 需检查退款是否正确处理

**待检查函数：**
- `AdminRevenueStats`
- `AdminFinancialReport`

### 4. 业务逻辑问题

#### 4.1 订单逻辑
- ✅ 优惠券过期检查已修复
- ⚠️ 优惠券竞态条件需手动修复（已文档化）
- ⚠️ 订单取消回滚需手动修复（已文档化）

#### 4.2 支付逻辑
- ✅ 支付回调重放已修复
- ✅ 支付金额验证已修复
- ⚠️ 需检查支付超时处理

#### 4.3 订阅逻辑
- ✅ 订阅重置已修复
- ⚠️ 需检查订阅过期处理
- ⚠️ 需检查设备限制逻辑

### 5. 代码结构问题

#### 5.1 文件组织
- ✅ 代码结构清晰
- ⚠️ admin.go 文件过大（3000+ 行）
- ⚠️ 建议拆分为多个文件

**建议拆分：**
```
internal/api/handlers/admin/
├── dashboard.go      # 仪表盘
├── users.go          # 用户管理
├── orders.go         # 订单管理
├── packages.go       # 套餐管理
├── nodes.go          # 节点管理
├── subscriptions.go  # 订阅管理
├── coupons.go        # 优惠券管理
├── stats.go          # 统计报表
└── settings.go       # 系统设置
```

#### 5.2 代码复用
- ⚠️ 存在重复的分页逻辑
- ⚠️ 存在重复的状态筛选逻辑
- ⚠️ 建议提取公共函数

### 6. 列表状态筛选问题

#### 6.1 前端状态筛选
**需检查的页面：**
- `/order/Index.vue` - 订单状态筛选
- `/admin/orders/Index.vue` - 管理员订单筛选
- `/admin/users/Index.vue` - 用户状态筛选
- `/admin/tickets/Index.vue` - 工单状态筛选

**常见问题：**
- ⚠️ 状态选项不完整
- ⚠️ 状态文本不一致
- ⚠️ 缺少"全部"选项
- ⚠️ 筛选后分页未重置

#### 6.2 后端状态筛选
**需检查：**
- 是否支持空值（查询全部）
- 是否支持多状态查询
- 是否有 SQL 注入风险（已检查，无问题）

---

## 🔧 具体修复任务

### 任务 1：修复收入统计使用错误字段

**文件：** `internal/api/handlers/admin.go`

**问题：** 使用 `amount` 而非 `final_amount`

**修复位置：**
1. AdminDashboard - 今日收入（line 63）
2. AdminDashboard - 本月收入（line 69）
3. AdminDashboard - 收入趋势（line 92）
4. AdminRevenueStats - 收入统计
5. AdminFinancialReport - 财务报表

**修复代码：**
```go
// 修改前
Select("COALESCE(SUM(amount), 0)").Scan(&revenue)

// 修改后
Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&revenue)
```

### 任务 2：排除退款订单

**修复：**
```go
// 修改前
Where("status = ?", "paid")

// 修改后
Where("status = ? OR status = ?", "paid", "completed").
Where("status != ?", "refunded")
```

### 任务 3：实现日志轮转

**新增文件：** `internal/utils/logger.go`

**实现：**
```go
package utils

import (
    "log"
    "gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() {
    log.SetOutput(&lumberjack.Logger{
        Filename:   "./logs/backend.log",
        MaxSize:    10, // MB
        MaxBackups: 3,
        MaxAge:     28, // days
        Compress:   true,
    })
}
```

### 任务 4：优化订单状态筛选

**前端修复：** `frontend/src/views/order/Index.vue`

**添加完整状态选项：**
```typescript
const statusFilters = [
  { label: '全部', value: '' },
  { label: '待支付', value: 'pending' },
  { label: '已支付', value: 'paid' },
  { label: '已完成', value: 'completed' },
  { label: '已取消', value: 'cancelled' },
  { label: '已退款', value: 'refunded' },
  { label: '已过期', value: 'expired' }
]
```

### 任务 5：添加统计缓存

**实现简单的内存缓存：**
```go
var (
    dashboardCache     interface{}
    dashboardCacheTime time.Time
    dashboardCacheMu   sync.RWMutex
)

func AdminDashboard(c *gin.Context) {
    // 检查缓存（5分钟有效）
    dashboardCacheMu.RLock()
    if time.Since(dashboardCacheTime) < 5*time.Minute && dashboardCache != nil {
        dashboardCacheMu.RUnlock()
        utils.Success(c, dashboardCache)
        return
    }
    dashboardCacheMu.RUnlock()

    // 查询数据...

    // 更新缓存
    dashboardCacheMu.Lock()
    dashboardCache = result
    dashboardCacheTime = time.Now()
    dashboardCacheMu.Unlock()
}
```

### 任务 6：拆分 admin.go 文件

**步骤：**
1. 创建 `internal/api/handlers/admin/` 目录
2. 按功能模块拆分文件
3. 更新路由引用
4. 测试所有功能

---

## 📊 优先级

### P0 - 立即修复
1. ✅ 修复收入统计字段错误
2. ✅ 排除退款订单
3. ✅ 修复订单状态筛选

### P1 - 高优先级
1. ⚠️ 实现日志轮转
2. ⚠️ 添加统计缓存
3. ⚠️ 优化状态筛选逻辑

### P2 - 中优先级
1. ⚠️ 拆分 admin.go 文件
2. ⚠️ 提取公共函数
3. ⚠️ 添加单元测试

---

## 🎯 预期效果

### 修复后
- ✅ 收入统计准确（使用实付金额）
- ✅ 退款订单正确排除
- ✅ 日志文件自动轮转
- ✅ 统计查询性能提升（缓存）
- ✅ 状态筛选完整准确
- ✅ 代码结构更清晰

---

## 📝 下一步

1. 立即修复 P0 问题
2. 测试所有修复
3. 部署到生产环境
4. 监控统计数据准确性

---

**创建时间**: 2026-03-02
**负责人**: Claude AI
**状态**: 📋 待执行
