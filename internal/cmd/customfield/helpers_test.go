package customfield

import (
	"time"

	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/planfix"
)

// fakeClient returns a client factory that points at srvURL with infinite rate
// limiting and zero backoff, making tests hermetic and fast.
func fakeClient(srvURL string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = srvURL
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}
