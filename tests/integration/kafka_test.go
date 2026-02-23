package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	kafkamodule "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/stretchr/testify/require"

	adapterskafka "github.com/nimafallahian/go-workflow/internal/adapters/kafka"
	"github.com/nimafallahian/go-workflow/internal/domain"
)

var (
	kafkaContainer *kafkamodule.KafkaContainer
	kafkaBrokers   []string
)

func TestMain(m *testing.M) {
	// If Docker is not available (common in restricted CI or IDE sandboxes),
	// skip spinning up Kafka and let tests decide whether to run.
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	kafkaContainer, err = kafkamodule.Run(ctx,
		"confluentinc/confluent-local:7.5.0",
		kafkamodule.WithClusterID("test-cluster"),
	)
	if err != nil {
		panic(err)
	}

	kafkaBrokers, err = kafkaContainer.Brokers(ctx)
	if err != nil {
		_ = kafkaContainer.Terminate(ctx)
		panic(err)
	}

	code := m.Run()

	_ = kafkaContainer.Terminate(context.Background())

	os.Exit(code)
}

func TestKafkaConsumerReadsMessageEvent(t *testing.T) {
	if len(kafkaBrokers) == 0 {
		t.Skip("kafka container not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const topic = "messages"
	groupID := "test-group-consumer"

	// Ensure topic exists before producing/consuming to avoid UNKNOWN_TOPIC_OR_PARTITION
	adminConn, err := kafkago.Dial("tcp", kafkaBrokers[0])
	require.NoError(t, err)
	err = adminConn.CreateTopics(kafkago.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	require.NoError(t, err)
	require.NoError(t, adminConn.Close())

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(kafkaBrokers...),
		Topic:        topic,
		Balancer:     &kafkago.LeastBytes{},
		RequiredAcks: kafkago.RequireAll,
	}
	defer func() { _ = writer.Close() }()

	event := domain.MessageEvent{
		ID: "msg-1",
		Payload: map[string]any{
			"foo": "bar",
		},
		Metadata: map[string]string{
			"source": "integration-test",
		},
		StatusCode: 200,
	}

	value, err := json.Marshal(event)
	require.NoError(t, err)

	err = writer.WriteMessages(ctx, kafkago.Message{
		Value: value,
	})
	require.NoError(t, err)

	consumer, err := adapterskafka.NewConsumer(kafkaBrokers, topic, groupID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	msgCh, errCh := consumer.Stream(ctx)

	select {
	case msg := <-msgCh:
		require.Equal(t, event.ID, msg.Event.ID)
		require.Equal(t, event.StatusCode, msg.Event.StatusCode)
		require.Equal(t, event.Payload["foo"], msg.Event.Payload["foo"])
		require.Equal(t, event.Metadata["source"], msg.Event.Metadata["source"])

		require.NoError(t, consumer.Acknowledge(ctx, msg))
	case err := <-errCh:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatalf("timeout waiting for message: %v", ctx.Err())
	}
}

