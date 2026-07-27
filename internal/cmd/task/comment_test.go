package task

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
)

// --- comment list tests ---

func TestRunCommentListDefaultTable(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","comments":[`+
			`{"id":1,"description":"First comment","dateTime":{"datetime":"2024-01-15"}},`+
			`{"id":2,"description":"Second comment","dateTime":{"datetime":"2024-01-16"}}`+
			`]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  5,
		offset: 0,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/15/comments/list" {
		t.Errorf("path = %q, want /task/15/comments/list", gotPath)
	}
	if !strings.Contains(gotBody, `"pageSize":5`) {
		t.Errorf("body missing pageSize: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"offset":0`) {
		t.Errorf("body missing offset: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"fields":"id,description,dateTime"`) {
		t.Errorf("body missing fields: %q", gotBody)
	}
	result := out.String()
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID header: %q", result)
	}
	if !strings.Contains(result, "CREATED") {
		t.Errorf("output missing CREATED header: %q", result)
	}
	if !strings.Contains(result, "COMMENT") {
		t.Errorf("output missing COMMENT header: %q", result)
	}
	if !strings.Contains(result, "First comment") {
		t.Errorf("output missing comment text: %q", result)
	}
}

func TestRunCommentListCustomFields(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","comments":[{"id":1,"description":"First comment","dateTime":{"datetime":"2024-01-15"}}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  100,
		offset: 0,
		fields: "id,description",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	if !strings.Contains(gotBody, `"fields":"id,description"`) {
		t.Errorf("body missing custom fields: %q", gotBody)
	}
	result := out.String()
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID column: %q", result)
	}
	if !strings.Contains(result, "DESCRIPTION") {
		t.Errorf("output missing DESCRIPTION column: %q", result)
	}
	if strings.Contains(result, "CREATED") {
		t.Errorf("output should not contain CREATED column when not in fields override: %q", result)
	}
}

func TestRunCommentListLongCommentTruncated(t *testing.T) {
	// Build a comment longer than 80 runes.
	longDesc := strings.Repeat("x", 100) // 100 chars > 80 rune limit
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"result": "success",
			"comments": []map[string]any{
				{"id": 1, "description": longDesc, "dateTime": map[string]any{"datetime": "2024-01-15"}},
			},
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  100,
		offset: 0,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	result := out.String()
	// The full long description should NOT appear verbatim.
	if strings.Contains(result, longDesc) {
		t.Errorf("long comment should be truncated, but full text appears in output: %q", result)
	}
	// The truncated version (first 80 runes + ellipsis) should be present.
	if !strings.Contains(result, strings.Repeat("x", 80)+"…") {
		t.Errorf("output should contain 80-rune truncated comment with ellipsis: %q", result)
	}
}

func TestRunCommentListNewlinesCollapsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"result": "success",
			"comments": []map[string]any{
				{"id": 1, "description": "line one\nline two\nline three", "dateTime": map[string]any{"datetime": "2024-01-15"}},
			},
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  100,
		offset: 0,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	result := out.String()
	// Newlines should be replaced with spaces (collapsed to single line).
	if strings.Contains(result, "\nline two") {
		t.Errorf("newlines in comment should be collapsed; got %q", result)
	}
	if !strings.Contains(result, "line one") {
		t.Errorf("comment text should still appear; got %q", result)
	}
}

func TestRunCommentListLongMultilineComment(t *testing.T) {
	// 90 runes including newlines: exercises collapse AND truncation together.
	longMultiline := strings.Repeat("ab\n", 30) // 90 runes
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"result": "success",
			"comments": []map[string]any{
				{"id": 1, "description": longMultiline, "dateTime": map[string]any{"datetime": "2024-01-15"}},
			},
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  100,
		offset: 0,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	result := out.String()
	// Truncated: the COMMENT cell must end in an ellipsis (90 runes > 80 limit).
	if !strings.Contains(result, "…") {
		t.Errorf("long multiline comment should be truncated with ellipsis: %q", result)
	}
	// Collapsed: the single-lined, truncated value (80 runes of "ab " pattern) must
	// appear, proving newlines were replaced with spaces before truncation.
	want := output.Truncate(strings.ReplaceAll(longMultiline, "\n", " "), 80)
	if !strings.Contains(result, want) {
		t.Errorf("output should contain collapsed+truncated comment %q; got %q", want, result)
	}
}

