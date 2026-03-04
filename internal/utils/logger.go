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

// InitLogger 初始化日志系统
func InitLogger() error {
	// 创建 logs 目录
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建日志文件（按日期）
	logFileName := filepath.Join(logDir, fmt.Sprintf("app-%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	logFile = file
	appLogger = log.New(file, "", log.LstdFlags)

	// 同时输出到控制台和文件
	log.SetOutput(file)

	LogInfo("========================================")
	LogInfo("日志系统已启动")
	LogInfo("日志文件: %s", logFileName)
	LogInfo("========================================")

	return nil
}

// CloseLogger 关闭日志系统
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// LogInfo 记录信息日志
func LogInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[INFO] %s", msg)
	}
	fmt.Printf("[INFO] %s\n", msg)
}

// LogError 记录错误日志
func LogError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[ERROR] %s", msg)
	}
	fmt.Printf("[ERROR] %s\n", msg)
}

// LogWarn 记录警告日志
func LogWarn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[WARN] %s", msg)
	}
	fmt.Printf("[WARN] %s\n", msg)
}

// LogDebug 记录调试日志
func LogDebug(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[DEBUG] %s", msg)
	}
	fmt.Printf("[DEBUG] %s\n", msg)
}

// LogPayment 记录支付相关日志
func LogPayment(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[PAYMENT] %s", msg)
	}
	fmt.Printf("[PAYMENT] %s\n", msg)
}

// LogCallback 记录回调相关日志
func LogCallback(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[CALLBACK] %s", msg)
	}
	fmt.Printf("[CALLBACK] %s\n", msg)
}

// LogOrder 记录订单相关日志
func LogOrder(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if appLogger != nil {
		appLogger.Printf("[ORDER] %s", msg)
	}
	fmt.Printf("[ORDER] %s\n", msg)
}
