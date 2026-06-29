package planfix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// JSON sends a request whose body is the JSON encoding of body (nil → no body),
// and returns the raw response bytes. On HTTP status >= 300 it returns *APIError.
func (c *Client) JSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var raw []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		raw = b
	}
	resp, err := c.Do(ctx, method, path, raw, nil)
	if err != nil {
		return nil, err
	}
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
