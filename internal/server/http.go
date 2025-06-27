package server

import (
	"context"

	"buf.build/go/protovalidate"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt2 "github.com/golang-jwt/jwt/v5"
	h "github.com/project-kessel/relations-api/api/kessel/relations/v1"
	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/conf"
	"github.com/project-kessel/relations-api/internal/server/middleware"
	"github.com/project-kessel/relations-api/internal/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/metric"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, relationships *service.RelationshipsService, health *service.HealthService, check *service.CheckService, subjects *service.LookupService, meter metric.Meter, logger log.Logger) (*http.Server, error) {
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
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			middleware.ValidationMiddleware(validator),
			logging.Server(logger),
			metrics.Server(
				metrics.WithSeconds(seconds),
				metrics.WithRequests(requests),
			),
		),
	}
	if c.Auth.EnableAuth {
		jwks, err := FetchJwks(c.Auth.JwksUrl)
		if err != nil {
			return nil, err
		}
		opts = append(opts, http.Middleware(
			selector.Server(
				jwt.Server(
					jwks.Keyfunc,
					jwt.WithSigningMethod(jwt2.SigningMethodRS256),
				)).
				Match(NewWhiteListMatcher).
				Build(),
		))
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	if c.Http.Pathprefix != "" {
		opts = append(opts, http.PathPrefix(c.Http.Pathprefix))
	}

	srv := http.NewServer(opts...)
	srv.HandlePrefix("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	v1beta1.RegisterKesselTupleServiceHTTPServer(srv, relationships)
	v1beta1.RegisterKesselCheckServiceHTTPServer(srv, check)
	h.RegisterKesselRelationsHealthServiceHTTPServer(srv, health)
	return srv, nil
}

func NewWhiteListMatcher(ctx context.Context, operation string) bool {
	whiteList := make(map[string]struct{})
	whiteList["/kessel.relations.v1.KesselRelationsHealthService/GetReadyz"] = struct{}{}
	whiteList["/kessel.relations.v1.KesselRelationsHealthService/GetLivez"] = struct{}{}
	whiteList["/grpc.health.v1.Health/Check"] = struct{}{}
	if _, ok := whiteList[operation]; ok {
		return false
	}
	return true
}

// Use the JWKS URL to create a JWKSet.
func FetchJwks(jwksURL string) (keyfunc.Keyfunc, error) {
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Fatalf("Failed to create JWK Set from resource at the given URL: %s.\nError: %s", jwksURL, err)
		return nil, err
	}
	return jwks, nil
}
