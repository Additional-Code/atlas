package http

import (
	"context"
	"fmt"
	"net/http"

	echo "github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
	"github.com/Additional-Code/atlas/internal/observability"
)

// Module exposes the HTTP server lifecycle to Fx.
var Module = fx.Module("http_server",
	fx.Provide(NewEcho),
	fx.Invoke(Run),
)

// NewEcho configures the Echo router with basic middleware.
func NewEcho(cfg config.Config, obs *observability.Manager, logger *zap.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		logger.Error("http request failed", zap.Error(err))
		c.Echo().DefaultHTTPErrorHandler(err, c)
	}

	if obs != nil && obs.TracingEnabled() {
		e.Use(otelecho.Middleware(cfg.Observability.ServiceName))
	}

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	if obs != nil && obs.MetricsEnabled() && obs.MetricsHandler() != nil {
		e.GET(cfg.Observability.PrometheusPath, echo.WrapHandler(obs.MetricsHandler()))
	}

	return e
}

// Run starts the HTTP server and ties it to the Fx lifecycle.
func Run(lc fx.Lifecycle, cfg config.Config, e *echo.Echo, logger *zap.Logger) {
	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)

	server := &http.Server{
		Addr:    addr,
		Handler: e,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting HTTP server", zap.String("addr", addr))
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("http server failed", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping HTTP server")
			return server.Shutdown(ctx)
		},
	})
}
