package observability

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	stdoutmetric "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	stdouttrace "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
)

const (
	defaultServiceVersion  = "0.1.0"
	defaultShutdownTimeout = 10 * time.Second
)

// Manager wires tracing and metrics providers.
type Manager struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	metricsHandler http.Handler
	cfg            config.Observability
	logger         *zap.Logger
}

// Module exposes the observability manager to Fx.
var Module = fx.Provide(NewManager)

// NewManager configures tracing and metrics providers based on configuration.
func NewManager(lc fx.Lifecycle, cfg config.Config, logger *zap.Logger) (*Manager, error) {
	ctx := context.Background()
	resource, err := sdkresource.New(ctx,
		sdkresource.WithFromEnv(),
		sdkresource.WithHost(),
		sdkresource.WithAttributes(
			semconv.ServiceName(cfg.Observability.ServiceName),
			semconv.ServiceVersion(defaultServiceVersion),
			attribute.String("service.environment", cfg.Observability.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	mgr := &Manager{
		cfg:    cfg.Observability,
		logger: logger,
	}

	if cfg.Observability.EnableTracing {
		if err := mgr.initTracing(ctx, resource); err != nil {
			return nil, err
		}
	}

	if cfg.Observability.EnableMetrics {
		if err := mgr.initMetrics(resource); err != nil {
			return nil, err
		}
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if tp := mgr.tracerProvider; tp != nil {
				otel.SetTracerProvider(tp)
				otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
					propagation.TraceContext{},
					propagation.Baggage{},
				))
			}
			if mp := mgr.meterProvider; mp != nil {
				otel.SetMeterProvider(mp)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			deadlineCtx, cancel := context.WithTimeout(ctx, defaultShutdownTimeout)
			defer cancel()

			var shutdownErr error
			if tp := mgr.tracerProvider; tp != nil {
				shutdownErr = errors.Join(shutdownErr, tp.Shutdown(deadlineCtx))
			}
			if mp := mgr.meterProvider; mp != nil {
				shutdownErr = errors.Join(shutdownErr, mp.Shutdown(deadlineCtx))
			}
			return shutdownErr
		},
	})

	return mgr, nil
}

// TracingEnabled reports whether tracing is active.
func (m *Manager) TracingEnabled() bool {
	return m.tracerProvider != nil && m.cfg.EnableTracing
}

// MetricsEnabled reports whether metrics are active.
func (m *Manager) MetricsEnabled() bool {
	return m.meterProvider != nil && m.cfg.EnableMetrics
}

// MetricsHandler exposes the Prometheus HTTP handler when metrics are enabled.
func (m *Manager) MetricsHandler() http.Handler {
	return m.metricsHandler
}

// PrometheusPath returns the configured metrics endpoint path.
func (m *Manager) PrometheusPath() string {
	return m.cfg.PrometheusPath
}

func (m *Manager) initTracing(ctx context.Context, resource *sdkresource.Resource) error {
	exporter, err := m.createTraceExporter(ctx)
	if err != nil {
		return err
	}
	if exporter == nil {
		return nil
	}

	td := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)
	m.tracerProvider = td
	return nil
}

func (m *Manager) createTraceExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	switch strings.ToLower(m.cfg.TraceExporter) {
	case "", "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp":
		if m.cfg.TraceEndpoint == "" {
			return nil, fmt.Errorf("OBS_OTLP_ENDPOINT must be set for otlp exporter")
		}
		clientOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(m.cfg.TraceEndpoint)}
		if m.cfg.TraceInsecure {
			clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
		}
		exporterCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return otlptracegrpc.New(exporterCtx, clientOpts...)
	default:
		if m.logger != nil {
			m.logger.Warn("unsupported trace exporter; tracing disabled", zap.String("exporter", m.cfg.TraceExporter))
		}
		return nil, nil
	}
}

func (m *Manager) initMetrics(resource *sdkresource.Resource) error {
	switch strings.ToLower(m.cfg.MetricsExporter) {
	case "prometheus":
		exporter, err := promexporter.New(promexporter.WithRegisterer(prometheus.DefaultRegisterer))
		if err != nil {
			return err
		}
		m.meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(exporter),
			sdkmetric.WithResource(resource),
		)
		m.metricsHandler = promhttp.Handler()
	case "stdout":
		exporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint(), stdoutmetric.WithWriter(os.Stdout))
		if err != nil {
			return err
		}
		reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(30*time.Second))
		m.meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
			sdkmetric.WithResource(resource),
		)
	default:
		if m.logger != nil {
			m.logger.Warn("unsupported metrics exporter; metrics disabled", zap.String("exporter", m.cfg.MetricsExporter))
		}
	}
	return nil
}
