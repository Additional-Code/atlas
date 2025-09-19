package migration

import "go.uber.org/fx"

// Module exposes the migrator via Fx.
var Module = fx.Provide(New)
