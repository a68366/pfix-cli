package cmdutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/planfix"
)

func TestCheckResult(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantErr  bool
		wantCode int
		wantMsg  string
	}{
		{"success", `{"result":"success"}`, false, 0, ""},
		{"no result key", `{"id":3}`, false, 0, ""},
		{"not json", `<html></html>`, false, 0, ""},
		{"empty body", ``, false, 0, ""},
		{"fail with code", `{"result":"fail","code":7000,"error":"File by id - 9 has already been deleted"}`, true, 7000, "already been deleted"},
		{"fail without message", `{"result":"fail"}`, true, 0, `result "fail"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckResult([]byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckResult(%q) = %v, wantErr %v", tt.body, err, tt.wantErr)
			}
			if !tt.wantErr {
				return
			}
			var apiErr *planfix.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error %T is not *planfix.APIError", err)
			}
			if apiErr.StatusCode != 200 || apiErr.Code != tt.wantCode {
				t.Errorf("status/code = %d/%d, want 200/%d", apiErr.StatusCode, apiErr.Code, tt.wantCode)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantMsg)
			}
		})
	}
}
