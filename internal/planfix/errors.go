package planfix

import (
	"encoding/json"
	"fmt"
)

// APIError represents a non-2xx response from the Planfix API.
type APIError struct {
	StatusCode int
	Code       int
	Message    string
	Body       []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("planfix API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("planfix API error (HTTP %d)", e.StatusCode)
}

// ParseError builds an APIError, extracting the message and app code from the body.
func ParseError(statusCode int, body []byte) *APIError {
	msg, code := parseBody(body)
	return &APIError{StatusCode: statusCode, Code: code, Message: msg, Body: body}
}

// parseBody extracts (message, appCode) from a Planfix error envelope. A body
// that is not valid JSON (e.g. an HTML error page) yields ("", 0) so the error
// renders as a clean "HTTP <status>" rather than dumping markup.
func parseBody(body []byte) (string, int) {
	if !json.Valid(body) {
		return "", 0
	}
	var probe struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
		Code    int             `json:"code"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return "", 0
	}
	msg := probe.Message
	if msg == "" && len(probe.Error) > 0 {
		var s string
		if json.Unmarshal(probe.Error, &s) == nil && s != "" {
			msg = s
		} else {
			var obj struct {
				Message string `json:"message"`
			}
			if json.Unmarshal(probe.Error, &obj) == nil {
				msg = obj.Message
			}
		}
	}
	return msg, probe.Code
}
