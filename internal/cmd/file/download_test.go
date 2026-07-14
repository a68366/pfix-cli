package file

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func downloadServer(t *testing.T, body []byte, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/download") {
			if status != 0 {
				w.WriteHeader(status)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(body)
			return
		}
		io.WriteString(w, `{"result":"success","file":{"id":1,"name":"pic.png","size":1}}`)
	}))
}

func baseDownloadOpts(srvURL string, out, errOut io.Writer) *downloadOptions {
	return &downloadOptions{id: 1, client: fakeClient(srvURL), out: out, errOut: errOut}
}

func TestDownloadByteExactToAutoName(t *testing.T) {
	body := []byte{0x89, 'P', 'N', 'G', 0xff} // no trailing newline
	srv := downloadServer(t, body, 0)
	defer srv.Close()

	dir := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	errOut := &strings.Builder{}
	o := baseDownloadOpts(srv.URL, &strings.Builder{}, errOut)
	if err := runDownload(context.Background(), o); err != nil {
		t.Fatalf("runDownload: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "pic.png"))
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("saved bytes = %v, want %v", got, body)
	}
	if !strings.Contains(errOut.String(), "Saved") {
		t.Errorf("missing Saved note: %q", errOut.String())
	}
}

func TestDownloadToStdout(t *testing.T) {
	body := []byte("RAWBYTES")
	srv := downloadServer(t, body, 0)
	defer srv.Close()

	out, errOut := &strings.Builder{}, &strings.Builder{}
	o := baseDownloadOpts(srv.URL, out, errOut)
	o.output = "-"
	if err := runDownload(context.Background(), o); err != nil {
		t.Fatalf("runDownload: %v", err)
	}
	if out.String() != string(body) {
		t.Fatalf("stdout = %q, want %q", out.String(), body)
	}
	if strings.Contains(errOut.String(), "Saved") {
		t.Errorf("stdout mode must not print a Saved note: %q", errOut.String())
	}
}

func TestDownloadRefusesExistingWithoutForce(t *testing.T) {
	srv := downloadServer(t, []byte("x"), 0)
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "keep.png")
	if err := os.WriteFile(dest, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := baseDownloadOpts(srv.URL, &strings.Builder{}, &strings.Builder{})
	o.output = dest
	if err := runDownload(context.Background(), o); err == nil || !strings.Contains(err.Error(), "exists") {
		t.Fatalf("want exists error, got %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "original" {
		t.Errorf("existing file was overwritten: %q", got)
	}
}

func TestDownloadForceOverwrites(t *testing.T) {
	srv := downloadServer(t, []byte("new"), 0)
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "keep.png")
	os.WriteFile(dest, []byte("original"), 0o644)
	o := baseDownloadOpts(srv.URL, &strings.Builder{}, &strings.Builder{})
	o.output, o.force = dest, true
	if err := runDownload(context.Background(), o); err != nil {
		t.Fatalf("runDownload: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "new" {
		t.Errorf("dest = %q, want new", got)
	}
}

func TestDownloadRemovesPartialOnAPIError(t *testing.T) {
	srv := downloadServer(t, nil, http.StatusNotFound)
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.png")
	o := baseDownloadOpts(srv.URL, &strings.Builder{}, &strings.Builder{})
	o.output = dest
	if err := runDownload(context.Background(), o); err == nil {
		t.Fatal("want error on 404")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("partial file not removed: stat err = %v", err)
	}
}

func TestDownloadRejectsJSON(t *testing.T) {
	o := baseDownloadOpts("http://unused", &strings.Builder{}, &strings.Builder{})
	o.json = true
	if err := runDownload(context.Background(), o); err == nil || !strings.Contains(err.Error(), "raw bytes") {
		t.Fatalf("want --json rejection, got %v", err)
	}
}
