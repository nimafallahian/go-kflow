package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nimafallahian/go-workflow/internal/domain"
	"github.com/nimafallahian/go-workflow/internal/ports"
)

type mockMessageConsumer struct {
	mock.Mock
}

func (m *mockMessageConsumer) Consume(ctx context.Context) (<-chan ports.KafkaMessage, <-chan error) {
	args := m.Called(ctx)
	return args.Get(0).(<-chan ports.KafkaMessage), args.Get(1).(<-chan error)
}

type mockDataIndexer struct {
	mock.Mock
}

func (m *mockDataIndexer) Index(ctx context.Context, events []domain.MessageEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func TestIndexerService_ValidMessageIndexedAndAcknowledged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgCh := make(chan ports.KafkaMessage, 1)
	errCh := make(chan error)

	consumer := &mockMessageConsumer{}
	consumer.On("Consume", mock.Anything).Return((<-chan ports.KafkaMessage)(msgCh), (<-chan error)(errCh))

	indexer := &mockDataIndexer{}
	indexer.On("Index", mock.Anything, mock.AnythingOfType("[]domain.MessageEvent")).Return(nil).Run(
		func(args mock.Arguments) {
			events := args.Get(1).([]domain.MessageEvent)
			require.Len(t, events, 1)
			require.Equal(t, "msg-1", events[0].ID)
		},
	)

	svc := NewIndexerService(consumer, indexer, 1)

	var commitMu sync.Mutex
	committed := false
	done := make(chan struct{})

	msg := ports.KafkaMessage{
		Event: domain.MessageEvent{
			ID:         "msg-1",
			StatusCode: 200,
		},
		Commit: func(context.Context) error {
			commitMu.Lock()
			committed = true
			commitMu.Unlock()
			close(done)
			return nil
		},
	}

	go svc.Start(ctx)

	msgCh <- msg
	close(msgCh)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for commit")
	}

	cancel()

	indexer.AssertExpectations(t)
	consumer.AssertExpectations(t)

	commitMu.Lock()
	defer commitMu.Unlock()
	require.True(t, committed, "message should have been acknowledged")
}

func TestIndexerService_ClientErrorMessageAcknowledgedNotIndexed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgCh := make(chan ports.KafkaMessage, 1)
	errCh := make(chan error)

	consumer := &mockMessageConsumer{}
	consumer.On("Consume", mock.Anything).Return((<-chan ports.KafkaMessage)(msgCh), (<-chan error)(errCh))

	indexer := &mockDataIndexer{}

	svc := NewIndexerService(consumer, indexer, 1)

	ackCh := make(chan struct{}, 1)

	msg := ports.KafkaMessage{
		Event: domain.MessageEvent{
			ID:         "msg-4xx",
			StatusCode: 400, // client error, non-retriable
		},
		Commit: func(context.Context) error {
			ackCh <- struct{}{}
			return nil
		},
	}

	go svc.Start(ctx)

	msgCh <- msg
	close(msgCh)

	select {
	case <-ackCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for commit on 4xx message")
	}

	cancel()

	indexer.AssertNotCalled(t, "Index", mock.Anything, mock.Anything)
	consumer.AssertExpectations(t)
}

func TestIndexerService_IndexerErrorDoesNotAcknowledge(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgCh := make(chan ports.KafkaMessage, 1)
	errCh := make(chan error)

	consumer := &mockMessageConsumer{}
	consumer.On("Consume", mock.Anything).Return((<-chan ports.KafkaMessage)(msgCh), (<-chan error)(errCh))

	indexer := &mockDataIndexer{}
	indexer.On("Index", mock.Anything, mock.AnythingOfType("[]domain.MessageEvent")).
		Return(errors.New("elasticsearch 5xx"))

	svc := NewIndexerService(consumer, indexer, 1)

	ackCh := make(chan struct{}, 1)

	msg := ports.KafkaMessage{
		Event: domain.MessageEvent{
			ID:         "msg-es-error",
			StatusCode: 200,
		},
		Commit: func(context.Context) error {
			ackCh <- struct{}{}
			return nil
		},
	}

	go svc.Start(ctx)

	msgCh <- msg
	close(msgCh)

	// Give the worker some time to process; we expect no acknowledgement.
	select {
	case <-ackCh:
		t.Fatal("did not expect message to be acknowledged on indexer error")
	case <-time.After(500 * time.Millisecond):
		// expected path: no ack within this window
	}

	cancel()

	indexer.AssertExpectations(t)
	consumer.AssertExpectations(t)
}

