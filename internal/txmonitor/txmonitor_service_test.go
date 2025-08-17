package txmonitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"deblock/internal/blockchain"
	"deblock/internal/pubsub"
	"deblock/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewTxMonitorService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockBlockchainClient := mocks.NewMockClient(ctrl)
	mockAddressWatcher := mocks.NewMockWatcher(ctrl)
	mockPublisher := mocks.NewMockPublisher(ctrl)
	mockDlock := mocks.NewMockDistributedLock(ctrl)

	service := NewTxMonitorService(logger, mockBlockchainClient, mockAddressWatcher, mockPublisher, mockDlock)

	assert.NotNil(t, service, "NewTxMonitorService should return a non-nil service")
}

func TestTxMonitorService_IsRunning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockBlockchainClient := mocks.NewMockClient(ctrl)
	mockAddressWatcher := mocks.NewMockWatcher(ctrl)
	mockPublisher := mocks.NewMockPublisher(ctrl)
	mockDlock := mocks.NewMockDistributedLock(ctrl)

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Expect blockchain client to subscribe to blocks when Start is called
	blockChan := make(chan blockchain.Block)
	errChan := make(chan error)
	mockBlockchainClient.EXPECT().SubscribeToBlocks(gomock.Any()).Return(blockChan, errChan).AnyTimes()

	service := NewTxMonitorService(logger, mockBlockchainClient, mockAddressWatcher, mockPublisher, mockDlock)

	// Initially not running
	assert.False(t, service.IsRunning(context.Background()), "Service should not be running initially")

	// Start the service
	err := service.Start(ctx)
	assert.NoError(t, err, "Start should not return an error")
	assert.True(t, service.IsRunning(context.Background()), "Service should be running after Start")

	// Stop the service
	err = service.Stop(ctx)
	assert.NoError(t, err, "Stop should not return an error")
	assert.False(t, service.IsRunning(context.Background()), "Service should not be running after Stop")
}

func TestTxMonitorService_IsTransactionRelevant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockBlockchainClient := mocks.NewMockClient(ctrl)
	mockAddressWatcher := mocks.NewMockWatcher(ctrl)
	mockPublisher := mocks.NewMockPublisher(ctrl)
	mockDlock := mocks.NewMockDistributedLock(ctrl)

	service := NewTxMonitorService(logger, mockBlockchainClient, mockAddressWatcher, mockPublisher, mockDlock).(*txMonitorService)

	ctx := context.Background()
	sourceAddr := "0x1234"
	destAddr := "0x5678"

	// Test transaction with source address watched
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), sourceAddr).Return(true)
	tx1 := blockchain.Transaction{
		Source:      sourceAddr,
		Destination: "0x9999",
		Amount:      big.NewInt(100),
		Fees:        big.NewInt(10),
		Hash:        "tx1hash",
	}
	assert.True(t, service.isTransactionRelevant(ctx, tx1), "Transaction with watched source address should be relevant")

	// Test transaction with destination address watched
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), sourceAddr).Return(false)
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), destAddr).Return(true)
	tx2 := blockchain.Transaction{
		Source:      sourceAddr,
		Destination: destAddr,
		Amount:      big.NewInt(200),
		Fees:        big.NewInt(20),
		Hash:        "tx2hash",
	}
	assert.True(t, service.isTransactionRelevant(ctx, tx2), "Transaction with watched destination address should be relevant")

	// Test transaction with no watched addresses
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), sourceAddr).Return(false)
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), destAddr).Return(false)
	tx3 := blockchain.Transaction{
		Source:      sourceAddr,
		Destination: destAddr,
		Amount:      big.NewInt(300),
		Fees:        big.NewInt(30),
		Hash:        "tx3hash",
	}
	assert.False(t, service.isTransactionRelevant(ctx, tx3), "Transaction with no watched addresses should not be relevant")
}

func TestTxMonitorService_ProcessBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockBlockchainClient := mocks.NewMockClient(ctrl)
	mockAddressWatcher := mocks.NewMockWatcher(ctrl)
	mockPublisher := mocks.NewMockPublisher(ctrl)
	mockDlock := mocks.NewMockDistributedLock(ctrl)

	service := NewTxMonitorService(logger, mockBlockchainClient, mockAddressWatcher, mockPublisher, mockDlock).(*txMonitorService)

	ctx := context.Background()
	blockHash := "block123"
	sourceAddr := "0x1234"
	destAddr := "0x5678"

	// Prepare block with a relevant transaction
	block := blockchain.Block{
		Number: big.NewInt(100),
		Hash:   blockHash,
		Transactions: []blockchain.Transaction{
			{
				Source:      sourceAddr,
				Destination: destAddr,
				Amount:      big.NewInt(100),
				Fees:        big.NewInt(10),
				Hash:        "tx1hash",
			},
		},
	}

	// Expect distributed lock to be acquired and released
	lockKey := fmt.Sprintf("block_lock_%s", blockHash)
	mockDlock.EXPECT().Lock(gomock.Any(), lockKey).Return(nil)
	mockDlock.EXPECT().Unlock(gomock.Any(), lockKey).Return(true, nil)

	// Expect address watcher to check transaction relevance
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), sourceAddr).Return(false)
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), destAddr).Return(true)

	// Expect publisher to publish the transaction
	expectedEvent := &pubsub.Transaction{
		Source:      sourceAddr,
		Destination: destAddr,
		Amount:      big.NewInt(100),
		Fees:        big.NewInt(10),
		Hash:        "tx1hash",
	}
	expectedMsg, _ := json.Marshal(expectedEvent)
	mockPublisher.EXPECT().Publish(gomock.Any(), pubsub.TopicTransaction, expectedMsg).Return(nil)

	// Process the block
	err := service.processBlock(ctx, block)
	assert.NoError(t, err, "processBlock should not return an error")
}

