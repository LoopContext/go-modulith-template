package validation

import (
	"context"
	"testing"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const testSuccessResponse = "success"

// mockRequestWithoutProto is a request that is not a proto message
type mockRequestWithoutProto struct {
	value string
}

func TestUnaryServerInterceptor_ValidRequest(t *testing.T) {
	interceptor := UnaryServerInterceptor()

	// Use a real proto message that should pass validation
	// Note: This test requires proto files to be generated first
	req := &authv1.GetProfileRequest{} // Empty request, should pass validation
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testSuccessResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Test/TestMethod",
	}

	resp, err := interceptor(context.Background(), req, info, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp != testSuccessResponse {
		t.Errorf("expected response 'success', got %v", resp)
	}
}

func TestUnaryServerInterceptor_InvalidRequest(t *testing.T) {
	interceptor := UnaryServerInterceptor()

	// Use a real proto message with invalid data
	// Create a request with invalid email format
	req := &authv1.ChangeEmailRequest{
		NewEmail: "invalid-email", // Invalid email format
	}
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "should not reach here", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Test/TestMethod",
	}

	resp, err := interceptor(context.Background(), req, info, handler)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if resp != nil {
		t.Errorf("expected nil response on validation error, got %v", resp)
	}

	// Check that it's an InvalidArgument error
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument code, got %v", st.Code())
	}

	if len(st.Message()) == 0 {
		t.Error("expected validation error message, got empty string")
	}
}

func TestUnaryServerInterceptor_RequestWithoutProto(t *testing.T) {
	interceptor := UnaryServerInterceptor()

	req := &mockRequestWithoutProto{value: "test"}
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testSuccessResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Test/TestMethod",
	}

	resp, err := interceptor(context.Background(), req, info, handler)
	if err != nil {
		t.Fatalf("expected no error for non-proto request, got %v", err)
	}

	if resp != testSuccessResponse {
		t.Errorf("expected response 'success', got %v", resp)
	}
}
