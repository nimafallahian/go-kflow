package service

import (
	"context"
	"sync"

	"github.com/nimafallahian/go-workflow/internal/domain"
	"github.com/nimafallahian/go-workflow/internal/ports"
)

// IndexerService orchestrates reading messages from Kafka, applying domain
// rules, indexing into the data store, and acknowledging offsets.
type IndexerService struct {
	consumer    ports.MessageConsumer
	indexer     ports.DataIndexer
	workerCount int
}

// NewIndexerService constructs a new IndexerService.
func NewIndexerService(consumer ports.MessageConsumer, indexer ports.DataIndexer, workerCount int) *IndexerService {
	if workerCount <= 0 {
		workerCount = 1
	}
	return &IndexerService{
		consumer:    consumer,
		indexer:     indexer,
		workerCount: workerCount,
	}
}

// Start begins consuming messages and processing them with a worker pool.
// It blocks until the context is cancelled.
func (s *IndexerService) Start(ctx context.Context) {
	msgCh, errCh := s.consumer.Consume(ctx)

	var wg sync.WaitGroup
	wg.Add(s.workerCount)

	for i := 0; i < s.workerCount; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-msgCh:
					if !ok {
						return
					}
					s.handleMessage(ctx, msg)
				}
			}
		}()
	}

	// Drain error channel in a separate goroutine to avoid blocking producers.
	go func() {
		for range errCh {
			// In this initial implementation we ignore adapter-level errors here.
		}
	}()

	<-ctx.Done()
	wg.Wait()
}

func (s *IndexerService) handleMessage(ctx context.Context, msg ports.KafkaMessage) {
	event := msg.Event

	// 4xx / non-retriable: skip indexing but acknowledge to clear from queue.
	if event.ShouldDeadLetterWithoutRetry() {
		if msg.Commit != nil {
			_ = msg.Commit(ctx)
		}
		return
	}

	// Messages that shouldn't be indexed are simply acknowledged.
	if !event.ShouldIndex() {
		if msg.Commit != nil {
			_ = msg.Commit(ctx)
		}
		return
	}

	// Index message; on error, do not acknowledge so Kafka can redeliver.
	if err := s.indexer.Index(ctx, []domain.MessageEvent{event}); err != nil {
		return
	}

	if msg.Commit != nil {
		_ = msg.Commit(ctx)
	}
}

