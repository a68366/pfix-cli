package task

import (
	"context"
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
