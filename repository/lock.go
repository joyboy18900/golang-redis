package repository

import (
	"context"
	"errors"
	"time"
)

var ErrLockNotAcquired = errors.New("repository: lock not acquired")

type Lock interface {
	Unlock(ctx context.Context) error
}

//go:generate go tool mockgen -destination=../mock/mock_repository/lock.go golang-redis/repository LockRepository
type LockRepository interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}
