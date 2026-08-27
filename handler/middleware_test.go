package handler_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"golang-redis/errs"
	"golang-redis/handler"
	"golang-redis/mock/mock_service"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func newRateLimitTestApp(rateLimiterSvc service.RateLimiterService) *fiber.App {
	app := fiber.New()
	app.Get("/ping", handler.RateLimitMiddleware(rateLimiterSvc), handler.Ping)
	return app
}

func TestRateLimitMiddleware(t *testing.T) {
	resetAt := time.Now().Add(10 * time.Second)

	tests := []struct {
		name       string
		decision   service.RateLimitDecision
		serviceErr error
		wantStatus int
	}{
		{
			name:       "allowed",
			decision:   service.RateLimitDecision{Allowed: true, Remaining: 3, ResetAt: resetAt},
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "rejected",
			decision:   service.RateLimitDecision{Allowed: false, Remaining: 0, ResetAt: resetAt},
			wantStatus: fiber.StatusTooManyRequests,
		},
		{
			name:       "service error",
			serviceErr: errs.NewUnexpectedError(),
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			rateLimiterSvc := mock_service.NewMockRateLimiterService(ctrl)

			var callErr error
			if tc.serviceErr != nil {
				callErr = tc.serviceErr
			}
			rateLimiterSvc.EXPECT().Allow(gomock.Any(), gomock.Any()).Return(tc.decision, callErr)

			app := newRateLimitTestApp(rateLimiterSvc)

			req := httptest.NewRequest(fiber.MethodGet, "/ping", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			if tc.serviceErr == nil {
				if got := resp.Header.Get("X-RateLimit-Remaining"); got == "" {
					t.Fatal("X-RateLimit-Remaining header not set")
				}
				if got := resp.Header.Get("X-RateLimit-Reset"); got == "" {
					t.Fatal("X-RateLimit-Reset header not set")
				}
			}
		})
	}
}
