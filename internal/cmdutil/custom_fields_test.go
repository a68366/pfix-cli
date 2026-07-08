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

func TestFormatCF(t *testing.T) {
	tests := []struct {
		name     string
		spec     CustomFieldSpec
		typeCode int
		want     map[string]any
		wantErr  string
	}{
		{name: "short text", spec: CustomFieldSpec{ID: 1, Value: "hi"}, typeCode: 0,
			want: map[string]any{"field": map[string]any{"id": 1}, "value": "hi"}},
		{name: "multiline", spec: CustomFieldSpec{ID: 2, Value: "a\nb"}, typeCode: 2,
			want: map[string]any{"field": map[string]any{"id": 2}, "value": "a\nb"}},
		{name: "number int", spec: CustomFieldSpec{ID: 3, Value: "42"}, typeCode: 1,
			want: map[string]any{"field": map[string]any{"id": 3}, "value": float64(42)}},
		{name: "number float", spec: CustomFieldSpec{ID: 3, Value: "4.5"}, typeCode: 1,
			want: map[string]any{"field": map[string]any{"id": 3}, "value": float64(4.5)}},
		{name: "enum", spec: CustomFieldSpec{ID: 4, Value: "5"}, typeCode: 8,
			want: map[string]any{"field": map[string]any{"id": 4}, "value": map[string]any{"id": 5}}},
		{name: "number rejects text", spec: CustomFieldSpec{ID: 3, Value: "abc"}, typeCode: 1, wantErr: "is a number"},
		{name: "enum rejects text", spec: CustomFieldSpec{ID: 4, Value: "abc"}, typeCode: 8, wantErr: "give an option id"},
		{name: "enum rejects zero", spec: CustomFieldSpec{ID: 4, Value: "0"}, typeCode: 8, wantErr: "give an option id"},
		{name: "unsupported type", spec: CustomFieldSpec{ID: 9, Value: "x"}, typeCode: 4, wantErr: "unsupported type 4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatCF(tt.spec, tt.typeCode)
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
		io.WriteString(w, `{"result":"success","customfields":[{"id":88206,"type":0},{"id":85984,"type":1},{"id":88210,"type":8}]}`)
	})
	specs := []CustomFieldSpec{{ID: 88206, Value: "hello"}, {ID: 85984, Value: "42"}, {ID: 88210, Value: "5"}}
	got, err := BuildCustomFieldData(context.Background(), c, "task", specs)
	if err != nil {
		t.Fatalf("BuildCustomFieldData: %v", err)
	}
	if gotPath != "/customfield/task" {
		t.Errorf("path = %q, want /customfield/task", gotPath)
	}
	if !strings.Contains(gotQuery, "fields=id%2Ctype") {
		t.Errorf("query = %q, want fields=id,type", gotQuery)
	}
	want := []map[string]any{
		{"field": map[string]any{"id": 88206}, "value": "hello"},
		{"field": map[string]any{"id": 85984}, "value": float64(42)},
		{"field": map[string]any{"id": 88210}, "value": map[string]any{"id": 5}},
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
