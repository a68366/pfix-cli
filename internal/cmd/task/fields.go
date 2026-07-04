package task

import (
	"fmt"
	"strconv"
	"strings"
)

// parsePriority validates the two-value Planfix priority enum
// case-insensitively and returns the canonical form the API expects.
// Validation is client-side because the API does not reject an unknown
// priority — it silently resets the field to NotUrgent.
func parsePriority(s string) (string, error) {
	switch strings.ToLower(s) {
	case "urgent":
		return "Urgent", nil
	case "noturgent":
		return "NotUrgent", nil
	}
	return "", fmt.Errorf("invalid priority %q: use Urgent or NotUrgent", s)
}

// parseCounterparty accepts a contact reference as a bare positive id (N) or
// the prefixed form contact:N, returning the Planfix counterparty value —
// {"id": N} for the bare form, {"id": "contact:N"} for the prefixed one.
func parseCounterparty(s string) (map[string]any, error) {
	num, prefixed := strings.CutPrefix(s, "contact:")
	id, err := strconv.Atoi(num)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid counterparty %q: use a contact id or contact:N", s)
	}
	if prefixed {
		return map[string]any{"id": s}, nil
	}
	return map[string]any{"id": id}, nil
}
