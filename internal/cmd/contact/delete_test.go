package contact

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

func TestRunDeleteRefusesBadInput(t *testing.T) {
	tests := []struct {
		name    string
		force   bool
		id      string
		wantErr string
	}{
		{"no force", false, "42", "--force"},
		{"bad id", true, "abc", "abc"},
		{"zero id", true, "0", "0"},
		{"negative id", true, "-3", "-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hit = true
			}))
			defer srv.Close()

			o := &deleteOptions{force: tt.force, client: fakeClient(srv.URL), out: &strings.Builder{}}
			err := runDelete(context.Background(), o, tt.id)
			if err == nil {
				t.Fatal("want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to mention %q", err, tt.wantErr)
			}
			if hit {
				t.Error("no request may be sent")
			}
		})
	}
}

func TestRunRestoreRefusesBadID(t *testing.T) {
	for _, id := range []string{"abc", "0", "-3"} {
		t.Run(id, func(t *testing.T) {
			hit := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hit = true
			}))
			defer srv.Close()

			o := &restoreOptions{client: fakeClient(srv.URL), out: &strings.Builder{}}
			if err := runRestore(context.Background(), o, id); err == nil {
				t.Fatal("want error")
			}
			if hit {
				t.Error("no request may be sent")
			}
		})
	}
}

// outputCase drives the shared -q/--json/--jq/human matrix for both commands.
type outputCase struct {
	name  string
	quiet bool
	json  bool
	jq    string
	want  string
	exact bool
}

func outputCases(human string) []outputCase {
	return []outputCase{
		{"human", false, false, "", human, true},
		{"quiet", true, false, "", "42\n", true},
		{"json", false, true, "", `"result": "success"`, false},
		{"jq", false, true, ".result", "success\n", true},
	}
}

func checkOutput(t *testing.T, tt outputCase, got string) {
	t.Helper()
	if tt.exact && got != tt.want {
		t.Errorf("output = %q, want %q", got, tt.want)
	}
	if !tt.exact && !strings.Contains(got, tt.want) {
		t.Errorf("output = %q, want it to contain %q", got, tt.want)
	}
}

func TestRunDeleteOutput(t *testing.T) {
	for _, tt := range outputCases("Deleted contact 42 (restore with: pfix contact restore 42)\n") {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				gotBody, _ = io.ReadAll(r.Body)
				_, _ = io.WriteString(w, `{"result":"success"}`)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &deleteOptions{force: true, quiet: tt.quiet, json: tt.json, jq: tt.jq, client: fakeClient(srv.URL), out: out}
			if err := runDelete(context.Background(), o, "42"); err != nil {
				t.Fatalf("runDelete: %v", err)
			}
			if gotMethod != http.MethodDelete || gotPath != "/contact/42" {
				t.Errorf("request = %s %s, want DELETE /contact/42", gotMethod, gotPath)
			}
			if len(gotBody) != 0 {
				t.Errorf("body = %q, want none", gotBody)
			}
			checkOutput(t, tt, out.String())
		})
	}
}

func TestRunRestoreOutput(t *testing.T) {
	for _, tt := range outputCases("Restored contact 42\n") {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				raw, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(raw, &gotBody); err != nil {
					t.Errorf("body %q is not JSON: %v", raw, err)
				}
				_, _ = io.WriteString(w, `{"result":"success"}`)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &restoreOptions{quiet: tt.quiet, json: tt.json, jq: tt.jq, client: fakeClient(srv.URL), out: out}
			if err := runRestore(context.Background(), o, "42"); err != nil {
				t.Fatalf("runRestore: %v", err)
			}
			if gotMethod != http.MethodPost || gotPath != "/contact/restore/" {
				t.Errorf("request = %s %s, want POST /contact/restore/", gotMethod, gotPath)
			}
			if len(gotBody) != 1 || gotBody["id"] != float64(42) {
				t.Errorf("body = %v, want exactly {\"id\":42}", gotBody)
			}
			checkOutput(t, tt, out.String())
		})
	}
}

