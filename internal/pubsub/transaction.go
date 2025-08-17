package pubsub

import "math/big"

// Transaction represents a generic blockchain transaction
type Transaction struct {
	Source      string
	Destination string
	Amount      *big.Int
	Fees        *big.Int
	Hash        string
}
