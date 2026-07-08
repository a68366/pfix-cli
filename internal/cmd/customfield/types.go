package customfield

import (
	"fmt"
	"strconv"
)

// typeNames maps the Planfix custom-field type code to its display name.
// Source: GET /customfield/type (a stable system catalog). Codes 18 and 19 are
// intentionally absent (the API skips them).
var typeNames = map[int]string{
	0:  "Short text",
	1:  "Number",
	2:  "Multi-line text",
	3:  "Date",
	4:  "Time",
	5:  "Date and time",
	6:  "Period of time",
	7:  "Checkbox",
	8:  "List",
	9:  "Directory entry",
	10: "Contact",
	11: "Employee",
	12: "Counterparty",
	13: "Group, employee, or contact",
	14: "List of users",
	15: "Set of directory values",
	16: "Task",
	17: "Task set",
	20: "Set of values",
	21: "Files",
	22: "Project",
	23: "Data tag summaries",
	24: "Calculated field",
	25: "Location",
	26: "Subtask total",
	27: "AI results field",
	28: "Date with time frame",
	29: "Totals field",
}

// typeName renders a custom-field type code (a float64 from decoded JSON) as its
// catalog name, falling back to the raw number for an unknown code and to a
// plain string for any non-numeric value.
func typeName(v any) string {
	f, ok := v.(float64)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	code := int(f)
	if n, ok := typeNames[code]; ok {
		return n
	}
	return strconv.Itoa(code)
}
