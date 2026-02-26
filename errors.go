package pay

import "fmt"

// APIError represents an error response from the v2 payment API (4xx/5xx).
// StatusCode is the HTTP status; Message is the human-readable body message.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
}

// ValidationError is returned when the SDK rejects a request before
// it reaches the API (e.g. nil request, empty intent ID).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return "validation: " + e.Message }
