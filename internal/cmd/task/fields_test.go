package task

import (
	"reflect"
	"strings"
	"testing"
)

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "Urgent", want: "Urgent"},
		{input: "NotUrgent", want: "NotUrgent"},
		{input: "urgent", want: "Urgent"},
		{input: "NOTURGENT", want: "NotUrgent"},
		{input: "VeryUrgent", wantErr: true},
		{input: "high", wantErr: true},
		{input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePriority(tt.input)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "invalid priority") {
					t.Fatalf("parsePriority(%q) error = %v, want it to contain %q", tt.input, err, "invalid priority")
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePriority(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parsePriority(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseCounterparty(t *testing.T) {
	tests := []struct {
		input   string
		want    map[string]any
		wantErr bool
	}{
		{input: "4", want: map[string]any{"id": 4}},
		{input: "contact:4", want: map[string]any{"id": "contact:4"}},
		{input: "abc", wantErr: true},
		{input: "contact:abc", wantErr: true},
		{input: "contact:", wantErr: true},
		{input: "0", wantErr: true},
		{input: "contact:0", wantErr: true},
		{input: "-3", wantErr: true},
		{input: "user:4", wantErr: true},
		{input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseCounterparty(tt.input)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "invalid counterparty") {
					t.Fatalf("parseCounterparty(%q) error = %v, want it to contain %q", tt.input, err, "invalid counterparty")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCounterparty(%q) unexpected error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCounterparty(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}
