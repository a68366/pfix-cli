package task

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

func TestRunCreateDefault(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":42}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body:   map[string]any{"name": "New task", "description": "A description"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/" {
		t.Errorf("path = %q, want /task/", gotPath)
	}
	if gotBody["name"] != "New task" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "New task")
	}
	if gotBody["description"] != "A description" {
		t.Errorf("body description = %v, want %q", gotBody["description"], "A description")
	}
	result := out.String()
	if result != "Created task 42\n" {
		t.Errorf("output = %q, want %q", result, "Created task 42\n")
	}
}

func TestRunCreateSendsBodyVerbatim(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":1}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body: map[string]any{
			"name":     "Full",
			"template": map[string]any{"id": 6},
			"assignees": map[string]any{
				"users":  []any{map[string]any{"id": "user:1"}},
				"groups": []any{},
			},
			"endDateTime": map[string]any{"date": "20-07-2026", "time": "18:00"},
		},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	tmpl, ok := gotBody["template"].(map[string]any)
	if !ok || tmpl["id"] != float64(6) {
		t.Errorf("body template = %#v, want id 6", gotBody["template"])
	}
	asg, ok := gotBody["assignees"].(map[string]any)
	if !ok {
		t.Fatalf("body assignees = %#v, want map", gotBody["assignees"])
	}
	users, ok := asg["users"].([]any)
	if !ok || len(users) != 1 || users[0].(map[string]any)["id"] != "user:1" {
		t.Errorf("assignees users = %#v, want [{id user:1}]", asg["users"])
	}
	end, ok := gotBody["endDateTime"].(map[string]any)
	if !ok || end["date"] != "20-07-2026" || end["time"] != "18:00" {
		t.Errorf("endDateTime = %#v, want 20-07-2026 18:00", gotBody["endDateTime"])
	}
}

func TestRunCreateQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":7}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body:   map[string]any{"name": "Task"},
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if out.String() != "7\n" {
		t.Errorf("quiet output = %q, want %q", out.String(), "7\n")
	}
}

func TestRunCreateJSON(t *testing.T) {
	raw := `{"result":"success","id":99}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, raw)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body:   map[string]any{"name": "Task"},
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) {
		t.Errorf("json output missing result field: %q", result)
	}
	if !strings.Contains(result, `"success"`) {
		t.Errorf("json output missing success value: %q", result)
	}
	if !strings.Contains(result, `"id"`) {
		t.Errorf("json output missing id field: %q", result)
	}
}

// TestCreateBodyFromCommandLine exercises the flags-to-body seam on the
// success path: field flags parsed from a real command line (including
// comma-split list flags) land in the body, and description is included
// only when non-empty.
func TestCreateBodyFromCommandLine(t *testing.T) {
	t.Run("field flags land in the body, empty description omitted", func(t *testing.T) {
		f := &taskFields{}
		cmd := &cobra.Command{}
		f.register(cmd, true)
		args := []string{"--project", "21", "--assignees", "user:1,group:3", "--priority", "urgent", "--end-date", "2026-07-20"}
		if err := cmd.Flags().Parse(args); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		body, err := createBody("Task", "", f, cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("createBody: %v", err)
		}
		want := map[string]any{
			"name":    "Task",
			"project": map[string]any{"id": 21},
			"assignees": map[string]any{
				"users":  []any{map[string]any{"id": "user:1"}},
				"groups": []any{map[string]any{"id": 3}},
			},
			"priority":    "Urgent",
			"endDateTime": map[string]any{"date": "20-07-2026"},
		}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body = %#v, want %#v", body, want)
		}
	})
	t.Run("description included when non-empty, no field flags", func(t *testing.T) {
		f := &taskFields{}
		cmd := &cobra.Command{}
		f.register(cmd, true)
		if err := cmd.Flags().Parse(nil); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		body, err := createBody("Task", "some text", f, cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("createBody: %v", err)
		}
		want := map[string]any{"name": "Task", "description": "some text"}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body = %#v, want %#v", body, want)
		}
	})
}

// TestCreateMissingName drives the Cobra command without --name and asserts the
// required-flag guard rejects it before RunE builds a client or sends a request.
func TestCreateMissingName(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newCreateCmd(g)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --name is missing")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention the missing 'name' flag, got: %q", err.Error())
	}
}

// TestCreateInvalidFieldFailsFast drives the Cobra command with an invalid
// field value and asserts the parse error surfaces before any client is built
// or request sent (the helper's message — not a config/transport error —
// proves where execution stopped).
func TestCreateInvalidFieldFailsFast(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "bad priority", args: []string{"--name", "x", "--priority", "sky-high"}, wantErr: "invalid priority"},
		{name: "bad people ref", args: []string{"--name", "x", "--assignees", "12"}, wantErr: "invalid people reference"},
		{name: "bad date", args: []string{"--name", "x", "--end-date", "tomorrow"}, wantErr: "invalid date"},
		{name: "zero template", args: []string{"--name", "x", "--template", "0"}, wantErr: "--template must be a positive number"},
		{name: "bad cf", args: []string{"--name", "x", "--cf", "abc"}, wantErr: "invalid --cf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &cmdutil.GlobalOpts{}
			cmd := newCreateCmd(g)
			cmd.SetArgs(tt.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// cfRouter serves the customfield defs GET and captures the task POST body.
func cfRouter(t *testing.T, defs string, gotBody *map[string]any, postSeen *bool) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/customfield/task":
			io.WriteString(w, defs)
		case r.Method == "POST":
			*postSeen = true
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, gotBody)
			io.WriteString(w, `{"result":"success","id":57}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}
}

