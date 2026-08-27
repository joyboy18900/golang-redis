package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

	pubsubRepo := repository.NewPubSubRepositoryRedis(redisClient)
	pubsubSvc := service.NewPubSubService(pubsubRepo)
	pubsubHdlr := handler.NewPubSubHandler(pubsubSvc)

	app := fiber.New()

	app.Post("/jobs/run", lockHdlr.RunCriticalSection)
	app.Get("/ping", handler.RateLimitMiddleware(rateLimiterSvc), handler.Ping)
	app.Post("/messages", pubsubHdlr.Publish)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	channel := viper.GetString("pubsub.channel")
	for i := 1; i <= viper.GetInt("pubsub.consumer_count"); i++ {
		name := "consumer-" + strconv.Itoa(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := pubsubSvc.RunConsumer(ctx, channel, name); err != nil && !errors.Is(err, context.Canceled) {
				logs.Error(err)
			}
		}()
	}

	go func() {
		port := viper.GetString("app.port")
		logs.Info("server started on port " + port)
		if err := app.Listen(":" + port); err != nil {
			logs.Error(err)
		}
	}()

	<-ctx.Done()
	logs.Info("shutting down")
	if err := app.Shutdown(); err != nil {
		logs.Error(err)
	}
	wg.Wait()
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
