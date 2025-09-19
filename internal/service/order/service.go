package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/cache"
	"github.com/Additional-Code/atlas/internal/config"
	"github.com/Additional-Code/atlas/internal/entity"
	"github.com/Additional-Code/atlas/internal/messaging"
	repo "github.com/Additional-Code/atlas/internal/repository/order"
	"github.com/Additional-Code/atlas/pkg/errorbank"
)

var serviceTracer = otel.Tracer("github.com/Additional-Code/atlas/service/order")

// Service encapsulates business logic around orders.
type Service struct {
	repo      *repo.Repository
	cache     cache.Store
	cacheTTL  time.Duration
	logger    *zap.Logger
	publisher messaging.Client
	messaging messagingConfig
}

// messagingConfig contains messaging specific knobs we care about.
type messagingConfig struct {
	enabled bool
	topic   string
}

// Params defines dependencies for constructing Service.
type Params struct {
	fx.In

	Repository *repo.Repository
	Cache      cache.Store
	Config     config.Config
	Logger     *zap.Logger
	Publisher  messaging.Client
}

// NewService wires a new Service instance.
func NewService(p Params) *Service {
	return &Service{
		repo:      p.Repository,
		cache:     p.Cache,
		cacheTTL:  p.Config.Cache.DefaultTTL,
		logger:    p.Logger,
		publisher: p.Publisher,
		messaging: messagingConfig{
			enabled: p.Config.Messaging.Enabled,
			topic:   p.Config.Messaging.Kafka.Topic,
		},
	}
}

// Get retrieves an order by id, consulting cache when available.
func (s *Service) Get(ctx context.Context, id int64) (*entity.Order, error) {
	ctx, span := serviceTracer.Start(ctx, "OrderService.Get", trace.WithAttributes(attribute.Int64("order.id", id)))
	defer span.End()

	if order, err := s.getFromCache(ctx, id); err == nil {
		return order, nil
	} else if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
		if s.logger != nil {
			s.logger.Warn("orders cache read failed", zap.Int64("id", id), zap.Error(err))
		}
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, errorbank.NotFound("order not found")
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository error")
		return nil, errorbank.Internal("failed to load order", errorbank.WithCause(err))
	}

	if err := s.storeInCache(ctx, order); err != nil {
		if s.logger != nil {
			s.logger.Warn("orders cache write failed", zap.Int64("id", id), zap.Error(err))
		}
	}

	return order, nil
}

// Create creates a new order in the database and refreshes cache state.
func (s *Service) Create(ctx context.Context, order *entity.Order) error {
	if order == nil {
		return errorbank.BadRequest("order payload is required")
	}
	if order.CreatedAt.IsZero() {
		now := time.Now().UTC()
		order.CreatedAt = now
		order.UpdatedAt = now
	}
	ctx, span := serviceTracer.Start(ctx, "OrderService.Create", trace.WithAttributes(attribute.String("order.number", order.Number)))
	defer span.End()

	if err := s.repo.Create(ctx, order); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository error")
		return errorbank.Internal("failed to create order", errorbank.WithCause(err))
	}

	if err := s.storeInCache(ctx, order); err != nil {
		if s.logger != nil {
			s.logger.Warn("orders cache write failed", zap.Int64("id", order.ID), zap.Error(err))
		}
	}

	s.publishOrderCreated(ctx, order)
	return nil
}

func (s *Service) publishOrderCreated(ctx context.Context, order *entity.Order) {
	if !s.messaging.enabled || s.publisher == nil {
		return
	}
	event := OrderCreatedEvent{
		ID:        order.ID,
		Number:    order.Number,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("marshal order created", zap.Error(err))
		}
		return
	}
	if err := s.publisher.Publish(ctx, []byte(fmt.Sprintf("order-%d", order.ID)), payload); err != nil {
		if s.logger != nil {
			s.logger.Error("publish order created", zap.Error(err))
		}
	}
}

func (s *Service) cacheKey(id int64) string {
	return fmt.Sprintf("orders:%d", id)
}

func (s *Service) getFromCache(ctx context.Context, id int64) (*entity.Order, error) {
	if s.cache == nil {
		return nil, cache.ErrCacheMiss
	}
	bytes, err := s.cache.Get(ctx, s.cacheKey(id))
	if err != nil {
		return nil, err
	}
	var order entity.Order
	if err := json.Unmarshal(bytes, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *Service) storeInCache(ctx context.Context, order *entity.Order) error {
	if s.cache == nil || order == nil {
		return nil
	}
	bytes, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, s.cacheKey(order.ID), bytes, s.cacheTTL)
}

// OrderCreatedEvent is emitted when a new order is persisted.
type OrderCreatedEvent struct {
	ID        int64     `json:"id"`
	Number    string    `json:"number"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
