package files

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/planfix"
)

func fakeClient(srvURL string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = srvURL
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}

func baseOpts(srvURL string, opts Options, out, errOut io.Writer) *listOptions {
	return &listOptions{
		Options: opts,
		id:      42,
		source:  "attached",
		client:  fakeClient(srvURL),
		out:     out,
		errOut:  errOut,
	}
}

func TestAttachedTaskPathAndColumns(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"result":"success","files":[{"id":7,"name":"a.pdf","size":11}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "task", DescriptionOnly: true}, out, &strings.Builder{})
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if gotPath != "/task/42/files" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (no onlyFromDescription by default)", gotQuery)
	}
	for _, want := range []string{"ID", "NAME", "SIZE", "7", "a.pdf", "11"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q: %q", want, out.String())
		}
	}
}

func TestAttachedDescriptionOnlyQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"result":"success","files":[]}`)
	}))
	defer srv.Close()

	o := baseOpts(srv.URL, Options{Type: "contact", DescriptionOnly: true}, &strings.Builder{}, &strings.Builder{})
	o.descriptionOnly = true
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(gotQuery, "onlyFromDescription=true") {
		t.Errorf("query = %q, want onlyFromDescription=true", gotQuery)
	}
}

func TestAttachedProjectPaging(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"result":"success","files":[]}`)
	}))
	defer srv.Close()

	o := baseOpts(srv.URL, Options{Type: "project", Paging: true}, &strings.Builder{}, &strings.Builder{})
	o.limit, o.offset = 50, 100
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(gotQuery, "pageSize=50") || !strings.Contains(gotQuery, "offset=100") {
		t.Errorf("query = %q, want pageSize=50 & offset=100", gotQuery)
	}
}

func TestAttachedJSONRawEcho(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","files":[{"id":7}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "task"}, out, &strings.Builder{})
	o.json = true
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(out.String(), `"files"`) {
		t.Errorf("json output missing envelope: %q", out.String())
	}
}

func TestAttachedEmptyNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","files":[]}`)
	}))
	defer srv.Close()

	errOut := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "task"}, &strings.Builder{}, errOut)
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(errOut.String(), "no files") {
		t.Errorf("stderr note missing: %q", errOut.String())
	}
}

func TestSourceValidation(t *testing.T) {
	o := baseOpts("http://unused", Options{Type: "task", DescriptionOnly: true}, &strings.Builder{}, &strings.Builder{})
	o.source = "bogus"
	if err := runFiles(context.Background(), o); err == nil || !strings.Contains(err.Error(), "invalid --source") {
		t.Fatalf("want invalid --source error, got %v", err)
	}

	o2 := baseOpts("http://unused", Options{Type: "task", DescriptionOnly: true}, &strings.Builder{}, &strings.Builder{})
	o2.source, o2.descriptionOnly = "inline", true
	if err := runFiles(context.Background(), o2); err == nil || !strings.Contains(err.Error(), "description-only") {
		t.Fatalf("want description-only+inline conflict, got %v", err)
	}

	o3 := baseOpts("http://unused", Options{Type: "project", Paging: true}, &strings.Builder{}, &strings.Builder{})
	o3.source, o3.pagingSet = "inline", true
	if err := runFiles(context.Background(), o3); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("want limit+inline conflict, got %v", err)
	}
}

func TestInlineProjectScrapesDescription(t *testing.T) {
	var gotPath, gotQuery string
	fileHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/project/"):
			gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
			io.WriteString(w, `{"result":"success","project":{"description":"<img src=\"?uniqueid=6340764\">"}}`)
		case strings.HasPrefix(r.URL.Path, "/file/"):
			fileHits++
			io.WriteString(w, `{"result":"success","file":{"id":6340764,"name":"editor.png","size":1}}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "project", Paging: true}, out, &strings.Builder{})
	o.source = "inline"
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if gotPath != "/project/42" || !strings.Contains(gotQuery, "fields=description") {
		t.Errorf("project scrape path=%q query=%q", gotPath, gotQuery)
	}
	if fileHits != 1 {
		t.Errorf("file lookups = %d, want 1", fileHits)
	}
	if !strings.Contains(out.String(), "editor.png") {
		t.Errorf("output missing resolved name: %q", out.String())
	}
}

func TestInlineContactScrapesCommentsNotDescription(t *testing.T) {
	var hitComments, hitContactGet bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/contact/42/comments/list":
			hitComments = true
			io.WriteString(w, `{"result":"success","comments":[{"description":"<img src=\"?uniqueid=99\">"}]}`)
		case r.URL.Path == "/contact/42":
			hitContactGet = true
			io.WriteString(w, `{"result":"success","contact":{"description":"plain"}}`)
		case strings.HasPrefix(r.URL.Path, "/file/"):
			io.WriteString(w, `{"result":"success","file":{"id":99,"name":"c.png","size":2}}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "contact", DescriptionOnly: true}, out, &strings.Builder{})
	o.source = "inline"
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !hitComments {
		t.Error("contact inline must read the comment feed")
	}
	if hitContactGet {
		t.Error("contact inline must NOT read the plaintext description field")
	}
	if !strings.Contains(out.String(), "c.png") {
		t.Errorf("output missing resolved name: %q", out.String())
	}
}

func TestInlineTaskPagesCommentsAndDedupes(t *testing.T) {
	var offsets []string
	fileHits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/task/42/comments/list" {
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), `"offset":0`) {
				offsets = append(offsets, "0")
				// 100 comments (a full page) all referencing id 5 → dedupe to one.
				var sb strings.Builder
				sb.WriteString(`{"result":"success","comments":[`)
				for i := 0; i < 100; i++ {
					if i > 0 {
						sb.WriteByte(',')
					}
					sb.WriteString(`{"description":"<img src=\"?uniqueid=5\">"}`)
				}
				sb.WriteString(`]}`)
				io.WriteString(w, sb.String())
				return
			}
			offsets = append(offsets, "100")
			io.WriteString(w, `{"result":"success","comments":[{"description":"<img src=\"?uniqueid=6\">"}]}`)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/file/") {
			fileHits[r.URL.Path]++
			io.WriteString(w, `{"result":"success","file":{"id":1,"name":"x.png","size":1}}`)
			return
		}
		t.Errorf("unexpected path %q", r.URL.Path)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "task", DescriptionOnly: true}, out, &strings.Builder{})
	o.source = "inline"
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if len(offsets) != 2 || offsets[0] != "0" || offsets[1] != "100" {
		t.Errorf("comment offsets = %v, want [0 100]", offsets)
	}
	if fileHits["/file/5"] != 1 || fileHits["/file/6"] != 1 {
		t.Errorf("file lookups = %v, want one each for /file/5 and /file/6", fileHits)
	}
}

func TestInlineComposedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/project/") {
			io.WriteString(w, `{"result":"success","project":{"description":"<img src=\"?uniqueid=1\">"}}`)
			return
		}
		io.WriteString(w, `{"result":"success","file":{"id":1,"name":"z.png","size":3}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "project", Paging: true}, out, &strings.Builder{})
	o.source, o.json = "inline", true
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	for _, want := range []string{`"result"`, `"files"`, `"z.png"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("composed json missing %q: %q", want, out.String())
		}
	}
}
