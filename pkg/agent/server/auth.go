package server

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const metadataKeyAuth = "authorization"

// authUnaryInterceptor validates Bearer token from incoming metadata when token is set.
func authUnaryInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if token == "" {
			return handler(ctx, req)
		}
		if err := validateTokenFromContext(ctx, token); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// authStreamInterceptor validates Bearer token for streaming RPCs when token is set.
func authStreamInterceptor(token string) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if token == "" {
			return handler(srv, ss)
		}
		if err := validateTokenFromContext(ss.Context(), token); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func validateTokenFromContext(ctx context.Context, expected string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get(metadataKeyAuth)
	if len(vals) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization")
	}
	raw := vals[0]
	if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return status.Error(codes.Unauthenticated, "invalid authorization format")
	}
	got := strings.TrimSpace(raw[7:])
	if got != expected {
		return status.Error(codes.Unauthenticated, "invalid token")
	}
	return nil
}
