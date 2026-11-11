package file

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// uploadAPI is a fake for the upload, from-url, and attach endpoints. It
// records every request; attachStatus/attachBody override the attach answer.
type uploadAPI struct {
	mu           sync.Mutex
	reqs         []string // "METHOD path?query"
	fileName     string
	fileBytes    []byte
	fromURLBody  map[string]any
	uploadStatus int
	uploadBody   string
	attachStatus int
	attachBody   string
}

func (a *uploadAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.reqs = append(a.reqs, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.URL.Path == "/file/":
			mr, err := r.MultipartReader()
			if err != nil {
				t.Errorf("multipart: %v", err)
				return
			}
			p, err := mr.NextPart()
			if err != nil {
				t.Errorf("part: %v", err)
				return
			}
			a.fileName = p.FileName()
			a.fileBytes, _ = io.ReadAll(p)
			if a.uploadStatus != 0 {
				w.WriteHeader(a.uploadStatus)
				io.WriteString(w, a.uploadBody)
				return
			}
			io.WriteString(w, `{"result":"success","id":555}`)
		case r.URL.Path == "/file/from-url/":
			if err := json.NewDecoder(r.Body).Decode(&a.fromURLBody); err != nil {
				t.Errorf("decode from-url body: %v", err)
			}
			if a.uploadStatus != 0 {
				w.WriteHeader(a.uploadStatus)
				io.WriteString(w, a.uploadBody)
				return
			}
			io.WriteString(w, `{"result":"success","id":556}`)
		case strings.Contains(r.URL.Path, "/attach/"):
			if a.attachStatus != 0 {
				w.WriteHeader(a.attachStatus)
				io.WriteString(w, a.attachBody)
				return
			}
			io.WriteString(w, `{"result":"success"}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUploadLocalFile(t *testing.T) {
	content := []byte{0x00, 0x01, 0xfe, '\n'}
	tests := []struct {
		name     string
		nameFlag string
		nameSet  bool
		attach   map[string]string
		quiet    bool
		json     bool
		jq       string
		wantName string
		wantOut  string
		wantReqs []string
	}{
		{name: "plain", wantName: "data.bin", wantOut: "Uploaded data.bin as file 555\n",
			wantReqs: []string{"POST /file/"}},
		{name: "--name override", nameFlag: "отчёт за май.txt", nameSet: true,
			wantName: "отчёт за май.txt", wantOut: "Uploaded отчёт за май.txt as file 555\n",
			wantReqs: []string{"POST /file/"}},
		{name: "attach task", attach: map[string]string{"task": "12"}, wantName: "data.bin",
			wantOut:  "Uploaded data.bin as file 555, attached to task 12\n",
			wantReqs: []string{"POST /file/", "POST /file/555/attach/task?id=12"}},
		{name: "attach contact", attach: map[string]string{"contact": "7"}, wantName: "data.bin",
			wantOut:  "Uploaded data.bin as file 555, attached to contact 7\n",
			wantReqs: []string{"POST /file/", "POST /file/555/attach/contact?id=7"}},
		{name: "attach project", attach: map[string]string{"project": "3"}, wantName: "data.bin",
			wantOut:  "Uploaded data.bin as file 555, attached to project 3\n",
			wantReqs: []string{"POST /file/", "POST /file/555/attach/project?id=3"}},
		{name: "quiet", quiet: true, attach: map[string]string{"task": "12"}, wantName: "data.bin",
			wantOut:  "555\n",
			wantReqs: []string{"POST /file/", "POST /file/555/attach/task?id=12"}},
		{name: "json", json: true, wantName: "data.bin",
			wantOut: "{\n  \"result\": \"success\",\n  \"id\": 555\n}\n", wantReqs: []string{"POST /file/"}},
		{name: "jq", json: true, jq: ".id", wantName: "data.bin",
			wantOut: "555\n", wantReqs: []string{"POST /file/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &uploadAPI{}
			srv := api.server(t)
			out := &strings.Builder{}
			o := &uploadOptions{
				path: writeTemp(t, "data.bin", content), name: tt.nameFlag, nameSet: tt.nameSet,
				attach: tt.attach, quiet: tt.quiet, json: tt.json, jq: tt.jq,
				client: fakeClient(srv.URL), out: out,
			}
			if err := runUpload(context.Background(), o); err != nil {
				t.Fatalf("runUpload: %v", err)
			}
			if api.fileName != tt.wantName {
				t.Errorf("filename = %q, want %q", api.fileName, tt.wantName)
			}
			if string(api.fileBytes) != string(content) {
				t.Errorf("bytes = %v, want %v", api.fileBytes, content)
			}
			if out.String() != tt.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tt.wantOut)
			}
			if strings.Join(api.reqs, "|") != strings.Join(tt.wantReqs, "|") {
				t.Errorf("requests = %v, want %v", api.reqs, tt.wantReqs)
			}
		})
	}
}

func TestUploadFromURL(t *testing.T) {
	tests := []struct {
		name     string
		nameFlag string
		nameSet  bool
		wantBody map[string]any
		wantOut  string
	}{
		{name: "url only", wantBody: map[string]any{"url": "https://example.test/spec.json"},
			wantOut: "Uploaded https://example.test/spec.json as file 556\n"},
		{name: "with name", nameFlag: "spec.json", nameSet: true,
			wantBody: map[string]any{"url": "https://example.test/spec.json", "name": "spec.json"},
			wantOut:  "Uploaded spec.json as file 556\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &uploadAPI{}
			srv := api.server(t)
			out := &strings.Builder{}
			o := &uploadOptions{fromURL: "https://example.test/spec.json", name: tt.nameFlag,
				nameSet: tt.nameSet, client: fakeClient(srv.URL), out: out}
			if err := runUpload(context.Background(), o); err != nil {
				t.Fatalf("runUpload: %v", err)
			}
			got, _ := json.Marshal(api.fromURLBody)
			want, _ := json.Marshal(tt.wantBody)
			if string(got) != string(want) {
				t.Errorf("body = %s, want %s", got, want)
			}
			if out.String() != tt.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tt.wantOut)
			}
		})
	}
}

func TestUploadFromURLNotRetriedOn5xx(t *testing.T) {
	api := &uploadAPI{uploadStatus: 500, uploadBody: `{"result":"fail","code":0,"error":"Rest API error"}`}
	srv := api.server(t)
	o := &uploadOptions{fromURL: "https://nowhere.invalid/x", client: fakeClient(srv.URL), out: &strings.Builder{}}
	if err := runUpload(context.Background(), o); err == nil {
		t.Fatal("want error")
	}
	if len(api.reqs) != 1 {
		t.Errorf("requests = %v, want exactly one", api.reqs)
	}
}

func TestUploadLocalNotRetriedOn5xx(t *testing.T) {
	api := &uploadAPI{uploadStatus: 500, uploadBody: `{"result":"fail","code":0,"error":"Rest API error"}`}
	srv := api.server(t)
	o := &uploadOptions{path: writeTemp(t, "a.txt", []byte("abc")), client: fakeClient(srv.URL), out: &strings.Builder{}}
	if err := runUpload(context.Background(), o); err == nil {
		t.Fatal("want error")
	}
	if len(api.reqs) != 1 {
		t.Errorf("requests = %v, want exactly one", api.reqs)
	}
}

func TestUploadRejectsBeforeAnyRequest(t *testing.T) {
	dir := t.TempDir()
	file := writeTemp(t, "a.txt", []byte("abc"))
	huge := filepath.Join(dir, "huge.bin")
	exact := filepath.Join(dir, "exact.bin")
	for p, size := range map[string]int64{huge: maxUploadSize + 1, exact: maxUploadSize} {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(p, size); err != nil { // sparse: no 100 MiB written
			t.Fatal(err)
		}
	}
	badBase := writeTemp(t, `back\slash.txt`, []byte("x"))

	tests := []struct {
		name    string
		o       uploadOptions
		wantErr string
	}{
		{"neither path nor url", uploadOptions{}, "exactly one of <path> or --from-url"},
		{"both path and url", uploadOptions{path: file, fromURL: "https://x"}, "exactly one of <path> or --from-url"},
		{"two attach targets", uploadOptions{path: file, attach: map[string]string{"task": "1", "project": "2"}}, "at most one of --task, --contact, --project"},
		{"zero attach id", uploadOptions{path: file, attach: map[string]string{"task": "0"}}, "--task: id must be a positive number"},
		{"non-numeric attach id", uploadOptions{path: file, attach: map[string]string{"contact": "abc"}}, "--contact: id must be a number"},
		{"missing file", uploadOptions{path: filepath.Join(dir, "nope.txt")}, "no such file"},
		{"directory", uploadOptions{path: dir}, "is a directory"},
		{"over 100 MiB", uploadOptions{path: huge}, "limits uploads to 100 MiB"},
		{"empty --name", uploadOptions{path: file, name: "  ", nameSet: true}, "--name must not be empty"},
		{"slash in --name", uploadOptions{path: file, name: "../up/x.txt", nameSet: true}, "must not contain a path separator"},
		{"backslash in --name", uploadOptions{path: file, name: `a\b.txt`, nameSet: true}, "must not contain a path separator"},
		{"newline in --name", uploadOptions{path: file, name: "a\r\nb.txt", nameSet: true}, "must not contain control characters"},
		{"dotdot --name", uploadOptions{fromURL: "https://x", name: "..", nameSet: true}, "must be a file name"},
		{"backslash in base name", uploadOptions{path: badBase}, "use --name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &uploadAPI{}
			srv := api.server(t)
			o := tt.o
			o.client, o.out = fakeClient(srv.URL), &strings.Builder{}
			err := runUpload(context.Background(), &o)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
			if len(api.reqs) != 0 {
				t.Errorf("requests = %v, want none", api.reqs)
			}
		})
	}

	// Exactly 100 MiB passes the pre-check (the request is answered by the fake).
	t.Run("exactly 100 MiB is sent", func(t *testing.T) {
		if testing.Short() {
			t.Skip("streams 100 MiB through the fake")
		}
		api := &uploadAPI{}
		srv := api.server(t)
		o := &uploadOptions{path: exact, client: fakeClient(srv.URL), out: &strings.Builder{}}
		if err := runUpload(context.Background(), o); err != nil {
			t.Fatalf("runUpload: %v", err)
		}
		if len(api.fileBytes) != maxUploadSize {
			t.Errorf("sent %d bytes, want %d", len(api.fileBytes), maxUploadSize)
		}
	})
}

func TestUploadAttachFailureNamesFileID(t *testing.T) {
	api := &uploadAPI{attachStatus: 400, attachBody: `{"result":"fail","code":1000,"error":"Task not found by id - 99"}`}
	srv := api.server(t)
	out := &strings.Builder{}
	o := &uploadOptions{path: writeTemp(t, "a.txt", []byte("abc")), attach: map[string]string{"task": "99"},
		client: fakeClient(srv.URL), out: out}
	err := runUpload(context.Background(), o)
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{"uploaded a.txt as file 555", "attaching it to task 99 failed", "no task with that id"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want containing %q", err, want)
		}
	}
	if out.String() != "" {
		t.Errorf("out = %q, want nothing on failure", out.String())
	}
}

func TestDescribeUploadErrorHints(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{7002, "100 MiB upload limit"},
		{7000, "no file with that id"},
		{7001, "no URL to fetch"},
		{1000, "no task with that id"},
		{43, "Idempotency-Key was already used"},
		{1, "pfix auth login"}, // falls through to the shared mapping
	}
	for _, tt := range tests {
		api := &uploadAPI{uploadStatus: 400, uploadBody: `{"result":"fail","code":` + itoa(tt.code) + `,"error":"Rest API error"}`}
		srv := api.server(t)
		o := &uploadOptions{fromURL: "https://example.test/x", client: fakeClient(srv.URL), out: &strings.Builder{}}
		err := runUpload(context.Background(), o)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("code %d: err = %v, want containing %q", tt.code, err, tt.want)
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
