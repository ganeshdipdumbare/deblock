package blockchain

import (
	"context"
	"math/big"
)

// Transaction represents a generic blockchain transaction
type Transaction struct {
	Source      string
	Destination string
	Amount      *big.Int
	Fees        *big.Int
	Hash        string
	BlockNumber *big.Int
}

// Block represents a generic blockchain block
type Block struct {
	Number       *big.Int
	Hash         string
	Timestamp    int64
	Difficulty   *big.Int
	Transactions []Transaction
}

// Client defines the interface for blockchain interactions
//
//go:generate go run go.uber.org/mock/mockgen@latest -source=blockchain.go -destination=../../mocks/mock_blockchain.go -package=mocks
type Client interface {
	// SubscribeToBlocks starts streaming new block headers
	SubscribeToBlocks(ctx context.Context) (<-chan Block, <-chan error)

	// GetBlockByNumber retrieves a block by its number
	GetBlockByNumber(ctx context.Context, number *big.Int) (*Block, error)

	// GetTransactionReceipt retrieves the receipt of a transaction
	GetTransactionReceipt(ctx context.Context, txHash string) (*Transaction, error)

	// Close terminates the connection to the blockchain
	Close(ctx context.Context) error
}
