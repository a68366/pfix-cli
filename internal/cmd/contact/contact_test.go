package contact

import "testing"

func TestContactCmdRegistersGroups(t *testing.T) {
	cmd := NewCmd(nil)
	found := false
	for _, c := range cmd.Commands() {
		if c.Name() == "groups" {
			found = true
		}
	}
	if !found {
		t.Errorf("contact command missing subcommand %q", "groups")
	}
}
