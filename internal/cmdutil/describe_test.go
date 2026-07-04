package cmdutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/planfix"
)

func TestDescribeAPIError(t *testing.T) {
	t.Run("code 1 unknown token adds a re-login hint", func(t *testing.T) {
		in := &planfix.APIError{StatusCode: 401, Code: 1, Message: "Access denied, unknown token"}
		got := DescribeAPIError(in)
		msg := got.Error()
		if !strings.Contains(msg, "Access denied, unknown token") {
			t.Errorf("hint dropped the original message: %q", msg)
		}
		if !strings.Contains(msg, "auth login") {
			t.Errorf("code 1 should point at `pfix auth login`: %q", msg)
		}
	})

	t.Run("code 5 scope denied explains the token is valid", func(t *testing.T) {
		in := &planfix.APIError{StatusCode: 405, Code: 5, Message: "Scope denied, method not allowed"}
		msg := DescribeAPIError(in).Error()
		if !strings.Contains(msg, "Scope denied") {
			t.Errorf("hint dropped the original message: %q", msg)
		}
		if !strings.Contains(msg, "scope") {
			t.Errorf("code 5 should mention scope: %q", msg)
		}
		if strings.Contains(msg, "auth login") {
			t.Errorf("code 5 is not a login problem: %q", msg)
		}
	})

	t.Run("other app codes pass through unchanged", func(t *testing.T) {
		in := &planfix.APIError{StatusCode: 400, Code: 1000, Message: "Task not found by id - 9"}
		got := DescribeAPIError(in)
		if got.Error() != in.Error() {
			t.Errorf("code 1000 was modified: %q", got.Error())
		}
	})

	t.Run("non-API errors pass through unchanged", func(t *testing.T) {
		in := errors.New("connection refused")
		got := DescribeAPIError(in)
		if got != in {
			t.Errorf("wrapped a non-API error: %v", got)
		}
	})
}
