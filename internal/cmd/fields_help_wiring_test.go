package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// find walks the command tree to the leaf named by path (e.g. "task", "list").
func find(t *testing.T, root *cobra.Command, path ...string) *cobra.Command {
	t.Helper()
	cur := root
	for _, name := range path {
		var next *cobra.Command
		for _, c := range cur.Commands() {
			if c.Name() == name {
				next = c
				break
			}
		}
		if next == nil {
			t.Fatalf("command %v not found (missing %q)", path, name)
		}
		cur = next
	}
	return cur
}

// TestFieldsHelpWiring asserts every in-scope read command surfaces its default
// fields and an "Available fields" block in its Long help. Guards against an
// unwired command or an emptied available-fields constant.
func TestFieldsHelpWiring(t *testing.T) {
	root := NewRootCmd()
	cases := []struct {
		path      []string
		wantCount string   // e.g. "Available fields (42):"; empty skips the block check (see below)
		wantCF    bool     // custom-field note present
		wantNames []string // field names that must appear in the block (spec 1.8.2 additions)
	}{
		{[]string{"task", "list"}, "Available fields (42):", true, nil},
		{[]string{"task", "view"}, "Available fields (45):", true, nil},
		{[]string{"comment", "view"}, "Available fields (23):", false, nil},
		{[]string{"task", "comment", "list"}, "Available fields (23):", false, nil},
		{[]string{"task", "checklist", "list"}, "Available fields (6):", false, nil},
		{[]string{"task", "checklist", "view"}, "Available fields (6):", false, nil},
		{[]string{"project", "list"}, "Available fields (23):", false, nil},
		{[]string{"project", "view"}, "Available fields (23):", false, nil},
		{[]string{"contact", "list"}, "Available fields (32):", false, []string{"taxId"}},
		{[]string{"contact", "view"}, "Available fields (38):", false, []string{"taxId", "timezone"}},
		{[]string{"user", "list"}, "Available fields (27):", false, []string{"temporaryReplacement"}},
		{[]string{"user", "view"}, "Available fields (27):", false, []string{"temporaryReplacement"}},
		{[]string{"report", "list"}, "Available fields (3):", false, nil},
		{[]string{"report", "view"}, "Available fields (3):", false, nil},
		{[]string{"datatag", "list"}, "Available fields (4):", false, nil},
		{[]string{"datatag", "view"}, "Available fields (4):", false, nil},
		{[]string{"object", "list"}, "Available fields (32):", false, nil},
		{[]string{"object", "view"}, "Available fields (32):", false, nil},
		{[]string{"customfield", "list"}, "Available fields (11):", false, []string{"listValuesColors", "mainValue"}},
		{[]string{"customfield", "types"}, "Available fields (2):", false, nil},
		{[]string{"user", "groups"}, "Available fields (2):", false, nil},
		{[]string{"contact", "groups"}, "Available fields (2):", false, nil},
		{[]string{"user", "positions"}, "Available fields (2):", false, nil},
		{[]string{"task", "processes"}, "Available fields (2):", false, nil},
		{[]string{"contact", "processes"}, "Available fields (2):", false, nil},
		{[]string{"task", "statuses"}, "Available fields (7):", false, nil},
		{[]string{"task", "files"}, "Available fields (4):", false, nil},
		{[]string{"contact", "files"}, "Available fields (4):", false, nil},
		{[]string{"project", "files"}, "Available fields (4):", false, nil},
		// template list has no fixed vocabulary — the selectable fields depend on
		// the object type being listed — so it carries no "Available fields"
		// block, only the Default fields line.
		{[]string{"template", "list"}, "", false, nil},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.path, "/"), func(t *testing.T) {
			cmd := find(t, root, tc.path...)
			long := cmd.Long
			if !strings.Contains(long, "Default fields: ") {
				t.Errorf("%v Long missing Default fields line:\n%s", tc.path, long)
			}
			if tc.wantCount != "" && !strings.Contains(long, tc.wantCount) {
				t.Errorf("%v Long missing %q:\n%s", tc.path, tc.wantCount, long)
			}
			for _, name := range tc.wantNames {
				if !strings.Contains(long, name) {
					t.Errorf("%v Long missing field %q:\n%s", tc.path, name, long)
				}
			}
			hasCF := strings.Contains(long, "pfix customfield list task")
			if hasCF != tc.wantCF {
				t.Errorf("%v custom-field note = %v, want %v", tc.path, hasCF, tc.wantCF)
			}
		})
	}
}
