package service

import (
	"context"
	"time"
)

type RateLimitDecision struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}

//go:generate go tool mockgen -destination=../mock/mock_service/ratelimiter.go golang-redis/service RateLimiterService
type RateLimiterService interface {
	Allow(ctx context.Context, identifier string) (RateLimitDecision, error)
}
