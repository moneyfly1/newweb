package handlers

import (
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminPaymentStats 支付统计
func AdminPaymentStats(c *gin.Context) {
	db := database.GetDB()

	// 获取时间范围参数
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	// 按支付方式统计
	type PaymentMethodStat struct {
		PaymentMethod string  `json:"payment_method"`
		OrderCount    int64   `json:"order_count"`
		TotalAmount   float64 `json:"total_amount"`
		SuccessCount  int64   `json:"success_count"`
		SuccessRate   float64 `json:"success_rate"`
	}

	var methodStats []PaymentMethodStat
	db.Model(&models.Order{}).
		Select("payment_method, COUNT(*) as order_count, SUM(COALESCE(final_amount, amount)) as total_amount, "+
			"SUM(CASE WHEN status IN ('paid', 'completed') THEN 1 ELSE 0 END) as success_count").
		Where("DATE(created_at) BETWEEN ? AND ?", startDate, endDate).
		Group("payment_method").
		Scan(&methodStats)

	// 计算成功率
	for i := range methodStats {
		if methodStats[i].OrderCount > 0 {
			methodStats[i].SuccessRate = float64(methodStats[i].SuccessCount) / float64(methodStats[i].OrderCount) * 100
		}
	}

	// 按日期统计支付趋势
	type DailyPaymentStat struct {
		Date        string  `json:"date"`
		OrderCount  int64   `json:"order_count"`
		TotalAmount float64 `json:"total_amount"`
	}

	var dailyStats []DailyPaymentStat
	db.Model(&models.Order{}).
		Select("DATE(payment_time) as date, COUNT(*) as order_count, SUM(COALESCE(final_amount, amount)) as total_amount").
		Where("status IN ? AND DATE(payment_time) BETWEEN ? AND ?", []string{"paid", "completed"}, startDate, endDate).
		Group("DATE(payment_time)").
		Order("date ASC").
		Scan(&dailyStats)

	// 支付失败原因统计
	type FailureReasonStat struct {
		Reason string `json:"reason"`
		Count  int64  `json:"count"`
	}

	var failureStats []FailureReasonStat
	db.Model(&models.Order{}).
		Select("COALESCE(failure_reason, '未知原因') as reason, COUNT(*) as count").
		Where("status = ? AND DATE(created_at) BETWEEN ? AND ?", "failed", startDate, endDate).
		Group("failure_reason").
		Order("count DESC").
		Limit(10).
		Scan(&failureStats)

	// 总体统计
	type OverallStat struct {
		TotalOrders   int64   `json:"total_orders"`
		SuccessOrders int64   `json:"success_orders"`
		FailedOrders  int64   `json:"failed_orders"`
		PendingOrders int64   `json:"pending_orders"`
		TotalAmount   float64 `json:"total_amount"`
		AverageAmount float64 `json:"average_amount"`
		SuccessRate   float64 `json:"success_rate"`
	}

	var overall OverallStat
	db.Model(&models.Order{}).
		Select("COUNT(*) as total_orders, "+
			"SUM(CASE WHEN status IN ('paid', 'completed') THEN 1 ELSE 0 END) as success_orders, "+
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_orders, "+
			"SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_orders, "+
			"SUM(CASE WHEN status IN ('paid', 'completed') THEN COALESCE(final_amount, amount) ELSE 0 END) as total_amount").
		Where("DATE(created_at) BETWEEN ? AND ?", startDate, endDate).
		Scan(&overall)

	if overall.SuccessOrders > 0 {
		overall.AverageAmount = overall.TotalAmount / float64(overall.SuccessOrders)
	}
	if overall.TotalOrders > 0 {
		overall.SuccessRate = float64(overall.SuccessOrders) / float64(overall.TotalOrders) * 100
	}

	// 支付方式使用趋势（按日期和支付方式）
	type PaymentTrend struct {
		Date          string `json:"date"`
		PaymentMethod string `json:"payment_method"`
		OrderCount    int64  `json:"order_count"`
	}

	var trends []PaymentTrend
	db.Model(&models.Order{}).
		Select("DATE(created_at) as date, payment_method, COUNT(*) as order_count").
		Where("DATE(created_at) BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at), payment_method").
		Order("date ASC").
		Scan(&trends)

	utils.Success(c, gin.H{
		"overall":        overall,
		"method_stats":   methodStats,
		"daily_stats":    dailyStats,
		"failure_stats":  failureStats,
		"payment_trends": trends,
		"start_date":     startDate,
		"end_date":       endDate,
	})
}

// AdminPaymentMethodComparison 支付方式对比
func AdminPaymentMethodComparison(c *gin.Context) {
	db := database.GetDB()

	// 获取时间范围
	days := c.DefaultQuery("days", "30")
	startDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	if days == "7" {
		startDate = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	} else if days == "90" {
		startDate = time.Now().AddDate(0, 0, -90).Format("2006-01-02")
	}

	type MethodComparison struct {
		PaymentMethod string  `json:"payment_method"`
		TotalOrders   int64   `json:"total_orders"`
		SuccessOrders int64   `json:"success_orders"`
		FailedOrders  int64   `json:"failed_orders"`
		TotalAmount   float64 `json:"total_amount"`
		AverageAmount float64 `json:"average_amount"`
		SuccessRate   float64 `json:"success_rate"`
		AverageTime   float64 `json:"average_time"` // 平均支付时间（分钟）
	}

	var comparisons []MethodComparison
	db.Model(&models.Order{}).
		Select("payment_method, "+
			"COUNT(*) as total_orders, "+
			"SUM(CASE WHEN status IN ('paid', 'completed') THEN 1 ELSE 0 END) as success_orders, "+
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_orders, "+
			"SUM(CASE WHEN status IN ('paid', 'completed') THEN COALESCE(final_amount, amount) ELSE 0 END) as total_amount, "+
			"AVG(CASE WHEN status IN ('paid', 'completed') AND payment_time IS NOT NULL THEN "+
			"TIMESTAMPDIFF(MINUTE, created_at, payment_time) ELSE NULL END) as average_time").
		Where("DATE(created_at) >= ?", startDate).
		Group("payment_method").
		Scan(&comparisons)

	// 计算成功率和平均金额
	for i := range comparisons {
		if comparisons[i].TotalOrders > 0 {
			comparisons[i].SuccessRate = float64(comparisons[i].SuccessOrders) / float64(comparisons[i].TotalOrders) * 100
		}
		if comparisons[i].SuccessOrders > 0 {
			comparisons[i].AverageAmount = comparisons[i].TotalAmount / float64(comparisons[i].SuccessOrders)
		}
	}

	utils.Success(c, gin.H{
		"comparisons": comparisons,
		"days":        days,
		"start_date":  startDate,
	})
}

// AdminPaymentAnalysis 支付分析详情
func AdminPaymentAnalysis(c *gin.Context) {
	db := database.GetDB()
	paymentMethod := c.Query("payment_method")

	if paymentMethod == "" {
		utils.BadRequest(c, "请指定支付方式")
		return
	}

	// 获取时间范围
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	// 按小时统计（用于分析高峰时段）
	type HourlyStat struct {
		Hour        int     `json:"hour"`
		OrderCount  int64   `json:"order_count"`
		SuccessRate float64 `json:"success_rate"`
	}

	var hourlyStats []HourlyStat
	db.Model(&models.Order{}).
		Select("HOUR(created_at) as hour, COUNT(*) as order_count, "+
			"SUM(CASE WHEN status IN ('paid', 'completed') THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as success_rate").
		Where("payment_method = ? AND DATE(created_at) BETWEEN ? AND ?", paymentMethod, startDate, endDate).
		Group("HOUR(created_at)").
		Order("hour ASC").
		Scan(&hourlyStats)

	// 金额分布
	type AmountRange struct {
		Range      string `json:"range"`
		OrderCount int64  `json:"order_count"`
	}

	var amountRanges []AmountRange
	db.Raw(`
		SELECT
			CASE
				WHEN amount < 10 THEN '0-10'
				WHEN amount < 50 THEN '10-50'
				WHEN amount < 100 THEN '50-100'
				WHEN amount < 200 THEN '100-200'
				WHEN amount < 500 THEN '200-500'
				ELSE '500+'
			END as range,
			COUNT(*) as order_count
		FROM orders
		WHERE payment_method = ? AND DATE(created_at) BETWEEN ? AND ?
		GROUP BY range
		ORDER BY
			CASE range
				WHEN '0-10' THEN 1
				WHEN '10-50' THEN 2
				WHEN '50-100' THEN 3
				WHEN '100-200' THEN 4
				WHEN '200-500' THEN 5
				ELSE 6
			END
	`, paymentMethod, startDate, endDate).Scan(&amountRanges)

	// 最近失败订单
	var recentFailures []models.Order
	db.Where("payment_method = ? AND status = ?", paymentMethod, "failed").
		Order("created_at DESC").
		Limit(10).
		Find(&recentFailures)

	utils.Success(c, gin.H{
		"payment_method":  paymentMethod,
		"hourly_stats":    hourlyStats,
		"amount_ranges":   amountRanges,
		"recent_failures": recentFailures,
		"start_date":      startDate,
		"end_date":        endDate,
	})
}
