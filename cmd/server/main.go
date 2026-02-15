package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cboard/v2/internal/api/router"
	"cboard/v2/internal/config"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 子命令: 重设管理员密码 (供安装脚本菜单「重设管理员密码」调用)
	if len(os.Args) >= 2 && os.Args[1] == "reset-password" {
		runResetPassword()
		return
	}

	log.Println("CBoard v2.0 启动中...")

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	log.Printf("项目: %s v%s", cfg.ProjectName, cfg.Version)

	// 初始化数据库
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 自动迁移
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建默认管理员
	createDefaultAdmin()

	// 确保所有用户都有订阅记录
	ensureUserSubscriptions()

	// 启动节点自动更新定时任务
	services.GetConfigUpdateService().StartSchedule()

	// 启动后台任务调度器（邮件队列、订阅过期检查、到期提醒等）
	services.GetScheduler().Start()

	// 设置路由
	r := router.SetupRouter(cfg)

	// 启动服务
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("服务启动: http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func createDefaultAdmin() {
	db := database.GetDB()
	var count int64
	db.Model(&models.User{}).Where("is_admin = ?", true).Count(&count)
	if count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("生成管理员密码失败: %v", err)
		return
	}

	admin := models.User{
		Username:                    "admin",
		Email:                       "admin@example.com",
		Password:                    string(hash),
		IsActive:                    true,
		IsVerified:                  true,
		IsAdmin:                     true,
		Theme:                       "light",
		Language:                    "zh-CN",
		Timezone:                    "Asia/Shanghai",
		EmailNotifications:          true,
		AbnormalLoginAlertEnabled:   true,
		PushNotifications:           true,
		DataSharing:                 true,
		Analytics:                   true,
		SpecialNodeSubscriptionType: "both",
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Printf("创建默认管理员失败: %v", err)
		return
	}

	// Auto-create subscription for admin
	subURL := utils.GenerateRandomString(32)
	db.Create(&models.Subscription{
		UserID:          admin.ID,
		SubscriptionURL: subURL,
		DeviceLimit:     3,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      time.Now(),
	})

	log.Println("已创建默认管理员: admin@example.com（请立即修改密码）")
}

// runResetPassword 从命令行参数解析 --email 和 --password，重置管理员密码后退出
func runResetPassword() {
	var email, password string
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--email" && i+1 < len(os.Args) {
			email = strings.TrimSpace(os.Args[i+1])
			i++
		} else if os.Args[i] == "--password" && i+1 < len(os.Args) {
			password = strings.TrimSpace(os.Args[i+1])
			i++
		}
	}
	if password == "" {
		log.Fatal("请提供 --password 参数")
	}
	if len(password) < 8 {
		log.Fatal("密码至少 8 位")
	}

	// 避免生产环境校验 CORS 导致无法执行
	if os.Getenv("CORS_ORIGINS") == "" {
		_ = os.Setenv("CORS_ORIGINS", "http://localhost")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	db := database.GetDB()
	var user models.User
	q := db.Model(&models.User{}).Where("is_admin = ?", true)
	if email != "" && email != "admin" {
		q = q.Where("email = ?", email)
	}
	err = q.First(&user).Error
	if err != nil {
		// 指定了邮箱但该邮箱没有管理员：创建新管理员（安装脚本常用场景）
		if email != "" && email != "admin" {
			localPart := email
			if idx := strings.Index(email, "@"); idx > 0 {
				localPart = strings.ReplaceAll(email[:idx], ".", "_")
			}
			if len(localPart) > 45 {
				localPart = localPart[:45]
			}
			username := "admin_" + localPart
			user = models.User{
				Username:                    username,
				Email:                       email,
				Password:                    string(hash),
				IsActive:                    true,
				IsVerified:                  true,
				IsAdmin:                     true,
				Theme:                       "light",
				Language:                    "zh-CN",
				Timezone:                    "Asia/Shanghai",
				EmailNotifications:          true,
				AbnormalLoginAlertEnabled:   true,
				PushNotifications:           true,
				DataSharing:                 true,
				Analytics:                   true,
				SpecialNodeSubscriptionType: "both",
			}
			if err := db.Create(&user).Error; err != nil {
				log.Fatalf("创建管理员失败: %v", err)
			}
			subURL := utils.GenerateRandomString(32)
			db.Create(&models.Subscription{
				UserID:          user.ID,
				SubscriptionURL: subURL,
				DeviceLimit:     3,
				IsActive:        true,
				Status:          "active",
				ExpireTime:      time.Now(),
			})
			log.Printf("已创建管理员 %s 并设置密码", user.Email)
			return
		}
		log.Fatalf("未找到管理员账号（可指定 --email 或留空使用第一个管理员）: %v", err)
	}

	if err := db.Model(&user).Update("password", string(hash)).Error; err != nil {
		log.Fatalf("更新密码失败: %v", err)
	}
	log.Printf("管理员 %s 密码已重置成功", user.Email)
}

func ensureUserSubscriptions() {
	db := database.GetDB()
	// Find all users without a subscription
	var users []models.User
	db.Where("id NOT IN (SELECT user_id FROM subscriptions)").Find(&users)
	for _, user := range users {
		subURL := utils.GenerateRandomString(32)
		sub := models.Subscription{
			UserID:          user.ID,
			SubscriptionURL: subURL,
			DeviceLimit:     3,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      time.Now(),
		}
		if err := db.Create(&sub).Error; err == nil {
			log.Printf("为用户 %s (ID:%d) 创建了订阅记录", user.Email, user.ID)
		}
	}
}
