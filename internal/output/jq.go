package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

// CompileJQ parses a jq expression, wrapping a syntax error with pfix
// context. Callers pass a non-empty expression.
func CompileJQ(expr string) (*gojq.Query, error) {
	q, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid --jq expression: %w", err)
	}
	return q, nil
}

// EmitJSON writes raw to w, optionally filtered by a jq expression. When
// jqExpr is empty it is identical to JSON(w, raw) (plain pretty-print,
// including the non-JSON passthrough). When jqExpr is non-empty, raw must be
// valid JSON; the compiled query runs over the decoded value and each result
// is written on its own line: a string result raw/unquoted (jq -r style),
// anything else as compact JSON.
func EmitJSON(w io.Writer, raw []byte, jqExpr string) error {
	if jqExpr == "" {
		return JSON(w, raw)
	}
	q, err := CompileJQ(jqExpr)
	if err != nil {
		return err
	}
	// Decode with UseNumber so integer ids beyond 2^53 survive as json.Number
	// instead of being rounded to float64. gojq accepts json.Number input and
	// normalizes it to int/*big.Int, so large ids pass through exactly and
	// arithmetic still works.
	var input any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&input); err != nil {
		return fmt.Errorf("--jq: response is not valid JSON: %w", err)
	}
	iter := q.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if e, ok := v.(error); ok {
			return fmt.Errorf("--jq error: %w", e)
		}
		if s, ok := v.(string); ok {
			fmt.Fprintln(w, s)
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
	}
	return nil
}
