package auth

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type logEntry struct {
	level   log.Level
	keyvals []any
}

// captureLogger records every Log call for assertion in tests.
type captureLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

func (c *captureLogger) Log(level log.Level, keyvals ...any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, logEntry{level: level, keyvals: keyvals})
	return nil
}

func (c *captureLogger) all() []logEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.entries
}

// keyvalMap converts a flat key-value slice into a map for field assertions.
func keyvalMap(keyvals []any) map[string]any {
	m := make(map[string]any, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		if k, ok := keyvals[i].(string); ok {
			m[k] = keyvals[i+1]
		}
	}
	return m
}

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

	const operation = "/kessel.relations.v1beta1.KesselTupleService/CreateTuples"

	jwtErrors := []error{
		jwt.ErrMissingJwtToken,
		jwt.ErrTokenInvalid,
		jwt.ErrTokenExpired,
		jwt.ErrTokenParseFail,
		jwt.ErrUnSupportSigningMethod,
		jwt.ErrMissingKeyFunc,
	}

	for _, jwtErr := range jwtErrors {
		t.Run(jwtErr.Error(), func(t *testing.T) {
			t.Parallel()

			cl := &captureLogger{}
			m := AuthFailureLoggingMiddleware(cl)

			handler := func(ctx context.Context, req any) (any, error) {
				return nil, jwtErr
			}

			_, err := m(handler)(ctxWithOperation(operation), nil)

			// Error is passed through unchanged.
			assert.Equal(t, jwtErr, err)

			// Exactly one log entry must have been emitted.
			entries := cl.all()
			require.Len(t, entries, 1)

			entry := entries[0]
			assert.Equal(t, log.LevelWarn, entry.level)

			fields := keyvalMap(entry.keyvals)
			assert.Equal(t, "AUTHENTICATE", fields["action"])
			assert.Equal(t, "api_endpoint", fields["resource_type"])
			assert.Equal(t, operation, fields["resource_id"])
			assert.Equal(t, "failure", fields["outcome"])
			assert.Equal(t, jwtErrorReason(jwtErr), fields["reason"])
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
