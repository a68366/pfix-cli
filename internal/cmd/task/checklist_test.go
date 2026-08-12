package task

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- checklist list tests ---

func TestRunChecklistListDefaultTable(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","items":[`+
			`{"id":29146,"name":"waba","isDone":true},`+
			`{"id":29149,"name":"amocrm","isDone":false}`+
			`]}`)
	}))
	defer srv.Close()

	out, errOut := &strings.Builder{}, &strings.Builder{}
	o := &checklistListOptions{
		taskID: 29141,
		limit:  100,
		client: fakeClient(srv.URL),
		out:    out,
		errOut: errOut,
	}
	if err := runChecklistList(context.Background(), o); err != nil {
		t.Fatalf("runChecklistList: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/29141/checklist/list" {
		t.Errorf("path = %q, want /task/29141/checklist/list", gotPath)
	}
	for _, want := range []string{`"pageSize":100`, `"offset":0`, `"fields":"id,name,isDone"`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("body %q missing %s", gotBody, want)
		}
	}
	result := out.String()
	for _, want := range []string{"ID", "DONE", "NAME", "29146", "waba", "amocrm", "true", "false"} {
		if !strings.Contains(result, want) {
			t.Errorf("output missing %q: %q", want, result)
		}
	}
	if errOut.String() != "" {
		t.Errorf("unexpected stderr: %q", errOut.String())
	}
}

func TestRunChecklistListPagingAndQuiet(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","items":[{"id":1,"name":"x","isDone":false}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistListOptions{
		taskID: 7,
		limit:  25,
		offset: 50,
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
		errOut: &strings.Builder{},
	}
	if err := runChecklistList(context.Background(), o); err != nil {
		t.Fatalf("runChecklistList: %v", err)
	}
	if !strings.Contains(gotBody, `"pageSize":25`) || !strings.Contains(gotBody, `"offset":50`) {
		t.Errorf("body = %q, want pageSize 25 / offset 50", gotBody)
	}
	if strings.Contains(out.String(), "NAME") {
		t.Errorf("quiet output should drop the header row: %q", out.String())
	}
}

func TestRunChecklistListFieldsOverride(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","items":[{"id":1,"name":"x","dateTime":{"datetime":"2026-08-06 10:00"}}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistListOptions{
		taskID: 7,
		limit:  100,
		fields: "id,name,dateTime",
		client: fakeClient(srv.URL),
		out:    out,
		errOut: &strings.Builder{},
	}
	if err := runChecklistList(context.Background(), o); err != nil {
		t.Fatalf("runChecklistList: %v", err)
	}
	if !strings.Contains(gotBody, `"fields":"id,name,dateTime"`) {
		t.Errorf("body = %q, want the overridden fields", gotBody)
	}
	result := out.String()
	if !strings.Contains(result, "DATETIME") {
		t.Errorf("output missing the derived DATETIME column: %q", result)
	}
	if strings.Contains(result, "DONE") {
		t.Errorf("output should not keep the default DONE column: %q", result)
	}
}

func TestRunChecklistListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","items":[{"id":1,"name":"x","isDone":true}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistListOptions{
		taskID: 7,
		limit:  100,
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
		errOut: &strings.Builder{},
	}
	if err := runChecklistList(context.Background(), o); err != nil {
		t.Fatalf("runChecklistList: %v", err)
	}
	if !strings.Contains(out.String(), `"items"`) {
		t.Errorf("json output missing the items envelope: %q", out.String())
	}
}

func TestRunChecklistListEmptyNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out, errOut := &strings.Builder{}, &strings.Builder{}
	o := &checklistListOptions{
		taskID: 42,
		limit:  100,
		client: fakeClient(srv.URL),
		out:    out,
		errOut: errOut,
	}
	if err := runChecklistList(context.Background(), o); err != nil {
		t.Fatalf("runChecklistList: %v", err)
	}
	if !strings.Contains(errOut.String(), "task 42 has no checklist items") {
		t.Errorf("stderr = %q, want the empty-checklist note", errOut.String())
	}
}

// --- checklist view tests ---

