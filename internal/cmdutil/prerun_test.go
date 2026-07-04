package cmdutil

import (
	"strings"
	"testing"
)

func TestGlobalOptsPreRun(t *testing.T) {
	tests := []struct {
		name       string
		jq         string
		wantJSON   bool
		wantErrHas string
	}{
		{name: "empty jq leaves json unchanged", jq: "", wantJSON: false},
		{name: "non-empty jq turns json on", jq: ".tasks[].id", wantJSON: true},
		{name: "invalid jq expression errors", jq: ".[", wantErrHas: "invalid --jq expression"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GlobalOpts{JQ: tt.jq}
			err := g.PreRun()
			if tt.wantErrHas != "" {
				if err == nil {
					t.Fatalf("PreRun() expected error containing %q, got nil", tt.wantErrHas)
				}
				if !strings.Contains(err.Error(), tt.wantErrHas) {
					t.Errorf("PreRun() error = %q, want it to contain %q", err.Error(), tt.wantErrHas)
				}
				return
			}
			if err != nil {
				t.Fatalf("PreRun() unexpected error: %v", err)
			}
			if g.JSON != tt.wantJSON {
				t.Errorf("PreRun() JSON = %v, want %v", g.JSON, tt.wantJSON)
			}
		})
	}
}
