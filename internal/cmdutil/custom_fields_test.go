package cmdutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/planfix"
)

// cfTestClient builds a planfix.Client pointed at an httptest server.
func cfTestClient(t *testing.T, h http.HandlerFunc) *planfix.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := planfix.New("example.test", "tok")
	c.BaseURL = srv.URL
	c.Limiter = rate.NewLimiter(rate.Inf, 1)
	c.Backoff = func(int) time.Duration { return 0 }
	return c
}

func TestParseCustomFieldSpecs(t *testing.T) {
	tests := []struct {
		name    string
		raw     []string
		want    []CustomFieldSpec
		wantErr string
	}{
		{name: "single", raw: []string{"88206=hello"}, want: []CustomFieldSpec{{ID: 88206, Value: "hello"}}},
		{name: "multiple", raw: []string{"1=a", "2=b"}, want: []CustomFieldSpec{{ID: 1, Value: "a"}, {ID: 2, Value: "b"}}},
		{name: "value with equals", raw: []string{"1=a=b"}, want: []CustomFieldSpec{{ID: 1, Value: "a=b"}}},
		{name: "value with comma", raw: []string{"1=a, b"}, want: []CustomFieldSpec{{ID: 1, Value: "a, b"}}},
		{name: "empty value", raw: []string{"1="}, want: []CustomFieldSpec{{ID: 1, Value: ""}}},
		{name: "no equals", raw: []string{"88206"}, wantErr: "use <id>=<value>"},
		{name: "non-numeric id", raw: []string{"abc=x"}, wantErr: "id must be a positive number"},
		{name: "zero id", raw: []string{"0=x"}, wantErr: "id must be a positive number"},
		{name: "leading zero id", raw: []string{"007=x"}, wantErr: "id must be a positive number"},
		{name: "duplicate id", raw: []string{"1=a", "1=b"}, wantErr: "more than once"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCustomFieldSpecs(tt.raw)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

// listDef is the definition of a type-8 list field. Its options deliberately mix
// numeric-looking labels (so a label can be told apart from a positional index)
// with a word and a multi-word label containing spaces.
var listDef = customFieldDef{ID: 4, Name: "List field test", Type: 8,
	EnumValues: []string{"1", "2", "3", "four", "five with space"}}

func TestFormatCF(t *testing.T) {
	tests := []struct {
		name    string
		spec    CustomFieldSpec
		def     customFieldDef
		want    map[string]any
		wantErr string
	}{
		{name: "short text", spec: CustomFieldSpec{ID: 1, Value: "hi"}, def: customFieldDef{ID: 1, Type: 0},
			want: map[string]any{"field": map[string]any{"id": 1}, "value": "hi"}},
		{name: "multiline", spec: CustomFieldSpec{ID: 2, Value: "a\nb"}, def: customFieldDef{ID: 2, Type: 2},
			want: map[string]any{"field": map[string]any{"id": 2}, "value": "a\nb"}},
		{name: "number int", spec: CustomFieldSpec{ID: 3, Value: "42"}, def: customFieldDef{ID: 3, Type: 1},
			want: map[string]any{"field": map[string]any{"id": 3}, "value": float64(42)}},
		{name: "number float", spec: CustomFieldSpec{ID: 3, Value: "4.5"}, def: customFieldDef{ID: 3, Type: 1},
			want: map[string]any{"field": map[string]any{"id": 3}, "value": float64(4.5)}},

		// A list is addressed by option label and sent as a bare string.
		{name: "list word label", spec: CustomFieldSpec{ID: 4, Value: "four"}, def: listDef,
			want: map[string]any{"field": map[string]any{"id": 4}, "value": "four"}},
		{name: "list numeric label", spec: CustomFieldSpec{ID: 4, Value: "1"}, def: listDef,
			want: map[string]any{"field": map[string]any{"id": 4}, "value": "1"}},
		{name: "list label with spaces", spec: CustomFieldSpec{ID: 4, Value: "five with space"}, def: listDef,
			want: map[string]any{"field": map[string]any{"id": 4}, "value": "five with space"}},

		{name: "number rejects text", spec: CustomFieldSpec{ID: 3, Value: "abc"}, def: customFieldDef{ID: 3, Type: 1}, wantErr: "is a number"},

		// Regression: "4" is an option *index*, not a label. Sending it used to
		// store the phantom value "4" instead of selecting "four".
		{name: "list rejects index", spec: CustomFieldSpec{ID: 4, Value: "4"}, def: listDef, wantErr: "has no such option"},
		{name: "list rejects unknown label", spec: CustomFieldSpec{ID: 4, Value: "five"}, def: listDef, wantErr: "has no such option"},
		// Labels match exactly — no trimming, no case folding.
		{name: "list rejects trailing space", spec: CustomFieldSpec{ID: 4, Value: "four "}, def: listDef, wantErr: "has no such option"},
		{name: "list rejects wrong case", spec: CustomFieldSpec{ID: 4, Value: "Four"}, def: listDef, wantErr: "has no such option"},
		// Options are quoted so a label containing spaces or a comma stays legible.
		{name: "list error quotes options", spec: CustomFieldSpec{ID: 4, Value: "five"}, def: listDef,
			wantErr: `"1", "2", "3", "four", "five with space"`},
		{name: "list without options", spec: CustomFieldSpec{ID: 4, Value: "x"}, def: customFieldDef{ID: 4, Name: "Empty", Type: 8},
			wantErr: "no options defined"},

		{name: "unsupported type", spec: CustomFieldSpec{ID: 9, Value: "x"}, def: customFieldDef{ID: 9, Type: 4}, wantErr: "unsupported type 4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatCF(tt.spec, tt.def)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBuildCustomFieldData(t *testing.T) {
	var gotPath, gotQuery string
	c := cfTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"result":"success","customfields":[
			{"id":88206,"name":"text","type":0},
			{"id":85984,"name":"num","type":1},
			{"id":88210,"name":"list","type":8,"enumValues":["1","2","3","four"]}]}`)
	})
	specs := []CustomFieldSpec{{ID: 88206, Value: "hello"}, {ID: 85984, Value: "42"}, {ID: 88210, Value: "four"}}
	got, err := BuildCustomFieldData(context.Background(), c, "task", specs)
	if err != nil {
		t.Fatalf("BuildCustomFieldData: %v", err)
	}
	if gotPath != "/customfield/task" {
		t.Errorf("path = %q, want /customfield/task", gotPath)
	}
	// enumValues must be requested, or a list value cannot be validated.
	if !strings.Contains(gotQuery, "fields=id%2Cname%2Ctype%2CenumValues") {
		t.Errorf("query = %q, want fields=id,name,type,enumValues", gotQuery)
	}
	want := []map[string]any{
		{"field": map[string]any{"id": 88206}, "value": "hello"},
		{"field": map[string]any{"id": 85984}, "value": float64(42)},
		{"field": map[string]any{"id": 88210}, "value": "four"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestBuildCustomFieldDataUnknownID(t *testing.T) {
	c := cfTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","customfields":[{"id":88206,"type":0}]}`)
	})
	_, err := BuildCustomFieldData(context.Background(), c, "task", []CustomFieldSpec{{ID: 999, Value: "x"}})
	if err == nil || !strings.Contains(err.Error(), "no custom field 999 for task") {
		t.Fatalf("err = %v, want unknown-id error", err)
	}
}

// A bad list option must fail before any write is attempted.
func TestBuildCustomFieldDataRejectsUnknownOption(t *testing.T) {
	c := cfTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","customfields":[{"id":88210,"name":"list","type":8,"enumValues":["1","2","3","four"]}]}`)
	})
	_, err := BuildCustomFieldData(context.Background(), c, "task", []CustomFieldSpec{{ID: 88210, Value: "4"}})
	if err == nil || !strings.Contains(err.Error(), "has no such option") {
		t.Fatalf("err = %v, want unknown-option error", err)
	}
	if !strings.Contains(err.Error(), `"1", "2", "3", "four"`) {
		t.Errorf("err = %v, want it to list the valid options", err)
	}
}
