package task

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// --- checklist delete tests ---

func TestRunChecklistDeleteRefusesBadInput(t *testing.T) {
	tests := []struct {
		name    string
		force   bool
		taskID  string
		itemID  string
		wantErr string
	}{
		{"no force", false, "12", "13", "--force"},
		{"bad task id", true, "abc", "13", "abc"},
		{"zero task id", true, "0", "13", "0"},
		{"bad item id", true, "12", "x", "x"},
		{"negative item id", true, "12", "-3", "-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hit = true
			}))
			defer srv.Close()

			o := &checklistDeleteOptions{force: tt.force, client: fakeClient(srv.URL), out: &strings.Builder{}}
			err := runChecklistDelete(context.Background(), o, tt.taskID, tt.itemID)
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

func TestRunChecklistDeleteOutput(t *testing.T) {
	tests := []struct {
		name  string
		quiet bool
		json  bool
		jq    string
		want  string
		exact bool
	}{
		{"human", false, false, "", "Deleted checklist item 13 from task 12\n", true},
		{"quiet", true, false, "", "13\n", true},
		{"json", false, true, "", `"result": "success"`, false},
		{"jq", false, true, ".result", "success\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				_, _ = io.WriteString(w, `{"result":"success"}`)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &checklistDeleteOptions{
				force: true, quiet: tt.quiet, json: tt.json, jq: tt.jq,
				client: fakeClient(srv.URL), out: out,
			}
			if err := runChecklistDelete(context.Background(), o, "12", "13"); err != nil {
				t.Fatalf("runChecklistDelete: %v", err)
			}
			if gotMethod != http.MethodDelete {
				t.Errorf("method = %q, want DELETE", gotMethod)
			}
			if gotPath != "/task/12/checklist/13" {
				t.Errorf("path = %q, want /task/12/checklist/13", gotPath)
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

func TestRunChecklistDeleteAPIErrors(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		want   []string
		reject []string
	}{
		{
			name: "unknown or already deleted item",
			body: `{"result":"fail","code":600,"error":"Checklist item not found by id - 13"}`,
			want: []string{"Checklist item not found by id - 13", "already deleted", "nested"},
		},
		{
			name: "item of another task",
			body: `{"result":"fail","code":6,"error":"Checklist item does not belong to task by id - 13"}`,
			want: []string{"does not belong to task", "pfix task checklist list"},
		},
		{
			name:   "unknown task",
			body:   `{"result":"fail","code":1000,"error":"Task not found by id - 12"}`,
			want:   []string{"Task not found by id - 12"},
			reject: []string{"already deleted", "checklist list"},
		},
		{
			name:   "scope denied passes through for the root to describe",
			body:   `{"result":"fail","code":5,"error":"Access denied"}`,
			want:   []string{"Access denied"},
			reject: []string{"already deleted", "checklist list"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &checklistDeleteOptions{force: true, client: fakeClient(srv.URL), out: out}
			err := runChecklistDelete(context.Background(), o, "12", "13")
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

func TestChecklistGroupRegistersDelete(t *testing.T) {
	cmd := newChecklistCmd(&cmdutil.GlobalOpts{})
	sub, _, err := cmd.Find([]string{"delete"})
	if err != nil || sub.Name() != "delete" {
		t.Fatalf("checklist delete not registered: %v", err)
	}
	if strings.Contains(cmd.Long, "no\ndelete") || strings.Contains(cmd.Long, "no delete") {
		t.Errorf("group help still claims there is no delete: %q", cmd.Long)
	}
	for _, w := range []string{"--force", "nested", "cannot be undone"} {
		if !strings.Contains(sub.Long, w) {
			t.Errorf("delete help missing %q: %q", w, sub.Long)
		}
	}
	if sub.Flags().Lookup("force") == nil {
		t.Error("delete has no --force flag")
	}
}
