package comment

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

// editServer answers the parent-resolution GET with parentJSON and records the
// following POST.
type editCapture struct {
	path  string
	query string
	body  map[string]any
}

func editServer(t *testing.T, parentJSON string, cap *editCapture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			io.WriteString(w, parentJSON)
			return
		}
		cap.path, cap.query = r.URL.Path, r.URL.RawQuery
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, &cap.body); err != nil {
			t.Errorf("unmarshal body %q: %v", raw, err)
		}
		io.WriteString(w, `{"result":"success"}`)
	}))
}

func TestEditResolvesTaskParent(t *testing.T) {
	cap := &editCapture{}
	srv := editServer(t, `{"result":"success","comment":{"id":11849892,"task":{"id":12}}}`, cap)
	defer srv.Close()

	out := &strings.Builder{}
	o := &editOptions{
		body:   map[string]any{"description": "new text"},
		client: fakeClient(srv.URL),
		out:    out,
		in:     strings.NewReader(""),
	}
	if err := runEdit(context.Background(), o, "11849892"); err != nil {
		t.Fatalf("runEdit: %v", err)
	}
	if cap.path != "/task/12/comments/11849892" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.body["description"] != "new text" {
		t.Errorf("body = %v", cap.body)
	}
	if !strings.Contains(out.String(), "Updated comment 11849892") {
		t.Errorf("output = %q", out.String())
	}
}

func TestEditResolvesContactParent(t *testing.T) {
	cap := &editCapture{}
	srv := editServer(t, `{"result":"success","comment":{"id":11849990,"contact":{"id":"contact:3","name":"Support"}}}`, cap)
	defer srv.Close()

	o := &editOptions{
		body:   map[string]any{"description": "x"},
		client: fakeClient(srv.URL),
		out:    &strings.Builder{},
		in:     strings.NewReader(""),
	}
	if err := runEdit(context.Background(), o, "11849990"); err != nil {
		t.Fatalf("runEdit: %v", err)
	}
	if cap.path != "/contact/contact:3/comments/11849990" {
		t.Errorf("path = %q", cap.path)
	}
}

func TestEditErrorsWithoutParent(t *testing.T) {
	cap := &editCapture{}
	srv := editServer(t, `{"result":"success","comment":{"id":7}}`, cap)
	defer srv.Close()

	o := &editOptions{
		body:   map[string]any{"description": "x"},
		client: fakeClient(srv.URL),
		out:    &strings.Builder{},
		in:     strings.NewReader(""),
	}
	if err := runEdit(context.Background(), o, "7"); err == nil {
		t.Fatal("want error when the comment has no task or contact")
	}
	if cap.path != "" {
		t.Errorf("no POST may be sent, got %q", cap.path)
	}
}

func TestEditSilentQuery(t *testing.T) {
	cap := &editCapture{}
	srv := editServer(t, `{"result":"success","comment":{"id":5,"task":{"id":1}}}`, cap)
	defer srv.Close()

	o := &editOptions{
		body:   map[string]any{"isPinned": true},
		silent: true,
		client: fakeClient(srv.URL),
		out:    &strings.Builder{},
		in:     strings.NewReader(""),
	}
	if err := runEdit(context.Background(), o, "5"); err != nil {
		t.Fatalf("runEdit: %v", err)
	}
	if cap.query != "silent=true" {
		t.Errorf("query = %q", cap.query)
	}
}

func TestEditReadsStdinWhenNoFlags(t *testing.T) {
	cap := &editCapture{}
	srv := editServer(t, `{"result":"success","comment":{"id":5,"task":{"id":1}}}`, cap)
	defer srv.Close()

	o := &editOptions{
		body:   map[string]any{},
		client: fakeClient(srv.URL),
		out:    &strings.Builder{},
		in:     strings.NewReader("piped text\n"),
	}
	if err := runEdit(context.Background(), o, "5"); err != nil {
		t.Fatalf("runEdit: %v", err)
	}
	if cap.body["description"] != "piped text" {
		t.Errorf("body = %v", cap.body)
	}
}

