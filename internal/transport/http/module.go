package http

import (
	"go.uber.org/fx"

	ordertransport "github.com/Additional-Code/atlas/internal/transport/http/order"
)

// Module aggregates all HTTP transport handlers.
var Module = fx.Options(
	ordertransport.Module,
)
