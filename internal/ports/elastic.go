package ports

import (
	"context"

	"github.com/nimafallahian/go-workflow/internal/domain"
)

// DataIndexer defines the system boundary for indexing domain events into
// a backing datastore such as Elasticsearch.
type DataIndexer interface {
	// Index requests that the given events be indexed. Implementations may use
	// bulk semantics under the hood and must honour the provided context.
	Index(ctx context.Context, events []domain.MessageEvent) error
}

