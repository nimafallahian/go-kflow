package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/nimafallahian/go-workflow/internal/domain"
	"github.com/nimafallahian/go-workflow/internal/ports"
)

// Consumer implements ports.MessageConsumer using segmentio/kafka-go.
type Consumer struct {
	reader *kafkago.Reader
}

// NewConsumer constructs a new Consumer configured for manual offset commits.
func NewConsumer(brokers []string, topic, groupID string) (*Consumer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("brokers must not be empty")
	}
	if topic == "" {
		return nil, fmt.Errorf("topic must not be empty")
	}
	if groupID == "" {
		return nil, fmt.Errorf("groupID must not be empty")
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		CommitInterval: 0, // manual commits only
	})

	return &Consumer{reader: reader}, nil
}

// Stream starts a goroutine that continuously reads from Kafka and pushes
// domain-mapped messages onto a channel until the context is cancelled.
func (c *Consumer) Stream(ctx context.Context) (<-chan ports.KafkaMessage, <-chan error) {
	msgCh := make(chan ports.KafkaMessage)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for {
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
					return
				}
				errCh <- err
				return
			}

			var event domain.MessageEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				errCh <- err
				return
			}

			kmsg := ports.KafkaMessage{
				Event: event,
				Commit: func(commitCtx context.Context) error {
					return c.reader.CommitMessages(commitCtx, m)
				},
			}

			select {
			case <-ctx.Done():
				return
			case msgCh <- kmsg:
			}
		}
	}()

	return msgCh, errCh
}

// Consume satisfies the ports.MessageConsumer interface by delegating to Stream.
func (c *Consumer) Consume(ctx context.Context) (<-chan ports.KafkaMessage, <-chan error) {
	return c.Stream(ctx)
}

// Acknowledge commits the offset for the given message using its embedded Commit
// function, if present.
func (c *Consumer) Acknowledge(ctx context.Context, msg ports.KafkaMessage) error {
	if msg.Commit == nil {
		return nil
	}
	return msg.Commit(ctx)
}

// Close releases the underlying reader resources.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

