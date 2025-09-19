package seeder

import "go.uber.org/fx"

// Module exposes Seeder through Fx.
var Module = fx.Provide(New)
