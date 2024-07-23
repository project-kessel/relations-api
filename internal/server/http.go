package server

import (
	h "github.com/project-kessel/relations-api/api/kessel/relations/v1"
	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/conf"
	"github.com/project-kessel/relations-api/internal/service"
	"go.opentelemetry.io/otel/metric"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			validate.Validator(),
			logging.Server(logger),
			metrics.Server(
				metrics.WithSeconds(seconds),
				metrics.WithRequests(requests),
			),
		),
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
	h.RegisterKesselHealthServiceHTTPServer(srv, health)
	return srv, nil
}
