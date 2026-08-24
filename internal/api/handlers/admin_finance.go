package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func AdminRevenueStats(c *gin.Context) {
	db := database.GetDB()
	var totalRevenue float64
	db.Model(&models.Order{}).Where("status IN ?", []string{"paid", "completed"}).Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&totalRevenue)
	var todayRevenue float64
	today := time.Now().Format("2006-01-02")
	db.Model(&models.Order{}).Where("status IN ? AND DATE(payment_time) = ?", []string{"paid", "completed"}, today).
		Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&todayRevenue)
	var monthRevenue float64
	monthStart := time.Now().Format("2006-01") + "-01"
	db.Model(&models.Order{}).Where("status IN ? AND payment_time >= ?", []string{"paid", "completed"}, monthStart).
		Select("COALESCE(SUM(COALESCE(final_amount, amount)), 0)").Scan(&monthRevenue)
	var orderCount int64
	db.Model(&models.Order{}).Where("status IN ?", []string{"paid", "completed"}).Count(&orderCount)
	utils.Success(c, gin.H{
		"total_revenue":     roundToTwoDecimals(totalRevenue),
		"today_revenue":     roundToTwoDecimals(todayRevenue),
		"monthly_revenue":   roundToTwoDecimals(monthRevenue),
		"paid_orders_count": orderCount,
	})
}

func AdminUserStats(c *gin.Context) {
	db := database.GetDB()
	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)
	var activeUsers int64
	db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUsers)
	var todayNew int64
	today := time.Now().Format("2006-01-02")
	db.Model(&models.User{}).Where("DATE(created_at) = ?", today).Count(&todayNew)
	var paidUsers int64
	db.Model(&models.Order{}).Where("status = ?", "paid").Distinct("user_id").Count(&paidUsers)
	utils.Success(c, gin.H{
		"total_users":     totalUsers,
		"active_users":    activeUsers,
		"today_new_users": todayNew,
		"paid_users":      paidUsers,
	})
}

