package object

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunListDefaultTable(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		// Two objects with FAT status to prove status.name is used.
		io.WriteString(w, `{"result":"success","objects":[{"id":1,"name":"Widget","status":{"id":1,"name":"New"},"priority":"Normal"},{"id":2,"name":"Gadget","status":{"id":2,"name":"Open"},"priority":"Low"}]}`)
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
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/object/list" {
		t.Errorf("path = %q, want /object/list", gotPath)
	}
	if !strings.Contains(gotBody, `"pageSize":25`) {
		t.Errorf("body missing pageSize 25: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"offset":0`) {
		t.Errorf("body missing offset 0: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"fields":"id,name,status,priority"`) {
		t.Errorf("body missing default fields: %q", gotBody)
	}
	result := out.String()
	for _, hdr := range []string{"ID", "NAME", "STATUS", "PRIORITY"} {
		if !strings.Contains(result, hdr) {
			t.Errorf("output missing header %q: %q", hdr, result)
		}
	}
	if !strings.Contains(result, "Widget") {
		t.Errorf("output missing object name Widget: %q", result)
	}
	// Proves status.name path is used (not status.id which would be "1")
	if !strings.Contains(result, "New") {
		t.Errorf("output missing status name New: %q", result)
	}
}

func TestRunListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","objects":[]}`)
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

func TestRunListQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","objects":[{"id":1,"name":"Widget","status":{"id":1,"name":"New"},"priority":"Normal"}]}`)
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
	if !strings.Contains(result, "Widget") {
		t.Errorf("quiet mode should still show data: %q", result)
	}
}

func TestRunListCustomFields(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","objects":[{"id":1,"name":"Widget"}]}`)
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
	if strings.Contains(result, "STATUS") {
		t.Errorf("output should not contain STATUS column when not in fields override: %q", result)
	}
	if strings.Contains(result, "PRIORITY") {
		t.Errorf("output should not contain PRIORITY column when not in fields override: %q", result)
	}
}

func TestRunListFilter(t *testing.T) {
	t.Run("valid filter forwarded in body", func(t *testing.T) {
		var gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			io.WriteString(w, `{"result":"success","objects":[]}`)
		}))
		defer srv.Close()

		out := &strings.Builder{}
		o := &listOptions{
			limit:  100,
			filter: `[{"type":1,"operator":"equal","value":5}]`,
			client: fakeClient(srv.URL),
			out:    out,
		}
		if err := runList(context.Background(), o); err != nil {
			t.Fatalf("runList: %v", err)
		}
		if !strings.Contains(gotBody, `"filters":[{`) {
			t.Errorf("body missing filters: %q", gotBody)
		}
	})

	t.Run("invalid filter errors before HTTP", func(t *testing.T) {
		requestMade := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestMade = true
			io.WriteString(w, `{"result":"success","objects":[]}`)
		}))
		defer srv.Close()

		out := &strings.Builder{}
		o := &listOptions{
			limit:  100,
			filter: "nope",
			client: fakeClient(srv.URL),
			out:    out,
		}
		err := runList(context.Background(), o)
		if err == nil || !strings.Contains(err.Error(), "invalid --filter JSON") {
			t.Fatalf("err = %v, want invalid --filter JSON", err)
		}
		if requestMade {
			t.Error("HTTP request should not be made when filter is invalid")
		}
	})
}
