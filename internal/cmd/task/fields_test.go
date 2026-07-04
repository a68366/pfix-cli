package task

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
		{input: "contact:+4", wantErr: true},
		{input: "007", wantErr: true},
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

// changedSet fakes cmd.Flags().Changed for apply tests.
func changedSet(names ...string) func(string) bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return func(n string) bool { return m[n] }
}

func TestTaskFieldsApply(t *testing.T) {
	t.Run("all fields set", func(t *testing.T) {
		f := &taskFields{
			template:     6,
			project:      21,
			parent:       31,
			status:       2,
			priority:     "urgent",
			counterparty: "contact:4",
			assignees:    []string{"user:1", "group:3"},
			auditors:     []string{"contact:4"},
			participants: []string{"user:2"},
			startDate:    "2026-07-08",
			endDate:      "2026-07-20 18:00",
		}
		body := map[string]any{"name": "x"}
		err := f.apply(body, changedSet("template", "project", "parent", "status",
			"priority", "counterparty", "assignees", "auditors", "participants",
			"start-date", "end-date"))
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		want := map[string]any{
			"name":         "x",
			"template":     map[string]any{"id": 6},
			"project":      map[string]any{"id": 21},
			"parent":       map[string]any{"id": 31},
			"status":       map[string]any{"id": 2},
			"priority":     "Urgent",
			"counterparty": map[string]any{"id": "contact:4"},
			"assignees": map[string]any{
				"users":  []any{map[string]any{"id": "user:1"}},
				"groups": []any{map[string]any{"id": 3}},
			},
			"auditors": map[string]any{
				"users":  []any{map[string]any{"id": "contact:4"}},
				"groups": []any{},
			},
			"participants": map[string]any{
				"users":  []any{map[string]any{"id": "user:2"}},
				"groups": []any{},
			},
			"startDateTime": map[string]any{"date": "08-07-2026"},
			"endDateTime":   map[string]any{"date": "20-07-2026", "time": "18:00"},
		}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body = %#v, want %#v", body, want)
		}
	})

	t.Run("unset flags leave body untouched", func(t *testing.T) {
		f := &taskFields{template: 6, priority: "urgent"}
		body := map[string]any{"name": "x"}
		if err := f.apply(body, changedSet()); err != nil {
			t.Fatalf("apply: %v", err)
		}
		if !reflect.DeepEqual(body, map[string]any{"name": "x"}) {
			t.Errorf("body = %#v, want name only", body)
		}
	})

	t.Run("non-positive id flag", func(t *testing.T) {
		f := &taskFields{project: 0}
		err := f.apply(map[string]any{}, changedSet("project"))
		if err == nil || !strings.Contains(err.Error(), "--project must be a positive number") {
			t.Fatalf("err = %v, want --project positive error", err)
		}
	})

	t.Run("invalid priority propagates", func(t *testing.T) {
		f := &taskFields{priority: "sky-high"}
		err := f.apply(map[string]any{}, changedSet("priority"))
		if err == nil || !strings.Contains(err.Error(), "invalid priority") {
			t.Fatalf("err = %v, want invalid priority", err)
		}
	})

	t.Run("invalid counterparty propagates", func(t *testing.T) {
		f := &taskFields{counterparty: "user:1"}
		err := f.apply(map[string]any{}, changedSet("counterparty"))
		if err == nil || !strings.Contains(err.Error(), "invalid counterparty") {
			t.Fatalf("err = %v, want invalid counterparty", err)
		}
	})

	t.Run("invalid people ref propagates", func(t *testing.T) {
		f := &taskFields{auditors: []string{"12"}}
		err := f.apply(map[string]any{}, changedSet("auditors"))
		if err == nil || !strings.Contains(err.Error(), "invalid people reference") {
			t.Fatalf("err = %v, want invalid people reference", err)
		}
	})

	t.Run("invalid date propagates", func(t *testing.T) {
		f := &taskFields{startDate: "tomorrow"}
		err := f.apply(map[string]any{}, changedSet("start-date"))
		if err == nil || !strings.Contains(err.Error(), "invalid date") {
			t.Fatalf("err = %v, want invalid date", err)
		}
	})

	t.Run("empty people list rejected", func(t *testing.T) {
		f := &taskFields{}
		err := f.apply(map[string]any{}, changedSet("assignees"))
		if err == nil || !strings.Contains(err.Error(), "--assignees requires at least one reference") {
			t.Fatalf("err = %v, want --assignees requires at least one reference", err)
		}
	})
}

func TestTaskFieldsRegister(t *testing.T) {
	shared := []string{"project", "parent", "status", "priority", "counterparty",
		"assignees", "auditors", "participants", "start-date", "end-date"}

	withTemplate := &cobra.Command{}
	(&taskFields{}).register(withTemplate, true)
	for _, name := range append([]string{"template"}, shared...) {
		if withTemplate.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered with withTemplate=true", name)
		}
	}

	withoutTemplate := &cobra.Command{}
	(&taskFields{}).register(withoutTemplate, false)
	if withoutTemplate.Flags().Lookup("template") != nil {
		t.Error("flag --template must not be registered with withTemplate=false")
	}
	for _, name := range shared {
		if withoutTemplate.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered with withTemplate=false", name)
		}
	}
}