func TestRunChecklistViewDetail(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.Query().Get("fields")
		io.WriteString(w, `{"result":"success","item":{"id":29149,"name":"amocrm","isDone":false,`+
			`"dateTime":{"datetime":"2026-08-06 10:00"},"assignees":{"users":[{"id":"user:1","name":"Ann"}],"groups":[]}}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistViewOptions{
		taskID: 29141,
		itemID: 29149,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistView(context.Background(), o); err != nil {
		t.Fatalf("runChecklistView: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/task/29141/checklist/29149" {
		t.Errorf("path = %q, want /task/29141/checklist/29149", gotPath)
	}
	if gotQuery != "id,name,isDone,dateTime,assignees" {
		t.Errorf("fields = %q, want the view defaults", gotQuery)
	}
	result := out.String()
	for _, want := range []string{"ID", "29149", "NAME", "amocrm", "DONE", "false", "CREATED", "2026-08-06 10:00"} {
		if !strings.Contains(result, want) {
			t.Errorf("detail missing %q: %q", want, result)
		}
	}
}

func TestRunChecklistViewFieldsOverride(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("fields")
		io.WriteString(w, `{"result":"success","item":{"id":5,"name":"only name"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistViewOptions{
		taskID: 1,
		itemID: 5,
		fields: "id,name",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistView(context.Background(), o); err != nil {
		t.Fatalf("runChecklistView: %v", err)
	}
	if gotQuery != "id,name" {
		t.Errorf("fields = %q, want id,name", gotQuery)
	}
	if strings.Contains(out.String(), "DONE") {
		t.Errorf("detail should drop the DONE row when not requested: %q", out.String())
	}
}

func TestRunChecklistViewJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","item":{"id":5,"name":"x","assignees":{"users":[{"id":"user:1"}],"groups":[]}}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistViewOptions{
		taskID: 1,
		itemID: 5,
		json:   true,
		jq:     "",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistView(context.Background(), o); err != nil {
		t.Fatalf("runChecklistView: %v", err)
	}
	if !strings.Contains(out.String(), `"assignees"`) {
		t.Errorf("json output missing assignees: %q", out.String())
	}
}

func TestRunChecklistListEmptyQuietNoNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","items":[]}`)
	}))
	defer srv.Close()

	errOut := &strings.Builder{}
	o := &checklistListOptions{
		taskID: 42,
		limit:  100,
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    &strings.Builder{},
		errOut: errOut,
	}
	if err := runChecklistList(context.Background(), o); err != nil {
		t.Fatalf("runChecklistList: %v", err)
	}
	if errOut.String() != "" {
		t.Errorf("quiet stderr = %q, want no note", errOut.String())
	}
}

// --- checklist body assembly tests ---

func TestChecklistBody(t *testing.T) {
	cases := []struct {
		name      string
		itemName  string
		done      bool
		assignees []string
		set       []string
		want      string
		wantErr   bool
	}{
		{name: "name only", itemName: "amocrm", set: []string{"name"}, want: `{"name":"amocrm"}`},
		{name: "done true", done: true, set: []string{"done"}, want: `{"isDone":true}`},
		{name: "done false is still sent", done: false, set: []string{"done"}, want: `{"isDone":false}`},
		{name: "unset flags are omitted", itemName: "x", done: true, want: `{}`},
		{
			name:      "assignees",
			assignees: []string{"user:5", "group:7"},
			set:       []string{"assignees"},
			want:      `{"assignees":{"groups":[{"id":7}],"users":[{"id":"user:5"}]}}`,
		},
		{name: "bad assignee ref", assignees: []string{"user5"}, set: []string{"assignees"}, wantErr: true},
		// --assignees="" parses as a zero-length slice, and the list is replaced
		// wholesale, so accepting it would silently clear the item's people.
		{name: "empty assignees list", assignees: []string{}, set: []string{"assignees"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := func(f string) bool {
				for _, s := range tc.set {
					if s == f {
						return true
					}
				}
				return false
			}
			got, err := checklistBody(tc.itemName, tc.done, tc.assignees, changed)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("checklistBody(%v) = %v, want error", tc.assignees, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("checklistBody: %v", err)
			}
			b, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tc.want {
				t.Errorf("body = %s, want %s", b, tc.want)
			}
		})
	}
}

// --- checklist add tests ---

