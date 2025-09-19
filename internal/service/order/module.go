package order

import "go.uber.org/fx"

// Module provides the order service to Fx.
var Module = fx.Provide(NewService)
