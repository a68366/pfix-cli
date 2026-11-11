package task

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// flakyServer answers the first request with a 502 and every later one with
// body, counting requests — so a retrying call succeeds on its second attempt
// while a single-attempt call fails after exactly one.
func flakyServer(t *testing.T, body string) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// Writes the server may already have applied when it answered 5xx must not be
// repeated: a retried create duplicates the record, a retried delete answers
// "not found" for a delete that succeeded. Idempotent updates keep retrying.
func TestTaskWriteRetrySemantics(t *testing.T) {
	tests := []struct {
		name     string
		run      func(srvURL string) error
		wantHits int32
		wantErr  bool
	}{
		{
			name: "create is sent once",
			run: func(u string) error {
				return runCreate(context.Background(), &createOptions{body: map[string]any{"name": "x"}, client: fakeClient(u), out: &strings.Builder{}})
			},
			wantHits: 1, wantErr: true,
		},
		{
			name: "comment add is sent once",
			run: func(u string) error {
				return runCommentAdd(context.Background(), &commentAddOptions{id: 7, body: "hi", client: fakeClient(u), out: &strings.Builder{}, in: strings.NewReader("")})
			},
			wantHits: 1, wantErr: true,
		},
		{
			name: "checklist add is sent once",
			run: func(u string) error {
				return runChecklistAdd(context.Background(), &checklistAddOptions{taskID: 7, body: map[string]any{"name": "x"}, client: fakeClient(u), out: &strings.Builder{}})
			},
			wantHits: 1, wantErr: true,
		},
		{
			name: "checklist delete is sent once",
			run: func(u string) error {
				return runChecklistDelete(context.Background(), &checklistDeleteOptions{force: true, client: fakeClient(u), out: &strings.Builder{}}, "7", "8")
			},
			wantHits: 1, wantErr: true,
		},
		{
			name: "update retries",
			run: func(u string) error {
				return runUpdate(context.Background(), &updateOptions{id: 7, body: map[string]any{"name": "x"}, client: fakeClient(u), out: &strings.Builder{}})
			},
			wantHits: 2,
		},
		{
			name: "checklist update retries",
			run: func(u string) error {
				return runChecklistUpdate(context.Background(), &checklistUpdateOptions{taskID: 7, itemID: 8, body: map[string]any{"name": "x"}, client: fakeClient(u), out: &strings.Builder{}})
			},
			wantHits: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, hits := flakyServer(t, `{"result":"success","id":9}`)
			err := tt.run(srv.URL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got := atomic.LoadInt32(hits); got != tt.wantHits {
				t.Errorf("requests = %d, want %d", got, tt.wantHits)
			}
		})
	}
}
