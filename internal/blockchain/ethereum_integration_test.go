//go:build integration
// +build integration

package blockchain

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Integration tests require actual Ethereum RPC and WebSocket URLs
// These can be set via environment variables
const (
	envRPCURL = "TEST_ETHEREUM_RPC_URL"
	envWSURL  = "TEST_ETHEREUM_WS_URL"
)

// EthereumClientTestSuite encapsulates the test suite for Ethereum client
type EthereumClientTestSuite struct {
	suite.Suite
	logger     *slog.Logger
	client     *EthereumClient
	rpcURL     string
	wsURL      string
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// SetupSuite prepares the test suite
func (s *EthereumClientTestSuite) SetupSuite() {
	// Skip if URLs are not set
	s.rpcURL = os.Getenv(envRPCURL)
	s.wsURL = os.Getenv(envWSURL)

	if s.rpcURL == "" || s.wsURL == "" {
		s.T().Skipf("Skipping test: %s and %s must be set. Current values - RPC: %q, WS: %q",
			envRPCURL, envWSURL, s.rpcURL, s.wsURL)
	}

	// Setup logger
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create Ethereum client
	var err error
	s.client, err = NewEthereumClient(s.logger, s.rpcURL, s.wsURL)
	s.Require().NoError(err)

	// Setup context
	s.ctx, s.cancelFunc = context.WithTimeout(context.Background(), 30*time.Second)
}

// TearDownSuite cleans up resources
func (s *EthereumClientTestSuite) TearDownSuite() {
	if s.client != nil {
		s.Require().NoError(s.client.Close(context.Background()))
	}
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}

// TestClientCreation tests Ethereum client creation
func (s *EthereumClientTestSuite) TestClientCreation() {
	s.Run("Successful Client Creation", func() {
		s.Require().NotNil(s.client)
		s.Require().NotNil(s.client.client)
		s.Require().NotNil(s.client.rpc)
	})

	s.Run("Invalid URLs", func() {
		invalidClient, err := NewEthereumClient(s.logger, "invalid-url", "invalid-url")
		s.Assert().Error(err)
		s.Assert().Nil(invalidClient)
	})
}

// TestSubscribeToBlocks tests block subscription functionality
func (s *EthereumClientTestSuite) TestSubscribeToBlocks() {
	blockChan, errChan := s.client.SubscribeToBlocks(s.ctx)

	// Collect blocks and errors
	var blocks []Block
	var errors []error

	for {
		select {
		case <-s.ctx.Done():
			// Timeout reached
			s.T().Logf("Collected %d blocks", len(blocks))
			s.Assert().NotEmpty(blocks, "Should have received at least one block")
			return
		case block := <-blockChan:
			blocks = append(blocks, block)
			s.T().Logf("Block received: #%s, Hash: %s, Transactions: %d",
				block.Number.String(), block.Hash, len(block.Transactions))

			// Stop after collecting a few blocks
			if len(blocks) >= 5 {
				return
			}
		case err := <-errChan:
			errors = append(errors, err)
			s.T().Logf("Error received: %v", err)
		}
	}
}

// TestGetBlockByNumber tests fetching blocks by number
func (s *EthereumClientTestSuite) TestGetBlockByNumber() {
	s.Run("Fetch Existing Block", func() {
		blockNumber := big.NewInt(4000000) // Use a known block number

		block, err := s.client.GetBlockByNumber(s.ctx, blockNumber)
		s.Require().NoError(err)
		s.Require().NotNil(block)

		s.Assert().Equal(blockNumber.String(), block.Number.String(), "Block number should match")
		s.Assert().NotEmpty(block.Hash, "Block hash should not be empty")
		s.Assert().NotZero(block.Timestamp, "Block timestamp should be non-zero")
	})

	s.Run("Fetch Non-Existent Block", func() {
		blockNumber := big.NewInt(999999999) // Extremely high block number

		block, err := s.client.GetBlockByNumber(s.ctx, blockNumber)
		s.Assert().Error(err)
		s.Assert().Nil(block)
	})
}

// TestGetTransactionReceipt tests fetching transaction receipts
func (s *EthereumClientTestSuite) TestGetTransactionReceipt() {
	s.Run("Fetch Existing Transaction Receipt", func() {
		// Get the latest block to find a recent transaction
		latestBlock, err := s.client.GetBlockByNumber(s.ctx, nil)
		s.Require().NoError(err)
		s.Require().NotNil(latestBlock)
		s.Require().NotEmpty(latestBlock.Transactions, "Latest block should have transactions")

		// Use the first transaction hash from the latest block
		txHash := latestBlock.Transactions[0].Hash

		s.T().Logf("Testing transaction hash: %s", txHash)

		tx, err := s.client.GetTransactionReceipt(s.ctx, txHash)
		s.Require().NoError(err, "Failed to get transaction receipt")
		s.Require().NotNil(tx, "Transaction receipt should not be nil")

		s.T().Logf("Transaction details:")
		s.T().Logf("  Source:      %s", tx.Source)
		s.T().Logf("  Destination: %s", tx.Destination)
		s.T().Logf("  Amount:      %s", tx.Amount.String())
		s.T().Logf("  Fees:        %s", tx.Fees.String())

		s.Assert().NotEmpty(tx.Source, "Transaction source should not be empty")
		s.Assert().NotNil(tx.Amount, "Transaction amount should not be nil")
		s.Assert().NotNil(tx.Fees, "Transaction fees should not be nil")
	})

	s.Run("Fetch Non-Existent Transaction Receipt", func() {
		// Use a non-existent transaction hash
		txHash := "0x0000000000000000000000000000000000000000000000000000000000000000"

		tx, err := s.client.GetTransactionReceipt(s.ctx, txHash)
		s.Assert().Error(err)
		s.Assert().Nil(tx)
	})
}

// TestConvertBlock tests block conversion functionality
func (s *EthereumClientTestSuite) TestConvertBlock() {
	s.Run("Convert Recent Block", func() {
		// Get the latest block
		block, err := s.client.GetBlockByNumber(s.ctx, nil)
		s.Require().NoError(err)
		s.Require().NotNil(block)

		s.T().Logf("Block details:")
		s.T().Logf("  Number:      %s", block.Number.String())
		s.T().Logf("  Hash:        %s", block.Hash)
		s.T().Logf("  Timestamp:   %d", block.Timestamp)
		s.T().Logf("  Transactions: %d", len(block.Transactions))

		s.Assert().NotNil(block.Number, "Block number should not be nil")
		s.Assert().NotEmpty(block.Hash, "Block hash should not be empty")
		s.Assert().NotZero(block.Timestamp, "Block timestamp should be non-zero")

		// Optional: Check transactions if present
		if len(block.Transactions) > 0 {
			tx := block.Transactions[0]
			s.T().Logf("First transaction:")
			s.T().Logf("  Hash:        %s", tx.Hash)
			s.T().Logf("  Source:      %s", tx.Source)
			s.T().Logf("  Destination: %s", tx.Destination)
			s.T().Logf("  Amount:      %s", tx.Amount.String())
		}
	})

	s.Run("Convert Specific Block", func() {
		// Use a known block number (e.g., a recent milestone block)
		blockNumber := big.NewInt(4000000)

		block, err := s.client.GetBlockByNumber(s.ctx, blockNumber)
		s.Require().NoError(err)
		s.Require().NotNil(block)

		s.Assert().Equal(blockNumber.String(), block.Number.String(), "Block number should match")
		s.Assert().NotEmpty(block.Hash, "Block hash should not be empty")
		s.Assert().NotZero(block.Timestamp, "Block timestamp should be non-zero")
	})
}

// Benchmark tests to measure performance
func BenchmarkSubscribeToBlocks_Integration(b *testing.B) {
	rpcURL := os.Getenv(envRPCURL)
	wsURL := os.Getenv(envWSURL)
	if rpcURL == "" || wsURL == "" {
		b.Skip("Skipping benchmark: Ethereum URLs not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	client, err := NewEthereumClient(logger, rpcURL, wsURL)
	require.NoError(b, err)
	defer client.Close(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		blockChan, _ := client.SubscribeToBlocks(ctx)

		// Consume blocks
		for {
			select {
			case <-ctx.Done():
				cancel()
				return
			case <-blockChan:
				// Do nothing, just consume
			}
		}
	}
}

// Run the test suite
func TestEthereumClientSuiteIntegration(t *testing.T) {
	suite.Run(t, new(EthereumClientTestSuite))
}
