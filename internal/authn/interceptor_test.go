package authn

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testResponse = "response"

type mockVerifier struct {
	verifyFunc func(ctx context.Context, token string) (*Claims, error)
}

func (m *mockVerifier) VerifyToken(ctx context.Context, token string) (*Claims, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, token)
	}

	return &Claims{UserID: "user-123", Role: "user"}, nil
}

func TestUnaryServerInterceptor_PublicMethod(t *testing.T) {
	cfg := InterceptorConfig{
		Verifier: nil,
		PublicMethods: map[string]struct{}{
			"/test.Service/PublicMethod": {},
		},
	}

	interceptor := UnaryServerInterceptor(cfg)

	called := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		called = true
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/PublicMethod",
	}

	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !called {
		t.Error("expected handler to be called")
	}

	if resp != "response" {
		t.Errorf("expected response 'response', got %v", resp)
	}
}

func TestUnaryServerInterceptor_NoVerifier(t *testing.T) {
	cfg := InterceptorConfig{
		Verifier:      nil,
		PublicMethods: map[string]struct{}{},
	}

	interceptor := UnaryServerInterceptor(cfg)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/PrivateMethod",
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err == nil {
		t.Fatal("expected error when verifier is not configured")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected code Internal, got %v", st.Code())
	}
}

func TestUnaryServerInterceptor_MissingToken(t *testing.T) {
	cfg := InterceptorConfig{
		Verifier:      &mockVerifier{},
		PublicMethods: map[string]struct{}{},
	}

	interceptor := UnaryServerInterceptor(cfg)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/PrivateMethod",
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err == nil {
		t.Fatal("expected error when token is missing")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected code Unauthenticated, got %v", st.Code())
	}
}

func TestUnaryServerInterceptor_InvalidToken(t *testing.T) {
	cfg := InterceptorConfig{
		Verifier: &mockVerifier{
			verifyFunc: func(_ context.Context, _ string) (*Claims, error) {
				return nil, errors.New("invalid token")
			},
		},
		PublicMethods: map[string]struct{}{},
	}

	interceptor := UnaryServerInterceptor(cfg)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/PrivateMethod",
	}

	md := metadata.New(map[string]string{
		"authorization": "Bearer invalid-token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, "request", info, handler)
	if err == nil {
		t.Fatal("expected error when token is invalid")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected code Unauthenticated, got %v", st.Code())
	}
}

func TestUnaryServerInterceptor_ValidToken(t *testing.T) {
	expectedClaims := &Claims{UserID: "user-123", Role: "admin"}

	cfg := createValidTokenInterceptorConfig(expectedClaims)
	interceptor := UnaryServerInterceptor(cfg)

	handlerCalled := false
	handler := createValidTokenHandler(t, &handlerCalled, expectedClaims)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/PrivateMethod",
	}

	ctx := createAuthContext("valid-token")

	resp, err := interceptor(ctx, "request", info, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}

	if resp != testResponse {
		t.Errorf("expected response %q, got %v", testResponse, resp)
	}
}

func createValidTokenInterceptorConfig(expectedClaims *Claims) InterceptorConfig {
	return InterceptorConfig{
		Verifier: &mockVerifier{
			verifyFunc: func(_ context.Context, token string) (*Claims, error) {
				if token != "valid-token" {
					return nil, errors.New("invalid token")
				}

				return expectedClaims, nil
			},
		},
		PublicMethods: map[string]struct{}{},
	}
}

func createValidTokenHandler(t *testing.T, handlerCalled *bool, expectedClaims *Claims) grpc.UnaryHandler {
	t.Helper()

	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		*handlerCalled = true

		// Verify claims are in context
		userID, ok := UserIDFromContext(ctx)
		if !ok {
			t.Error("expected user ID in context")
		}

		if userID != expectedClaims.UserID {
			t.Errorf("expected user ID %s, got %s", expectedClaims.UserID, userID)
		}

		role, ok := RoleFromContext(ctx)
		if !ok {
			t.Error("expected role in context")
		}

		if role != expectedClaims.Role {
			t.Errorf("expected role %s, got %s", expectedClaims.Role, role)
		}

		return testResponse, nil
	}
}

func createAuthContext(token string) context.Context {
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})

	return metadata.NewIncomingContext(context.Background(), md)
}

func TestBearerTokenFromMetadata(t *testing.T) {
	tests := getBearerTokenTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext(tt.metadata)
			token, err := bearerTokenFromMetadata(ctx)
			verifyBearerTokenResult(t, token, err, tt.wantToken, tt.wantErr)
		})
	}
}

type bearerTokenTestCase struct {
	name      string
	metadata  map[string]string
	wantToken string
	wantErr   bool
}

func getBearerTokenTestCases() []bearerTokenTestCase {
	return []bearerTokenTestCase{
		{name: "valid bearer token", metadata: map[string]string{"authorization": "Bearer valid-token-123"}, wantToken: "valid-token-123", wantErr: false},
		{name: "bearer with mixed case", metadata: map[string]string{"authorization": "BEARER token-abc"}, wantToken: "token-abc", wantErr: false},
		{name: "bearer with extra spaces", metadata: map[string]string{"authorization": "  Bearer   token-xyz  "}, wantToken: "token-xyz", wantErr: false},
		{name: "missing metadata", metadata: nil, wantToken: "", wantErr: true},
		{name: "missing authorization header", metadata: map[string]string{}, wantToken: "", wantErr: true},
		{name: "empty authorization header", metadata: map[string]string{"authorization": ""}, wantToken: "", wantErr: true},
		{name: "authorization header with only spaces", metadata: map[string]string{"authorization": "   "}, wantToken: "", wantErr: true},
		{name: "not a bearer token", metadata: map[string]string{"authorization": "Basic dXNlcjpwYXNz"}, wantToken: "", wantErr: true},
		{name: "bearer with empty token", metadata: map[string]string{"authorization": "Bearer "}, wantToken: "", wantErr: true},
		{name: "bearer with only spaces after", metadata: map[string]string{"authorization": "Bearer    "}, wantToken: "", wantErr: true},
	}
}

func createTestContext(mdMap map[string]string) context.Context {
	if mdMap != nil {
		md := metadata.New(mdMap)
		return metadata.NewIncomingContext(context.Background(), md)
	}

	return context.Background()
}

func verifyBearerTokenResult(t *testing.T, token string, err error, wantToken string, wantErr bool) {
	t.Helper()

	if wantErr {
		if err == nil {
			t.Error("expected error, got nil")
		}
	} else {
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if token != wantToken {
			t.Errorf("expected token %s, got %s", wantToken, token)
		}
	}
}