func TestRunCommentListJSON(t *testing.T) {
	raw := `{"result":"success","comments":[]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, raw)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  100,
		offset: 0,
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) {
		t.Errorf("json output missing result field: %q", result)
	}
	if !strings.Contains(result, `"success"`) {
		t.Errorf("json output missing success value: %q", result)
	}
}

func TestRunCommentListQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","comments":[{"id":1,"description":"A comment","dateTime":{"datetime":"2024-01-15"}}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  100,
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	result := out.String()
	if strings.Contains(result, "ID") || strings.Contains(result, "CREATED") || strings.Contains(result, "COMMENT") {
		t.Errorf("quiet mode should not show headers: %q", result)
	}
	if !strings.Contains(result, "A comment") {
		t.Errorf("quiet mode should still show data: %q", result)
	}
}

func TestRunCommentListOffset(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"result":"success","comments":[]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{
		id:     15,
		limit:  10,
		offset: 20,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	if !strings.Contains(gotBody, `"pageSize":10`) {
		t.Errorf("body missing pageSize 10: %q", gotBody)
	}
	if !strings.Contains(gotBody, `"offset":20`) {
		t.Errorf("body missing offset 20: %q", gotBody)
	}
}

// --- comment add tests ---

func TestRunCommentAddDefault(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":99}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentAddOptions{
		id:     15,
		body:   "Hi",
		client: fakeClient(srv.URL),
		out:    out,
		// Non-empty stdin: --body must take priority and stdin must be ignored.
		in: strings.NewReader("unexpected stdin content"),
	}
	if err := runCommentAdd(context.Background(), o); err != nil {
		t.Fatalf("runCommentAdd: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/15/comments/" {
		t.Errorf("path = %q, want /task/15/comments/", gotPath)
	}
	if gotBody["description"] != "Hi" {
		t.Errorf("body description = %v, want %q (stdin must be ignored when --body set)", gotBody["description"], "Hi")
	}
	if out.String() != "Added comment 99\n" {
		t.Errorf("output = %q, want %q", out.String(), "Added comment 99\n")
	}
}

func TestRunCommentAddQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":99}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentAddOptions{
		id:     15,
		body:   "Hi",
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
		in:     strings.NewReader(""),
	}
	if err := runCommentAdd(context.Background(), o); err != nil {
		t.Fatalf("runCommentAdd: %v", err)
	}
	if out.String() != "99\n" {
		t.Errorf("quiet output = %q, want %q", out.String(), "99\n")
	}
}

func TestRunCommentAddJSON(t *testing.T) {
	raw := `{"result":"success","id":99}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, raw)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentAddOptions{
		id:     15,
		body:   "Hi",
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
		in:     strings.NewReader(""),
	}
	if err := runCommentAdd(context.Background(), o); err != nil {
		t.Fatalf("runCommentAdd: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) {
		t.Errorf("json output missing result field: %q", result)
	}
	if !strings.Contains(result, `"success"`) {
		t.Errorf("json output missing success value: %q", result)
	}
	if !strings.Contains(result, `"id"`) {
		t.Errorf("json output missing id field: %q", result)
	}
}

func TestRunCommentAddStdin(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":7}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentAddOptions{
		id:     15,
		body:   "", // empty — should fall through to stdin
		client: fakeClient(srv.URL),
		out:    out,
		in:     strings.NewReader("from stdin\n"),
	}
	if err := runCommentAdd(context.Background(), o); err != nil {
		t.Fatalf("runCommentAdd: %v", err)
	}
	if gotBody["description"] != "from stdin" {
		t.Errorf("body description = %v, want %q", gotBody["description"], "from stdin")
	}
	if out.String() != "Added comment 7\n" {
		t.Errorf("output = %q, want %q", out.String(), "Added comment 7\n")
	}
}

