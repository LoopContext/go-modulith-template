// Package errors provides domain error types and gRPC status code mapping.
//
//nolint:revive // Package name 'errors' is intentional for domain errors
package errors

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DomainError represents a domain-specific error with a type and message.
type DomainError struct {
	Type    ErrorType
	Message string
	Err     error
}

// ErrorType represents the category of error.
type ErrorType int

const (
	// ErrorTypeUnknown represents an unknown error type.
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeNotFound represents a resource not found error.
	ErrorTypeNotFound
	// ErrorTypeAlreadyExists represents a resource already exists error.
	ErrorTypeAlreadyExists
	// ErrorTypeValidation represents a validation error.
	ErrorTypeValidation
	// ErrorTypeUnauthorized represents an authentication error.
	ErrorTypeUnauthorized
	// ErrorTypeForbidden represents an authorization error.
	ErrorTypeForbidden
	// ErrorTypeConflict represents a conflict error.
	ErrorTypeConflict
	// ErrorTypeInternal represents an internal server error.
	ErrorTypeInternal
	// ErrorTypeUnavailable represents a service unavailable error.
	ErrorTypeUnavailable
)

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}

	return e.Message
}

// Unwrap returns the wrapped error.
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NotFound creates a new not found error.
func NotFound(message string) error {
	return &DomainError{
		Type:    ErrorTypeNotFound,
		Message: message,
	}
}

// NotFoundf creates a new not found error with formatting.
func NotFoundf(format string, args ...interface{}) error {
	return &DomainError{
		Type:    ErrorTypeNotFound,
		Message: fmt.Sprintf(format, args...),
	}
}

// AlreadyExists creates a new already exists error.
func AlreadyExists(message string) error {
	return &DomainError{
		Type:    ErrorTypeAlreadyExists,
		Message: message,
	}
}

// AlreadyExistsf creates a new already exists error with formatting.
func AlreadyExistsf(format string, args ...interface{}) error {
	return &DomainError{
		Type:    ErrorTypeAlreadyExists,
		Message: fmt.Sprintf(format, args...),
	}
}

// Validation creates a new validation error.
func Validation(message string) error {
	return &DomainError{
		Type:    ErrorTypeValidation,
		Message: message,
	}
}

// Validationf creates a new validation error with formatting.
func Validationf(format string, args ...interface{}) error {
	return &DomainError{
		Type:    ErrorTypeValidation,
		Message: fmt.Sprintf(format, args...),
	}
}

// Unauthorized creates a new unauthorized error.
func Unauthorized(message string) error {
	return &DomainError{
		Type:    ErrorTypeUnauthorized,
		Message: message,
	}
}

// Forbidden creates a new forbidden error.
func Forbidden(message string) error {
	return &DomainError{
		Type:    ErrorTypeForbidden,
		Message: message,
	}
}

// Conflict creates a new conflict error.
func Conflict(message string) error {
	return &DomainError{
		Type:    ErrorTypeConflict,
		Message: message,
	}
}

// Internal creates a new internal error wrapping another error.
func Internal(message string, err error) error {
	return &DomainError{
		Type:    ErrorTypeInternal,
		Message: message,
		Err:     err,
	}
}

// Internalf creates a new internal error with formatting.
func Internalf(err error, format string, args ...interface{}) error {
	return &DomainError{
		Type:    ErrorTypeInternal,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

// Unavailable creates a new unavailable error.
func Unavailable(message string, err error) error {
	return &DomainError{
		Type:    ErrorTypeUnavailable,
		Message: message,
		Err:     err,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return &DomainError{
			Type:    domainErr.Type,
			Message: message,
			Err:     err,
		}
	}

	return &DomainError{
		Type:    ErrorTypeInternal,
		Message: message,
		Err:     err,
	}
}

// ToGRPC converts a domain error to a gRPC status error.
//
//nolint:wrapcheck // This function intentionally returns unwrapped gRPC status errors
func ToGRPC(err error) error {
	if err == nil {
		return nil
	}

	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		// Not a domain error, return as internal error
		return status.Error(codes.Internal, "internal server error")
	}

	code := mapErrorTypeToGRPCCode(domainErr.Type)

	return status.Error(code, domainErr.Message)
}

// mapErrorTypeToGRPCCode maps domain error types to gRPC codes.
func mapErrorTypeToGRPCCode(errType ErrorType) codes.Code {
	switch errType {
	case ErrorTypeNotFound:
		return codes.NotFound
	case ErrorTypeAlreadyExists:
		return codes.AlreadyExists
	case ErrorTypeValidation:
		return codes.InvalidArgument
	case ErrorTypeUnauthorized:
		return codes.Unauthenticated
	case ErrorTypeForbidden:
		return codes.PermissionDenied
	case ErrorTypeConflict:
		return codes.Aborted
	case ErrorTypeUnavailable:
		return codes.Unavailable
	case ErrorTypeInternal:
		return codes.Internal
	default:
		return codes.Unknown
	}
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == ErrorTypeNotFound
	}

	return false
}

// IsAlreadyExists checks if an error is an already exists error.
func IsAlreadyExists(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == ErrorTypeAlreadyExists
	}

	return false
}

// IsValidation checks if an error is a validation error.
func IsValidation(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == ErrorTypeValidation
	}

	return false
}

