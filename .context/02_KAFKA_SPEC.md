# Policy: Kafka Consumer & Error Handling

## Connection & Offsets
- Use `segmentio/kafka-go`.
- **Offset Policy**: Manual commits only. `CommitMessages` must happen *after* successful indexing in Elasticsearch.

## Message Status Logic (The Catch)
The consumer must inspect headers or payload for a `status` field:
- **Status 2xx/Default**: Proceed to Indexing.
- **Status 4xx**: Log as "Invalid Data" and move to Dead Letter Queue (DLQ). Do not retry.
- **Status 5xx**: Implement Exponential Backoff (3 retries). If all fail, move to DLQ.

## Graceful Shutdown
- Listen for `ctx.Done()`. Finish the current processing batch before exiting.
