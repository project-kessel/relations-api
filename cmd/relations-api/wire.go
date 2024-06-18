//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"relations-api/internal/biz"
	"relations-api/internal/conf"
	"relations-api/internal/data"
	"relations-api/internal/server"
	"relations-api/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, service.ProviderSet, biz.ProviderSet, data.ProviderSet, newApp))
}
