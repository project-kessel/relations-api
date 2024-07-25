package auth

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strings"
)

type authKey struct{}

const (
	// bearerWord the bearer key word for authorization
	bearerWord string = "bearer"

	// authorizationKey holds the key used to store the JWT Token in the request tokenHeader.
	authorizationKey string = "authorization"
)

type authOptions struct {
	signingMethod jwtv5.SigningMethod
	claims        func() jwtv5.Claims
	tokenHeader   map[string]interface{}
}

type AuthOption func(*authOptions)

func WithClaims(claimsFunc func() jwtv5.Claims) AuthOption {
	return func(o *authOptions) {
		o.claims = claimsFunc
	}
}

// WithTokenHeader withe customer tokenHeader for client side
func WithTokenHeader(header map[string]interface{}) AuthOption {
	return func(o *authOptions) {
		o.tokenHeader = header
	}
}

func WithSigningMethod(signingMethod jwtv5.SigningMethod) AuthOption {
	return func(o *authOptions) {
		o.signingMethod = signingMethod
	}
}

// StreamAuthInterceptor is a gRPC stream server interceptor for JWT authentication.
func StreamAuthInterceptor(keyFunc jwtv5.Keyfunc, opts ...AuthOption) grpc.StreamServerInterceptor {
	o := &authOptions{
		signingMethod: jwtv5.SigningMethodRS256,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var newCtx context.Context
		var err error
		newCtx = ss.Context()
		if keyFunc == nil {
			return jwt.ErrMissingKeyFunc
		}
		md, ok := metadata.FromIncomingContext(newCtx)
		if !ok {
			return jwt.ErrMissingJwtToken
		}
		authHeader, ok := md[authorizationKey]
		if !ok || len(authHeader) == 0 {
			return jwt.ErrMissingJwtToken
		}
		auths := strings.SplitN(authHeader[0], " ", 2)
		if len(auths) != 2 || !strings.EqualFold(auths[0], bearerWord) {
			return jwt.ErrMissingJwtToken
		}
		jwtToken := auths[1]
		var (
			tokenInfo *jwtv5.Token
		)
		if o.claims != nil {
			tokenInfo, err = jwtv5.ParseWithClaims(jwtToken, o.claims(), keyFunc)
		} else {
			tokenInfo, err = jwtv5.Parse(jwtToken, keyFunc)
		}
		if err != nil {
			if errors.Is(err, jwtv5.ErrTokenMalformed) || errors.Is(err, jwtv5.ErrTokenUnverifiable) {
				return jwt.ErrTokenInvalid
			}
			if errors.Is(err, jwtv5.ErrTokenNotValidYet) || errors.Is(err, jwtv5.ErrTokenExpired) {
				return jwt.ErrTokenExpired
			}
			return jwt.ErrTokenParseFail
		}
		if !tokenInfo.Valid {
			return jwt.ErrTokenInvalid
		}
		if tokenInfo.Method != o.signingMethod {
			return jwt.ErrUnSupportSigningMethod
		}
		newCtx = NewContext(newCtx, tokenInfo.Claims)
		wrappedStream := &authServerStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrappedStream)
	}
}

type authServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (a *authServerStream) Context() context.Context {
	return a.ctx
}

func NewContext(ctx context.Context, info jwtv5.Claims) context.Context {
	return context.WithValue(ctx, authKey{}, info)
}

func FromContext(ctx context.Context) (jwtv5.Claims, bool) {
	claims, ok := ctx.Value(authKey{}).(jwtv5.Claims)
	return claims, ok
}
