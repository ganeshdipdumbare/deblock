package dlock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedsyncLockTestSuite encapsulates the test suite for distributed locking
type RedsyncLockTestSuite struct {
	suite.Suite
	ctx            context.Context
	redisContainer testcontainers.Container
	redisAddr      string
}

// SetupSuite starts the Redis container before all tests
func (s *RedsyncLockTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Define Redis container request
	req := testcontainers.ContainerRequest{
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	var err error
	s.redisContainer, err = testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err, "Failed to start Redis container")

	// Get container host and port
	host, err := s.redisContainer.Host(s.ctx)
	s.Require().NoError(err, "Failed to get container host")

	port, err := s.redisContainer.MappedPort(s.ctx, "6379")
	s.Require().NoError(err, "Failed to get mapped port")

	// Construct Redis address
	s.redisAddr = host + ":" + port.Port()
}

// TearDownSuite stops the Redis container after all tests
func (s *RedsyncLockTestSuite) TearDownSuite() {
	if s.redisContainer != nil {
		s.Require().NoError(s.redisContainer.Terminate(s.ctx), "Failed to terminate Redis container")
	}
}

// TestLockAndUnlock tests basic lock and unlock functionality for a single block
func (s *RedsyncLockTestSuite) TestLockAndUnlock() {
	lock := NewRedsyncLock(s.redisAddr)
	ctx := context.Background()
	lockKey := "block_123"

	// First lock attempt should succeed
	err := lock.Lock(ctx, lockKey)
	s.Require().NoError(err, "First lock attempt should succeed")

	// Unlock the lock
	unlocked, err := lock.Unlock(ctx, lockKey)
	s.Require().NoError(err, "Unlock should not return an error")
	s.Require().NotEqual(false, unlocked, "Unlock should return true or false")
}

// TestConcurrentLockAttempt simulates attempting to lock a block that is already locked
func (s *RedsyncLockTestSuite) TestConcurrentLockAttempt() {
	lock1 := NewRedsyncLock(s.redisAddr)
	lock2 := NewRedsyncLock(s.redisAddr)
	ctx := context.Background()
	lockKey := "block_456"

	// First lock should succeed
	err := lock1.Lock(ctx, lockKey)
	s.Require().NoError(err, "First lock attempt should succeed")

	// Second lock attempt should fail or timeout
	secondLockCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = lock2.Lock(secondLockCtx, lockKey)
	s.Require().Error(err, "Second lock attempt should fail")

	// Unlock the first lock
	unlocked, err := lock1.Unlock(ctx, lockKey)
	s.Require().NoError(err, "First unlock should not return an error")
	s.Require().NotEqual(false, unlocked, "First unlock should return true or false")
}

// Run the test suite
func TestRedsyncLockSuite(t *testing.T) {
	suite.Run(t, new(RedsyncLockTestSuite))
}
