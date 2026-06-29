package project

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunListDefaultTable(t *testing.T) {
	var gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		// Status IDs 10 and 11 do not overlap with project IDs 1 and 2,
		// so assertions on status values are unambiguous.
		io.WriteString(w, `{"result":"success","projects":[{"id":1,"name":"Alpha","owner":{"name":"Alice"},"status":{"id":10}},{"id":2,"name":"Beta","owner":{"name":"Bob"},"status":{"id":11}}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		limit:  100,
		offset: 0,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.Contains(gotBody, `"pageSize":100`) {
		t.Errorf("body missing pageSize: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"fields":"id,name,description,owner,status"`) {
		t.Errorf("body missing fields: %q", gotBody)
	}
	result := out.String()
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID header: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME header: %q", result)
	}
	if !strings.Contains(result, "OWNER") {
		t.Errorf("output missing OWNER header: %q", result)
	}
	if !strings.Contains(result, "STATUS") {
		t.Errorf("output missing STATUS header: %q", result)
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("output missing owner name Alice: %q", result)
	}
	// Status ID 10 appears only in the STATUS column (project IDs are 1 and 2).
	if !strings.Contains(result, "10") {
		t.Errorf("output missing status id 10: %q", result)
	}
	if !strings.Contains(result, "Alpha") {
		t.Errorf("output missing project name Alpha: %q", result)
	}
}

func TestRunListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","projects":[]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		limit:  100,
		offset: 0,
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) {
		t.Errorf("json output missing result field: %q", result)
	}
	if !strings.Contains(result, `"success"`) {
		t.Errorf("json output missing success value: %q", result)
	}
}

func TestRunListCustomLimit(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","projects":[]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		limit:  25,
		offset: 0,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(gotBody, `"pageSize":25`) {
		t.Errorf("body missing pageSize 25: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"offset":0`) {
		t.Errorf("body missing offset 0: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"fields":"id,name,description,owner,status"`) {
		t.Errorf("body missing default fields: %q", gotBody)
	}
}

func TestRunListQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","projects":[{"id":1,"name":"Project 1","owner":{"name":"Alice"},"status":{"id":1}}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		limit:  100,
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	result := out.String()
	if strings.Contains(result, "ID") || strings.Contains(result, "NAME") {
		t.Errorf("quiet mode should not show headers: %q", result)
	}
	if !strings.Contains(result, "Project 1") {
		t.Errorf("quiet mode should still show data: %q", result)
	}
}

func TestRunListCustomFields(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","projects":[{"id":5,"name":"Gamma"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		limit:  100,
		offset: 0,
		fields: "id,name",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(gotBody, `"fields":"id,name"`) {
		t.Errorf("body missing custom fields: %q", gotBody)
	}
	result := out.String()
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID column: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME column: %q", result)
	}
	if strings.Contains(result, "OWNER") {
		t.Errorf("output should not contain OWNER column when not in fields override: %q", result)
	}
	if strings.Contains(result, "STATUS") {
		t.Errorf("output should not contain STATUS column when not in fields override: %q", result)
	}
}
