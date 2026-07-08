package cmdutil

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/a68366/pfix-cli/internal/planfix"
)

// CustomFieldSpec is one parsed `--cf <id>=<value>` entry: the numeric custom
// field id and its raw (unformatted) value.
type CustomFieldSpec struct {
	ID    int
	Value string
}

// ParseCustomFieldSpecs parses raw "id=value" flag values into specs. It splits
// on the first '=' only (so a value may contain '=' or ','), validates a
// positive-integer id, and rejects a missing '=', a bad id, or a duplicate id.
// Validation is structural and offline — value typing happens later against the
// field's definition (see BuildCustomFieldData).
func ParseCustomFieldSpecs(raw []string) ([]CustomFieldSpec, error) {
	specs := make([]CustomFieldSpec, 0, len(raw))
	seen := make(map[int]bool, len(raw))
	for _, s := range raw {
		idStr, value, found := strings.Cut(s, "=")
		if !found {
			return nil, fmt.Errorf("invalid --cf %q: use <id>=<value>", s)
		}
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 || idStr != strconv.Itoa(id) {
			return nil, fmt.Errorf("invalid --cf %q: id must be a positive number", s)
		}
		if seen[id] {
			return nil, fmt.Errorf("invalid --cf %q: custom field %d given more than once", s, id)
		}
		seen[id] = true
		specs = append(specs, CustomFieldSpec{ID: id, Value: value})
	}
	return specs, nil
}

// BuildCustomFieldData resolves each spec's value shape from the object type's
// field definitions and returns the Planfix customFieldData array. It issues one
// GET /customfield/<objectType>?fields=id,type, then formats each spec by its
// field's numeric type code. Errors (before any write) on an unknown id, a value
// that doesn't fit its field's type, or an unsupported type code. objectType is
// an internal literal ("task"), URL-safe by construction.
func BuildCustomFieldData(ctx context.Context, c *planfix.Client, objectType string, specs []CustomFieldSpec) ([]map[string]any, error) {
	path := "customfield/" + objectType + "?fields=" + url.QueryEscape("id,type")
	raw, err := c.JSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		CustomFields []struct {
			ID   int `json:"id"`
			Type int `json:"type"`
		} `json:"customfields"`
	}
	if err := DecodeJSON(raw, &env); err != nil {
		return nil, err
	}
	typeByID := make(map[int]int, len(env.CustomFields))
	for _, cf := range env.CustomFields {
		typeByID[cf.ID] = cf.Type
	}
	data := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		typeCode, ok := typeByID[spec.ID]
		if !ok {
			return nil, fmt.Errorf("no custom field %d for %s", spec.ID, objectType)
		}
		entry, err := formatCF(spec, typeCode)
		if err != nil {
			return nil, err
		}
		data = append(data, entry)
	}
	return data, nil
}

// formatCF turns one spec into a customFieldData entry, shaping the value by the
// field's numeric type code: 0/2 (text) -> string; 1 (number) -> JSON number;
// 8 (list/enum) -> {"id": optionId}. Unsupported codes error.
func formatCF(spec CustomFieldSpec, typeCode int) (map[string]any, error) {
	field := map[string]any{"id": spec.ID}
	switch typeCode {
	case 0, 2: // short text, multiline text
		return map[string]any{"field": field, "value": spec.Value}, nil
	case 1: // number
		n, err := strconv.ParseFloat(spec.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid --cf %d=%q: field %d is a number", spec.ID, spec.Value, spec.ID)
		}
		return map[string]any{"field": field, "value": n}, nil
	case 8: // list / enum
		opt, err := strconv.Atoi(spec.Value)
		if err != nil || opt <= 0 || spec.Value != strconv.Itoa(opt) {
			return nil, fmt.Errorf("invalid --cf %d=%q: field %d is a list; give an option id", spec.ID, spec.Value, spec.ID)
		}
		return map[string]any{"field": field, "value": map[string]any{"id": opt}}, nil
	default:
		return nil, fmt.Errorf("custom field %d has unsupported type %d; set it via 'pfix api'", spec.ID, typeCode)
	}
}