func TestRunCreateWithCF(t *testing.T) {
	var gotBody map[string]any
	postSeen := false
	defs := `{"result":"success","customfields":[{"id":88206,"type":0},{"id":85984,"type":1},` +
		`{"id":88210,"name":"list","type":8,"enumValues":["1","2","3","four"]}]}`
	srv := httptest.NewServer(cfRouter(t, defs, &gotBody, &postSeen))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body: map[string]any{"name": "T"},
		cfSpecs: []cmdutil.CustomFieldSpec{
			{ID: 88206, Value: "hello"},
			{ID: 85984, Value: "42"},
			{ID: 88210, Value: "four"},
		},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	data, ok := gotBody["customFieldData"].([]any)
	if !ok || len(data) != 3 {
		t.Fatalf("customFieldData = %#v, want 3 entries", gotBody["customFieldData"])
	}
	// text -> string
	e0 := data[0].(map[string]any)
	if e0["field"].(map[string]any)["id"] != float64(88206) || e0["value"] != "hello" {
		t.Errorf("entry 0 = %#v", e0)
	}
	// number -> JSON number
	if data[1].(map[string]any)["value"] != float64(42) {
		t.Errorf("entry 1 value = %#v, want 42", data[1])
	}
	// list -> the option label, as a bare string
	e2 := data[2].(map[string]any)
	if e2["value"] != "four" {
		t.Errorf("entry 2 = %#v, want value \"four\"", e2)
	}
}

func TestRunCreateCFUnknownID(t *testing.T) {
	var gotBody map[string]any
	postSeen := false
	defs := `{"result":"success","customfields":[{"id":88206,"type":0}]}`
	srv := httptest.NewServer(cfRouter(t, defs, &gotBody, &postSeen))
	defer srv.Close()

	o := &createOptions{
		body:    map[string]any{"name": "T"},
		cfSpecs: []cmdutil.CustomFieldSpec{{ID: 999, Value: "x"}},
		client:  fakeClient(srv.URL),
		out:     io.Discard,
	}
	err := runCreate(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "no custom field 999 for task") {
		t.Fatalf("err = %v, want unknown-id error", err)
	}
	if postSeen {
		t.Error("no task POST should be sent when a cf id is unknown")
	}
}

func TestCreateCFCommaPreserved(t *testing.T) {
	f := &taskFields{}
	cmd := &cobra.Command{}
	f.register(cmd, true)
	if err := cmd.Flags().Parse([]string{"--cf", "88206=a, b"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	specs, err := f.customFieldSpecs(cmd.Flags().Changed)
	if err != nil {
		t.Fatalf("customFieldSpecs: %v", err)
	}
	want := []cmdutil.CustomFieldSpec{{ID: 88206, Value: "a, b"}}
	if !reflect.DeepEqual(specs, want) {
		t.Errorf("specs = %#v, want %#v", specs, want)
	}
}
