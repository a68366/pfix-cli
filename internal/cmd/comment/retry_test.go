package comment

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// A 5xx on delete must not be retried: if the first attempt deleted the
// comment, a retry answers an error and misreports a success as a failure.
// Edit sets the same values again, so its POST keeps retrying.
func TestCommentWriteRetrySemantics(t *testing.T) {
	tests := []struct {
		name     string
		run      func(srvURL string) error
		wantHits int32 // write requests only; edit's resolving GET is not counted
		wantErr  bool
	}{
		{
			name: "delete is sent once",
			run: func(u string) error {
				return runDelete(context.Background(), &deleteOptions{force: true, client: fakeClient(u), out: &strings.Builder{}}, "12")
			},
			wantHits: 1, wantErr: true,
		},
		{
			name: "edit retries",
			run: func(u string) error {
				o := &editOptions{body: map[string]any{"isPinned": true}, client: fakeClient(u), out: &strings.Builder{}, in: strings.NewReader("")}
				return runEdit(context.Background(), o, "12")
			},
			wantHits: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					io.WriteString(w, `{"result":"success","comment":{"id":12,"task":{"id":3}}}`)
					return
				}
				if atomic.AddInt32(&hits, 1) == 1 {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				io.WriteString(w, `{"result":"success","id":12}`)
			}))
			defer srv.Close()

			err := tt.run(srv.URL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got := atomic.LoadInt32(&hits); got != tt.wantHits {
				t.Errorf("requests = %d, want %d", got, tt.wantHits)
			}
		})
	}
}
