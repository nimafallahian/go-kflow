package ports

import (
	"context"

	"github.com/nimafallahian/go-workflow/internal/domain"
)

// KafkaMessage represents a unit of work received from Kafka along with
// a mechanism to commit its offset once processing has completed.
type KafkaMessage struct {
	Event domain.MessageEvent

	// Commit commits the underlying Kafka message offset after successful processing.
	Commit func(ctx context.Context) error
}

// MessageConsumer exposes a streaming interface for consuming Kafka messages.
// Implementations must be goroutine-safe and compatible with select-based loops.
type MessageConsumer interface {
	// Consume returns a read-only channel of KafkaMessage instances and a channel
	// for terminal errors from the consumer loop. Both channels must be closed
	// when the provided context is cancelled or the consumer shuts down.
	Consume(ctx context.Context) (<-chan KafkaMessage, <-chan error)
}

