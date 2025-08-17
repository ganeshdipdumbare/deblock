package address

import (
	"context"
	"sync"
)

type inMemoryAddressWatcher struct {
	watchedAddresses map[string]bool
	mu               sync.RWMutex
}

func NewInMemoryAddressWatcher() *inMemoryAddressWatcher {
	return &inMemoryAddressWatcher{
		watchedAddresses: make(map[string]bool),
	}
}

func (w *inMemoryAddressWatcher) IsWatched(_ context.Context, address string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watchedAddresses[address]
}

func (w *inMemoryAddressWatcher) AddAddresses(_ context.Context, addresses []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, address := range addresses {
		w.watchedAddresses[address] = true
	}
}

func (w *inMemoryAddressWatcher) RemoveAddresses(_ context.Context, addresses []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, address := range addresses {
		delete(w.watchedAddresses, address)
	}
}

func (w *inMemoryAddressWatcher) GetWatchedAddresses(_ context.Context) []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	addresses := make([]string, 0, len(w.watchedAddresses))
	for address := range w.watchedAddresses {
		addresses = append(addresses, address)
	}
	return addresses
}
