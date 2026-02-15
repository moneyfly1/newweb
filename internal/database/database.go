package database

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard/v2/internal/config"
	"cboard/v2/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库实例
var DB *gorm.DB

// InitDatabase 根据配置初始化数据库连接
// 支持 sqlite、mysql、postgres 三种数据库类型
func InitDatabase(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	dialector, err := buildDialector(cfg)
	if err != nil {
		return fmt.Errorf("构建数据库驱动失败: %w", err)
	}

	// 根据 Debug 模式配置 GORM 日志级别
	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库实例失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 验证连接可用
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Printf("数据库连接成功 (driver=%s)", DB.Dialector.Name())
	return nil
}

// buildDialector 根据 DatabaseURL 或独立字段构建 GORM Dialector
func buildDialector(cfg *config.Config) (gorm.Dialector, error) {
	dbURL := strings.TrimSpace(cfg.DatabaseURL)

	switch {
	case strings.HasPrefix(dbURL, "sqlite"):
		return buildSQLite(dbURL)
	case strings.HasPrefix(dbURL, "mysql"):
		return buildMySQL(cfg)
	case strings.HasPrefix(dbURL, "postgres"):
		return buildPostgres(cfg)
	default:
		// 未指定或无法识别时，默认使用 SQLite
		return buildSQLite("sqlite:///./cboard.db")
	}
}

// buildSQLite 解析 sqlite:///path/to/db 格式并返回 SQLite Dialector
func buildSQLite(dbURL string) (gorm.Dialector, error) {
	// 去掉 "sqlite:///" 前缀，提取文件路径
	dbPath := strings.TrimPrefix(dbURL, "sqlite:///")
	dbPath = strings.TrimPrefix(dbPath, "sqlite://")
	if dbPath == "" {
		dbPath = "cboard.db"
	}

	// 相对路径转为基于当前工作目录的绝对路径
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(".", dbPath)
	}

	// 确保父目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建 SQLite 数据库目录失败: %w", err)
	}

	log.Printf("使用 SQLite 数据库: %s", dbPath)
	return sqlite.Open(dbPath), nil
}

// buildMySQL 构建 MySQL DSN
// 优先从 DatabaseURL 解析，若 URL 仅为 "mysql://" 标识则使用独立字段
func buildMySQL(cfg *config.Config) (gorm.Dialector, error) {
	dbURL := cfg.DatabaseURL

	// 尝试从完整 URL 解析: mysql://user:pass@host:port/dbname
	if strings.Contains(dbURL, "@") {
		parsed, err := url.Parse(dbURL)
		if err == nil && parsed.Host != "" {
			password, _ := parsed.User.Password()
			dbName := strings.TrimPrefix(parsed.Path, "/")
			dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				parsed.User.Username(), password, parsed.Host, dbName,
			)
			log.Printf("使用 MySQL 数据库: %s@%s/%s", parsed.User.Username(), parsed.Host, dbName)
			return mysql.Open(dsn), nil
		}
	}

	// 回退到独立配置字段
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQLUser, cfg.MySQLPassword,
		cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDatabase,
	)
	log.Printf("使用 MySQL 数据库: %s@%s:%d/%s", cfg.MySQLUser, cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDatabase)
	return mysql.Open(dsn), nil
}

// buildPostgres 构建 PostgreSQL DSN
// 优先从 DatabaseURL 解析，若 URL 仅为 "postgres://" 标识则使用独立字段
func buildPostgres(cfg *config.Config) (gorm.Dialector, error) {
	dbURL := cfg.DatabaseURL

	// 尝试从完整 URL 解析: postgres://user:pass@host:port/dbname
	if strings.Contains(dbURL, "@") {
		parsed, err := url.Parse(dbURL)
		if err == nil && parsed.Host != "" {
			password, _ := parsed.User.Password()
			dbName := strings.TrimPrefix(parsed.Path, "/")
			dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
				parsed.Hostname(), parsed.User.Username(), password, dbName, parsed.Port(),
			)
			log.Printf("使用 PostgreSQL 数据库: %s@%s/%s", parsed.User.Username(), parsed.Host, dbName)
			return postgres.Open(dsn), nil
		}
	}

	// 回退到独立配置字段
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Asia/Shanghai",
		cfg.PostgresServer, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDB,
	)
	log.Printf("使用 PostgreSQL 数据库: %s@%s/%s", cfg.PostgresUser, cfg.PostgresServer, cfg.PostgresDB)
	return postgres.Open(dsn), nil
}

// AutoMigrate 自动迁移所有数据模型表结构
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化，请先调用 InitDatabase")
	}

	err := DB.AutoMigrate(
		// 用户相关
		&models.User{},
		&models.UserLevel{},
		&models.TokenBlacklist{},

		// 订阅与设备
		&models.Subscription{},
		&models.SubscriptionReset{},
		&models.Device{},

		// 节点
		&models.Node{},
		&models.CustomNode{},
		&models.UserCustomNode{},

		// 订单与套餐
		&models.Order{},
		&models.Package{},

		// 支付
		&models.PaymentTransaction{},
		&models.PaymentCallback{},
		&models.PaymentConfig{},

		// 优惠券
		&models.Coupon{},
		&models.CouponUsage{},

		// 邀请
		&models.InviteCode{},
		&models.InviteRelation{},

		// 通知与邮件
		&models.Notification{},
		&models.EmailTemplate{},
		&models.EmailQueue{},

		// 工单
		&models.Ticket{},
		&models.TicketReply{},
		&models.TicketAttachment{},
		&models.TicketRead{},

		// 充值
		&models.RechargeRecord{},

		// 系统配置
		&models.SystemConfig{},
		&models.Announcement{},
		&models.ThemeConfig{},

		// 审计与安全
		&models.AuditLog{},
		&models.LoginAttempt{},
		&models.LoginHistory{},
		&models.UserActivity{},
		&models.VerificationCode{},
		&models.VerificationAttempt{},

		// 日志
		&models.RegistrationLog{},
		&models.SubscriptionLog{},
		&models.BalanceLog{},
		&models.CommissionLog{},
		&models.SystemLog{},

		// 卡密（v2 新增）
		&models.RedeemCode{},
		&models.RedeemRecord{},

		// 签到
		&models.CheckIn{},

		// 盲盒
		&models.MysteryBoxPool{},
		&models.MysteryBoxPrize{},
		&models.MysteryBoxRecord{},
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}

// GetDB 获取全局数据库实例
func GetDB() *gorm.DB {
	return DB
}
