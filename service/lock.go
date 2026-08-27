package service

import (
	"context"
	"time"
)

type RunCriticalSectionRequest struct {
	JobID      string `json:"job_id"`
	HoldMillis int    `json:"hold_millis"`
}

type RunCriticalSectionResponse struct {
	JobID      string    `json:"job_id"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

//go:generate go tool mockgen -destination=../mock/mock_service/lock.go golang-redis/service LockService
type LockService interface {
	RunCriticalSection(ctx context.Context, req RunCriticalSectionRequest) (*RunCriticalSectionResponse, error)
}