func TestRunChecklistAdd(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","id":29151}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistAddOptions{
		taskID: 29141,
		body:   map[string]any{"name": "new item", "isDone": true},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistAdd(context.Background(), o); err != nil {
		t.Fatalf("runChecklistAdd: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/29141/checklist" {
		t.Errorf("path = %q, want /task/29141/checklist", gotPath)
	}
	if !strings.Contains(gotBody, `"name":"new item"`) || !strings.Contains(gotBody, `"isDone":true`) {
		t.Errorf("body = %q, want name and isDone", gotBody)
	}
	if !strings.Contains(out.String(), "29151") {
		t.Errorf("output = %q, want the new item id", out.String())
	}
}

func TestRunChecklistAddQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":29151}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistAddOptions{
		taskID: 1,
		body:   map[string]any{"name": "x"},
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistAdd(context.Background(), o); err != nil {
		t.Fatalf("runChecklistAdd: %v", err)
	}
	if out.String() != "29151\n" {
		t.Errorf("quiet output = %q, want just the id", out.String())
	}
}

// The create response has no documented schema. When it carries no id, say so
// plainly rather than reporting a fabricated 0.
func TestRunChecklistAddWithoutID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistAddOptions{
		taskID: 7,
		body:   map[string]any{"name": "x"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistAdd(context.Background(), o); err != nil {
		t.Fatalf("runChecklistAdd: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "0") {
		t.Errorf("output = %q, should not print a zero id", got)
	}
	if !strings.Contains(got, "task 7") {
		t.Errorf("output = %q, want the task named", got)
	}

	quiet := &strings.Builder{}
	o.out, o.quiet = quiet, true
	if err := runChecklistAdd(context.Background(), o); err != nil {
		t.Fatalf("runChecklistAdd: %v", err)
	}
	if quiet.String() != "" {
		t.Errorf("quiet output = %q, want nothing when the API returns no id", quiet.String())
	}
}

func TestRunChecklistAddJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":8}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistAddOptions{
		taskID: 1,
		body:   map[string]any{"name": "x"},
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistAdd(context.Background(), o); err != nil {
		t.Fatalf("runChecklistAdd: %v", err)
	}
	if !strings.Contains(out.String(), `"result": "success"`) {
		t.Errorf("json output = %q, want the raw response", out.String())
	}
}

// --- checklist update tests ---

func TestRunChecklistUpdate(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistUpdateOptions{
		taskID: 29141,
		itemID: 29149,
		body:   map[string]any{"isDone": false},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistUpdate(context.Background(), o); err != nil {
		t.Fatalf("runChecklistUpdate: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/29141/checklist/29149" {
		t.Errorf("path = %q, want /task/29141/checklist/29149", gotPath)
	}
	if gotBody != `{"isDone":false}` {
		t.Errorf("body = %q, want only the flag that was set", gotBody)
	}
	if !strings.Contains(out.String(), "29149") {
		t.Errorf("output = %q, want the item id", out.String())
	}
}

func TestRunChecklistUpdateNothingToChange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be made when nothing was set")
	}))
	defer srv.Close()

	o := &checklistUpdateOptions{
		taskID: 1,
		itemID: 2,
		body:   map[string]any{},
		client: fakeClient(srv.URL),
		out:    &strings.Builder{},
	}
	err := runChecklistUpdate(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "nothing to change") {
		t.Fatalf("err = %v, want a nothing-to-change error", err)
	}
}

// The update endpoint reports per-field rejections in a failures array while
// still answering 200, so a non-empty failures list must not read as success.
func TestRunChecklistUpdateFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","failures":[{"field":"assignees","error":"user not found"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistUpdateOptions{
		taskID: 1,
		itemID: 2,
		body:   map[string]any{"assignees": map[string]any{}},
		client: fakeClient(srv.URL),
		out:    out,
	}
	err := runChecklistUpdate(context.Background(), o)
	if err == nil {
		t.Fatalf("err = nil, want the failures reported as an error")
	}
	for _, want := range []string{"assignees", "user not found"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want it to mention %q", err, want)
		}
	}
	if out.String() != "" {
		t.Errorf("output = %q, want no success line", out.String())
	}
}

// Under --json the response still passes through unmodified, and the failure
// then sets the exit code — output and error, not one or the other.
func TestRunChecklistUpdateFailuresJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","failures":[{"field":"name","error":"too long"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistUpdateOptions{
		taskID: 1,
		itemID: 2,
		body:   map[string]any{"name": "x"},
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	err := runChecklistUpdate(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Fatalf("err = %v, want the failure reported", err)
	}
	if !strings.Contains(out.String(), `"failures"`) {
		t.Errorf("json output = %q, want the raw response passed through", out.String())
	}
}

// The create response is undocumented, so the same failures contract is applied
// to add: a rejected create must not print as a success.
func TestRunChecklistAddFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","failures":[{"field":"assignees","error":"user not found"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistAddOptions{
		taskID: 1,
		body:   map[string]any{"name": "x"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	err := runChecklistAdd(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("err = %v, want the failure reported", err)
	}
	if out.String() != "" {
		t.Errorf("output = %q, want no success line", out.String())
	}
}

func TestRunChecklistUpdateQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &checklistUpdateOptions{
		taskID: 1,
		itemID: 2,
		body:   map[string]any{"name": "x"},
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runChecklistUpdate(context.Background(), o); err != nil {
		t.Fatalf("runChecklistUpdate: %v", err)
	}
	if out.String() != "2\n" {
		t.Errorf("quiet output = %q, want just the item id", out.String())
	}
}
