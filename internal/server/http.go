package server

import (
	h "github.com/project-kessel/relations-api/api/health/v1"
	v0 "github.com/project-kessel/relations-api/api/relations/v0"
	"github.com/project-kessel/relations-api/internal/conf"
	"github.com/project-kessel/relations-api/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, relationships *service.RelationshipsService, health *service.HealthService, check *service.CheckService, subjects *service.LookupService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			validate.Validator(),
			logging.Server(logger),
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
	v0.RegisterKesselTupleServiceHTTPServer(srv, relationships)
	v0.RegisterKesselCheckServiceHTTPServer(srv, check)
	h.RegisterKesselHealthHTTPServer(srv, health)
	return srv
}
