package cmdutil

import (
	"reflect"
	"strings"
	"testing"
)

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "****"},
		{"abc", "****"},
		{"abcd", "****"},
		{"abcde", "****bcde"},
		{"1234567890", "****7890"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MaskToken(tt.input)
			if got != tt.want {
				t.Errorf("MaskToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  int
		wantErr string
	}{
		{name: "valid integer", input: "15", wantID: 15},
		{name: "non-numeric", input: "abc", wantErr: "number"},
		{name: "empty string", input: "", wantErr: "number"},
		{name: "float", input: "1.5", wantErr: "number"},
		{name: "zero", input: "0", wantErr: "positive"},
		{name: "negative", input: "-5", wantErr: "positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateID(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ValidateID(%q) expected error containing %q, got nil", tt.input, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateID(%q) error = %q, want it to contain %q", tt.input, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateID(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.wantID {
				t.Errorf("ValidateID(%q) = %d, want %d", tt.input, got, tt.wantID)
			}
		})
	}
}

func TestFieldsCSV(t *testing.T) {
	tests := []struct {
		name     string
		override string
		def      string
		want     string
	}{
		{name: "override wins", override: "a", def: "b", want: "a"},
		{name: "empty override uses def", override: "", def: "b", want: "b"},
		{name: "both empty", override: "", def: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FieldsCSV(tt.override, tt.def)
			if got != tt.want {
				t.Errorf("FieldsCSV(%q, %q) = %q, want %q", tt.override, tt.def, got, tt.want)
			}
		})
	}
}

