package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"

	"github.com/nimafallahian/go-workflow/internal/domain"
)

// Error values returned by the indexer for callers to react to.
var (
	ErrTooManyRequests = fmt.Errorf("elasticsearch: too many requests (429)")
	ErrServerError     = fmt.Errorf("elasticsearch: server error (5xx)")
)

// Indexer implements ports.DataIndexer using the Elasticsearch Bulk API.
type Indexer struct {
	client *elasticsearch.Client
	index  string
}

// NewIndexer constructs a new Indexer.
func NewIndexer(client *elasticsearch.Client, index string) (*Indexer, error) {
	if client == nil {
		return nil, fmt.Errorf("client must not be nil")
	}
	if index == "" {
		return nil, fmt.Errorf("index must not be empty")
	}
	return &Indexer{
		client: client,
		index:  index,
	}, nil
}

// Index implements ports.DataIndexer by sending documents via the Bulk API
// with refresh=wait_for (suitable for tests).
func (i *Indexer) Index(ctx context.Context, events []domain.MessageEvent) error {
	if len(events) == 0 {
		return nil
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	for _, evt := range events {
		// Action line
		meta := map[string]any{
			"index": map[string]any{
				"_index": i.index,
				"_id":    evt.ID,
			},
		}
		if err := enc.Encode(meta); err != nil {
			return fmt.Errorf("encode bulk meta: %w", err)
		}

		// Document line
		if err := enc.Encode(evt); err != nil {
			return fmt.Errorf("encode bulk doc: %w", err)
		}
	}

	res, err := i.client.Bulk(
		bytes.NewReader(buf.Bytes()),
		i.client.Bulk.WithContext(ctx),
		i.client.Bulk.WithRefresh("wait_for"),
	)
	if err != nil {
		return fmt.Errorf("bulk request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode == http.StatusConflict {
		// 409 Conflict: idempotent conflict, ignore per policy.
		return nil
	}

	if res.StatusCode == http.StatusTooManyRequests {
		return ErrTooManyRequests
	}

	if res.StatusCode >= 500 && res.StatusCode <= 599 {
		return ErrServerError
	}

	if res.IsError() {
		return fmt.Errorf("bulk error: %s", res.String())
	}

	// Inspect per-item errors in the bulk response.
	var body struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Status int `json:"status"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		// If decoding fails, surface a generic error.
		return fmt.Errorf("decode bulk response: %w", err)
	}

	if !body.Errors {
		return nil
	}

	for _, item := range body.Items {
		for _, v := range item {
			switch {
			case v.Status == http.StatusConflict:
				// 409 Conflict: ignore.
				continue
			case v.Status == http.StatusTooManyRequests:
				return ErrTooManyRequests
			case v.Status >= 500 && v.Status <= 599:
				return ErrServerError
			}
		}
	}

	return nil
}

