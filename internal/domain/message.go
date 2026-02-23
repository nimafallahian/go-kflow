package domain

// MessageEvent is the core domain model that mirrors contracts/message.json.
type MessageEvent struct {
	ID         string            `json:"id"`
	Payload    map[string]any    `json:"payload"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	StatusCode int               `json:"status_code"`
}

// StatusCategory represents the coarse classification of an HTTP-like status code.
type StatusCategory int

const (
	StatusCategoryUnknown StatusCategory = iota
	StatusCategorySuccess
	StatusCategoryClientError
	StatusCategoryServerError
)

// StatusCategory returns the category of the event's StatusCode.
// A zero value status is treated as success (default forward-to-index behaviour).
func (e MessageEvent) StatusCategory() StatusCategory {
	code := e.StatusCode

	if code == 0 {
		return StatusCategorySuccess
	}

	switch {
	case code >= 200 && code <= 299:
		return StatusCategorySuccess
	case code >= 400 && code <= 499:
		return StatusCategoryClientError
	case code >= 500 && code <= 599:
		return StatusCategoryServerError
	default:
		return StatusCategoryUnknown
	}
}

// IsRetriable returns true when processing should be retried on failure.
// Per Kafka policy, only 5xx server errors are retriable.
func (e MessageEvent) IsRetriable() bool {
	return e.StatusCategory() == StatusCategoryServerError
}

// ShouldIndex returns true when the message should be sent to the indexer.
// Default/2xx statuses are indexable.
func (e MessageEvent) ShouldIndex() bool {
	cat := e.StatusCategory()
	return cat == StatusCategorySuccess
}

// ShouldDeadLetterWithoutRetry returns true when the message is invalid and
// should be sent directly to the DLQ without retries (4xx client errors).
func (e MessageEvent) ShouldDeadLetterWithoutRetry() bool {
	return e.StatusCategory() == StatusCategoryClientError
}

