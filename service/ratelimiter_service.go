package service

import (
	"context"
	"time"

	"golang-redis/errs"
	"golang-redis/logs"
	"golang-redis/repository"
)

type rateLimiterService struct {
	repo   repository.RateLimiterRepository
	limit  int
	window time.Duration
}

func NewRateLimiterService(repo repository.RateLimiterRepository, limit int, window time.Duration) RateLimiterService {
	return rateLimiterService{repo: repo, limit: limit, window: window}
}

func (s rateLimiterService) Allow(ctx context.Context, identifier string) (RateLimitDecision, error) {
	if identifier == "" {
		return RateLimitDecision{}, errs.NewValidationError("identifier is required")
	}

	result, err := s.repo.Allow(ctx, identifier, s.limit, s.window)
	if err != nil {
		logs.Error(err)
		return RateLimitDecision{}, errs.NewUnexpectedError()
	}

	return RateLimitDecision{Allowed: result.Allowed, Remaining: result.Remaining, ResetAt: result.ResetAt}, nil
}
