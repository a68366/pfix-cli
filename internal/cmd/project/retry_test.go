package project

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// A 5xx on create must not be retried: the first attempt may already have
// created the project, and a retry would create a duplicate.
func TestRunCreateSendsOnce(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		io.WriteString(w, `{"result":"success","id":9}`)
	}))
	defer srv.Close()

	o := &createOptions{name: "x", client: fakeClient(srv.URL), out: &strings.Builder{}}
	if err := runCreate(context.Background(), o); err == nil {
		t.Fatal("want error")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}
