package cmdutil

import (
	"strings"
	"testing"
)

func TestFieldsHelp(t *testing.T) {
	t.Run("default only when available empty", func(t *testing.T) {
		got := FieldsHelp("List reports", "id,name", "", "")
		if !strings.HasPrefix(got, "List reports\n") {
			t.Errorf("want short description as first line, got:\n%s", got)
		}
		if !strings.Contains(got, "Default fields: id,name") {
			t.Errorf("want default fields line, got:\n%s", got)
		}
		if strings.Contains(got, "Available fields") {
			t.Errorf("want no Available block when available is empty, got:\n%s", got)
		}
		if strings.Contains(got, "customfield list") {
			t.Errorf("want no custom-field note when cfResource is empty, got:\n%s", got)
		}
	})

	t.Run("available block shows count", func(t *testing.T) {
		got := FieldsHelp("View a report", "id,name,fields", "id,name,fields", "")
		if !strings.Contains(got, "Available fields (3):") {
			t.Errorf("want Available fields (3), got:\n%s", got)
		}
		if !strings.Contains(got, "id, name, fields") {
			t.Errorf("want space-separated field list, got:\n%s", got)
		}
	})

	t.Run("custom-field note references resource", func(t *testing.T) {
		got := FieldsHelp("List tasks", "id,name", "id,name,status", "task")
		if !strings.Contains(got, "pfix customfield list task") {
			t.Errorf("want custom-field note for task, got:\n%s", got)
		}
	})

	t.Run("long list wraps with indented continuation", func(t *testing.T) {
		fields := make([]string, 0, 40)
		for i := 0; i < 40; i++ {
			fields = append(fields, "fieldNameNumber")
		}
		avail := strings.Join(fields, ",")
		got := FieldsHelp("List tasks", "id,name", avail, "")
		if !strings.Contains(got, "\n  ") {
			t.Errorf("want a two-space-indented continuation line, got:\n%s", got)
		}
		for _, line := range strings.Split(got, "\n") {
			if len(line) > 80 {
				t.Errorf("line exceeds wrap budget (%d): %q", len(line), line)
			}
		}
	})
}
