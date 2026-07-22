package auth

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	kratosMiddleware "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
func StreamAuthInterceptor(logger log.Logger, keyFunc jwtv5.Keyfunc, opts ...AuthOption) grpc.StreamServerInterceptor {
	o := &authOptions{
		signingMethod: jwtv5.SigningMethodRS256,
	}
	for _, opt := range opts {
		opt(o)
	}

	// Authentication failure - SEC-MON-REQ-1 compliance (EOI-7 invalid_login, EOI-8 authorization_failure)
	logAuthFailure := func(ctx context.Context, operation, reason string) {
		log.NewHelper(log.WithContext(ctx, logger)).Warnw(
			"msg", "Authentication failed",
			"action", "AUTHENTICATE",
			"resource_type", "api_endpoint",
			"resource_id", operation,
			"outcome", "failure",
			"reason", reason,
		)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var newCtx context.Context
		var err error
		newCtx = ss.Context()
		if keyFunc == nil {
			logAuthFailure(newCtx, info.FullMethod, "missing_key_func")
			return jwt.ErrMissingKeyFunc
		}
		md, ok := metadata.FromIncomingContext(newCtx)
		if !ok {
			logAuthFailure(newCtx, info.FullMethod, "missing_token")
			return jwt.ErrMissingJwtToken
		}
		authHeader, ok := md[authorizationKey]
		if !ok || len(authHeader) == 0 {
			logAuthFailure(newCtx, info.FullMethod, "missing_token")
			return jwt.ErrMissingJwtToken
		}
		auths := strings.SplitN(authHeader[0], " ", 2)
		if len(auths) != 2 || !strings.EqualFold(auths[0], bearerWord) {
			logAuthFailure(newCtx, info.FullMethod, "missing_token")
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
				logAuthFailure(newCtx, info.FullMethod, "token_invalid")
				return jwt.ErrTokenInvalid
			}
			if errors.Is(err, jwtv5.ErrTokenNotValidYet) || errors.Is(err, jwtv5.ErrTokenExpired) {
				logAuthFailure(newCtx, info.FullMethod, "token_expired")
				return jwt.ErrTokenExpired
			}
			logAuthFailure(newCtx, info.FullMethod, "token_parse_failed")
			return jwt.ErrTokenParseFail
		}
		if !tokenInfo.Valid {
			logAuthFailure(newCtx, info.FullMethod, "token_invalid")
			return jwt.ErrTokenInvalid
		}
		if tokenInfo.Method != o.signingMethod {
			logAuthFailure(newCtx, info.FullMethod, "unsupported_signing_method")
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

// AuthFailureLoggingMiddleware logs JWT authentication failures for unary RPCs (SEC-MON-REQ-1 compliance).
func AuthFailureLoggingMiddleware(logger log.Logger) kratosMiddleware.Middleware {
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			resp, err := handler(ctx, req)
			if err != nil {
				if reason := jwtErrorReason(err); reason != "" {
					operation := "unknown"
					if tr, ok := transport.FromServerContext(ctx); ok {
						operation = tr.Operation()
					}
					log.NewHelper(log.WithContext(ctx, logger)).Warnw(
						"msg", "Authentication failed",
						"action", "AUTHENTICATE",
						"resource_type", "api_endpoint",
						"resource_id", operation,
						"outcome", "failure",
						"reason", reason,
					)
				}
			}
			return resp, err
		}
	}
}

func jwtErrorReason(err error) string {
	switch {
	case jwt.ErrMissingKeyFunc.Is(err):
		return "missing_key_func"
	case jwt.ErrMissingJwtToken.Is(err):
		return "missing_token"
	case jwt.ErrTokenInvalid.Is(err):
		return "token_invalid"
	case jwt.ErrTokenExpired.Is(err):
		return "token_expired"
	case jwt.ErrTokenParseFail.Is(err):
		return "token_parse_failed"
	case jwt.ErrUnSupportSigningMethod.Is(err):
		return "unsupported_signing_method"
	default:
		return ""
	}
}
