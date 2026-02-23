# Policy: Elasticsearch Adapter

## Indexing Strategy
- Use the **Bulk API** for all indexing operations to ensure K8s pod efficiency.
- **Refresh Policy**: `wait_for` during integration tests, `false` in production.

## Error Mapping
- **409 Conflict**: Log and ignore (idempotency).
- **429 Too Many Requests**: Trigger backoff in the Service layer.
- **5xx**: Return a retriable error to the Kafka consumer.
