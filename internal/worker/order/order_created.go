package order

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
	"github.com/Additional-Code/atlas/internal/messaging"
	ordersvc "github.com/Additional-Code/atlas/internal/service/order"
	"github.com/Additional-Code/atlas/internal/worker"
)

var workerTracer = otel.Tracer("github.com/Additional-Code/atlas/worker/order")

// Module registers order-related worker handlers.
var Module = fx.Module("worker_order",
	fx.Provide(
		fx.Annotate(
			NewOrderCreatedHandler,
			fx.ResultTags(`group:"worker.handlers"`),
		),
	),
)

// NewOrderCreatedHandler sets up a worker handler that logs order creations.
func NewOrderCreatedHandler(logger *zap.Logger, cfg config.Config) worker.HandlerRegistration {
	handler := func(ctx context.Context, msg messaging.Message) error {
		ctx, span := workerTracer.Start(ctx, "worker.orders.process", trace.WithAttributes(
			attribute.String("messaging.topic", msg.Topic),
		))
		defer span.End()

		var event ordersvc.OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Error("failed to decode order created", zap.Error(err))

			span.RecordError(err)
			span.SetStatus(codes.Error, "decode error")
			return err
		}
		logger.Info("order created event processed",
			zap.Int64("id", event.ID),
			zap.String("number", event.Number),
			zap.String("status", event.Status),
		)

		return nil
	}

	return worker.HandlerRegistration{
		Topic:   cfg.Messaging.Kafka.Topic,
		Handler: handler,
	}
}
