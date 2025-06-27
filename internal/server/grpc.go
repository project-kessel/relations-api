package server

import (
	"buf.build/go/protovalidate"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	h "github.com/project-kessel/relations-api/api/kessel/relations/v1"
	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/conf"
	"github.com/project-kessel/relations-api/internal/server/middleware"
	"github.com/project-kessel/relations-api/internal/server/middleware/auth"
	"github.com/project-kessel/relations-api/internal/service"

	"go.opentelemetry.io/otel/metric"

	kratosmiddleware "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	kesselMetrics "github.com/project-kessel/relations-api/internal/server/middleware/metrics"
	kesselRecovery "github.com/project-kessel/relations-api/internal/server/middleware/recovery"
	googlegrpc "google.golang.org/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, relations *service.RelationshipsService, health *service.HealthService, check *service.CheckService, subjects *service.LookupService, meter metric.Meter, logger log.Logger) (*grpc.Server, error) {
	requests, err := metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil, err
	}
	seconds, err := metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil, err
	}
	validator, err := protovalidate.New()
	if err != nil {
		return nil, err
	}

	unaryMiddleware := []kratosmiddleware.Middleware{
		recovery.Recovery(),
		middleware.ValidationMiddleware(validator),
		logging.Server(logger),
		metrics.Server(
			metrics.WithSeconds(seconds),
			metrics.WithRequests(requests),
		),
	}
	streamingMiddleware := []googlegrpc.StreamServerInterceptor{
		middleware.StreamLogInterceptor(logger),
		middleware.StreamValidationInterceptor(validator),
		kesselRecovery.StreamRecoveryInterceptor(logger),
		kesselMetrics.StreamMetricsInterceptor(
			kesselMetrics.WithSeconds(seconds),
			kesselMetrics.WithRequests(requests),
		),
	}

	if c.Auth.EnableAuth {
		jwks, err := FetchJwks(c.Auth.JwksUrl)
		if err != nil {
			return nil, err
		}

		unaryMiddleware = append(unaryMiddleware,
			selector.Server(jwt.Server(jwks.Keyfunc,
				jwt.WithSigningMethod(jwtv5.SigningMethodRS256))).
				Match(NewWhiteListMatcher).
				Build(),
		)
		streamingMiddleware = append(streamingMiddleware, auth.StreamAuthInterceptor(
			jwks.Keyfunc,
			auth.WithSigningMethod(jwtv5.SigningMethodRS256)))
	}

	var opts = []grpc.ServerOption{
		grpc.Middleware(
			unaryMiddleware...,
		),
		grpc.Options(googlegrpc.ChainStreamInterceptor(
			streamingMiddleware...,
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
	h.RegisterKesselRelationsHealthServiceServer(srv, health)
	v1beta1.RegisterKesselLookupServiceServer(srv, subjects)
	return srv, nil
}
