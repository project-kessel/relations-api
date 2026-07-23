package service

import (
	"context"
	"testing"

	kratosJwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	localAuth "github.com/project-kessel/relations-api/internal/server/middleware/auth"
	"github.com/stretchr/testify/assert"
)

func TestExtractPrincipal_KratosJwtContext(t *testing.T) {
	t.Parallel()

	claims := jwtv5.MapClaims{"sub": "user@example.com"}
	ctx := kratosJwt.NewContext(context.Background(), claims)

	assert.Equal(t, "user@example.com", extractPrincipal(ctx))
}

func TestExtractPrincipal_LocalAuthContext(t *testing.T) {
	t.Parallel()

	claims := jwtv5.MapClaims{"sub": "stream-user@example.com"}
	ctx := localAuth.NewContext(context.Background(), claims)

	assert.Equal(t, "stream-user@example.com", extractPrincipal(ctx))
}

func TestExtractPrincipal_KratosJwtTakesPrecedence(t *testing.T) {
	t.Parallel()

	kratosClaims := jwtv5.MapClaims{"sub": "kratos-user"}
	localClaims := jwtv5.MapClaims{"sub": "local-user"}
	ctx := kratosJwt.NewContext(context.Background(), kratosClaims)
	ctx = localAuth.NewContext(ctx, localClaims)

	assert.Equal(t, "kratos-user", extractPrincipal(ctx))
}

func TestExtractPrincipal_NoContext(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", extractPrincipal(context.Background()))
}

func TestExtractPrincipal_MissingSubClaim(t *testing.T) {
	t.Parallel()

	claims := jwtv5.MapClaims{"email": "user@example.com"}
	ctx := kratosJwt.NewContext(context.Background(), claims)

	assert.Equal(t, "", extractPrincipal(ctx))
}

func TestExtractPrincipal_NonStringSubClaim(t *testing.T) {
	t.Parallel()

	claims := jwtv5.MapClaims{"sub": 12345}
	ctx := kratosJwt.NewContext(context.Background(), claims)

	assert.Equal(t, "", extractPrincipal(ctx))
}

func TestExtractSub_ValidMapClaims(t *testing.T) {
	t.Parallel()

	claims := jwtv5.MapClaims{"sub": "alice"}
	assert.Equal(t, "alice", extractSub(claims))
}

func TestExtractSub_NonMapClaims(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", extractSub("not-a-claims-object"))
}

func TestExtractSub_NilClaims(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", extractSub(nil))
}
