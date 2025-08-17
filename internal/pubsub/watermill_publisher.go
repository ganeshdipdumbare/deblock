package pubsub

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
)

// kafkaWatermillPublisher implements the Publisher interface using Watermill with Kafka
type kafkaWatermillPublisher struct {
	logger         *slog.Logger
	kafkaPublisher message.Publisher
}

func NewKafkaWatermillPublisher(logger *slog.Logger, brokers []string) (*kafkaWatermillPublisher, error) {
	publisher, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:   brokers,
			Marshaler: kafka.DefaultMarshaler{},
		},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, err
	}
	return &kafkaWatermillPublisher{
		logger:         logger,
		kafkaPublisher: publisher,
	}, nil
}

func (p *kafkaWatermillPublisher) Publish(ctx context.Context, topic string, msg []byte) error {
	watermillMsg := message.NewMessage(watermill.NewUUID(), msg)
	return p.kafkaPublisher.Publish(topic, watermillMsg)
}

func (p *kafkaWatermillPublisher) Close(_ context.Context) error {
	return p.kafkaPublisher.Close()
}
