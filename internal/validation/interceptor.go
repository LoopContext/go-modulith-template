// Package validation provides gRPC interceptors for automatic protobuf message validation.
package validation

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"buf.build/go/protovalidate"
)

var (
	// validator is the global protovalidate validator instance
	validator protovalidate.Validator
)

func init() {
	var err error
	validator, err = protovalidate.New()
	if err != nil {
		// If initialization fails, validator will be nil and we'll skip validation
		// This should not happen in normal operation
		panic(fmt.Sprintf("failed to initialize protovalidate: %v", err))
	}
}

// UnaryServerInterceptor validates incoming gRPC requests using protovalidate.
//
// The interceptor validates protobuf messages using the protovalidate library
// at runtime. Validation errors are converted to gRPC InvalidArgument status errors.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if the request is a protobuf message
		if msg, ok := req.(proto.Message); ok && validator != nil {
			if err := validator.Validate(msg); err != nil {
				// Convert validation error to gRPC InvalidArgument
				// protovalidate errors are already well-formatted
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("validation failed: %v", err))
			}
		}

		// If validation passes (or request is not a proto message), proceed
		return handler(ctx, req)
	}
}

