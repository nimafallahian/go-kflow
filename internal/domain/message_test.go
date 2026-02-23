package domain

import "testing"

func TestMessageEvent_StatusCategory(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected StatusCategory
	}{
		{name: "zero treated as success", code: 0, expected: StatusCategorySuccess},
		{name: "2xx lower bound", code: 200, expected: StatusCategorySuccess},
		{name: "2xx upper bound", code: 299, expected: StatusCategorySuccess},
		{name: "4xx lower bound", code: 400, expected: StatusCategoryClientError},
		{name: "4xx upper bound", code: 499, expected: StatusCategoryClientError},
		{name: "5xx lower bound", code: 500, expected: StatusCategoryServerError},
		{name: "5xx upper bound", code: 599, expected: StatusCategoryServerError},
		{name: "unknown below range", code: 199, expected: StatusCategoryUnknown},
		{name: "unknown above range", code: 600, expected: StatusCategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := MessageEvent{StatusCode: tt.code}
			if got := e.StatusCategory(); got != tt.expected {
				t.Fatalf("StatusCategory() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestMessageEvent_IsRetriable(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected bool
	}{
		{name: "zero not retriable", code: 0, expected: false},
		{name: "2xx not retriable", code: 200, expected: false},
		{name: "4xx not retriable", code: 404, expected: false},
		{name: "5xx retriable", code: 503, expected: true},
		{name: "unknown not retriable", code: 750, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := MessageEvent{StatusCode: tt.code}
			if got := e.IsRetriable(); got != tt.expected {
				t.Fatalf("IsRetriable() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestMessageEvent_ShouldIndex(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected bool
	}{
		{name: "zero indexes", code: 0, expected: true},
		{name: "2xx indexes", code: 201, expected: true},
		{name: "4xx does not index", code: 422, expected: false},
		{name: "5xx does not index", code: 503, expected: false},
		{name: "unknown does not index", code: 750, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := MessageEvent{StatusCode: tt.code}
			if got := e.ShouldIndex(); got != tt.expected {
				t.Fatalf("ShouldIndex() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestMessageEvent_ShouldDeadLetterWithoutRetry(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected bool
	}{
		{name: "zero not DLQ", code: 0, expected: false},
		{name: "2xx not DLQ", code: 200, expected: false},
		{name: "4xx DLQ without retry", code: 400, expected: true},
		{name: "5xx not DLQ without retry", code: 500, expected: false},
		{name: "unknown not DLQ", code: 750, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := MessageEvent{StatusCode: tt.code}
			if got := e.ShouldDeadLetterWithoutRetry(); got != tt.expected {
				t.Fatalf("ShouldDeadLetterWithoutRetry() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

