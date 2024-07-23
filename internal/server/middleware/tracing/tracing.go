package tracing

import (
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/transport"
)

// StreamTracingInterceptor returns a new server middleware for OpenTelemetry.
func StreamTracingInterceptor(opts ...tracing.Option) grpc.StreamServerInterceptor {
	tracer := tracing.NewTracer(trace.SpanKindServer, opts...)
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx := ss.Context()
		if tr, ok := transport.FromServerContext(ctx); ok {
			var span trace.Span
			ctx, span = tracer.Start(ctx, tr.Operation(), tr.RequestHeader())
			//Using nil requests and responses skips recording request and response size
			//Alternatively could sum request/response sizes for the stream
			setServerSpan(ctx, span, nil)
			defer func() { tracer.End(ctx, span, nil, err) }()
		}
		return handler(srv, ss)
	}
}
