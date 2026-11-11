package file

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenameRejectsBeforeAnyRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		newName string
		wantErr string
	}{
		{"bad id", "abc", "x.txt", "abc"},
		{"zero id", "0", "x.txt", "0"},
		{"empty name", "12", "", "must not be empty"},
		{"blank name", "12", "   ", "must not be empty"},
		{"slash", "12", "../x.txt", "path separator"},
		{"backslash", "12", `a\b.txt`, "path separator"},
		{"dot dot", "12", "..", "must be a file name"},
		{"control char", "12", "a\nb.txt", "control characters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hit = true
			}))
			defer srv.Close()

			o := &renameOptions{client: fakeClient(srv.URL), out: &strings.Builder{}}
			err := runRename(context.Background(), o, tt.id, tt.newName)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to mention %q", err, tt.wantErr)
			}
			if hit {
				t.Error("no request may be sent")
			}
		})
	}
}

func TestRenameRequestAndOutput(t *testing.T) {
	tests := []struct {
		name  string
		quiet bool
		json  bool
		jq    string
		want  string
		exact bool
	}{
		{"human", false, false, "", "Renamed file 12 to отчёт за май.txt\n", true},
		{"quiet", true, false, "", "12\n", true},
		{"json", false, true, "", `"result": "success"`, false},
		{"jq", false, true, ".result", "success\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath, gotBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				b, _ := io.ReadAll(r.Body)
				gotBody = string(b)
				_, _ = io.WriteString(w, `{"result":"success"}`)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &renameOptions{quiet: tt.quiet, json: tt.json, jq: tt.jq, client: fakeClient(srv.URL), out: out}
			if err := runRename(context.Background(), o, "12", "отчёт за май.txt"); err != nil {
				t.Fatalf("runRename: %v", err)
			}
			if gotMethod != http.MethodPost || gotPath != "/file/12" {
				t.Errorf("request = %s %s, want POST /file/12", gotMethod, gotPath)
			}
			if gotBody != `{"name":"отчёт за май.txt"}` {
				t.Errorf("body = %s, want only the name", gotBody)
			}
			if tt.exact && out.String() != tt.want {
				t.Errorf("output = %q, want %q", out.String(), tt.want)
			}
			if !tt.exact && !strings.Contains(out.String(), tt.want) {
				t.Errorf("output = %q, want it to contain %q", out.String(), tt.want)
			}
		})
	}
}

func TestRenameAPIErrors(t *testing.T) {
	for _, tt := range fileWriteErrorCases {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &renameOptions{client: fakeClient(srv.URL), out: out}
			checkWriteError(t, runRename(context.Background(), o, "12", "x.txt"), tt.want, tt.reject)
			if out.Len() != 0 {
				t.Errorf("nothing may be printed on failure, got %q", out.String())
			}
		})
	}
}

// Renaming is idempotent, so a transient 5xx is retried.
func TestRenameRetries5xx(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	o := &renameOptions{client: fakeClient(srv.URL), out: &strings.Builder{}}
	if err := runRename(context.Background(), o, "12", "x.txt"); err != nil {
		t.Fatalf("runRename: %v", err)
	}
	if hits != 2 {
		t.Errorf("requests = %d, want 2", hits)
	}
}
