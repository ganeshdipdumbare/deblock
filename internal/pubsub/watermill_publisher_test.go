package pubsub

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

// WatermillPublisherTestSuite encapsulates the test suite for Watermill Kafka Publisher
type WatermillPublisherTestSuite struct {
	suite.Suite
	ctx            context.Context
	kafkaContainer *kafka.KafkaContainer
	brokers        []string
	logger         *slog.Logger
	publisher      *kafkaWatermillPublisher
}

// SetupSuite starts the Kafka container before all tests
func (s *WatermillPublisherTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Create Kafka container with explicit configuration
	var err error
	s.kafkaContainer, err = kafka.Run(
		s.ctx,
		"confluentinc/cp-kafka:7.4.1",
		kafka.WithClusterID("test-cluster"),
	)
	s.Require().NoError(err, "Failed to start Kafka container")

	// Get Kafka broker address
	s.brokers, err = s.kafkaContainer.Brokers(s.ctx)
	s.Require().NoError(err, "Failed to get Kafka brokers")
	s.Require().NotEmpty(s.brokers, "Kafka brokers should not be empty")

	// Setup logger
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create publisher
	s.publisher, err = NewKafkaWatermillPublisher(s.logger, s.brokers)
	s.Require().NoError(err, "Failed to create Kafka publisher")
}

// TearDownSuite stops the Kafka container after all tests
func (s *WatermillPublisherTestSuite) TearDownSuite() {
	if s.kafkaContainer != nil {
		s.Require().NoError(s.kafkaContainer.Terminate(s.ctx), "Failed to terminate Kafka container")
	}
}

// TestPublishMessage tests publishing a single message
func (s *WatermillPublisherTestSuite) TestPublishMessage() {
	topic := "test-topic"
	message := []byte("test message")

	err := s.publisher.Publish(s.ctx, topic, message)
	s.Require().NoError(err, "Failed to publish message")
}

// TestPublishMultipleMessages tests publishing multiple messages
func (s *WatermillPublisherTestSuite) TestPublishMultipleMessages() {
	topic := "multiple-messages-topic"
	messages := [][]byte{
		[]byte("message 1"),
		[]byte("message 2"),
		[]byte("message 3"),
	}

	for _, msg := range messages {
		err := s.publisher.Publish(s.ctx, topic, msg)
		s.Require().NoError(err, "Failed to publish message")
	}
}

// TestPublishWithEmptyMessage tests publishing an empty message
func (s *WatermillPublisherTestSuite) TestPublishWithEmptyMessage() {
	topic := "empty-message-topic"
	message := []byte{}

	err := s.publisher.Publish(s.ctx, topic, message)
	s.Require().NoError(err, "Failed to publish empty message")
}

// Run the test suite
func TestWatermillPublisherSuite(t *testing.T) {
	suite.Run(t, new(WatermillPublisherTestSuite))
}
