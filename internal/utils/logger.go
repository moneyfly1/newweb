package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	logFile   *os.File
	appLogger *log.Logger
)

// InitLogger 初始化日志系统：日志写入 logs/ 目录下按日期命名的文件，同时保留控制台输出。
func InitLogger() error {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	logFileName := filepath.Join(logDir, fmt.Sprintf("app-%s.log", time.Now().Format("2006-01-02")))
	// #nosec G304 -- log path is fixed under internal "logs" directory.
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	logFile = file
	appLogger = log.New(file, "", log.LstdFlags)
	// 其余包直接使用标准库 log 时，也统一写入日志文件
	log.SetOutput(file)

	LogInfo("========================================")
	LogInfo("日志系统已启动")
	LogInfo("日志文件: %s", logFileName)
	LogInfo("========================================")

	return nil
}

// CloseLogger 关闭日志系统。
func CloseLogger() {
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
		appLogger = nil
	}
}

// writeLog 输出一条带级别标签的日志：写入日志文件，同时打印到控制台。
func writeLog(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[%s] %s", level, msg)
	}
	fmt.Printf("[%s] %s\n", level, msg)
}

// LogInfo 记录信息日志
func LogInfo(format string, args ...interface{}) { writeLog("INFO", format, args...) }

// LogError 记录错误日志
func LogError(format string, args ...interface{}) { writeLog("ERROR", format, args...) }

// LogPayment 记录支付相关日志
func LogPayment(format string, args ...interface{}) { writeLog("PAYMENT", format, args...) }

// LogCallback 记录支付回调相关日志
func LogCallback(format string, args ...interface{}) { writeLog("CALLBACK", format, args...) }

// LogOrder 记录订单相关日志
func LogOrder(format string, args ...interface{}) { writeLog("ORDER", format, args...) }