func TestValidateObjectType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "task", input: "task"},
		{name: "project", input: "project"},
		{name: "empty", input: "", wantErr: "required"},
		{name: "uppercase Task", input: "Task", wantErr: "lowercase"},
		{name: "alphanumeric", input: "a1", wantErr: "lowercase"},
		{name: "with space", input: "a b", wantErr: "lowercase"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateObjectType(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ValidateObjectType(%q) expected error containing %q, got nil", tt.input, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateObjectType(%q) error = %q, want it to contain %q", tt.input, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateObjectType(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

func TestApplyFilter(t *testing.T) {
	t.Run("empty is no-op", func(t *testing.T) {
		b := map[string]any{"x": 1}
		if err := ApplyFilter(b, ""); err != nil {
			t.Fatal(err)
		}
		if _, ok := b["filters"]; ok {
			t.Fatal("filters should not be set")
		}
	})
	t.Run("valid array", func(t *testing.T) {
		b := map[string]any{}
		if err := ApplyFilter(b, `[{"type":1,"operator":"equal","value":5}]`); err != nil {
			t.Fatal(err)
		}
		arr, ok := b["filters"].([]any)
		if !ok || len(arr) != 1 {
			t.Fatalf("filters = %#v", b["filters"])
		}
	})
	t.Run("valid object", func(t *testing.T) {
		b := map[string]any{}
		if err := ApplyFilter(b, `{"a":1}`); err != nil {
			t.Fatal(err)
		}
		obj, ok := b["filters"].(map[string]any)
		if !ok || obj["a"] != float64(1) {
			t.Fatalf("filters = %#v", b["filters"])
		}
	})
	t.Run("invalid json errors", func(t *testing.T) {
		b := map[string]any{}
		err := ApplyFilter(b, "not json")
		if err == nil || !strings.Contains(err.Error(), "invalid --filter JSON") {
			t.Fatalf("err = %v", err)
		}
		if _, ok := b["filters"]; ok {
			t.Fatal("filters should not be set on error")
		}
	})
}

func TestDecodeJSON(t *testing.T) {
	t.Run("invalid JSON returns decode response error", func(t *testing.T) {
		var x map[string]any
		err := DecodeJSON([]byte("{bad"), &x)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "decode response") {
			t.Errorf("error = %q, want it to contain %q", err.Error(), "decode response")
		}
	})

	t.Run("valid JSON decodes correctly", func(t *testing.T) {
		var v struct {
			Name string `json:"name"`
			ID   int    `json:"id"`
		}
		err := DecodeJSON([]byte(`{"name":"test","id":42}`), &v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.Name != "test" || v.ID != 42 {
			t.Errorf("decoded %+v, want name=test id=42", v)
		}
	})
}

func TestParsePeople(t *testing.T) {
	tests := []struct {
		name       string
		refs       []string
		wantUsers  []any
		wantGroups []any
		wantErr    string
	}{
		{
			name:       "single user",
			refs:       []string{"user:1"},
			wantUsers:  []any{map[string]any{"id": "user:1"}},
			wantGroups: []any{},
		},
		{
			name:       "single contact",
			refs:       []string{"contact:4"},
			wantUsers:  []any{map[string]any{"id": "contact:4"}},
			wantGroups: []any{},
		},
		{
			name:       "single group",
			refs:       []string{"group:3"},
			wantUsers:  []any{},
			wantGroups: []any{map[string]any{"id": 3}},
		},
		{
			name: "mixed refs preserve order",
			refs: []string{"user:1", "contact:4", "group:3", "user:2"},
			wantUsers: []any{
				map[string]any{"id": "user:1"},
				map[string]any{"id": "contact:4"},
				map[string]any{"id": "user:2"},
			},
			wantGroups: []any{map[string]any{"id": 3}},
		},
		{
			name:       "empty input yields empty lists",
			refs:       nil,
			wantUsers:  []any{},
			wantGroups: []any{},
		},
		{name: "bare number", refs: []string{"12"}, wantErr: "invalid people reference"},
		{name: "unknown prefix", refs: []string{"team:3"}, wantErr: "invalid people reference"},
		{name: "empty ref", refs: []string{""}, wantErr: "invalid people reference"},
		{name: "missing id", refs: []string{"user:"}, wantErr: "positive"},
		{name: "non-numeric id", refs: []string{"user:abc"}, wantErr: "positive"},
		{name: "zero id", refs: []string{"user:0"}, wantErr: "positive"},
		{name: "negative group id", refs: []string{"group:-1"}, wantErr: "positive"},
		{name: "leading plus", refs: []string{"user:+7"}, wantErr: "positive"},
		{name: "leading zeros", refs: []string{"group:007"}, wantErr: "positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePeople(tt.refs)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ParsePeople(%v) expected error containing %q, got nil", tt.refs, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ParsePeople(%v) error = %q, want it to contain %q", tt.refs, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePeople(%v) unexpected error: %v", tt.refs, err)
			}
			want := map[string]any{"users": tt.wantUsers, "groups": tt.wantGroups}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ParsePeople(%v) = %#v, want %#v", tt.refs, got, want)
			}
		})
	}
}

func TestParseTimePoint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]any
		wantErr bool
	}{
		{name: "date only", input: "2026-07-10", want: map[string]any{"date": "10-07-2026"}},
		{name: "date and time", input: "2026-07-10 18:30", want: map[string]any{"date": "10-07-2026", "time": "18:30"}},
		{name: "T separator", input: "2026-07-10T18:30", want: map[string]any{"date": "10-07-2026", "time": "18:30"}},
		{name: "leap day", input: "2024-02-29", want: map[string]any{"date": "29-02-2024"}},
		{name: "invalid calendar date", input: "2026-13-40", wantErr: true},
		{name: "non-leap february 29", input: "2023-02-29", wantErr: true},
		{name: "output format rejected as input", input: "10-07-2026", wantErr: true},
		{name: "bad time", input: "2026-07-10 25:00", wantErr: true},
		{name: "seconds not accepted", input: "2026-07-10 18:30:15", wantErr: true},
		{name: "garbage", input: "next tuesday", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimePoint(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseTimePoint(%q) expected error, got %#v", tt.input, got)
				}
				if !strings.Contains(err.Error(), "invalid date") {
					t.Errorf("ParseTimePoint(%q) error = %q, want it to contain %q", tt.input, err.Error(), "invalid date")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTimePoint(%q) unexpected error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTimePoint(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}
