//go:build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/conf"
	"github.com/project-kessel/relations-api/internal/data"
	"github.com/project-kessel/relations-api/internal/server"
	"github.com/project-kessel/relations-api/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, service.ProviderSet, biz.ProviderSet, data.ProviderSet, newApp))
}
