package main_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"golang-redis/handler"
	"golang-redis/repository"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
)

func TestRateLimiter_RejectsOverLimitAndResets(t *testing.T) {
	client := connectTestRedis(t)
	defer client.Close()

	const (
		limit  = 3
		window = 300 * time.Millisecond
	)

	client.Del(context.Background(), "rate_limit:0.0.0.0")

	repo := repository.NewRateLimiterRepositoryRedis(client)
	svc := service.NewRateLimiterService(repo, limit, window)

	app := fiber.New()
	app.Get("/ping", handler.RateLimitMiddleware(svc), handler.Ping)

	request := func() int {
		req := httptest.NewRequest(fiber.MethodGet, "/ping", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		return resp.StatusCode
	}

	for i := range limit {
		if status := request(); status != fiber.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i+1, status, fiber.StatusOK)
		}
	}

	if status := request(); status != fiber.StatusTooManyRequests {
		t.Fatalf("request over limit status = %d, want %d", status, fiber.StatusTooManyRequests)
	}

	time.Sleep(window + 100*time.Millisecond)

	if status := request(); status != fiber.StatusOK {
		t.Fatalf("request after window reset status = %d, want %d", status, fiber.StatusOK)
	}
}
