package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"cboard/v2/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient 全局 Redis 客户端
var RedisClient *redis.Client

// InitRedis 初始化 Redis 连接
func InitRedis(cfg *config.Config) error {
	if cfg.RedisAddr == "" {
		log.Println("REDIS_ADDR 未配置，将不使用 Redis 缓存")
		return nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: 100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis 连接失败: %w", err)
	}

	log.Printf("Redis 连接成功 (%s)", cfg.RedisAddr)
	RedisClient = client
	return nil
}

// GetRedis 返回全局 Redis 客户端
func GetRedis() *redis.Client {
	return RedisClient
}

// CloseRedis 关闭 Redis 连接
func CloseRedis() {
	if RedisClient != nil {
		_ = RedisClient.Close()
	}
}
