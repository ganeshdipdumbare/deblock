package address

import "context"

// Watcher defines an interface for filtering addresses
//
//go:generate go run go.uber.org/mock/mockgen@latest -source=address_watcher.go -destination=../../mocks/mock_address_watcher.go -package=mocks
type Watcher interface {
	// IsWatched checks if an address is being monitored
	IsWatched(ctx context.Context, address string) bool

	// AddAddresses adds new addresses to watch
	AddAddresses(ctx context.Context, addresses []string)

	// RemoveAddresses removes addresses from being watched
	RemoveAddresses(ctx context.Context, addresses []string)

	// GetWatchedAddresses returns all currently watched addresses
	GetWatchedAddresses(ctx context.Context) []string
}
