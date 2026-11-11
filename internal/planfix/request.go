package planfix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// JSON sends a request whose body is the JSON encoding of body (nil → no body),
// and returns the raw response bytes. On HTTP status >= 300 it returns *APIError.
// Connection errors and 5xx are retried (see Do).
func (c *Client) JSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	if c.Retries <= 0 {
		return nil, fmt.Errorf("planfix: Retries must be > 0")
	}
	return c.jsonCall(ctx, method, path, body, c.Retries)
}

// JSONOnce is JSON with exactly one attempt: a connection error or 5xx is
// returned rather than retried. Use it for non-idempotent writes, where a retry
// could repeat a request the server already acted on.
func (c *Client) JSONOnce(ctx context.Context, method, path string, body any) ([]byte, error) {
	return c.jsonCall(ctx, method, path, body, 1)
}

func (c *Client) jsonCall(ctx context.Context, method, path string, body any, attempts int) ([]byte, error) {
	var raw []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		raw = b
	}
	resp, err := c.attempt(ctx, method, path, raw, nil, c.HTTP, attempts)
	if err != nil {
		return nil, err
	}
	return readJSONResponse(resp)
}

// readJSONResponse reads and closes resp's body, mapping status >= 300 to
// *APIError.
func readJSONResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, ParseError(resp.StatusCode, data)
	}
	return data, nil
}
