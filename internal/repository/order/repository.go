package order

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Additional-Code/atlas/internal/database"
	"github.com/Additional-Code/atlas/internal/entity"
)

var repoTracer = otel.Tracer("github.com/Additional-Code/atlas/repository/order")

// ErrNotFound is returned when an order is missing.
var ErrNotFound = errors.New("order not found")

// Repository encapsulates read/write access for orders.
type Repository struct {
	writer *bun.DB
	reader *bun.DB
}

// NewRepository wires a repository backed by configured database connections.
func NewRepository(conns *database.Connections) *Repository {
	return &Repository{
		writer: conns.Writer,
		reader: conns.Reader,
	}
}

// Create persists a new order using the write connection.
func (r *Repository) Create(ctx context.Context, order *entity.Order) error {
	if order == nil {
		return errors.New("nil order")
	}
	ctx, span := repoTracer.Start(ctx, "OrderRepository.Create", trace.WithAttributes(attribute.String("order.number", order.Number)))
	defer span.End()

	_, err := r.writer.NewInsert().Model(order).Exec(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "insert failed")
	}
	return err
}

// GetByID fetches an order by primary key using the read replica when available.
func (r *Repository) GetByID(ctx context.Context, id int64) (*entity.Order, error) {
	ctx, span := repoTracer.Start(ctx, "OrderRepository.GetByID", trace.WithAttributes(attribute.Int64("order.id", id)))
	defer span.End()

	order := new(entity.Order)
	err := r.reader.NewSelect().Model(order).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		span.SetStatus(codes.Error, "not found")
		return nil, ErrNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "select failed")
		return nil, err
	}
	return order, nil
}
