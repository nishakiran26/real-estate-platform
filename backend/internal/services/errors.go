// Package services holds the business logic. Services depend on repository
// interfaces, never on Echo or the database driver, so they stay testable and
// the same logic can be reused by a future gRPC or worker entrypoint.
package services

import (
	"errors"
	"fmt"
	"net/http"
)

// Kind classifies a service error so the HTTP layer can pick a status code
// without inspecting error strings.
type Kind int

const (
	KindInternal Kind = iota
	KindInvalid
	KindNotFound
	KindConflict
	KindForbidden
	KindUnauthorized
	KindTooMany
)

// Error is the single error type crossing the service boundary.
type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// HTTPStatus maps an error to its response code; unknown errors become 500.
func HTTPStatus(err error) int {
	var se *Error
	if !errors.As(err, &se) {
		return http.StatusInternalServerError
	}
	switch se.Kind {
	case KindInvalid:
		return http.StatusBadRequest
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindForbidden:
		return http.StatusForbidden
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindTooMany:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// Message returns a client-safe message. Internal errors are deliberately
// generic so implementation details never leak to API consumers.
func Message(err error) string {
	var se *Error
	if errors.As(err, &se) && se.Kind != KindInternal {
		return se.Message
	}
	return "An unexpected error occurred"
}

// Constructors for each error kind.
func Invalid(msg string) error      { return &Error{Kind: KindInvalid, Message: msg} }
func NotFound(msg string) error     { return &Error{Kind: KindNotFound, Message: msg} }
func Conflict(msg string) error     { return &Error{Kind: KindConflict, Message: msg} }
func Forbidden(msg string) error    { return &Error{Kind: KindForbidden, Message: msg} }
func Unauthorized(msg string) error { return &Error{Kind: KindUnauthorized, Message: msg} }
func TooMany(msg string) error      { return &Error{Kind: KindTooMany, Message: msg} }

// Internal wraps an unexpected failure, preserving the cause for the logs.
func Internal(msg string, err error) error {
	return &Error{Kind: KindInternal, Message: msg, Err: err}
}
