// internal/dlock/redsync_dlock.go
package dlock

import (
	"context"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	goredislib "github.com/redis/go-redis/v9"
)

// redsyncLock implements DistributedLock
type redsyncLock struct {
	rs    *redsync.Redsync
	mutex *redsync.Mutex
}

// NewRedsyncLock creates a new RedsyncLock
func NewRedsyncLock(addr string) *redsyncLock {
	// Create Redis client
	redisClient := goredislib.NewClient(&goredislib.Options{
		Addr: addr,
	})
	pool := goredis.NewPool(redisClient)

	return &redsyncLock{
		rs: redsync.New(pool),
	}
}

// Lock attempts to acquire a distributed lock
func (l *redsyncLock) Lock(ctx context.Context, key string) error {
	mutex := l.rs.NewMutex(key)
	l.mutex = mutex
	return mutex.LockContext(ctx)
}

// Unlock releases the distributed lock
func (l *redsyncLock) Unlock(ctx context.Context, key string) (bool, error) {
	return l.mutex.UnlockContext(ctx)
}
