package customfield

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunListDefaultTable(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"result":"success","customfields":[{"id":85984,"name":"Sum","type":1},{"id":85986,"name":"Date","type":0}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		objectType: "task",
		client:     fakeClient(srv.URL),
		out:        out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/customfield/task" {
		t.Errorf("path = %q, want /customfield/task", gotPath)
	}
	if !strings.Contains(gotQuery, "fields=id%2Cname%2Ctype") && !strings.Contains(gotQuery, "fields=id,name,type") {
		t.Errorf("query missing fields=id,name,type: %q", gotQuery)
	}
	result := out.String()
	for _, hdr := range []string{"ID", "NAME", "TYPE"} {
		if !strings.Contains(result, hdr) {
			t.Errorf("output missing header %q: %q", hdr, result)
		}
	}
	if !strings.Contains(result, "Sum") {
		t.Errorf("output missing field name Sum: %q", result)
	}
	if !strings.Contains(result, "Date") {
		t.Errorf("output missing field name Date: %q", result)
	}
}

func TestRunListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","customfields":[{"id":85984,"name":"Sum","type":1}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		objectType: "task",
		json:       true,
		client:     fakeClient(srv.URL),
		out:        out,
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

func TestRunListQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","customfields":[{"id":85984,"name":"Sum","type":1}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		objectType: "task",
		quiet:      true,
		client:     fakeClient(srv.URL),
		out:        out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	result := out.String()
	if strings.Contains(result, "ID") || strings.Contains(result, "NAME") || strings.Contains(result, "TYPE") {
		t.Errorf("quiet mode should not show headers: %q", result)
	}
	if !strings.Contains(result, "Sum") {
		t.Errorf("quiet mode should still show data: %q", result)
	}
}

func TestRunListCustomFields(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"result":"success","customfields":[{"id":85984,"name":"Sum","type":1}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		objectType: "task",
		fields:     "id",
		client:     fakeClient(srv.URL),
		out:        out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if gotQuery != "fields=id" {
		t.Errorf("query = %q, want fields=id", gotQuery)
	}
	result := out.String()
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID column: %q", result)
	}
	if strings.Contains(result, "NAME") {
		t.Errorf("output should not contain NAME column when not in fields override: %q", result)
	}
}

func TestRunListInvalidType(t *testing.T) {
	tests := []struct {
		name       string
		objectType string
	}{
		{name: "empty", objectType: ""},
		{name: "uppercase", objectType: "Bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestSent := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestSent = true
				io.WriteString(w, `{"result":"success","customfields":[]}`)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &listOptions{
				objectType: tt.objectType,
				client:     fakeClient(srv.URL),
				out:        out,
			}
			err := runList(context.Background(), o)
			if err == nil {
				t.Fatalf("runList(%q) expected error, got nil", tt.objectType)
			}
			if requestSent {
				t.Errorf("runList(%q) sent HTTP request before validating type", tt.objectType)
			}
		})
	}
}

func TestRunListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","customfields":[]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{
		objectType: "task",
		client:     fakeClient(srv.URL),
		out:        out,
	}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	result := out.String()
	// Header should still be present for non-quiet empty result
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing header ID for empty result: %q", result)
	}
}
