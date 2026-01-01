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
	Code    ErrorCode
	Message string
	Err     error
	Details map[string]string
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

// ErrorCode represents a stable, machine-readable error code for API clients.
// These codes are documented and remain stable across API versions.
type ErrorCode string

// Common error codes for API responses.
// Format: DOMAIN_ACTION_REASON (e.g., AUTH_LOGIN_INVALID_CREDENTIALS)
//
//nolint:gosec // These are error code constants, not credentials
const (
	// General errors
	CodeUnknown          ErrorCode = "UNKNOWN"
	CodeInternalError    ErrorCode = "INTERNAL_ERROR"
	CodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeUnavailable      ErrorCode = "SERVICE_UNAVAILABLE"

	// Authentication errors
	CodeAuthRequired         ErrorCode = "AUTH_REQUIRED"
	CodeAuthInvalidToken     ErrorCode = "AUTH_INVALID_TOKEN"
	CodeAuthTokenExpired     ErrorCode = "AUTH_TOKEN_EXPIRED"
	CodeAuthInvalidCreds     ErrorCode = "AUTH_INVALID_CREDENTIALS"
	CodeAuthSessionExpired   ErrorCode = "AUTH_SESSION_EXPIRED"
	CodeAuthSessionRevoked   ErrorCode = "AUTH_SESSION_REVOKED"
	CodeAuthMagicCodeExpired ErrorCode = "AUTH_MAGIC_CODE_EXPIRED"
	CodeAuthMagicCodeInvalid ErrorCode = "AUTH_MAGIC_CODE_INVALID"

	// Authorization errors
	CodeForbidden              ErrorCode = "FORBIDDEN"
	CodeInsufficientPermission ErrorCode = "INSUFFICIENT_PERMISSION"
	CodeNotOwner               ErrorCode = "NOT_RESOURCE_OWNER"

	// User errors
	CodeUserNotFound     ErrorCode = "USER_NOT_FOUND"
	CodeUserAlreadyExist ErrorCode = "USER_ALREADY_EXISTS"
	CodeUserSuspended    ErrorCode = "USER_SUSPENDED"

	// Rate limiting errors
	CodeRateLimited ErrorCode = "RATE_LIMITED"
	CodeQuotaExceed ErrorCode = "QUOTA_EXCEEDED"
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

// GetCode returns the error code, or a default based on the error type.
func (e *DomainError) GetCode() ErrorCode {
	if e.Code != "" {
		return e.Code
	}

	// Return default code based on type
	return mapErrorTypeToCode(e.Type)
}

// WithCode returns a copy of the error with the specified error code.
func (e *DomainError) WithCode(code ErrorCode) *DomainError {
	return &DomainError{
		Type:    e.Type,
		Code:    code,
		Message: e.Message,
		Err:     e.Err,
		Details: e.Details,
	}
}

// WithDetails returns a copy of the error with additional details.
func (e *DomainError) WithDetails(details map[string]string) *DomainError {
	return &DomainError{
		Type:    e.Type,
		Code:    e.Code,
		Message: e.Message,
		Err:     e.Err,
		Details: details,
	}
}

// mapErrorTypeToCode maps error types to default error codes.
func mapErrorTypeToCode(errType ErrorType) ErrorCode {
	switch errType {
	case ErrorTypeNotFound:
		return CodeNotFound
	case ErrorTypeAlreadyExists:
		return CodeAlreadyExists
	case ErrorTypeValidation:
		return CodeValidationFailed
	case ErrorTypeUnauthorized:
		return CodeAuthRequired
	case ErrorTypeForbidden:
		return CodeForbidden
	case ErrorTypeConflict:
		return CodeConflict
	case ErrorTypeUnavailable:
		return CodeUnavailable
	case ErrorTypeInternal:
		return CodeInternalError
	default:
		return CodeUnknown
	}
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

// WithCode creates a new error with a specific error code.
// This is useful for more specific error identification by API clients.
func WithCode(code ErrorCode, message string) error {
	errType := mapCodeToErrorType(code)

	return &DomainError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// WithCodeAndError creates a new error with a specific error code and wrapped error.
func WithCodeAndError(code ErrorCode, message string, err error) error {
	errType := mapCodeToErrorType(code)

	return &DomainError{
		Type:    errType,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// mapCodeToErrorType maps error codes to error types.
func mapCodeToErrorType(code ErrorCode) ErrorType {
	switch code {
	case CodeNotFound, CodeUserNotFound:
		return ErrorTypeNotFound
	case CodeAlreadyExists, CodeUserAlreadyExist:
		return ErrorTypeAlreadyExists
	case CodeValidationFailed:
		return ErrorTypeValidation
	case CodeAuthRequired, CodeAuthInvalidToken, CodeAuthTokenExpired,
		CodeAuthInvalidCreds, CodeAuthSessionExpired, CodeAuthSessionRevoked,
		CodeAuthMagicCodeExpired, CodeAuthMagicCodeInvalid:
		return ErrorTypeUnauthorized
	case CodeForbidden, CodeInsufficientPermission, CodeNotOwner:
		return ErrorTypeForbidden
	case CodeConflict:
		return ErrorTypeConflict
	case CodeUnavailable:
		return ErrorTypeUnavailable
	case CodeRateLimited, CodeQuotaExceed:
		return ErrorTypeForbidden // Rate limiting is a form of access denial
	default:
		return ErrorTypeInternal
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
// The error code is included in the message format: "[CODE] message"
//
//nolint:wrapcheck // This function intentionally returns unwrapped gRPC status errors
func ToGRPC(err error) error {
	if err == nil {
		return nil
	}

	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		// Not a domain error, return as internal error
		return status.Error(codes.Internal, "[INTERNAL_ERROR] internal server error")
	}

	grpcCode := mapErrorTypeToGRPCCode(domainErr.Type)
	errorCode := domainErr.GetCode()

	// Format: [ERROR_CODE] message
	message := fmt.Sprintf("[%s] %s", errorCode, domainErr.Message)

	return status.Error(grpcCode, message)
}

// ToGRPCWithDetails converts a domain error to a gRPC status error with details.
// Use this when you need to include additional error information.
//
//nolint:wrapcheck // This function intentionally returns unwrapped gRPC status errors
func ToGRPCWithDetails(err error) error {
	// For now, use the same as ToGRPC
	// In the future, we can add google.rpc.ErrorInfo or other details
	return ToGRPC(err)
}

// GetErrorCode extracts the error code from an error, or returns CodeUnknown.
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		return ""
	}

	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.GetCode()
	}

	return CodeUnknown
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