func TestTxMonitorService_ProcessBlock_PublishError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockBlockchainClient := mocks.NewMockClient(ctrl)
	mockAddressWatcher := mocks.NewMockWatcher(ctrl)
	mockPublisher := mocks.NewMockPublisher(ctrl)
	mockDlock := mocks.NewMockDistributedLock(ctrl)

	service := NewTxMonitorService(logger, mockBlockchainClient, mockAddressWatcher, mockPublisher, mockDlock).(*txMonitorService)

	ctx := context.Background()
	blockHash := "block123"
	sourceAddr := "0x1234"
	destAddr := "0x5678"

	// Prepare block with a relevant transaction
	block := blockchain.Block{
		Number: big.NewInt(100),
		Hash:   blockHash,
		Transactions: []blockchain.Transaction{
			{
				Source:      sourceAddr,
				Destination: destAddr,
				Amount:      big.NewInt(100),
				Fees:        big.NewInt(10),
				Hash:        "tx1hash",
			},
		},
	}

	// Expect distributed lock to be acquired and released
	lockKey := fmt.Sprintf("block_lock_%s", blockHash)
	mockDlock.EXPECT().Lock(gomock.Any(), lockKey).Return(nil)
	mockDlock.EXPECT().Unlock(gomock.Any(), lockKey).Return(true, nil)

	// Expect address watcher to check transaction relevance
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), sourceAddr).Return(false)
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), destAddr).Return(true)

	// Expect publisher to fail publishing the transaction
	expectedEvent := &pubsub.Transaction{
		Source:      sourceAddr,
		Destination: destAddr,
		Amount:      big.NewInt(100),
		Fees:        big.NewInt(10),
		Hash:        "tx1hash",
	}
	expectedMsg, _ := json.Marshal(expectedEvent)
	mockPublisher.EXPECT().Publish(gomock.Any(), pubsub.TopicTransaction, expectedMsg).Return(errors.New("publish error"))

	// Process the block
	err := service.processBlock(ctx, block)
	assert.NoError(t, err, "processBlock should not return an error even if publish fails")
}

func TestTxMonitorService_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockBlockchainClient := mocks.NewMockClient(ctrl)
	mockAddressWatcher := mocks.NewMockWatcher(ctrl)
	mockPublisher := mocks.NewMockPublisher(ctrl)
	mockDlock := mocks.NewMockDistributedLock(ctrl)

	service := NewTxMonitorService(logger, mockBlockchainClient, mockAddressWatcher, mockPublisher, mockDlock).(*txMonitorService)

	// Use a single context for the entire test
	ctx := context.Background()

	// Create channels for block subscription
	blockChan := make(chan blockchain.Block, 1)
	errChan := make(chan error, 1)

	// Expect blockchain client to subscribe to blocks
	mockBlockchainClient.EXPECT().SubscribeToBlocks(gomock.Any()).Return(blockChan, errChan)

	// Start the service
	err := service.Start(ctx)
	assert.NoError(t, err, "Start should not return an error")

	// Simulate a block being received
	sourceAddr := "0x1234"
	destAddr := "0x5678"
	block := blockchain.Block{
		Number: big.NewInt(100),
		Hash:   "block123",
		Transactions: []blockchain.Transaction{
			{
				Source:      sourceAddr,
				Destination: destAddr,
				Amount:      big.NewInt(100),
				Fees:        big.NewInt(10),
				Hash:        "tx1hash",
			},
		},
	}

	// Expect address watcher to check transaction relevance
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), sourceAddr).Return(false)
	mockAddressWatcher.EXPECT().IsWatched(gomock.Any(), destAddr).Return(true)

	// Expect distributed lock to be acquired and released
	lockKey := fmt.Sprintf("block_lock_%s", block.Hash)
	mockDlock.EXPECT().Lock(gomock.Any(), lockKey).Return(nil)
	mockDlock.EXPECT().Unlock(gomock.Any(), lockKey).Return(true, nil)

	// Expect publisher to publish the transaction
	expectedEvent := &pubsub.Transaction{
		Source:      sourceAddr,
		Destination: destAddr,
		Amount:      big.NewInt(100),
		Fees:        big.NewInt(10),
		Hash:        "tx1hash",
	}
	expectedMsg, _ := json.Marshal(expectedEvent)
	mockPublisher.EXPECT().Publish(gomock.Any(), pubsub.TopicTransaction, expectedMsg).Return(nil)

	// Send a block through the channel
	blockChan <- block

	// Wait a short time to allow processing
	time.Sleep(100 * time.Millisecond)

	// Stop the service
	err = service.Stop(ctx)
	assert.NoError(t, err, "Stop should not return an error")
}
