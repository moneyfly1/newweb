package main

import (
	"fmt"
	"log"
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

	log.Println("已创建默认管理员: admin@example.com / admin123")
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
