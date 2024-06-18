package data

import (
	"github.com/google/wire"
	"relations-api/internal/biz"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewSpiceDbRepository, wire.Bind(new(biz.ZanzibarRepository), new(*SpiceDbRepository)))
