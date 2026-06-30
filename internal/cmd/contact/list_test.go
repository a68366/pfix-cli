package contact

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
		// IDs 4 and 7 do not overlap with email strings, so assertions are unambiguous.
		io.WriteString(w, `{"result":"success","contacts":[{"id":4,"name":"Alice","lastname":"Smith","email":"alice@example.com","isCompany":false},{"id":7,"name":"Bob","lastname":"Jones","email":"bob@example.com","isCompany":false}]}`)
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
	if !strings.Contains(gotBody, `"pageSize":25`) {
		t.Errorf("body missing pageSize 25: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"offset":0`) {
		t.Errorf("body missing offset 0: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"fields":"id,name,lastname,email,isCompany"`) {
		t.Errorf("body missing default fields: %q", gotBody)
	}
	result := out.String()
	for _, hdr := range []string{"ID", "NAME", "LASTNAME", "EMAIL", "COMPANY"} {
		if !strings.Contains(result, hdr) {
			t.Errorf("output missing header %q: %q", hdr, result)
		}
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("output missing contact name Alice: %q", result)
	}
	if !strings.Contains(result, "alice@example.com") {
		t.Errorf("output missing email alice@example.com: %q", result)
	}
}

func TestRunListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","contacts":[]}`)
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
		io.WriteString(w, `{"result":"success","contacts":[{"id":4,"name":"Alice","lastname":"Smith","email":"alice@example.com","isCompany":false}]}`)
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
	if !strings.Contains(result, "Alice") {
		t.Errorf("quiet mode should still show data: %q", result)
	}
}

func TestRunListCustomFields(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","contacts":[{"id":4,"name":"Alice"}]}`)
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
	if strings.Contains(result, "LASTNAME") {
		t.Errorf("output should not contain LASTNAME column when not in fields override: %q", result)
	}
	if strings.Contains(result, "EMAIL") {
		t.Errorf("output should not contain EMAIL column when not in fields override: %q", result)
	}
}
