package blockchain

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// EthereumClient implements the Client interface for Ethereum
type EthereumClient struct {
	logger *slog.Logger
	client *ethclient.Client
	rpc    *rpc.Client
}

// NewEthereumClient creates a new Ethereum blockchain client
func NewEthereumClient(logger *slog.Logger, rpcURL, wsURL string) (*EthereumClient, error) {
	c, err := ethclient.Dial(wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}
	rc, err := rpc.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw rpc client: %w", err)
	}
	return &EthereumClient{logger: logger, client: c, rpc: rc}, nil
}

// SubscribeToBlocks starts streaming new blocks converted to generic Block type
func (e *EthereumClient) SubscribeToBlocks(ctx context.Context) (<-chan Block, <-chan error) {
	// Buffered channel ensures the last block can be queued during shutdown without blocking
	out := make(chan Block, 1)
	errC := make(chan error, 1)

	headers := make(chan *types.Header)
	sub, err := e.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		errC <- fmt.Errorf("failed to subscribe to new heads: %w", err)
		close(out)
		close(errC)
		return out, errC
	}

	go func() {
		defer sub.Unsubscribe()
		defer close(out)
		defer close(errC)

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-sub.Err():
				errC <- fmt.Errorf("subscription error: %w", err)
				return
			case h := <-headers:
				if h == nil {
					continue
				}
				// Use a bounded context decoupled from the subscription cancel to finish the last block
				convCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				blk, err := e.blockFromHeader(convCtx, h)
				cancel()
				if err != nil {
					e.logger.Error("failed to fetch block", "error", err, "number", h.Number)
					continue
				}
				select {
				case out <- *blk:
				case <-ctx.Done():
					// If shutting down and nobody is receiving, drop the block to avoid blocking
					return
				}
			}
		}
	}()

	return out, errC
}

// GetBlockByNumber retrieves a block by its number
func (e *EthereumClient) GetBlockByNumber(ctx context.Context, number *big.Int) (*Block, error) {
	ethBlock, err := e.client.BlockByNumber(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by number: %w", err)
	}
	return e.convertBlock(ctx, ethBlock)
}

// GetTransactionReceipt retrieves a transaction and computes fees (using effective gas price)
func (e *EthereumClient) GetTransactionReceipt(ctx context.Context, txHash string) (*Transaction, error) {
	hash := common.HexToHash(txHash)
	receipt, err := e.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx receipt: %w", err)
	}
	tx, _, err := e.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx: %w", err)
	}

	signer := types.LatestSignerForChainID(tx.ChainId())
	from, err := types.Sender(signer, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to derive sender: %w", err)
	}

	var to string
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	fees := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))

	return &Transaction{
		Source:      from.Hex(),
		Destination: to,
		Amount:      tx.Value(),
		Fees:        fees,
		Hash:        txHash,
		BlockNumber: receipt.BlockNumber,
	}, nil
}

// Close terminates the connection to the blockchain
func (e *EthereumClient) Close(_ context.Context) error {
	e.client.Close()
	if e.rpc != nil {
		e.rpc.Close()
	}
	return nil
}

// blockFromHeader fetches and converts a full block given its header
func (e *EthereumClient) blockFromHeader(ctx context.Context, h *types.Header) (*Block, error) {
	ethBlock, err := e.client.BlockByHash(ctx, h.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash: %w", err)
	}
	return e.convertBlock(ctx, ethBlock)
}

// convertTransaction converts an Ethereum transaction to our generic Transaction type
func (e *EthereumClient) convertTransaction(tx *types.Transaction, receipt *types.Receipt, blockNumber *big.Int) (*Transaction, error) {
	signer := types.LatestSignerForChainID(tx.ChainId())
	from, err := types.Sender(signer, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to derive sender: %w", err)
	}

	var to string
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	fees := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))

	return &Transaction{
		Source:      from.Hex(),
		Destination: to,
		Amount:      tx.Value(),
		Fees:        fees,
		Hash:        tx.Hash().Hex(),
		BlockNumber: blockNumber,
	}, nil
}

// convertBlock converts an Ethereum block to our generic Block type
func (e *EthereumClient) convertBlock(ctx context.Context, ethBlock *types.Block) (*Block, error) {
	txs := make([]Transaction, 0, len(ethBlock.Transactions()))

	// Fetch all receipts efficiently
	receipts, err := e.getBlockReceipts(ctx, ethBlock)
	if err != nil {
		e.logger.Warn("failed to get block receipts in bulk, will degrade", "error", err)
	}

	// Index receipts by tx hash for O(1) lookup
	receiptByHash := make(map[common.Hash]*types.Receipt, len(receipts))
	for _, r := range receipts {
		if r != nil {
			receiptByHash[r.TxHash] = r
		}
	}

	for _, tx := range ethBlock.Transactions() {
		receipt := receiptByHash[tx.Hash()]
		if receipt == nil {
			// If not available (provider missing method), try single-call as last resort
			r, rErr := e.client.TransactionReceipt(ctx, tx.Hash())
			if rErr != nil {
				e.logger.Warn("missing receipt for tx", "hash", tx.Hash().Hex(), "error", rErr)
				continue
			}
			receipt = r
		}

		convertedTx, err := e.convertTransaction(tx, receipt, ethBlock.Number())
		if err != nil {
			e.logger.Warn("failed to convert transaction", "hash", tx.Hash().Hex(), "error", err)
			continue
		}

		txs = append(txs, *convertedTx)
	}

	b := &Block{
		Number:       ethBlock.Number(),
		Hash:         ethBlock.Hash().Hex(),
		Timestamp:    int64(ethBlock.Time()),
		Difficulty:   ethBlock.Difficulty(),
		Transactions: txs,
	}
	return b, nil
}

// getBlockReceipts retrieves all receipts for a block using eth_getBlockReceipts
func (e *EthereumClient) getBlockReceipts(ctx context.Context, ethBlock *types.Block) ([]*types.Receipt, error) {
	if e.rpc == nil {
		return nil, fmt.Errorf("rpc client not initialized")
	}

	var receipts []*types.Receipt
	if err := e.rpc.CallContext(ctx, &receipts, "eth_getBlockReceipts", ethBlock.Hash()); err != nil {
		return nil, fmt.Errorf("failed to get block receipts: %w", err)
	}

	e.logger.Debug("Successfully fetched block receipts using eth_getBlockReceipts",
		"block", ethBlock.Number(),
		"receipt_count", len(receipts),
		"method", "eth_getBlockReceipts")

	return receipts, nil
}
