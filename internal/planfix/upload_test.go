package planfix_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/a68366/pfix-cli/internal/buildinfo"
	"github.com/a68366/pfix-cli/internal/planfix"
)

func TestUploadSendsOneMultipartFilePart(t *testing.T) {
	content := []byte{0x00, 0xff, '\r', '\n', 'x'} // binary, incl. CRLF
	tests := []struct {
		name     string
		filename string
	}{
		{"ascii", "report.pdf"},
		{"spaces and cyrillic", "отчёт за май.txt"},
		{"quote", `a"b.txt`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotName, gotField, gotAuth, gotUA, gotMethod, gotPath string
			var gotBytes []byte
			var parts int
			var gotLen int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth, gotUA = r.Header.Get("Authorization"), r.Header.Get("User-Agent")
				gotMethod, gotPath, gotLen = r.Method, r.URL.Path, r.ContentLength
				mr, err := r.MultipartReader()
				if err != nil {
					t.Errorf("multipart reader: %v", err)
					return
				}
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Errorf("next part: %v", err)
						return
					}
					parts++
					gotField, gotName = p.FormName(), p.FileName()
					gotBytes, _ = io.ReadAll(p)
				}
				io.WriteString(w, `{"result":"success","id":124}`)
			}))
			defer srv.Close()

			raw, err := fastClient(srv.URL).Upload(context.Background(), "file/", tt.filename, bytes.NewReader(content), int64(len(content)))
			if err != nil {
				t.Fatalf("Upload: %v", err)
			}
			if string(raw) != `{"result":"success","id":124}` {
				t.Errorf("raw = %s", raw)
			}
			if gotMethod != "POST" || gotPath != "/file/" {
				t.Errorf("request = %s %s, want POST /file/", gotMethod, gotPath)
			}
			if gotLen <= int64(len(content)) {
				t.Errorf("Content-Length = %d, want an exact length (not chunked)", gotLen)
			}
			if parts != 1 || gotField != "file" {
				t.Errorf("parts = %d, field = %q; want one part named file", parts, gotField)
			}
			if gotName != tt.filename {
				t.Errorf("filename = %q, want %q", gotName, tt.filename)
			}
			if !bytes.Equal(gotBytes, content) {
				t.Errorf("bytes = %v, want %v", gotBytes, content)
			}
			if gotAuth != "Bearer secret" {
				t.Errorf("auth = %q", gotAuth)
			}
			if want := "pfix/" + buildinfo.Version; gotUA != want {
				t.Errorf("User-Agent = %q, want %q", gotUA, want)
			}
		})
	}
}

func TestUploadMakesExactlyOneAttempt(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"result":"fail","code":0,"error":"Rest API error"}`)
	}))
	defer srv.Close()

	_, err := fastClient(srv.URL).Upload(context.Background(), "file/", "a.txt", strings.NewReader("abc"), 3)
	var apiErr *planfix.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
		t.Fatalf("err = %v, want a 500 APIError", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1 (no retry)", got)
	}
}

func TestUploadShortContentFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":1}`)
	}))
	defer srv.Close()

	// The file shrank between stat and read: the declared length cannot be met.
	_, err := fastClient(srv.URL).Upload(context.Background(), "file/", "a.txt", strings.NewReader("ab"), 10)
	if err == nil {
		t.Fatal("want error for content shorter than size")
	}
}

func TestUploadRoutesThroughProxy(t *testing.T) {
	var proxied int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&proxied, 1)
		_, _ = io.Copy(io.Discard, r.Body)
		io.WriteString(w, `{"result":"success","id":7}`)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}

	// Non-routable base: success proves Upload honors c.Proxy.
	c := fastClient("http://origin.invalid/rest")
	c.Proxy = func(*http.Request) (*url.URL, error) { return proxyURL, nil }
	if _, err := c.Upload(context.Background(), "file/", "a.txt", strings.NewReader("abc"), 3); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if got := atomic.LoadInt32(&proxied); got != 1 {
		t.Fatalf("proxy calls = %d, want 1", got)
	}
}

func TestJSONOnceDoesNotRetry5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := fastClient(srv.URL)
	if _, err := c.JSONOnce(context.Background(), "POST", "file/from-url/", map[string]string{"url": "x"}); err == nil {
		t.Fatal("want error on 500")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("JSONOnce calls = %d, want 1", got)
	}

	// Control: JSON keeps retrying through the same server.
	atomic.StoreInt32(&calls, 0)
	if _, err := c.JSON(context.Background(), "POST", "task/list", nil); err == nil {
		t.Fatal("want error on 500")
	}
	if got := atomic.LoadInt32(&calls); got != int32(c.Retries) {
		t.Fatalf("JSON calls = %d, want %d", got, c.Retries)
	}
}
