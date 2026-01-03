//nolint:revive // Package name 'errors' is intentional for domain errors test
package errors

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDomainError(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		err := NotFound("user not found")
		if err.Error() != "user not found" {
			t.Errorf("expected 'user not found', got '%s'", err.Error())
		}

		if !IsNotFound(err) {
			t.Error("expected IsNotFound to be true")
		}
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		err := AlreadyExists("user already exists")
		if !IsAlreadyExists(err) {
			t.Error("expected IsAlreadyExists to be true")
		}
	})

	t.Run("Validation", func(t *testing.T) {
		err := Validationf("invalid email: %s", "test")
		if !IsValidation(err) {
			t.Error("expected IsValidation to be true")
		}
	})

	t.Run("Internal with wrapped error", func(t *testing.T) {
		baseErr := errors.New("database connection failed")
		err := Internal("failed to query", baseErr)

		if !errors.Is(err, baseErr) {
			t.Error("expected wrapped error to be retrievable")
		}
	})
}

//nolint:funlen // Test table is necessarily long
func TestToGRPC(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "not found",
			err:          NotFound("resource not found"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "already exists",
			err:          AlreadyExists("resource exists"),
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "validation",
			err:          Validation("invalid input"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "unauthorized",
			err:          Unauthorized("not authenticated"),
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "forbidden",
			err:          Forbidden("access denied"),
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "conflict",
			err:          Conflict("resource conflict"),
			expectedCode: codes.Aborted,
		},
		{
			name:         "internal",
			err:          Internal("internal error", errors.New("db error")),
			expectedCode: codes.Internal,
		},
		{
			name:         "unavailable",
			err:          Unavailable("service unavailable", nil),
			expectedCode: codes.Unavailable,
		},
		{
			name:         "non-domain error",
			err:          errors.New("some error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := ToGRPC(tt.err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}

			if st.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
			}
		})
	}
}

func TestWrap(t *testing.T) {
	t.Run("wrap domain error", func(t *testing.T) {
		baseErr := NotFound("user not found")
		wrapped := Wrap(baseErr, "failed to get user")

		if !IsNotFound(wrapped) {
			t.Error("expected wrapped error to preserve NotFound type")
		}
	})

	t.Run("wrap non-domain error", func(t *testing.T) {
		baseErr := errors.New("some error")
		wrapped := Wrap(baseErr, "operation failed")

		var domainErr *DomainError
		if !errors.As(wrapped, &domainErr) {
			t.Error("expected wrapped error to be DomainError")
		}

		if domainErr.Type != ErrorTypeInternal {
			t.Error("expected wrapped non-domain error to be Internal type")
		}
	})

	t.Run("wrap nil error", func(t *testing.T) {
		wrapped := Wrap(nil, "some message")
		if wrapped != nil {
			t.Error("expected nil when wrapping nil error")
		}
	})
}
