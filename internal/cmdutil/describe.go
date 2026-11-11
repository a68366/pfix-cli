package cmdutil

import (
	"errors"
	"net/http"

	"github.com/a68366/pfix-cli/internal/planfix"
)

// AuthContext says where the active credentials came from, so an auth hint can
// name the fix that actually applies.
type AuthContext struct {
	// Profile is the --profile flag value, "" when the flag was not given. A
	// bare `pfix auth login` already targets a profile chosen by PFIX_PROFILE or
	// current_profile, so only the flag needs repeating in the hint.
	Profile string
	// TokenFromEnv reports that PFIX_TOKEN is set. It overrides the profile's
	// token, so re-running `pfix auth login` would not change the outcome.
	TokenFromEnv bool
}

type hintKind int

const (
	hintUnknownToken hintKind = iota + 1 // app code 1
	hintScopeDenied                      // app code 5
	hintRejected                         // HTTP 401 without a recognized app code
)

// hintedError is a Planfix API error carrying an auth hint. It is a distinct
// type so an error that already has its hint is recognized and not described
// twice.
type hintedError struct {
	err  error
	kind hintKind
	ctx  AuthContext
}

func (e *hintedError) Error() string { return e.err.Error() + " — " + e.hint() }

func (e *hintedError) Unwrap() error { return e.err }

func (e *hintedError) hint() string {
	switch e.kind {
	case hintScopeDenied:
		return "the token is valid but lacks the scope for this action"
	case hintRejected:
		return "the token was rejected; " + e.relogin() + "; if it keeps failing, the account may no longer exist"
	default:
		return "the token was rejected; " + e.relogin()
	}
}

func (e *hintedError) relogin() string {
	if e.ctx.TokenFromEnv {
		return "PFIX_TOKEN is set and overrides the profile's token, so correct or unset it"
	}
	login := "pfix auth login"
	if e.ctx.Profile != "" {
		login += " --profile " + e.ctx.Profile
	}
	return "run `" + login + "` to re-authenticate"
}

// noHintError marks an error whose auth failure the command has already
// explained in its own terms; DescribeAPIError leaves it alone.
type noHintError struct{ err error }

func (e *noHintError) Error() string { return e.err.Error() }

func (e *noHintError) Unwrap() error { return e.err }

// WithoutAuthHint marks err so DescribeAPIError adds no auth hint to it — for
// a command whose rejected token is not the active profile's, such as the one
// `pfix auth login` was just given, where "run `pfix auth login`" would be
// wrong.
func WithoutAuthHint(err error) error {
	if err == nil {
		return nil
	}
	return &noHintError{err: err}
}

// DescribeAPIError augments a Planfix *APIError with an actionable hint for
// the auth failures: an unknown token (app code 1), a valid token that lacks
// the scope for the action (app code 5), and an HTTP 401 with no recognized
// app code — the API's answer when it rejects a token without its usual JSON
// envelope, as seen for an account that no longer exists. Any other error,
// including nil and a non-APIError, is returned unchanged, as is an error that
// already carries a hint (so the hint never appears twice) or was marked with
// WithoutAuthHint.
//
// The root command applies it to every command's error, so commands return
// API errors as they are and need not call it themselves.
func DescribeAPIError(err error, ctx AuthContext) error {
	var hinted *hintedError
	var noHint *noHintError
	if errors.As(err, &hinted) || errors.As(err, &noHint) {
		return err
	}
	var apiErr *planfix.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	var kind hintKind
	switch {
	case apiErr.Code == 1:
		kind = hintUnknownToken
	case apiErr.Code == 5:
		kind = hintScopeDenied
	case apiErr.StatusCode == http.StatusUnauthorized:
		kind = hintRejected
	default:
		return err
	}
	return &hintedError{err: err, kind: kind, ctx: ctx}
}
