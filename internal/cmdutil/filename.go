package cmdutil

import (
	"fmt"
	"strings"
)

// SafeFileName validates an API-supplied file name for use as a local path
// segment. It rejects rather than sanitizes: names are attacker-influenced, and
// silently rewriting "../../etc/passwd" to "passwd" is a security trap. Unicode
// and spaces are allowed. Empty, ".", "..", and any name containing a path
// separator or NUL are rejected. Windows reserved device names (CON, PRN, …)
// are out of scope and not checked.
func SafeFileName(name string) (string, error) {
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("unsafe file name %q from the API: pass -o <path> to choose an output path", name)
	}
	if strings.ContainsAny(name, `/\`+"\x00") {
		return "", fmt.Errorf("unsafe file name %q from the API: pass -o <path> to choose an output path", name)
	}
	return name, nil
}
