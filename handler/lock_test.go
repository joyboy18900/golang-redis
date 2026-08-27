package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"golang-redis/errs"
	"golang-redis/handler"
	"golang-redis/mock/mock_service"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func newLockTestApp(lockSvc service.LockService) *fiber.App {
	app := fiber.New()
	lockHdlr := handler.NewLockHandler(lockSvc)
	app.Post("/jobs/run", lockHdlr.RunCriticalSection)
	return app
}

func TestLockHandler_RunCriticalSection(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(lockSvc *mock_service.MockLockService)
		wantStatus int
	}{
		{
			name: "success",
			setup: func(lockSvc *mock_service.MockLockService) {
				lockSvc.EXPECT().RunCriticalSection(gomock.Any(), service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 1}).
					Return(&service.RunCriticalSectionResponse{JobID: "job-1"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
		{
			name: "already running maps to 409",
			setup: func(lockSvc *mock_service.MockLockService) {
				lockSvc.EXPECT().RunCriticalSection(gomock.Any(), service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 1}).
					Return(nil, errs.NewConflictError("job job-1 is already running"))
			},
			wantStatus: fiber.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			lockSvc := mock_service.NewMockLockService(ctrl)
			tc.setup(lockSvc)

			app := newLockTestApp(lockSvc)

			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 1}); err != nil {
				t.Fatalf("encode request body: %v", err)
			}

			req := httptest.NewRequest(fiber.MethodPost, "/jobs/run", &buf)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestLockHandler_RunCriticalSection_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	lockSvc := mock_service.NewMockLockService(ctrl)
	app := newLockTestApp(lockSvc)

	req := httptest.NewRequest(fiber.MethodPost, "/jobs/run", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusUnprocessableEntity)
	}
}
