package errorbank

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
)

// Kind enumerates supported application error categories.
type Kind string

const (
	KindBadRequest          Kind = "bad_request"
	KindConflict            Kind = "conflict"
	KindNotFound            Kind = "not_found"
	KindUnprocessableEntity Kind = "unprocessable_entity"
	KindInternal            Kind = "internal"
)

// AppError captures rich error context shared across transports.
type AppError struct {
	kind    Kind
	message string
	details map[string]any
	cause   error
}

// Option mutates an AppError during construction.
type Option func(*AppError)

// WithCause attaches an underlying error.
func WithCause(err error) Option {
	return func(appErr *AppError) {
		appErr.cause = err
	}
}

// WithDetail adds a single named detail value.
func WithDetail(key string, value any) Option {
	return func(appErr *AppError) {
		if appErr.details == nil {
			appErr.details = make(map[string]any)
		}
		appErr.details[key] = value
	}
}

// WithDetails merges multiple detail values.
func WithDetails(details map[string]any) Option {
	return func(appErr *AppError) {
		if len(details) == 0 {
			return
		}
		if appErr.details == nil {
			appErr.details = make(map[string]any)
		}
		for k, v := range details {
			appErr.details[k] = v
		}
	}
}

// New constructs a new AppError with the supplied kind and message.
func New(kind Kind, message string, opts ...Option) *AppError {
	if message == "" {
		message = string(kind)
	}
	appErr := &AppError{kind: kind, message: message}
	for _, opt := range opts {
		opt(appErr)
	}
	return appErr
}

// Error satisfies the error interface.
func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Unwrap exposes the wrapped cause for errors.Is/errors.As.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Kind returns the error category.
func (e *AppError) Kind() Kind {
	if e == nil {
		return KindInternal
	}
	return e.kind
}

// Message returns the human-readable message.
func (e *AppError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// Details returns optional metadata about the error.
func (e *AppError) Details() map[string]any {
	if e == nil {
		return nil
	}
	return e.details
}

// StatusCode resolves the HTTP status for the error kind.
func (e *AppError) StatusCode() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	switch e.kind {
	case KindBadRequest:
		return http.StatusBadRequest
	case KindConflict:
		return http.StatusConflict
	case KindNotFound:
		return http.StatusNotFound
	case KindUnprocessableEntity:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// GRPCCode maps the error kind onto a gRPC status code.
func (e *AppError) GRPCCode() codes.Code {
	if e == nil {
		return codes.Internal
	}
	switch e.kind {
	case KindBadRequest:
		return codes.InvalidArgument
	case KindConflict:
		return codes.AlreadyExists
	case KindNotFound:
		return codes.NotFound
	case KindUnprocessableEntity:
		return codes.FailedPrecondition
	default:
		return codes.Internal
	}
}

// BadRequest constructs a 400 error.
func BadRequest(message string, opts ...Option) *AppError {
	return New(KindBadRequest, message, opts...)
}

// Conflict constructs a 409 error.
func Conflict(message string, opts ...Option) *AppError {
	return New(KindConflict, message, opts...)
}

// NotFound constructs a 404 error.
func NotFound(message string, opts ...Option) *AppError {
	return New(KindNotFound, message, opts...)
}

// Unprocessable constructs a 422 error.
func Unprocessable(message string, opts ...Option) *AppError {
	return New(KindUnprocessableEntity, message, opts...)
}

// Internal constructs a generic 500 error.
func Internal(message string, opts ...Option) *AppError {
	return New(KindInternal, message, opts...)
}

// From returns an AppError for any error input, wrapping unexpected values.
func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("internal error", WithCause(err))
}
