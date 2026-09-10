// Package passworddb implements immutable, operator-owned SHA-1 corpus snapshots.
// It has no network, password input, or result-output responsibilities.
package passworddb

import (
	"context"
	"errors"
	"os"
)

// Error contains only a fixed safe classification, never paths or lookup keys.
type Error struct {
	Kind  string
	cause error
}

func (e *Error) Error() string  { return "offline password database: " + e.Kind }
func (e *Error) Unwrap() error  { return e.cause }
func problem(kind string) error { return &Error{Kind: kind} }

// SafeError removes arbitrary error text at the database/provider boundary.
func SafeError(err error) error {
	if err == nil {
		return nil
	}
	var own *Error
	if errors.As(err, &own) {
		return own
	}
	switch {
	case errors.Is(err, context.Canceled):
		return &Error{Kind: "cancelled", cause: context.Canceled}
	case errors.Is(err, context.DeadlineExceeded):
		return &Error{Kind: "timeout", cause: context.DeadlineExceeded}
	case errors.Is(err, os.ErrNotExist):
		return problem("missing")
	case errors.Is(err, os.ErrPermission):
		return problem("permission_denied")
	default:
		return problem("io_failure")
	}
}
func sanitize(err *error) { *err = SafeError(*err) }
