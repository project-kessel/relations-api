package metrics

// Taken from Kratos: middleware/metrics/metrics.go

import (
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	metricLabelKind      = "kind"
	metricLabelOperation = "operation"
	metricLabelCode      = "code"
	metricLabelReason    = "reason"
)

type Option func(*options)

type options struct { // Duplicated from https://github.com/go-kratos/kratos/blob/main/middleware/metrics/metrics.go b/c no export
	requests metric.Int64Counter
	seconds  metric.Float64Histogram
}

// WithRequests with requests counter.
func WithRequests(c metric.Int64Counter) Option {
	return func(o *options) {
		o.requests = c
	}
}

// WithSeconds with seconds histogram.
func WithSeconds(histogram metric.Float64Histogram) Option {
	return func(o *options) {
		o.seconds = histogram
	}
}

func StreamMetricsInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	op := options{}
	for _, o := range opts {
		o(&op)
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if op.seconds == nil && op.requests == nil {
			return handler(srv, ss)
		}

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

		err := handler(srv, ss)
		if se := errors.FromError(err); se != nil {
			code = int(se.Code)
			reason = se.Reason
		}

		if op.requests != nil {
			op.requests.Add(
				ctx, 1,
				metric.WithAttributes(
					attribute.String(metricLabelKind, kind),
					attribute.String(metricLabelOperation, operation),
					attribute.Int(metricLabelCode, code),
					attribute.String(metricLabelReason, reason),
				),
			)
		}

		if op.seconds != nil {
			op.seconds.Record(
				ctx, time.Since(startTime).Seconds(),
				metric.WithAttributes(
					attribute.String(metricLabelKind, kind),
					attribute.String(metricLabelOperation, operation),
				),
			)
		}

		return err
	}
}
