package server

import (
	h "github.com/project-kessel/relations-api/api/kessel/relations/v1"
	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/conf"
	"github.com/project-kessel/relations-api/internal/server/middleware"
	"github.com/project-kessel/relations-api/internal/service"
	"go.opentelemetry.io/otel"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	kesselMetrics "github.com/project-kessel/relations-api/internal/server/middleware/metrics"
	kesselRecovery "github.com/project-kessel/relations-api/internal/server/middleware/recovery"
	googlegrpc "google.golang.org/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, relations *service.RelationshipsService, health *service.HealthService, check *service.CheckService, subjects *service.LookupService, logger log.Logger) (*grpc.Server, error) {
	meter := otel.Meter("meter")
	requests, err := metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil, err
	}
	seconds, err := metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil, err
	}
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			validate.Validator(),
			logging.Server(logger),
			metrics.Server(
				metrics.WithSeconds(seconds),
				metrics.WithRequests(requests),
			),
		),
		grpc.Options(googlegrpc.ChainStreamInterceptor(
			middleware.StreamLogInterceptor(logger),
			middleware.StreamValidationInterceptor(),
			kesselRecovery.StreamRecoveryInterceptor(logger),
			kesselMetrics.StreamMetricsInterceptor(
				kesselMetrics.WithSeconds(seconds),
				kesselMetrics.WithRequests(requests),
			),
		)),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1beta1.RegisterKesselTupleServiceServer(srv, relations)
	v1beta1.RegisterKesselCheckServiceServer(srv, check)
	h.RegisterKesselHealthServiceServer(srv, health)
	v1beta1.RegisterKesselLookupServiceServer(srv, subjects)
	return srv, nil
}
