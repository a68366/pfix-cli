package cmdutil

import (
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
