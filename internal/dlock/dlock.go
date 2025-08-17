// internal/dlock/dlock.go
package dlock

import (
	"context"
	"time"
)

// DistributedLock represents an interface for distributed locking
//
//go:generate go run go.uber.org/mock/mockgen@latest -source=dlock.go -destination=../../mocks/mock_dlock.go -package=mocks
type DistributedLock interface {
	// Lock attempts to acquire the lock
	Lock(ctx context.Context, key string) error

	// Unlock releases the lock
	Unlock(ctx context.Context, key string) (bool, error)
}

// LockOption allows configuring lock behavior
type LockOption func(*lockConfig)

type lockConfig struct {
	expiry time.Duration
}

// WithExpiry sets the lock expiration time
func WithExpiry(duration time.Duration) LockOption {
	return func(cfg *lockConfig) {
		cfg.expiry = duration
	}
}
