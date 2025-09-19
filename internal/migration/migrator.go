package migration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
	"github.com/Additional-Code/atlas/internal/database"
)

const migrationsDir = "db/migrations/sql"

// Migrator wraps goose operations.
type Migrator struct {
	db     *bun.DB
	logger *zap.Logger
}

// New constructs a goose-backed migrator.
func New(cfg config.Config, conns *database.Connections, logger *zap.Logger) (*Migrator, error) {
	dialect, err := gooseDialect(cfg.Database.Driver)
	if err != nil {
		return nil, err
	}

	if err := goose.SetDialect(dialect); err != nil {
		return nil, err
	}

	return &Migrator{
		db:     conns.Writer,
		logger: logger,
	}, nil
}

// Up applies all pending migrations.
func (m *Migrator) Up(ctx context.Context) error {
	if err := goose.UpContext(ctx, m.db.DB, migrationsDir); err != nil {
		if isNoMigrationErr(err) {
			m.logger.Info("no migrations to apply")

			return nil
		}
		return err
	}

	m.logger.Info("migrations applied")

	return nil
}

// Down rolls back migrations. Steps <=0 defaults to 1; all=true rolls everything back.
func (m *Migrator) Down(ctx context.Context, steps int, all bool) error {
	if all {
		if err := goose.DownToContext(ctx, m.db.DB, migrationsDir, 0); err != nil {
			if isNoMigrationErr(err) {
				m.logger.Info("no migrations to rollback")

				return nil
			}
			return err
		}
		m.logger.Info("migrations rolled back", zap.String("mode", "all"))

		return nil
	}

	if steps <= 0 {
		steps = 1
	}

	for i := 0; i < steps; i++ {
		if err := goose.DownContext(ctx, m.db.DB, migrationsDir); err != nil {
			if isNoMigrationErr(err) {
				m.logger.Info("no migrations to rollback")

				return nil
			}
			return err
		}
	}

	m.logger.Info("migrations rolled back", zap.Int("steps", steps))

	return nil
}

func gooseDialect(driver string) (string, error) {
	switch driver {
	case "postgres", "pg":
		return "postgres", nil
	case "mysql":
		return "mysql", nil
	case "sqlite", "sqlite3":
		return "sqlite3", nil
	default:
		return "", fmt.Errorf("unsupported goose dialect for driver %s", driver)
	}
}

func isNoMigrationErr(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, goose.ErrNoNextVersion) || errors.Is(err, goose.ErrNoMigrationFiles) {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "no migrations")
}
