package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang-redis/logs"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func main() {
	initConfig()

	redisClient := initRedis()
	defer redisClient.Close()

	app := fiber.New()

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
