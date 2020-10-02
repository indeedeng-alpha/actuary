package auth

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewStatic(authorizations map[string]string) grpc_auth.AuthFunc {
	return func(ctx context.Context) (context.Context, error) {
		token, err := grpc_auth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}

		if token == "" {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token was not provided")
		}

		client, ok := authorizations[token]
		if !ok || client == "" {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is invalid")
		}

		return context.WithValue(ctx, "actuary.client", client), nil
	}
}

