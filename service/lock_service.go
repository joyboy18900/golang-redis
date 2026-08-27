package service

import (
	"context"
	"errors"
	"time"

	"golang-redis/errs"
	"golang-redis/logs"
	"golang-redis/repository"
)

const defaultHoldDuration = 2 * time.Second

type lockService struct {
	repo    repository.LockRepository
	lockTTL time.Duration
}

func NewLockService(repo repository.LockRepository, lockTTL time.Duration) LockService {
	return lockService{repo: repo, lockTTL: lockTTL}
}

func (s lockService) RunCriticalSection(ctx context.Context, req RunCriticalSectionRequest) (*RunCriticalSectionResponse, error) {
	if req.JobID == "" {
		return nil, errs.NewValidationError("job_id is required")
	}

	hold := defaultHoldDuration
	if req.HoldMillis > 0 {
		hold = time.Duration(req.HoldMillis) * time.Millisecond
	}
	if hold > s.lockTTL {
		return nil, errs.NewValidationError("hold_millis must not exceed the lock ttl")
	}

	lock, err := s.repo.Acquire(ctx, req.JobID, s.lockTTL)
	if errors.Is(err, repository.ErrLockNotAcquired) {
		return nil, errs.NewConflictError("job " + req.JobID + " is already running")
	}
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}
	defer func() {
		if unlockErr := lock.Unlock(context.Background()); unlockErr != nil {
			logs.Error(unlockErr)
		}
	}()

	started := time.Now()
	select {
	case <-time.After(hold):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &RunCriticalSectionResponse{JobID: req.JobID, StartedAt: started, FinishedAt: time.Now()}, nil
}
