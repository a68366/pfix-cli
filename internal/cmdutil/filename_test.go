package cmdutil

import "testing"

func TestSafeFileNameAccepts(t *testing.T) {
	for _, name := range []string{"report.pdf", "héllo 日.pdf", "a b c.txt"} {
		got, err := SafeFileName(name)
		if err != nil {
			t.Errorf("SafeFileName(%q) unexpected error: %v", name, err)
		}
		if got != name {
			t.Errorf("SafeFileName(%q) = %q, want unchanged", name, got)
		}
	}
}

func TestSafeFileNameRejects(t *testing.T) {
	for _, name := range []string{"", ".", "..", "a/b", `a\b`, "../../etc/passwd", "x\x00y"} {
		if _, err := SafeFileName(name); err == nil {
			t.Errorf("SafeFileName(%q) = nil error, want rejection", name)
		}
	}
}
