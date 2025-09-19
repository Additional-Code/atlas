package order

import "go.uber.org/fx"

// Module provides the order repository to Fx.
var Module = fx.Provide(NewRepository)
