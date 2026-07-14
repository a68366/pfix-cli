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
		wantCount string // e.g. "Available fields (42):"
		wantCF    bool   // custom-field note present
	}{
		{[]string{"task", "list"}, "Available fields (42):", true},
		{[]string{"task", "view"}, "Available fields (45):", true},
		{[]string{"project", "list"}, "Available fields (23):", false},
		{[]string{"project", "view"}, "Available fields (23):", false},
		{[]string{"contact", "list"}, "Available fields (31):", false},
		{[]string{"contact", "view"}, "Available fields (36):", false},
		{[]string{"user", "list"}, "Available fields (26):", false},
		{[]string{"user", "view"}, "Available fields (26):", false},
		{[]string{"report", "list"}, "Available fields (3):", false},
		{[]string{"report", "view"}, "Available fields (3):", false},
		{[]string{"datatag", "list"}, "Available fields (4):", false},
		{[]string{"datatag", "view"}, "Available fields (4):", false},
		{[]string{"object", "list"}, "Available fields (32):", false},
		{[]string{"object", "view"}, "Available fields (32):", false},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.path, "/"), func(t *testing.T) {
			cmd := find(t, root, tc.path...)
			long := cmd.Long
			if !strings.Contains(long, "Default fields: ") {
				t.Errorf("%v Long missing Default fields line:\n%s", tc.path, long)
			}
			if !strings.Contains(long, tc.wantCount) {
				t.Errorf("%v Long missing %q:\n%s", tc.path, tc.wantCount, long)
			}
			hasCF := strings.Contains(long, "pfix customfield list task")
			if hasCF != tc.wantCF {
				t.Errorf("%v custom-field note = %v, want %v", tc.path, hasCF, tc.wantCF)
			}
		})
	}
}
