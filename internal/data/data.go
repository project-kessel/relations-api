package data

import (
	"ciam-rebac/internal/biz"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewSpiceDbRepository, wire.Bind(new(biz.ZanzibarRepository), new(*SpiceDbRepository)))
