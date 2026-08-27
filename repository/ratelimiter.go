package repository

import (
	"context"
	"time"
)

type RateLimitResult struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}

//go:generate go tool mockgen -destination=../mock/mock_repository/ratelimiter.go golang-redis/repository RateLimiterRepository
type RateLimiterRepository interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitResult, error)
}
