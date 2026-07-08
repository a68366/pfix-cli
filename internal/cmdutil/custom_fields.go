package cmdutil

import (
	"context"
	"fmt"
	"net/url"
	"slices"
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

// customFieldDef is one custom-field definition from GET /customfield/<type>:
// its numeric id, display name, type code, and — for a list field — the option
// labels it accepts.
type customFieldDef struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Type       int      `json:"type"`
	EnumValues []string `json:"enumValues"`
}

// cfDefFields are the definition fields needed to type a value. enumValues is
// required: a list field is addressed by option label, so its allowed labels
// must be known before the write (see formatCF).
const cfDefFields = "id,name,type,enumValues"

// BuildCustomFieldData resolves each spec's value shape from the object type's
// field definitions and returns the Planfix customFieldData array. It issues one
// GET /customfield/<objectType>, then formats each spec by its field's numeric
// type code. Errors (before any write) on an unknown id, a value that doesn't
// fit its field's type, an option label outside a list field's set, or an
// unsupported type code. objectType is an internal literal ("task"), URL-safe by
// construction.
func BuildCustomFieldData(ctx context.Context, c *planfix.Client, objectType string, specs []CustomFieldSpec) ([]map[string]any, error) {
	path := "customfield/" + objectType + "?fields=" + url.QueryEscape(cfDefFields)
	raw, err := c.JSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		CustomFields []customFieldDef `json:"customfields"`
	}
	if err := DecodeJSON(raw, &env); err != nil {
		return nil, err
	}
	defByID := make(map[int]customFieldDef, len(env.CustomFields))
	for _, cf := range env.CustomFields {
		defByID[cf.ID] = cf
	}
	data := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		def, ok := defByID[spec.ID]
		if !ok {
			return nil, fmt.Errorf("no custom field %d for %s", spec.ID, objectType)
		}
		entry, err := formatCF(spec, def)
		if err != nil {
			return nil, err
		}
		data = append(data, entry)
	}
	return data, nil
}

// formatCF turns one spec into a customFieldData entry, shaping the value by the
// field's numeric type code: 0/2 (text) -> string; 1 (number) -> JSON number;
// 8 (list) -> the option label, as a bare string. Unsupported codes error.
//
// A list field is addressed by its option *label*, not by a numeric id: the REST
// API exposes a list's options only as strings (enumValues) and never as ids.
// It also does no validation of its own — an unrecognized value is accepted and
// stored verbatim, so `--cf <id>=4` on a list would silently store the text "4"
// rather than select an option. pfix therefore checks the label up front.
func formatCF(spec CustomFieldSpec, def customFieldDef) (map[string]any, error) {
	field := map[string]any{"id": spec.ID}
	switch def.Type {
	case 0, 2: // short text, multiline text
		return map[string]any{"field": field, "value": spec.Value}, nil
	case 1: // number
		n, err := strconv.ParseFloat(spec.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid --cf %d=%q: field %d is a number", spec.ID, spec.Value, spec.ID)
		}
		return map[string]any{"field": field, "value": n}, nil
	case 8: // list
		if len(def.EnumValues) == 0 {
			return nil, fmt.Errorf("custom field %d %q is a list with no options defined; set it via 'pfix api'", spec.ID, def.Name)
		}
		if !slices.Contains(def.EnumValues, spec.Value) {
			return nil, fmt.Errorf("invalid --cf %d=%q: field %d %q has no such option; valid options: %s",
				spec.ID, spec.Value, spec.ID, def.Name, quoteAll(def.EnumValues))
		}
		return map[string]any{"field": field, "value": spec.Value}, nil
	default:
		return nil, fmt.Errorf("custom field %d has unsupported type %d; set it via 'pfix api'", spec.ID, def.Type)
	}
}

// quoteAll renders option labels as a comma-separated list of quoted strings.
// Labels may themselves contain spaces or commas, so an unquoted join would not
// show where one option ends and the next begins.
func quoteAll(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	return strings.Join(quoted, ", ")
}