func TestEditErrorsWithNothingToChange(t *testing.T) {
	o := &editOptions{
		body:   map[string]any{},
		client: fakeClient("http://unused"),
		out:    &strings.Builder{},
		in:     strings.NewReader(""),
	}
	if err := runEdit(context.Background(), o, "5"); err == nil {
		t.Fatal("want error when nothing was passed")
	}
}

func TestEditQuiet(t *testing.T) {
	cap := &editCapture{}
	srv := editServer(t, `{"result":"success","comment":{"id":5,"task":{"id":1}}}`, cap)
	defer srv.Close()

	out := &strings.Builder{}
	o := &editOptions{
		body:   map[string]any{"description": "x"},
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
		in:     strings.NewReader(""),
	}
	if err := runEdit(context.Background(), o, "5"); err != nil {
		t.Fatalf("runEdit: %v", err)
	}
	if out.String() != "5\n" {
		t.Errorf("quiet output = %q", out.String())
	}
}

// Only flags the user actually set may reach the request body — the API applies
// partial updates, so an unset flag must not clobber the stored value.
func TestEditBodyOnlyIncludesChangedFlags(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		pinned  bool
		hidden  bool
		changed map[string]bool
		want    map[string]any
	}{
		{
			name: "body only", body: "hello",
			changed: map[string]bool{"body": true},
			want:    map[string]any{"description": "hello"},
		},
		{
			name: "pin only", pinned: true,
			changed: map[string]bool{"pinned": true},
			want:    map[string]any{"isPinned": true},
		},
		{
			name: "explicit unpin", pinned: false,
			changed: map[string]bool{"pinned": true},
			want:    map[string]any{"isPinned": false},
		},
		{
			name: "body and hidden", body: "t", hidden: true,
			changed: map[string]bool{"body": true, "hidden": true},
			want:    map[string]any{"description": "t", "isHidden": true},
		},
		{
			name:    "nothing",
			changed: map[string]bool{},
			want:    map[string]any{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := editBody(tc.body, tc.pinned, tc.hidden, func(f string) bool { return tc.changed[f] })
			if len(got) != len(tc.want) {
				t.Fatalf("body = %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("body[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

// TestEditBodyFromCommandLine drives the real `comment edit` flags through
// Cobra's parser rather than a hand-rolled changed map, pinning that
// editBody's string literals ("body", "pinned", "hidden") match the flag
// names newEditCmd actually registers. A mismatch here would be a silent
// no-op that, for --body, falls through to a stdin read — which hangs on a
// terminal.
func TestEditBodyFromCommandLine(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want map[string]any
	}{
		{"body only", []string{"--body", "Corrected"}, map[string]any{"description": "Corrected"}},
		{"pin only", []string{"--pinned"}, map[string]any{"isPinned": true}},
		{"explicit unpin", []string{"--pinned=false"}, map[string]any{"isPinned": false}},
		{"body and hidden", []string{"--body", "t", "--hidden"}, map[string]any{"description": "t", "isHidden": true}},
		{"nothing", nil, map[string]any{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := &cmdutil.GlobalOpts{}
			cmd := newEditCmd(g)
			if err := cmd.Flags().Parse(tc.args); err != nil {
				t.Fatalf("parse: %v", err)
			}
			body, err := cmd.Flags().GetString("body")
			if err != nil {
				t.Fatalf("GetString(body): %v", err)
			}
			pinned, err := cmd.Flags().GetBool("pinned")
			if err != nil {
				t.Fatalf("GetBool(pinned): %v", err)
			}
			hidden, err := cmd.Flags().GetBool("hidden")
			if err != nil {
				t.Fatalf("GetBool(hidden): %v", err)
			}
			got := editBody(body, pinned, hidden, cmd.Flags().Changed)
			if len(got) != len(tc.want) {
				t.Fatalf("editBody = %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("editBody[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
