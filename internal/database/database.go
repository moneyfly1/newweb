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
	if DB.Dialector.Name() == "sqlite" {
		// SQLite with WAL mode supports better concurrency
		sqlDB.SetMaxOpenConns(50)
	} else {
		sqlDB.SetMaxOpenConns(100)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

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
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("创建 SQLite 数据库目录失败: %w", err)
	}

	// WAL mode + pragmas for concurrent web use
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_cache_size=-20000&_synchronous=NORMAL&_foreign_keys=ON"
	log.Printf("使用 SQLite 数据库: %s", dbPath)
	return sqlite.Open(dsn), nil
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

	var existingUserLevelColumns []string
	if DB.Migrator().HasTable(&models.UserLevel{}) {
		columnTypes, err := DB.Migrator().ColumnTypes(&models.UserLevel{})
		if err != nil {
			return fmt.Errorf("读取 user_levels 表结构失败: %w", err)
		}
		for _, ct := range columnTypes {
			existingUserLevelColumns = append(existingUserLevelColumns, ct.Name())
		}
	}

	err := DB.AutoMigrate(
		// 用户相关
		&models.User{},
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
		&models.PaymentNonce{},

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

		// 扩展日志（新增）
		&models.OrderLog{},
		&models.PaymentLog{},
		&models.CouponLog{},
		&models.NodeLog{},
		&models.UserActionLog{},
		&models.AdminActionLog{},
		&models.DeviceLog{},
		&models.TicketLog{},
		&models.InviteLog{},
		&models.ConfigChangeLog{},
		&models.SecurityLog{},
		&models.APILog{},
		&models.DatabaseLog{},
		&models.EmailLog{},
		&models.NotificationLog{},

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

	if len(existingUserLevelColumns) > 0 {
		hasDeviceLimit := false
		for _, name := range existingUserLevelColumns {
			if strings.EqualFold(name, "device_limit") {
				hasDeviceLimit = true
				break
			}
		}
		if hasDeviceLimit {
			if err := restoreUserLevelDeviceLimitColumn(); err != nil {
				return err
			}
		} else {
			if err := DB.AutoMigrate(&models.UserLevel{}); err != nil {
				return fmt.Errorf("数据库迁移 user_levels 失败: %w", err)
			}
		}
	} else {
		if err := DB.AutoMigrate(&models.UserLevel{}); err != nil {
			return fmt.Errorf("数据库迁移 user_levels 失败: %w", err)
		}
	}

	log.Println("数据库迁移完成")
	return nil
}

type legacyUserLevel struct {
	ID             uint      `gorm:"primaryKey"`
	LevelName      string    `gorm:"column:level_name"`
	LevelOrder     int       `gorm:"column:level_order"`
	MinConsumption float64   `gorm:"column:min_consumption"`
	DiscountRate   float64   `gorm:"column:discount_rate"`
	DeviceLimit    int       `gorm:"column:device_limit"`
	Benefits       *string   `gorm:"column:benefits"`
	IconURL        *string   `gorm:"column:icon_url"`
	Color          string    `gorm:"column:color"`
	IsActive       bool      `gorm:"column:is_active"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (legacyUserLevel) TableName() string { return "user_levels" }

func restoreUserLevelDeviceLimitColumn() error {
	if DB.Migrator().HasColumn(&legacyUserLevel{}, "device_limit") {
		return nil
	}
	if err := DB.Migrator().AddColumn(&legacyUserLevel{}, "device_limit"); err != nil {
		return fmt.Errorf("恢复 user_levels.device_limit 字段失败: %w", err)
	}
	return nil
}

// GetDB 获取全局数据库实例
func GetDB() *gorm.DB {
	return DB
}

// Close closes the underlying database connection pool.
func Close() {
	if DB == nil {
		return
	}
	if sqlDB, err := DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
