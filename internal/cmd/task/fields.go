package task

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
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
	if err != nil || id <= 0 || num != strconv.Itoa(id) {
		return nil, fmt.Errorf("invalid counterparty %q: use a contact id or contact:N", s)
	}
	if prefixed {
		return map[string]any{"id": s}, nil
	}
	return map[string]any{"id": id}, nil
}

// taskFields holds the typed field flags shared by `task create` and
// `task update`.
type taskFields struct {
	template     int
	project      int
	parent       int
	status       int
	priority     string
	counterparty string
	startDate    string
	endDate      string
	assignees    []string
	auditors     []string
	participants []string
}

// register adds the shared field flags to cmd. withTemplate controls the
// create-only --template flag (a task's template cannot be changed later).
func (f *taskFields) register(cmd *cobra.Command, withTemplate bool) {
	fl := cmd.Flags()
	if withTemplate {
		fl.IntVar(&f.template, "template", 0, "Task template ID")
	}
	fl.IntVar(&f.project, "project", 0, "Project ID")
	fl.IntVar(&f.parent, "parent", 0, "Parent task ID")
	fl.IntVar(&f.status, "status", 0, "Status ID")
	fl.StringVar(&f.priority, "priority", "", "Task priority: Urgent or NotUrgent")
	fl.StringVar(&f.counterparty, "counterparty", "", "Counterparty: a contact id or contact:N")
	fl.StringSliceVar(&f.assignees, "assignees", nil, "Assignees: user:N, contact:N, or group:N (comma-separated; replaces the list on update)")
	fl.StringSliceVar(&f.auditors, "auditors", nil, "Auditors: user:N, contact:N, or group:N (comma-separated; replaces the list on update)")
	fl.StringSliceVar(&f.participants, "participants", nil, "Participants: user:N, contact:N, or group:N (comma-separated; replaces the list on update)")
	fl.StringVar(&f.startDate, "start-date", "", `Start date: YYYY-MM-DD, "YYYY-MM-DD HH:MM", or YYYY-MM-DDTHH:MM (time is interpreted in the account timezone)`)
	fl.StringVar(&f.endDate, "end-date", "", `End date: YYYY-MM-DD, "YYYY-MM-DD HH:MM", or YYYY-MM-DDTHH:MM (time is interpreted in the account timezone)`)
}

// apply validates every flag reported set by `set` (cmd.Flags().Changed) and
// adds its Planfix body value to body. It returns the first validation error
// without touching the network.
func (f *taskFields) apply(body map[string]any, set func(string) bool) error {
	ids := []struct {
		flag string
		val  int
	}{
		{"template", f.template},
		{"project", f.project},
		{"parent", f.parent},
		{"status", f.status},
	}
	// set() is false for flags the command never registered (e.g. template on update),
	// so unregistered flags are skipped safely.
	for _, x := range ids {
		if !set(x.flag) {
			continue
		}
		if x.val <= 0 {
			return fmt.Errorf("--%s must be a positive number, got %d", x.flag, x.val)
		}
		body[x.flag] = map[string]any{"id": x.val}
	}
	if set("priority") {
		p, err := parsePriority(f.priority)
		if err != nil {
			return err
		}
		body["priority"] = p
	}
	if set("counterparty") {
		c, err := parseCounterparty(f.counterparty)
		if err != nil {
			return err
		}
		body["counterparty"] = c
	}
	people := []struct {
		flag string
		refs []string
	}{
		{"assignees", f.assignees},
		{"auditors", f.auditors},
		{"participants", f.participants},
	}
	for _, x := range people {
		if !set(x.flag) {
			continue
		}
		if len(x.refs) == 0 {
			return fmt.Errorf("--%s requires at least one reference", x.flag)
		}
		v, err := cmdutil.ParsePeople(x.refs)
		if err != nil {
			return err
		}
		body[x.flag] = v
	}
	dates := []struct {
		flag  string
		field string
		val   string
	}{
		{"start-date", "startDateTime", f.startDate},
		{"end-date", "endDateTime", f.endDate},
	}
	for _, x := range dates {
		if !set(x.flag) {
			continue
		}
		v, err := cmdutil.ParseTimePoint(x.val)
		if err != nil {
			return err
		}
		body[x.field] = v
	}
	return nil
}