func AdminFinancialReport(c *gin.Context) {
	db := database.GetDB()

	period := c.DefaultQuery("period", "month")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Default date range
	now := time.Now()
	var start, end time.Time
	if startDate != "" {
		s, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			utils.BadRequest(c, "start_date 格式错误，应为 YYYY-MM-DD")
			return
		}
		start = s
	} else {
		start = now.AddDate(0, 0, -29)
	}
	if endDate != "" {
		e, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			utils.BadRequest(c, "end_date 格式错误，应为 YYYY-MM-DD")
			return
		}
		end = e.Add(24*time.Hour - time.Second)
	} else {
		end = now
	}
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	// ---- Summary ----
	var totalRevenue float64
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	var totalOrders int64
	db.Model(&models.Order{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Count(&totalOrders)

	var paidOrders int64
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Count(&paidOrders)
	var refundedOrders int64
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(updated_at) >= ? AND DATE(updated_at) <= ?", "refunded", startStr, endStr).
		Count(&refundedOrders)

	var avgOrderAmount float64
	if paidOrders > 0 {
		avgOrderAmount = totalRevenue / float64(paidOrders)
	}

	var totalRecharge float64
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRecharge)

	var totalRechargeCount int64
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Count(&totalRechargeCount)

	var newUsers int64
	db.Model(&models.User{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Count(&newUsers)

	var newSubscriptions int64
	db.Model(&models.Subscription{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Count(&newSubscriptions)

	summary := gin.H{
		"total_revenue":        roundToTwoDecimals(totalRevenue),
		"total_orders":         totalOrders,
		"paid_orders":          paidOrders,
		"refunded_orders":      refundedOrders,
		"average_order_amount": roundToTwoDecimals(avgOrderAmount),
		"total_recharge":       roundToTwoDecimals(totalRecharge),
		"total_recharge_count": totalRechargeCount,
		"new_users":            newUsers,
		"new_subscriptions":    newSubscriptions,
	}

	// ---- Revenue Chart ----
	var dateExpr string
	switch period {
	case "day":
		dateExpr = "DATE(payment_time)"
	case "week":
		dateExpr = "DATE(payment_time, 'weekday 0', '-6 days')"
	default:
		dateExpr = "strftime('%Y-%m', payment_time)"
	}

	type ChartPoint struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
		Orders  int64   `json:"orders"`
	}
	var revenueChart []ChartPoint
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Select(dateExpr + " as date, COALESCE(SUM(amount), 0) as revenue, COUNT(*) as orders").
		Group(dateExpr).
		Order("date ASC").
		Scan(&revenueChart)
	// Recharge per period for chart
	type RechargePoint struct {
		Date     string  `json:"date"`
		Recharge float64 `json:"recharge"`
	}
	var rechargeByDate []RechargePoint
	var rechargeDateExpr string
	switch period {
	case "day":
		rechargeDateExpr = "DATE(paid_at)"
	case "week":
		rechargeDateExpr = "DATE(paid_at, 'weekday 0', '-6 days')"
	default:
		rechargeDateExpr = "strftime('%Y-%m', paid_at)"
	}
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Select(rechargeDateExpr + " as date, COALESCE(SUM(amount), 0) as recharge").
		Group(rechargeDateExpr).
		Order("date ASC").
		Scan(&rechargeByDate)

	rechargeMap := make(map[string]float64)
	for _, r := range rechargeByDate {
		rechargeMap[r.Date] = r.Recharge
	}
	type ChartPointFull struct {
		Date     string  `json:"date"`
		Revenue  float64 `json:"revenue"`
		Orders   int64   `json:"orders"`
		Recharge float64 `json:"recharge"`
	}
	chartFull := make([]ChartPointFull, 0, len(revenueChart))
	for _, cp := range revenueChart {
		chartFull = append(chartFull, ChartPointFull{
			Date:     cp.Date,
			Revenue:  cp.Revenue,
			Orders:   cp.Orders,
			Recharge: rechargeMap[cp.Date],
		})
	}

	// ---- Payment Method Stats ----
	type PaymentMethodStat struct {
		Method string  `json:"method"`
		Count  int64   `json:"count"`
		Amount float64 `json:"amount"`
	}
	var paymentMethodStats []PaymentMethodStat
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ? AND payment_method_name IS NOT NULL", "paid", startStr, endStr).
		Select("COALESCE(payment_method_name, '未知') as method, COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Group("payment_method_name").
		Order("amount DESC").
		Scan(&paymentMethodStats)
	// ---- Package Stats ----
	type PackageStat struct {
		PackageName string  `json:"package_name"`
		Count       int64   `json:"count"`
		Amount      float64 `json:"amount"`
	}
	var packageStats []PackageStat
	db.Model(&models.Order{}).
		Joins("LEFT JOIN packages ON packages.id = orders.package_id").
		Where("orders.status = ? AND DATE(orders.payment_time) >= ? AND DATE(orders.payment_time) <= ?", "paid", startStr, endStr).
		Select("COALESCE(packages.name, '未知套餐') as package_name, COUNT(*) as count, COALESCE(SUM(orders.amount), 0) as amount").
		Group("orders.package_id").
		Order("amount DESC").
		Scan(&packageStats)

	// ---- Top Users ----
	type TopUser struct {
		UserID     uint    `json:"user_id"`
		Username   string  `json:"username"`
		TotalSpent float64 `json:"total_spent"`
		OrderCount int64   `json:"order_count"`
	}
	var topUsers []TopUser
	db.Model(&models.Order{}).
		Joins("LEFT JOIN users ON users.id = orders.user_id").
		Where("orders.status = ? AND DATE(orders.payment_time) >= ? AND DATE(orders.payment_time) <= ?", "paid", startStr, endStr).
		Select("orders.user_id, COALESCE(users.username, '未知') as username, COALESCE(SUM(orders.amount), 0) as total_spent, COUNT(*) as order_count").
		Group("orders.user_id").
		Order("total_spent DESC").
		Limit(10).
		Scan(&topUsers)

	utils.Success(c, gin.H{
		"summary":              summary,
		"revenue_chart":        chartFull,
		"payment_method_stats": paymentMethodStats,
		"package_stats":        packageStats,
		"top_users":            topUsers,
	})
}

func AdminExportFinancialReport(c *gin.Context) {
	db := database.GetDB()

	period := c.DefaultQuery("period", "month")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	now := time.Now()
	var start, end time.Time
	if startDate != "" {
		s, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			utils.BadRequest(c, "start_date 格式错误")
			return
		}
		start = s
	} else {
		start = now.AddDate(0, 0, -29)
	}
	if endDate != "" {
		e, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			utils.BadRequest(c, "end_date 格式错误")
			return
		}
		end = e.Add(24*time.Hour - time.Second)
	} else {
		end = now
	}
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	var dateExpr string
	switch period {
	case "day":
		dateExpr = "DATE(payment_time)"
	case "week":
		dateExpr = "DATE(payment_time, 'weekday 0', '-6 days')"
	default:
		dateExpr = "strftime('%Y-%m', payment_time)"
	}

	type Row struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
		Orders  int64   `json:"orders"`
	}
	var rows []Row
	db.Model(&models.Order{}).
		Where("status = ? AND DATE(payment_time) >= ? AND DATE(payment_time) <= ?", "paid", startStr, endStr).
		Select(dateExpr + " as date, COALESCE(SUM(amount), 0) as revenue, COUNT(*) as orders").
		Group(dateExpr).
		Order("date ASC").
		Scan(&rows)

	// Recharge per period
	var rechargeDateExpr string
	switch period {
	case "day":
		rechargeDateExpr = "DATE(paid_at)"
	case "week":
		rechargeDateExpr = "DATE(paid_at, 'weekday 0', '-6 days')"
	default:
		rechargeDateExpr = "strftime('%Y-%m', paid_at)"
	}
	type RRow struct {
		Date     string  `json:"date"`
		Recharge float64 `json:"recharge"`
	}
	var rrows []RRow
	db.Model(&models.RechargeRecord{}).
		Where("status = ? AND DATE(paid_at) >= ? AND DATE(paid_at) <= ?", "paid", startStr, endStr).
		Select(rechargeDateExpr + " as date, COALESCE(SUM(amount), 0) as recharge").
		Group(rechargeDateExpr).
		Order("date ASC").
		Scan(&rrows)
	rechargeMap := make(map[string]float64)
	for _, r := range rrows {
		rechargeMap[r.Date] = r.Recharge
	}

	// New users per period
	var userDateExpr string
	switch period {
	case "day":
		userDateExpr = "DATE(created_at)"
	case "week":
		userDateExpr = "DATE(created_at, 'weekday 0', '-6 days')"
	default:
		userDateExpr = "strftime('%Y-%m', created_at)"
	}
	type URow struct {
		Date     string `json:"date"`
		NewUsers int64  `json:"new_users"`
	}
	var urows []URow
	db.Model(&models.User{}).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", startStr, endStr).
		Select(userDateExpr + " as date, COUNT(*) as new_users").
		Group(userDateExpr).
		Order("date ASC").
		Scan(&urows)
	userMap := make(map[string]int64)
	for _, u := range urows {
		userMap[u.Date] = u.NewUsers
	}

	filename := fmt.Sprintf("financial_report_%s.csv", now.Format("2006-01-02"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	// BOM for Excel UTF-8
	if _, err := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		utils.InternalError(c, "导出失败")
		return
	}
	if _, err := c.Writer.WriteString("日期,收入,订单数,充值,新用户\n"); err != nil {
		utils.InternalError(c, "导出失败")
		return
	}
	for _, row := range rows {
		line := fmt.Sprintf("%s,%.2f,%d,%.2f,%d\n",
			sanitizeCSVCell(row.Date), row.Revenue, row.Orders, rechargeMap[row.Date], userMap[row.Date])
		if _, err := c.Writer.WriteString(line); err != nil {
			utils.InternalError(c, "导出失败")
			return
		}
	}
}

// ==================== Region Statistics ====================

func AdminRegionStats(c *gin.Context) {
	db := database.GetDB()

	type RawRegion struct {
		Location string
		Count    int64
	}

	var rawRegions []RawRegion
	db.Model(&models.LoginHistory{}).
		Select("COALESCE(location, '未知') as location, COUNT(DISTINCT user_id) as count").
		Where("location IS NOT NULL AND location != ''").
		Group("location").
		Order("count DESC").
		Find(&rawRegions)

	// 解析 location 字段（兼容旧 JSON 格式和新纯文本格式），按 国家|省份|城市 聚合
	type RegionKey struct {
		Country  string
		Province string
		City     string
	}
	aggregated := make(map[RegionKey]int64)

	for _, r := range rawRegions {
		loc := strings.TrimSpace(r.Location)
		var country, province, city string

		if strings.HasPrefix(loc, "{") {
			// 旧 JSON 格式: {"country":"中国","country_code":"CN","city":"深圳","region":"广东",...}
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(loc), &obj); err == nil {
				if v, ok := obj["country"].(string); ok {
					country = v
				}
				if v, ok := obj["region"].(string); ok {
					province = v
				}
				if v, ok := obj["city"].(string); ok {
					city = v
				}
			}
		} else {
			// 新纯文本格式: "国家 省份 城市"
			parts := strings.Fields(loc)
			if len(parts) >= 1 {
				country = parts[0]
			}
			if len(parts) >= 3 {
				province = parts[1]
				city = parts[2]
			} else if len(parts) == 2 {
				city = parts[1]
			}
		}

		if country == "" {
			country = "未知"
		}

		key := RegionKey{Country: country, Province: province, City: city}
		aggregated[key] += r.Count
	}

	// 转为切片并排序
	type RegionResult struct {
		Country  string `json:"country"`
		Province string `json:"province"`
		City     string `json:"city"`
		Count    int64  `json:"count"`
	}

	results := make([]RegionResult, 0, len(aggregated))
	for k, v := range aggregated {
		results = append(results, RegionResult{
			Country:  k.Country,
			Province: k.Province,
			City:     k.City,
			Count:    v,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Count > results[j].Count
	})

	if len(results) > 30 {
		results = results[:30]
	}

	utils.Success(c, results)
}

// ==================== Batch Operations ====================