func TestRunCommentAddEmptyBodyAndStdin(t *testing.T) {
	tests := []struct {
		name  string
		stdin string
	}{
		{"empty stdin", ""},
		{"whitespace-only stdin", "   \n\t  \r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestSent := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestSent = true
				io.WriteString(w, `{"result":"success","id":1}`)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &commentAddOptions{
				id:     15,
				body:   "",
				client: fakeClient(srv.URL),
				out:    out,
				in:     strings.NewReader(tt.stdin),
			}
			err := runCommentAdd(context.Background(), o)
			if err == nil {
				t.Fatal("expected error when --body is empty and stdin has no content")
			}
			if !strings.Contains(err.Error(), "body") {
				t.Errorf("error should mention body requirement, got: %q", err.Error())
			}
			if requestSent {
				t.Error("no HTTP request should be sent when body is empty")
			}
		})
	}
}

// TestCommentListNonNumericID drives the Cobra command with a non-numeric id,
// asserting RunE rejects it before sending any request.
func TestCommentListNonNumericID(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newCommentCmd(g)
	cmd.SetArgs([]string{"list", "abc"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error on non-numeric id")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("error should mention 'number', got: %q", err.Error())
	}
}

// TestCommentAddNonNumericID drives the Cobra command with a non-numeric id.
func TestCommentAddNonNumericID(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newCommentCmd(g)
	cmd.SetArgs([]string{"add", "abc", "--body", "hi"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error on non-numeric id")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("error should mention 'number', got: %q", err.Error())
	}
}

func TestCommentListDefaultBodyHasNoTypeList(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Errorf("unmarshal %q: %v", raw, err)
		}
		io.WriteString(w, `{"result":"success","comments":[{"id":1,"description":"a"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{id: 12, limit: 100, client: fakeClient(srv.URL), out: out}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	if _, ok := got["typeList"]; ok {
		t.Errorf("typeList must be absent by default: %v", got)
	}
	if got["fields"] != commentListFields {
		t.Errorf("fields = %v, want %q", got["fields"], commentListFields)
	}
	if strings.Contains(out.String(), "DELETED") {
		t.Errorf("DELETED column present by default:\n%s", out.String())
	}
}

func TestCommentAddSendsOnlySetFlags(t *testing.T) {
	var got map[string]any
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Errorf("unmarshal %q: %v", raw, err)
		}
		io.WriteString(w, `{"result":"success","id":77}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentAddOptions{
		id:     12,
		body:   "hello",
		extra:  map[string]any{"isPinned": true},
		silent: true,
		client: fakeClient(srv.URL),
		out:    out,
		in:     strings.NewReader(""),
	}
	if err := runCommentAdd(context.Background(), o); err != nil {
		t.Fatalf("runCommentAdd: %v", err)
	}
	if got["description"] != "hello" {
		t.Errorf("description = %v", got["description"])
	}
	if got["isPinned"] != true {
		t.Errorf("isPinned = %v, want true", got["isPinned"])
	}
	if _, ok := got["isHidden"]; ok {
		t.Errorf("isHidden must be absent when unset: %v", got)
	}
	if gotQuery != "silent=true" {
		t.Errorf("query = %q", gotQuery)
	}
	if !strings.Contains(out.String(), "77") {
		t.Errorf("output = %q", out.String())
	}
}

func TestCommentAddExtras(t *testing.T) {
	cases := []struct {
		name    string
		pinned  bool
		hidden  bool
		changed map[string]bool
		want    map[string]any
	}{
		{"none", false, false, map[string]bool{}, map[string]any{}},
		{"pinned", true, false, map[string]bool{"pinned": true}, map[string]any{"isPinned": true}},
		{"hidden false explicit", false, false, map[string]bool{"hidden": true}, map[string]any{"isHidden": false}},
		{"both", true, true, map[string]bool{"pinned": true, "hidden": true}, map[string]any{"isPinned": true, "isHidden": true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := commentAddExtras(tc.pinned, tc.hidden, func(f string) bool { return tc.changed[f] })
			if len(got) != len(tc.want) {
				t.Fatalf("extras = %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("extras[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

// TestCommentAddFlagsFromCommandLine drives the real `comment add` flags
// through Cobra's parser rather than a hand-rolled changed map, pinning that
// commentAddExtras's string literals ("pinned", "hidden") match the flag
// names newCommentAddCmd actually registers. It also confirms that when
// --silent is omitted, the request carries no query string at all — silent=true
// must appear only when the user actually passes --silent.
func TestCommentAddFlagsFromCommandLine(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want map[string]any
	}{
		{"none", nil, map[string]any{}},
		{"pinned", []string{"--pinned"}, map[string]any{"isPinned": true}},
		{"hidden false explicit", []string{"--hidden=false"}, map[string]any{"isHidden": false}},
		{"both", []string{"--pinned", "--hidden"}, map[string]any{"isPinned": true, "isHidden": true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := &cmdutil.GlobalOpts{}
			cmd := newCommentAddCmd(g)
			if err := cmd.Flags().Parse(tc.args); err != nil {
				t.Fatalf("parse: %v", err)
			}
			pinned, err := cmd.Flags().GetBool("pinned")
			if err != nil {
				t.Fatalf("GetBool(pinned): %v", err)
			}
			hidden, err := cmd.Flags().GetBool("hidden")
			if err != nil {
				t.Fatalf("GetBool(hidden): %v", err)
			}
			got := commentAddExtras(pinned, hidden, cmd.Flags().Changed)
			if len(got) != len(tc.want) {
				t.Fatalf("extras = %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("extras[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}

	var gotQuery string
	requestSent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestSent = true
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"result":"success","id":1}`)
	}))
	defer srv.Close()

	g := &cmdutil.GlobalOpts{}
	cmd := newCommentAddCmd(g)
	if err := cmd.Flags().Parse([]string{"--body", "hi"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	silent, err := cmd.Flags().GetBool("silent")
	if err != nil {
		t.Fatalf("GetBool(silent): %v", err)
	}
	if silent {
		t.Fatal("silent must default to false when --silent is not passed")
	}
	pinned, _ := cmd.Flags().GetBool("pinned")
	hidden, _ := cmd.Flags().GetBool("hidden")
	body, _ := cmd.Flags().GetString("body")
	o := &commentAddOptions{
		id:     12,
		body:   body,
		extra:  commentAddExtras(pinned, hidden, cmd.Flags().Changed),
		silent: silent,
		client: fakeClient(srv.URL),
		out:    &strings.Builder{},
		in:     strings.NewReader(""),
	}
	if err := runCommentAdd(context.Background(), o); err != nil {
		t.Fatalf("runCommentAdd: %v", err)
	}
	if !requestSent {
		t.Fatal("request was never sent")
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (no ?silent= when --silent is omitted)", gotQuery)
	}
}

func TestCommentListIncludeDeleted(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Errorf("unmarshal %q: %v", raw, err)
		}
		io.WriteString(w, `{"result":"success","comments":[
			{"id":2,"description":"gone","isDeleted":true},
			{"id":1,"description":"live","isDeleted":false}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &commentListOptions{id: 12, limit: 100, includeDeleted: true, client: fakeClient(srv.URL), out: out}
	if err := runCommentList(context.Background(), o); err != nil {
		t.Fatalf("runCommentList: %v", err)
	}
	if got["typeList"] != "Deleted" {
		t.Errorf("typeList = %v, want Deleted", got["typeList"])
	}
	fields, _ := got["fields"].(string)
	if !strings.Contains(fields, "isDeleted") {
		t.Errorf("fields = %q, want isDeleted included", fields)
	}
	for _, want := range []string{"DELETED", "true", "gone", "live"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("table missing %q:\n%s", want, out.String())
		}
	}
}
