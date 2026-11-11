package cmdutil

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/a68366/pfix-cli/internal/planfix"
)

// CheckResult fails a 2xx response whose body reports a result other than
// "success". Some endpoints answer HTTP 200 with {"result":"fail","code":N,...}
// (deleting an already-deleted file does), which planfix.Client.JSON passes
// through as success. The error is a *planfix.APIError carrying the app code,
// so code-specific hints apply to it exactly as to a 4xx. A body that is not a
// JSON object, or that has no "result" key, is not judged.
func CheckResult(raw []byte) error {
	var probe struct {
		Result *string `json:"result"`
	}
	if json.Unmarshal(raw, &probe) != nil || probe.Result == nil || *probe.Result == "success" {
		return nil
	}
	apiErr := planfix.ParseError(http.StatusOK, raw)
	if apiErr.Message == "" {
		apiErr.Message = fmt.Sprintf("result %q", *probe.Result)
	}
	return apiErr
}
