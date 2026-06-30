package task

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
		io.WriteString(w, `{"result":"success","tasks":[{"id":1,"name":"Fix bug","status":{"name":"New"},"priority":"high","dateTime":{"datetime":"2024-01-15"}},{"id":2,"name":"Write docs","status":{"name":"In Progress"},"priority":"normal","dateTime":{"datetime":"2024-01-16"}}]}`)
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
	if !strings.Contains(gotBody, `"fields":"id,name,status,priority,dateTime"`) {
		t.Errorf("body missing fields: %q", gotBody)
	}
	result := out.String()
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID header: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME header: %q", result)
	}
	if !strings.Contains(result, "New") {
		t.Errorf("output missing status New: %q", result)
	}
	if !strings.Contains(result, "Fix bug") {
		t.Errorf("output missing task name: %q", result)
	}
}

func TestRunListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","tasks":[]}`)
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
		io.WriteString(w, `{"result":"success","tasks":[]}`)
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
}

func TestRunListQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","tasks":[{"id":1,"name":"Task 1","status":{"name":"New"}}]}`)
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
	if !strings.Contains(result, "Task 1") {
		t.Errorf("quiet mode should still show data: %q", result)
	}
}

func TestRunListFilter(t *testing.T) {
	t.Run("valid filter forwarded in body", func(t *testing.T) {
		var gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			io.WriteString(w, `{"result":"success","tasks":[]}`)
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
			io.WriteString(w, `{"result":"success","tasks":[]}`)
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
