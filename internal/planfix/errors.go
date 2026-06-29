package planfix

import (
	"encoding/json"
	"fmt"
	"strings"
)

// APIError represents a non-2xx response from the Planfix API.
type APIError struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("planfix API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("planfix API error (HTTP %d)", e.StatusCode)
}

// ParseError builds an APIError, extracting a human message from the body when possible.
func ParseError(statusCode int, body []byte) *APIError {
	return &APIError{StatusCode: statusCode, Message: extractMessage(body), Body: body}
}

// extractMessage tries common error shapes, falling back to a trimmed body.
// NOTE: confirm Planfix's exact error envelope against the live API and extend here.
func extractMessage(body []byte) string {
	var probe struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(body, &probe); err == nil {
		if probe.Message != "" {
			return probe.Message
		}
		if len(probe.Error) > 0 {
			var s string
			if json.Unmarshal(probe.Error, &s) == nil && s != "" {
				return s
			}
			var obj struct {
				Message string `json:"message"`
			}
			if json.Unmarshal(probe.Error, &obj) == nil && obj.Message != "" {
				return obj.Message
			}
		}
	}
	s := strings.TrimSpace(string(body))
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
