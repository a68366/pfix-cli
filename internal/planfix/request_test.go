package planfix

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	c := New("example.test", "tok")
	c.BaseURL = srv.URL
	return c
}

func TestJSON_SuccessReturnsRawBytes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
		io.WriteString(w, `{"result":"success","id":7}`)
	}))
	defer srv.Close()

	raw, err := newTestClient(srv).JSON(context.Background(), "POST", "task/", map[string]any{"name": "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != "POST" || gotPath != "/task/" {
		t.Fatalf("method/path = %s %s", gotMethod, gotPath)
	}
	var sent map[string]any
	if json.Unmarshal(gotBody, &sent); sent["name"] != "x" {
		t.Fatalf("body not marshaled: %s", gotBody)
	}
	if string(raw) != `{"result":"success","id":7}` {
		t.Fatalf("raw = %s", raw)
	}
}

func TestJSON_NilBodySendsNoBody(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		n = len(b)
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	if _, err := newTestClient(srv).JSON(context.Background(), "GET", "task/1", nil); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected empty body, got %d bytes", n)
	}
}

func TestJSON_ErrorStatusReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"result":"fail","code":1000,"error":"Task not found by id - 9"}`)
	}))
	defer srv.Close()
	_, err := newTestClient(srv).JSON(context.Background(), "GET", "task/9", nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 || apiErr.Code != 1000 || apiErr.Message != "Task not found by id - 9" {
		t.Fatalf("got %+v", apiErr)
	}
}
