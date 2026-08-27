package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	goredisdriver "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	goredislib "github.com/redis/go-redis/v9"
)

type lockRepositoryRedis struct {
	rs *redsync.Redsync
}

func NewLockRepositoryRedis(client *goredislib.Client) LockRepository {
	return lockRepositoryRedis{rs: redsync.New(goredisdriver.NewPool(client))}
}

func (r lockRepositoryRedis) Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	mutex := r.rs.NewMutex(lockKey(key), redsync.WithExpiry(ttl))

	if err := mutex.TryLockContext(ctx); err != nil {
		var taken *redsync.ErrTaken
		if errors.Is(err, redsync.ErrFailed) || errors.As(err, &taken) {
			return nil, ErrLockNotAcquired
		}
		return nil, fmt.Errorf("acquire lock %s: %w", key, err)
	}

	return redsyncLock{mutex: mutex}, nil
}

type redsyncLock struct {
	mutex *redsync.Mutex
}

func (l redsyncLock) Unlock(ctx context.Context) error {
	ok, err := l.mutex.UnlockContext(ctx)
	if err != nil {
		return fmt.Errorf("unlock: %w", err)
	}
	if !ok {
		return errors.New("unlock: lock was not held")
	}
	return nil
}

func lockKey(key string) string {
	return "lock:" + key
}
