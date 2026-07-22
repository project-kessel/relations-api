package service

import (
	"context"

	kratosJwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	localAuth "github.com/project-kessel/relations-api/internal/server/middleware/auth"
)

func extractPrincipal(ctx context.Context) string {
	if token, ok := kratosJwt.FromContext(ctx); ok {
		if sub := extractSub(token); sub != "" {
			return sub
		}
	}
	if claims, ok := localAuth.FromContext(ctx); ok {
		if sub := extractSub(claims); sub != "" {
			return sub
		}
	}
	return ""
}

func extractSub(claims any) string {
	if mc, ok := claims.(jwtv5.MapClaims); ok {
		if sub, ok := mc["sub"].(string); ok {
			return sub
		}
	}
	return ""
}
