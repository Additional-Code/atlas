package seeder

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/database"
	"github.com/Additional-Code/atlas/internal/entity"
)

// Seeder performs database seeding for local/dev setups.
type Seeder struct {
	db     *bun.DB
	logger *zap.Logger
}

// New constructs a Seeder backed by the primary database connection.
func New(conns *database.Connections, logger *zap.Logger) *Seeder {
	return &Seeder{db: conns.Writer, logger: logger}
}

// Orders seeds example orders if they are missing.
func (s *Seeder) Orders(ctx context.Context) error {
	now := time.Now().UTC()
	samples := []entity.Order{
		{Number: "ORDER-1000", Status: "pending", CreatedAt: now, UpdatedAt: now},
		{Number: "ORDER-1001", Status: "processing", CreatedAt: now, UpdatedAt: now},
	}

	for _, sample := range samples {
		order := sample
		_, err := s.db.NewInsert().Model(&order).
			On("CONFLICT (number) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	if s.logger != nil {
		s.logger.Info("seeded orders", zap.Int("count", len(samples)))
	}
	return nil
}
