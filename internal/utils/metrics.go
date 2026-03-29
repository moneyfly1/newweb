package utils

import (
	"time"
)

type PerformanceMetrics struct {
	RequestCount   int64
	ErrorCount     int64
	TotalDuration  time.Duration
	AverageDuration time.Duration
}

var metrics = &PerformanceMetrics{}

func RecordRequest(duration time.Duration, hasError bool) {
	metrics.RequestCount++
	metrics.TotalDuration += duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.RequestCount)

	if hasError {
		metrics.ErrorCount++
	}
}

func GetMetrics() *PerformanceMetrics {
	return metrics
}
