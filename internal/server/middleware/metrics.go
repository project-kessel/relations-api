package middleware

// Taken from Kratos: middleware/metrics/metrics.go

import (
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metrics"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/grpc"
)

// WithRequests with requests counter.
func WithRequests(c metrics.Counter) Option {
	return func(o *options) {
		o.requests = c
	}
}

// WithSeconds with seconds histogram.
func WithSeconds(c metrics.Observer) Option {
	return func(o *options) {
		o.seconds = c
	}
}

func StreamMetricsInterceptor(logger log.Logger, opts ...Option) grpc.StreamServerInterceptor {
	op := options{}
	for _, o := range opts {
		o(&op)
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var (
			code      int
			reason    string
			kind      string
			operation string
		)
		ctx := ss.Context()
		startTime := time.Now()

		if info, ok := transport.FromServerContext(ctx); ok {
			kind = info.Kind().String()
			operation = info.Operation()
		}

		wrapper := &requestInterceptingWrapper{ServerStream: ss}
		err := handler(srv, wrapper)

		if se := errors.FromError(err); se != nil {
			code = int(se.Code)
			reason = se.Reason
		}

		if op.requests != nil {
			op.requests.With(kind, operation, strconv.Itoa(code), reason).Inc()
		}

		if op.seconds != nil {
			op.seconds.With(kind, operation).Observe(time.Since(startTime).Seconds())
		}

		return err
	}
}
