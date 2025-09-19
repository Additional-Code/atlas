package logger

import (
	"context"
	"strings"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Additional-Code/atlas/internal/config"
)

// Module exposes a configured Zap logger to the Fx container.
var Module = fx.Provide(New)

// New builds a production Zap logger; callers own the cleanup via Fx lifecycle.
func New(lc fx.Lifecycle, cfg config.Config) (*zap.Logger, error) {
	observability := cfg.Observability
	level := zapcore.InfoLevel
	if err := level.Set(strings.ToLower(observability.LogLevel)); err != nil {
		level = zapcore.InfoLevel
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = zap.NewAtomicLevelAt(level)
	zapCfg.Encoding = observability.LogEncoding
	zapCfg.EncoderConfig.TimeKey = "ts"
	zapCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339Nano)
	zapCfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	zapCfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	if observability.LogEncoding == "console" {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(level)
		zapCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	logger = logger.With(
		zap.String("service", observability.ServiceName),
		zap.String("environment", observability.Environment),
	)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return logger.Sync()
		},
	})

	return logger, nil
}
