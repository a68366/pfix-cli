package task

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusesByProcess(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"result":"success","statuses":[{"id":0,"name":"Draft","isActive":false},{"id":1,"name":"New","isActive":true}]}`)
	}))
	defer srv.Close()

	var buf, errBuf strings.Builder
	o := &statusesOptions{process: 40720, client: fakeClient(srv.URL), out: &buf, errOut: &errBuf}
	if err := runStatuses(context.Background(), o); err != nil {
		t.Fatalf("runStatuses: %v", err)
	}
	if gotPath != "/process/task/40720/statuses" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "fields=id%2Cname%2CisActive") && !strings.Contains(gotQuery, "fields=id,name,isActive") {
		t.Errorf("query = %q", gotQuery)
	}
	if !strings.Contains(buf.String(), "Draft") || !strings.Contains(buf.String(), "New") {
		t.Errorf("output missing rows: %q", buf.String())
	}
	for _, hdr := range []string{"ID", "NAME", "ACTIVE"} {
		if !strings.Contains(buf.String(), hdr) {
			t.Errorf("output missing header %q: %q", hdr, buf.String())
		}
	}
}

func TestStatusesByTaskResolvesProcessId(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/task/24196":
			io.WriteString(w, `{"result":"success","task":{"id":24196,"processId":234106}}`)
		case "/process/task/234106/statuses":
			io.WriteString(w, `{"result":"success","statuses":[{"id":2,"name":"In progress","isActive":true}]}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	var buf, errBuf strings.Builder
	o := &statusesOptions{taskID: "24196", client: fakeClient(srv.URL), out: &buf, errOut: &errBuf}
	if err := runStatuses(context.Background(), o); err != nil {
		t.Fatalf("runStatuses: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/task/24196" || paths[1] != "/process/task/234106/statuses" {
		t.Errorf("paths = %v", paths)
	}
	if !strings.Contains(buf.String(), "In progress") {
		t.Errorf("output missing row: %q", buf.String())
	}
}

func TestStatusesNeitherArg(t *testing.T) {
	o := &statusesOptions{client: fakeClient("http://unused"), out: io.Discard, errOut: io.Discard}
	err := runStatuses(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "provide a task id or --process") {
		t.Fatalf("err = %v, want neither-arg error", err)
	}
}

func TestStatusesBothArgs(t *testing.T) {
	o := &statusesOptions{taskID: "5", process: 40720, client: fakeClient("http://unused"), out: io.Discard, errOut: io.Discard}
	err := runStatuses(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("err = %v, want mutual-exclusion error", err)
	}
}

func TestStatusesMissingProcessId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","task":{"id":5}}`)
	}))
	defer srv.Close()

	o := &statusesOptions{taskID: "5", client: fakeClient(srv.URL), out: io.Discard, errOut: io.Discard}
	err := runStatuses(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "could not determine the process for task 5") {
		t.Fatalf("err = %v, want missing-processId error", err)
	}
}

func TestStatusesEmptyNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","statuses":[]}`)
	}))
	defer srv.Close()

	var buf, errBuf strings.Builder
	o := &statusesOptions{process: 999999, client: fakeClient(srv.URL), out: &buf, errOut: &errBuf}
	if err := runStatuses(context.Background(), o); err != nil {
		t.Fatalf("runStatuses: %v", err)
	}
	if !strings.Contains(errBuf.String(), "process 999999 has no statuses") {
		t.Errorf("stderr note missing: %q", errBuf.String())
	}
}

func TestStatusesInvalidTaskID(t *testing.T) {
	o := &statusesOptions{taskID: "abc", client: fakeClient("http://unused"), out: io.Discard, errOut: io.Discard}
	if err := runStatuses(context.Background(), o); err == nil {
		t.Fatalf("want error for non-numeric task id")
	}
}

func TestStatusesNegativeProcess(t *testing.T) {
	o := &statusesOptions{process: -5, client: fakeClient("http://unused"), out: io.Discard, errOut: io.Discard}
	err := runStatuses(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "--process must be a positive number") {
		t.Fatalf("err = %v, want negative-process error", err)
	}
}
