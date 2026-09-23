package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"cboard/v2/internal/config"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"
)

// runSetDomain 从命令行写入域名相关设置（供安装脚本使用）。
//
// 为什么需要它：域名是「配置」不是「代码」，日常换域名在后台点一下就行；
// 但**重新搭建**时（新服务器/重装）安装脚本只能建 nginx 与证书，
// 数据库里的 site_url / 订阅域名 / 备用域名还需要人工进后台补，
// 很容易漏配。这里给安装脚本一个可编程入口，一次写全：
//
//	./cboard set-domain --site https://new.moneyfly.top \
//	    --sub https://sub.dollarsfly.top \
//	    --mirrors https://dollarsfly.top,https://new.moneyfly.top \
//	    --backup https://dollarsfly.top
//
// 只写这四个设置项；支付回调相关设置（pay_*_notify_url / payment_public_base_url）
// 一律不碰，避免换域名把支付回调搞挂。
func runSetDomain() {
	var site, sub, backup, mirrors string
	for i := 2; i < len(os.Args); i++ {
		next := func() string {
			if i+1 < len(os.Args) {
				i++
				return strings.TrimSpace(os.Args[i])
			}
			return ""
		}
		switch os.Args[i] {
		case "--site":
			site = next()
		case "--sub":
			sub = next()
		case "--backup":
			backup = next()
		case "--mirrors":
			mirrors = next()
		}
	}
	if site == "" && sub == "" && backup == "" && mirrors == "" {
		log.Fatal("用法: cboard set-domain --site https://a.com --sub https://sub.a.com --mirrors https://b.com --backup https://b.com")
	}

	if os.Getenv("CORS_ORIGINS") == "" {
		_ = os.Setenv("CORS_ORIGINS", "http://localhost")
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	db := database.GetDB()

	save := func(key, value string) {
		if value == "" {
			return
		}
		r := db.Model(&models.SystemConfig{}).Where("`key` = ?", key).Updates(map[string]interface{}{"value": value})
		if r.Error == nil && r.RowsAffected == 0 {
			db.Create(&models.SystemConfig{Key: key, Value: value, Category: ""})
		}
		fmt.Printf("  已设置 %-22s = %s\n", key, value)
	}

	save("site_url", services.NormalizeDomainInput(site))
	save("subscription_domain", services.NormalizeDomainInput(sub))
	save("backup_site_url", services.NormalizeDomainInput(backup))
	if mirrors != "" {
		list := services.ParseMirrorDomains(mirrors)
		save("subscription_mirrors", services.MirrorsToSetting(list))
	}

	utils.InvalidateSettingsCache()
	current := services.GetSubscriptionDomainSetting()
	fmt.Println("当前域名配置：")
	fmt.Printf("  站点域名     = %s\n", current.SiteURL)
	fmt.Printf("  订阅专用域名 = %s\n", current.SubscriptionDomain)
	fmt.Printf("  备用订阅域名 = %s\n", strings.Join(current.Mirrors, ", "))
	fmt.Printf("  备用站点域名 = %s\n", current.BackupSiteURL)
	fmt.Printf("  实际生效基址 = %s\n", services.SubscriptionBaseURL())
	fmt.Println("提示：支付回调设置未改动；如需体检请在后台「系统设置 → 域名设置 → 一键应用并体检」。")
}
