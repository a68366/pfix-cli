package planfix

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// uploadHeaderTimeout caps the wait for the upload response's headers once the
// body has been fully sent. The server answers only after storing the file,
// which takes several seconds near the 100 MiB limit, so this is more generous
// than Stream's.
const uploadHeaderTimeout = 60 * time.Second

// Upload POSTs size bytes read from content to path as multipart/form-data, in
// a single part named "file" that carries filename, and returns the raw
// response bytes. On HTTP status >= 300 it returns *APIError.
//
// Upload makes exactly one attempt: an upload is not idempotent, and the API
// answers a malformed upload with a 5xx that the retrying Do path would repeat.
// The body is streamed, not buffered: the multipart framing around the file is
// fixed, so the exact Content-Length is computed up front and the file bytes
// are read straight from content. If content yields fewer than size bytes the
// request fails rather than sending a short body. Like Stream, it runs with no
// whole-request timeout (a large body would otherwise be cut off mid-transfer)
// and resolves its proxy through c.Proxy.
func (c *Client) Upload(ctx context.Context, path, filename string, content io.Reader, size int64) ([]byte, error) {
	if size < 0 {
		return nil, fmt.Errorf("planfix: upload size must be >= 0, got %d", size)
	}
	head, tail, contentType, err := multipartFraming(filename)
	if err != nil {
		return nil, err
	}
	if err := c.Limiter.Wait(ctx); err != nil {
		return nil, err
	}
	body := io.MultiReader(bytes.NewReader(head), io.LimitReader(content, size), bytes.NewReader(tail))
	req, err := c.newRequest(ctx, http.MethodPost, path, body, contentType, nil)
	if err != nil {
		return nil, err
	}
	req.ContentLength = int64(len(head)) + size + int64(len(tail))

	hc := &http.Client{Transport: c.newTransport(uploadHeaderTimeout)}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed (not retried): %w", err)
	}
	return readJSONResponse(resp)
}

// multipartFraming returns the bytes that precede and follow the file content
// in a one-part multipart/form-data body, plus that body's Content-Type.
func multipartFraming(filename string) (head, tail []byte, contentType string, err error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if _, err := w.CreateFormFile("file", filename); err != nil {
		return nil, nil, "", fmt.Errorf("build multipart body: %w", err)
	}
	head = bytes.Clone(buf.Bytes())
	buf.Reset()
	if err := w.Close(); err != nil {
		return nil, nil, "", fmt.Errorf("build multipart body: %w", err)
	}
	return head, bytes.Clone(buf.Bytes()), w.FormDataContentType(), nil
}
