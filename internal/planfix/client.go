package planfix

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/buildinfo"
)

// Client is a thin Planfix REST transport: auth, throttle, retry.
type Client struct {
	Domain    string
	Token     string
	BaseURL   string // overrides https://<domain>/rest (used in tests)
	HTTP      *http.Client
	Limiter   *rate.Limiter
	Retries   int
	Backoff   func(attempt int) time.Duration
	UserAgent string // sent as the User-Agent header unless a caller overrides it
}

// New returns a Client with sane defaults (~5 req/s, 3 attempts).
func New(domain, token string) *Client {
	return &Client{
		Domain:    domain,
		Token:     token,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		Limiter:   rate.NewLimiter(rate.Limit(5), 1),
		Retries:   3,
		Backoff:   defaultBackoff,
		UserAgent: "pfix/" + buildinfo.Version,
	}
}

func defaultBackoff(attempt int) time.Duration {
	return time.Duration(attempt) * 500 * time.Millisecond
}

// URL builds the absolute REST URL for a path like "task/123" or "/rest/task/123".
func (c *Client) URL(path string) string {
	base := c.BaseURL
	if base == "" {
		base = "https://" + c.Domain + "/rest"
	}
	p := strings.TrimPrefix(path, "/")
	p = strings.TrimPrefix(p, "rest/")
	return strings.TrimRight(base, "/") + "/" + p
}

// Do sends an authenticated request, throttling and retrying transient failures.
// It returns the response for any HTTP status; only transport errors return err.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	if c.Retries <= 0 {
		return nil, fmt.Errorf("planfix: Retries must be > 0")
	}
	url := c.URL(path)
	for attempt := 1; attempt <= c.Retries; attempt++ {
		if err := c.Limiter.Wait(ctx); err != nil {
			return nil, err
		}
		var r io.Reader
		if len(body) > 0 {
			r = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, r)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.UserAgent != "" {
			req.Header.Set("User-Agent", c.UserAgent)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			if attempt < c.Retries {
				time.Sleep(c.Backoff(attempt))
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", c.Retries, err)
		}
		if resp.StatusCode >= 500 && attempt < c.Retries {
			resp.Body.Close()
			time.Sleep(c.Backoff(attempt))
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("planfix: request did not complete")
}
