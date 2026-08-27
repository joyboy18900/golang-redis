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

const testLockTTL = time.Second

type stubLock struct {
	unlockErr error
	unlocked  bool
}

func (l *stubLock) Unlock(ctx context.Context) error {
	l.unlocked = true
	return l.unlockErr
}

func TestLockService_RunCriticalSection(t *testing.T) {
	tests := []struct {
		name      string
		req       service.RunCriticalSectionRequest
		setup     func(lockRepo *mock_repository.MockLockRepository)
		wantErr   error
		wantJobID string
	}{
		{
			name: "success",
			req:  service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 1},
			setup: func(lockRepo *mock_repository.MockLockRepository) {
				lockRepo.EXPECT().Acquire(gomock.Any(), "job-1", testLockTTL).
					Return(&stubLock{}, nil)
			},
			wantJobID: "job-1",
		},
		{
			name:    "missing job id",
			req:     service.RunCriticalSectionRequest{},
			setup:   func(lockRepo *mock_repository.MockLockRepository) {},
			wantErr: errs.AppError{Code: 422},
		},
		{
			name:    "hold exceeds ttl",
			req:     service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 5000},
			setup:   func(lockRepo *mock_repository.MockLockRepository) {},
			wantErr: errs.AppError{Code: 422},
		},
		{
			name: "lock already held",
			req:  service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 1},
			setup: func(lockRepo *mock_repository.MockLockRepository) {
				lockRepo.EXPECT().Acquire(gomock.Any(), "job-1", testLockTTL).
					Return(nil, repository.ErrLockNotAcquired)
			},
			wantErr: errs.AppError{Code: 409},
		},
		{
			name: "repository error",
			req:  service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 1},
			setup: func(lockRepo *mock_repository.MockLockRepository) {
				lockRepo.EXPECT().Acquire(gomock.Any(), "job-1", testLockTTL).
					Return(nil, errors.New("redis down"))
			},
			wantErr: errs.AppError{Code: 500},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			lockRepo := mock_repository.NewMockLockRepository(ctrl)
			tc.setup(lockRepo)

			svc := service.NewLockService(lockRepo, testLockTTL)
			got, err := svc.RunCriticalSection(context.Background(), tc.req)

			if tc.wantErr != nil {
				var appErr errs.AppError
				if !errors.As(err, &appErr) || appErr.Code != tc.wantErr.(errs.AppError).Code {
					t.Fatalf("RunCriticalSection() error = %v, want AppError code %d", err, tc.wantErr.(errs.AppError).Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("RunCriticalSection() unexpected error = %v", err)
			}
			if got.JobID != tc.wantJobID {
				t.Fatalf("RunCriticalSection() job id = %q, want %q", got.JobID, tc.wantJobID)
			}
		})
	}
}

func TestLockService_RunCriticalSection_UnlocksOnSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	lockRepo := mock_repository.NewMockLockRepository(ctrl)
	lock := &stubLock{}
	lockRepo.EXPECT().Acquire(gomock.Any(), "job-1", testLockTTL).Return(lock, nil)

	svc := service.NewLockService(lockRepo, testLockTTL)
	if _, err := svc.RunCriticalSection(context.Background(), service.RunCriticalSectionRequest{JobID: "job-1", HoldMillis: 1}); err != nil {
		t.Fatalf("RunCriticalSection() unexpected error = %v", err)
	}

	if !lock.unlocked {
		t.Fatal("RunCriticalSection() did not unlock after completing")
	}
}
