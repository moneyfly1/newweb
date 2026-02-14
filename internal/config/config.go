package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// AppConfig 全局配置实例
var AppConfig *Config

// Config 应用全局配置结构体
type Config struct {
	// 项目基本信息
	ProjectName string `mapstructure:"PROJECT_NAME"`
	Version     string `mapstructure:"VERSION"`
	BaseURL     string `mapstructure:"BASE_URL"`

	// 数据库配置（支持 SQLite / MySQL / PostgreSQL）
	DatabaseURL     string `mapstructure:"DATABASE_URL"`
	MySQLHost       string `mapstructure:"MYSQL_HOST"`
	MySQLPort       int    `mapstructure:"MYSQL_PORT"`
	MySQLUser       string `mapstructure:"MYSQL_USER"`
	MySQLPassword   string `mapstructure:"MYSQL_PASSWORD"`
	MySQLDatabase   string `mapstructure:"MYSQL_DATABASE"`
	PostgresServer  string `mapstructure:"POSTGRES_SERVER"`
	PostgresUser    string `mapstructure:"POSTGRES_USER"`
	PostgresPass    string `mapstructure:"POSTGRES_PASS"`
	PostgresDB      string `mapstructure:"POSTGRES_DB"`

	// JWT / 安全配置
	SecretKey                string `mapstructure:"SECRET_KEY"`
	JWTAlgorithm             string `mapstructure:"JWT_ALGORITHM"`
	AccessTokenExpireMinutes int    `mapstructure:"ACCESS_TOKEN_EXPIRE_MINUTES"`
	RefreshTokenExpireDays   int    `mapstructure:"REFRESH_TOKEN_EXPIRE_DAYS"`

	// 邮件 SMTP 配置
	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     int    `mapstructure:"SMTP_PORT"`
	SMTPUser     string `mapstructure:"SMTP_USERNAME"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	FromEmail    string `mapstructure:"SMTP_FROM_EMAIL"`
	FromName     string `mapstructure:"SMTP_FROM_NAME"`
	SMTPTLS      bool   `mapstructure:"SMTP_TLS"`

	// 服务器配置
	Host  string `mapstructure:"HOST"`
	Port  int    `mapstructure:"PORT"`
	Debug bool   `mapstructure:"DEBUG"`

	// 订阅与设备限制
	SubscriptionURLPrefix string `mapstructure:"SUBSCRIPTION_URL_PREFIX"`
	DeviceLimitDefault    int    `mapstructure:"DEVICE_LIMIT_DEFAULT"`

	// 文件上传
	UploadDir   string `mapstructure:"UPLOAD_DIR"`
	MaxFileSize int64  `mapstructure:"MAX_FILE_SIZE"`

	// 调度与性能
	DisableScheduleTasks bool `mapstructure:"DISABLE_SCHEDULE_TASKS"`
	OptimizeForLowEnd    bool `mapstructure:"OPTIMIZE_FOR_LOW_END"`

	// 设备升级价格（每月）
	DeviceUpgradePricePerMonth float64 `mapstructure:"DEVICE_UPGRADE_PRICE_PER_MONTH"`

	// Telegram 机器人（v2 新增）
	TelegramBotToken  string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	TelegramWebhookURL string `mapstructure:"TELEGRAM_WEBHOOK_URL"`

	// CORS 允许的来源
	CorsOrigins []string `mapstructure:"CORS_ORIGINS"`
}

// generateSecretKey 生成指定字节数的随机密钥（hex 编码）
func generateSecretKey(bytes int) string {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("无法生成随机密钥: %v", err))
	}
	return hex.EncodeToString(b)
}

// setDefaults 设置所有配置项的默认值
func setDefaults() {
	viper.SetDefault("PROJECT_NAME", "CBoard")
	viper.SetDefault("VERSION", "2.0.0")
	viper.SetDefault("BASE_URL", "http://localhost:8000")

	// 数据库默认使用 SQLite
	viper.SetDefault("DATABASE_URL", "sqlite:///./cboard.db")
	viper.SetDefault("MYSQL_PORT", 3306)

	// JWT 默认值
	viper.SetDefault("JWT_ALGORITHM", "HS256")
	viper.SetDefault("ACCESS_TOKEN_EXPIRE_MINUTES", 1440)
	viper.SetDefault("REFRESH_TOKEN_EXPIRE_DAYS", 7)

	// SMTP 默认值
	viper.SetDefault("SMTP_PORT", 587)
	viper.SetDefault("SMTP_TLS", true)

	// 服务器默认值
	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("PORT", 8000)
	viper.SetDefault("DEBUG", false)

	// 业务默认值
	viper.SetDefault("DEVICE_LIMIT_DEFAULT", 3)
	viper.SetDefault("UPLOAD_DIR", "uploads")
	viper.SetDefault("MAX_FILE_SIZE", 10485760) // 10MB
	viper.SetDefault("DISABLE_SCHEDULE_TASKS", false)
	viper.SetDefault("OPTIMIZE_FOR_LOW_END", false)
	viper.SetDefault("DEVICE_UPGRADE_PRICE_PER_MONTH", 0)
}

// LoadConfig 从 .env 文件和环境变量加载配置
// 优先级：环境变量 > .env 文件 > 默认值
func LoadConfig() (*Config, error) {
	setDefaults()

	// 读取 .env 文件（不存在也不报错）
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	if err := viper.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("读取配置文件失败: %w", err)
			}
		}
	}

	// 环境变量覆盖
	viper.AutomaticEnv()

	// 处理 CORS_ORIGINS（逗号分隔字符串 -> 切片）
	if origins := viper.GetString("CORS_ORIGINS"); origins != "" {
		parts := strings.Split(origins, ",")
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				cleaned = append(cleaned, s)
			}
		}
		viper.Set("CORS_ORIGINS", cleaned)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// SecretKey 未设置时自动生成（至少 32 字符）
	if cfg.SecretKey == "" {
		cfg.SecretKey = generateSecretKey(32) // 64 hex chars
		fmt.Println("[config] 警告: SECRET_KEY 未设置，已自动生成随机密钥（仅适用于开发环境）")
	}
	if len(cfg.SecretKey) < 32 {
		return nil, fmt.Errorf("SECRET_KEY 长度不足，至少需要 32 个字符（当前 %d）", len(cfg.SecretKey))
	}

	// 生产环境额外校验
	if !cfg.Debug {
		if err := validateProduction(&cfg); err != nil {
			return nil, fmt.Errorf("生产环境配置校验失败: %w", err)
		}
	}

	AppConfig = &cfg
	return &cfg, nil
}

// GetSecretKey 返回当前配置的 SecretKey
func GetSecretKey() string {
	if AppConfig != nil {
		return AppConfig.SecretKey
	}
	return ""
}

// validateProduction 生产环境配置校验
func validateProduction(cfg *Config) error {
	var errs []string

	if cfg.SecretKey == "change-me-to-a-strong-random-32-bytes" {
		errs = append(errs, "请更换默认的 SECRET_KEY")
	}
	if cfg.BaseURL == "" || cfg.BaseURL == "http://localhost:8000" {
		errs = append(errs, "生产环境请设置正确的 BASE_URL")
	}
	if len(cfg.CorsOrigins) == 0 {
		errs = append(errs, "生产环境请设置 CORS_ORIGINS")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
