package txmonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"deblock/internal/address"
	"deblock/internal/blockchain"
	"deblock/internal/dlock"
	"deblock/internal/pubsub"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=txmonitor_service.go -destination=../../mocks/mock_txmonitor_service.go -package=mocks
type TxMonitorService interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning(ctx context.Context) bool
}

type txMonitorService struct {
	logger           *slog.Logger
	blockchainClient blockchain.Client
	addressWatcher   address.Watcher
	publisher        pubsub.Publisher
	dlock            dlock.DistributedLock

	mu         sync.RWMutex
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
}

func NewTxMonitorService(logger *slog.Logger, blockchainClient blockchain.Client, addressWatcher address.Watcher, publisher pubsub.Publisher, dlock dlock.DistributedLock) TxMonitorService {
	return &txMonitorService{
		logger:           logger,
		blockchainClient: blockchainClient,
		addressWatcher:   addressWatcher,
		publisher:        publisher,
		dlock:            dlock,
		mu:               sync.RWMutex{},
		cancelFunc:       nil,
		wg:               sync.WaitGroup{},
		isRunning:        false,
	}
}

// Start begins monitoring blockchain transactions
func (m *txMonitorService) Start(ctx context.Context) error {
	m.logger.Info("Starting transaction monitor")

	if m.isRunning {
		m.logger.Info("Transaction monitor is already running")
		return nil
	}

	// Create a long-lived context with a timeout
	monitorCtx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	m.mu.Lock()
	m.cancelFunc = cancel
	m.isRunning = true
	m.mu.Unlock()

	// Subscribe to blocks
	blockChan, errChan := m.blockchainClient.SubscribeToBlocks(monitorCtx)
	m.logger.Info("Subscribed to blocks",
		"context_cancelled", monitorCtx.Err() != nil,
		"block_channel_nil", blockChan == nil,
		"error_channel_nil", errChan == nil,
	)

	go func() {
		defer func() {
			m.logger.Info("Block subscription goroutine ending")
			// Resources are owned by the caller (main). Do not close here to allow graceful drain.
		}()

		for {
			select {
			case <-monitorCtx.Done():
				m.logger.Info("Monitor context cancelled, stopping block subscription")
				return
			case err := <-errChan:
				m.logger.Error("Block subscription error",
					"error", err,
					"error_type", fmt.Sprintf("%T", err),
				)
				return
			case block, ok := <-blockChan:
				if !ok {
					m.logger.Warn("Block channel closed unexpectedly")
					return
				}
				// Debug: comprehensive block info on arrival
				m.logger.Debug("New block received",
					"number", block.Number,
					"hash", block.Hash,
					"tx_count", len(block.Transactions),
					"timestamp", block.Timestamp,
				)
				// Process block synchronously but track completion
				m.wg.Add(1)
				if err := m.processBlock(monitorCtx, block); err != nil {
					m.logger.Error("Failed to process block",
						"blockNumber", block.Number,
						"error", err,
					)
				}
				m.wg.Done()
			}
		}
	}()

	return nil
}

// processBlock processes transactions in a block
func (m *txMonitorService) processBlock(ctx context.Context, block blockchain.Block) error {
	// Process each transaction in the block
	m.logger.Debug("Processing block transactions", "number", block.Number, "tx_count", len(block.Transactions))

	// Acquire lock
	lockKey := fmt.Sprintf("block_lock_%s", block.Hash)
	if err := m.dlock.Lock(ctx, lockKey); err != nil {
		m.logger.Warn("Other instance is processing block", "error", err, "blockNumber", block.Number)
		return nil
	}
	defer m.dlock.Unlock(ctx, lockKey)

	relevantTxCount := 0
	for _, tx := range block.Transactions {
		// Check if transaction involves watched addresses
		if !m.isTransactionRelevant(ctx, tx) {
			continue
		}

		relevantTxCount++

		// Create Kafka event
		event := &pubsub.Transaction{
			Source:      tx.Source,
			Destination: tx.Destination,
			Amount:      tx.Amount,
			Fees:        tx.Fees,
			Hash:        tx.Hash,
		}

		// Publish event
		msg, err := json.Marshal(event)
		if err != nil {
			m.logger.Error("Failed to marshal transaction event", "error", err)
			continue
		}
		if err := m.publisher.Publish(ctx, pubsub.TopicTransaction, msg); err != nil {
			m.logger.Error("Failed to publish transaction event",
				"error", err,
				"txHash", tx.Hash,
			)
		}

		// Debug: log each relevant transaction
		m.logger.Debug("Relevant tx",
			"hash", tx.Hash,
			"from", tx.Source,
			"to", tx.Destination,
			"amount_wei", tx.Amount.String(),
			"fees_wei", tx.Fees.String(),
		)
	}

	return nil
}

// isTransactionRelevant checks if the transaction involves watched addresses
func (m *txMonitorService) isTransactionRelevant(ctx context.Context, tx blockchain.Transaction) bool {
	return m.addressWatcher.IsWatched(ctx, tx.Source) || m.addressWatcher.IsWatched(ctx, tx.Destination)
}

// Stop halts the transaction monitoring
func (m *txMonitorService) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.isRunning = false

	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	// Wait for in-flight block processing to drain
	m.wg.Wait()

	return nil
}

func (m *txMonitorService) IsRunning(_ context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}
