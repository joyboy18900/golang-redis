package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang-redis/errs"
	"golang-redis/mock/mock_repository"
	"golang-redis/repository"
	"golang-redis/service"

	"go.uber.org/mock/gomock"
)

const (
	testRateLimit  = 5
	testRateWindow = 10 * time.Second
)

func TestRateLimiterService_Allow(t *testing.T) {
	resetAt := time.Now().Add(testRateWindow)

	tests := []struct {
		name      string
		identity  string
		setup     func(repo *mock_repository.MockRateLimiterRepository)
		wantErr   error
		wantAllow bool
		wantRem   int
	}{
		{
			name:     "allowed",
			identity: "1.2.3.4",
			setup: func(repo *mock_repository.MockRateLimiterRepository) {
				repo.EXPECT().Allow(gomock.Any(), "1.2.3.4", testRateLimit, testRateWindow).
					Return(repository.RateLimitResult{Allowed: true, Remaining: 2, ResetAt: resetAt}, nil)
			},
			wantAllow: true,
			wantRem:   2,
		},
		{
			name:     "rejected over limit",
			identity: "1.2.3.4",
			setup: func(repo *mock_repository.MockRateLimiterRepository) {
				repo.EXPECT().Allow(gomock.Any(), "1.2.3.4", testRateLimit, testRateWindow).
					Return(repository.RateLimitResult{Allowed: false, Remaining: 0, ResetAt: resetAt}, nil)
			},
			wantAllow: false,
			wantRem:   0,
		},
		{
			name:     "missing identifier",
			identity: "",
			setup:    func(repo *mock_repository.MockRateLimiterRepository) {},
			wantErr:  errs.AppError{Code: 422},
		},
		{
			name:     "repository error",
			identity: "1.2.3.4",
			setup: func(repo *mock_repository.MockRateLimiterRepository) {
				repo.EXPECT().Allow(gomock.Any(), "1.2.3.4", testRateLimit, testRateWindow).
					Return(repository.RateLimitResult{}, errors.New("redis down"))
			},
			wantErr: errs.AppError{Code: 500},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock_repository.NewMockRateLimiterRepository(ctrl)
			tc.setup(repo)

			svc := service.NewRateLimiterService(repo, testRateLimit, testRateWindow)
			got, err := svc.Allow(context.Background(), tc.identity)

			if tc.wantErr != nil {
				var appErr errs.AppError
				if !errors.As(err, &appErr) || appErr.Code != tc.wantErr.(errs.AppError).Code {
					t.Fatalf("Allow() error = %v, want AppError code %d", err, tc.wantErr.(errs.AppError).Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("Allow() unexpected error = %v", err)
			}
			if got.Allowed != tc.wantAllow {
				t.Fatalf("Allow() allowed = %v, want %v", got.Allowed, tc.wantAllow)
			}
			if got.Remaining != tc.wantRem {
				t.Fatalf("Allow() remaining = %d, want %d", got.Remaining, tc.wantRem)
			}
		})
	}
}