func TestContactWriteAPIErrors(t *testing.T) {
	tests := []struct {
		name    string
		restore bool
		status  int
		body    string
		want    []string
		reject  []string
	}{
		{
			name:   "delete: unknown or already deleted",
			status: http.StatusBadRequest,
			body:   `{"result":"fail","code":2000,"error":"Contact not found by id - 42"}`,
			want:   []string{"Contact not found by id - 42", "already deleted", "pfix contact restore"},
		},
		{
			name:   "delete: 200 with result fail is an error",
			status: http.StatusOK,
			body:   `{"result":"fail","code":2000,"error":"Contact not found by id - 42"}`,
			want:   []string{"Contact not found by id - 42", "already deleted"},
		},
		{
			name:   "delete: scope denied passes through for the root to describe",
			status: http.StatusBadRequest,
			body:   `{"result":"fail","code":5,"error":"Access denied"}`,
			want:   []string{"Access denied"},
			reject: []string{"already deleted"},
		},
		{
			name:    "restore: unknown contact",
			restore: true,
			status:  http.StatusBadRequest,
			body:    `{"result":"fail","code":2000,"error":"Contact not found by id - 42"}`,
			want:    []string{"Contact not found by id - 42", "no contact with that id"},
			reject:  []string{"already deleted"},
		},
		{
			name:    "restore: contact not deleted",
			restore: true,
			status:  http.StatusBadRequest,
			body:    `{"result":"fail","code":2002,"error":"Contact not deleted by id - 42"}`,
			want:    []string{"Contact not deleted by id - 42", "nothing to restore"},
		},
		{
			name:    "restore: 200 with result fail is an error",
			restore: true,
			status:  http.StatusOK,
			body:    `{"result":"fail","code":2002,"error":"Contact not deleted by id - 42"}`,
			want:    []string{"nothing to restore"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			var err error
			if tt.restore {
				err = runRestore(context.Background(), &restoreOptions{client: fakeClient(srv.URL), out: out}, "42")
			} else {
				err = runDelete(context.Background(), &deleteOptions{force: true, client: fakeClient(srv.URL), out: out}, "42")
			}
			if err == nil {
				t.Fatal("want error")
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error = %v, want it to contain %q", err, w)
				}
			}
			for _, r := range tt.reject {
				if strings.Contains(err.Error(), r) {
					t.Errorf("error = %v, must not contain %q", err, r)
				}
			}
			if out.Len() != 0 {
				t.Errorf("nothing may be printed on failure, got %q", out.String())
			}
		})
	}
}

// Neither write may be retried: a retry after a call the server already
// carried out would answer "not found"/"not deleted" and misreport it.
func TestContactWritesSendOnce(t *testing.T) {
	for _, restore := range []bool{false, true} {
		hits := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			w.WriteHeader(http.StatusBadGateway)
		}))
		var err error
		if restore {
			err = runRestore(context.Background(), &restoreOptions{client: fakeClient(srv.URL), out: &strings.Builder{}}, "42")
		} else {
			err = runDelete(context.Background(), &deleteOptions{force: true, client: fakeClient(srv.URL), out: &strings.Builder{}}, "42")
		}
		srv.Close()
		if err == nil {
			t.Errorf("restore=%v: want error", restore)
		}
		if hits != 1 {
			t.Errorf("restore=%v: requests = %d, want 1", restore, hits)
		}
	}
}

func TestContactRegistersDeleteAndRestore(t *testing.T) {
	cmd := NewCmd(&cmdutil.GlobalOpts{})
	del, _, err := cmd.Find([]string{"delete"})
	if err != nil || del.Name() != "delete" {
		t.Fatalf("contact delete not registered: %v", err)
	}
	for _, w := range []string{"--force", "trash", "pfix contact restore"} {
		if !strings.Contains(del.Long, w) {
			t.Errorf("delete help missing %q: %q", w, del.Long)
		}
	}
	if del.Flags().Lookup("force") == nil {
		t.Error("delete has no --force flag")
	}
	res, _, err := cmd.Find([]string{"restore"})
	if err != nil || res.Name() != "restore" {
		t.Fatalf("contact restore not registered: %v", err)
	}
	if res.Flags().Lookup("force") != nil {
		t.Error("restore is not destructive and must not take --force")
	}
}
