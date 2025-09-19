package order

import (
	"go.uber.org/fx"

	"github.com/labstack/echo/v4"
)

// Module wires HTTP order handlers.
var Module = fx.Options(
	fx.Provide(NewHandler),
	fx.Invoke(func(e *echo.Echo, h *Handler) {
		Register(e, h)
	}),
)
