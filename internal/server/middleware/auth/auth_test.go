package auth

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
)

// mockTransporter satisfies transport.Transporter for injecting operation names in tests.
type mockTransporter struct {
	operation string
}

func (m *mockTransporter) Kind() transport.Kind      { return transport.KindGRPC }
func (m *mockTransporter) Endpoint() string          { return "" }
func (m *mockTransporter) Operation() string         { return m.operation }
func (m *mockTransporter) RequestHeader() transport.Header { return nil }
func (m *mockTransporter) ReplyHeader() transport.Header   { return nil }

func ctxWithOperation(operation string) context.Context {
	return transport.NewServerContext(context.Background(), &mockTransporter{operation: operation})
}

func TestAuthFailureLoggingMiddleware_PassesThroughSuccess(t *testing.T) {
	t.Parallel()

	logger := log.NewStdLogger(io.Discard)
	m := AuthFailureLoggingMiddleware(logger)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	resp, err := m(handler)(ctxWithOperation("/test/Method"), nil)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestAuthFailureLoggingMiddleware_PassesThroughNonJwtError(t *testing.T) {
	t.Parallel()

	logger := log.NewStdLogger(io.Discard)
	m := AuthFailureLoggingMiddleware(logger)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, assert.AnError
	}

	_, err := m(handler)(ctxWithOperation("/test/Method"), nil)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestAuthFailureLoggingMiddleware_LogsAndPassesThroughJwtErrors(t *testing.T) {
	t.Parallel()

	jwtErrors := []error{
		jwt.ErrMissingJwtToken,
		jwt.ErrTokenInvalid,
		jwt.ErrTokenExpired,
		jwt.ErrTokenParseFail,
		jwt.ErrUnSupportSigningMethod,
		jwt.ErrMissingKeyFunc,
	}

	logger := log.NewStdLogger(io.Discard)
	m := AuthFailureLoggingMiddleware(logger)

	for _, jwtErr := range jwtErrors {
		t.Run(jwtErr.Error(), func(t *testing.T) {
			t.Parallel()

			handler := func(ctx context.Context, req any) (any, error) {
				return nil, jwtErr
			}

			_, err := m(handler)(ctxWithOperation("/kessel.relations.v1beta1.KesselTupleService/CreateTuples"), nil)
			assert.Equal(t, jwtErr, err)
		})
	}
}

func TestJwtErrorReason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      error
		expected string
	}{
		{"missing_key_func", jwt.ErrMissingKeyFunc, "missing_key_func"},
		{"missing_token", jwt.ErrMissingJwtToken, "missing_token"},
		{"token_invalid", jwt.ErrTokenInvalid, "token_invalid"},
		{"token_expired", jwt.ErrTokenExpired, "token_expired"},
		{"token_parse_failed", jwt.ErrTokenParseFail, "token_parse_failed"},
		{"unsupported_signing_method", jwt.ErrUnSupportSigningMethod, "unsupported_signing_method"},
		{"non_jwt_error", assert.AnError, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, jwtErrorReason(tc.err))
		})
	}
}
