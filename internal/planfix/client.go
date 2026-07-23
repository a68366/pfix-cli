package planfix

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// Proxy selects the proxy for every request; default
	// http.ProxyFromEnvironment. Both the shared HTTP client and Stream's
	// timeout-free client resolve it through this field at request time, so
	// overriding it (or setting it to nil to disable proxying) governs all
	// requests.
	Proxy func(*http.Request) (*url.URL, error)
}

// New returns a Client with sane defaults (~5 req/s, 3 attempts).
func New(domain, token string) *Client {
	c := &Client{
		Domain:    domain,
		Token:     token,
		Limiter:   rate.NewLimiter(rate.Limit(5), 1),
		Retries:   3,
		Backoff:   defaultBackoff,
		UserAgent: "pfix/" + buildinfo.Version,
		Proxy:     http.ProxyFromEnvironment,
	}
	c.HTTP = &http.Client{Timeout: 30 * time.Second, Transport: c.newTransport(0)}
	return c
}

// newTransport builds an HTTP transport that resolves its proxy through c.Proxy
// at request time — so overriding c.Proxy governs every request — and clones the
// standard transport for its connection-pool defaults. respHeaderTimeout caps
// the wait for a response's headers (0 disables it).
func (c *Client) newTransport(respHeaderTimeout time.Duration) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		if c.Proxy == nil {
			return nil, nil
		}
		return c.Proxy(req)
	}
	t.ResponseHeaderTimeout = respHeaderTimeout
	return t
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

// Do sends an authenticated request, throttling and retrying transient
// failures. It returns the response for any HTTP status; only transport errors
// return err.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	return c.do(ctx, method, path, body, headers, c.HTTP)
}

// Stream sends an authenticated GET and returns the response with its Body
// unread; the caller must close it. Unlike Do/JSON it runs against a client
// with no whole-request timeout (only a 30s response-header timeout), so
// reading a large body is not cut off by a deadline. It shares Do's proxy
// handling via c.Proxy (see the Proxy field). Redirects to object storage are
// followed by net/http, which drops the Authorization header on the cross-host
// hop.
func (c *Client) Stream(ctx context.Context, path string) (*http.Response, error) {
	hc := &http.Client{Transport: c.newTransport(30 * time.Second)}
	return c.do(ctx, http.MethodGet, path, nil, nil, hc)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, headers map[string]string, hc *http.Client) (*http.Response, error) {
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

		resp, err := hc.Do(req)
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
