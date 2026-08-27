package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang-redis/handler"
	"golang-redis/logs"
	"golang-redis/repository"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func main() {
	initConfig()

	redisClient := initRedis()
	defer redisClient.Close()

	lockTTL := time.Duration(viper.GetInt("lock.ttl_seconds")) * time.Second
	rateLimit := viper.GetInt("rate_limit.limit")
	rateWindow := time.Duration(viper.GetInt("rate_limit.window_seconds")) * time.Second

	lockRepo := repository.NewLockRepositoryRedis(redisClient)
	lockSvc := service.NewLockService(lockRepo, lockTTL)
	lockHdlr := handler.NewLockHandler(lockSvc)

	rateLimiterRepo := repository.NewRateLimiterRepositoryRedis(redisClient)
	rateLimiterSvc := service.NewRateLimiterService(rateLimiterRepo, rateLimit, rateWindow)

	app := fiber.New()

	app.Post("/jobs/run", lockHdlr.RunCriticalSection)
	app.Get("/ping", handler.RateLimitMiddleware(rateLimiterSvc), handler.Ping)

	port := viper.GetString("app.port")
	logs.Info("server started on port " + port)
	if err := app.Listen(":" + port); err != nil {
		logs.Error(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("read config: %w", err))
	}
}

func initRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     viper.GetString("redis.addr"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic(fmt.Errorf("ping redis: %w", err))
	}

	return client
}
