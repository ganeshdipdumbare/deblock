package pubsub

import "context"

// Publisher defines the interface for publishing messages
//
//go:generate go run go.uber.org/mock/mockgen@latest -source=publisher.go -destination=../../mocks/mock_publisher.go -package=mocks
type Publisher interface {
	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, message []byte) error

	// Close closes the publisher
	Close(ctx context.Context) error
}
