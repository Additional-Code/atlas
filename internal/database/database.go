package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/schema"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
)

// Connections bundles writer and reader bun instances.
type Connections struct {
	Writer *bun.DB
	Reader *bun.DB
}

// Module registers the database connections with Fx.
var Module = fx.Provide(New)

// New establishes writer and reader pools backed by Bun.
func New(lc fx.Lifecycle, cfg config.Config, logger *zap.Logger) (*Connections, error) {
	dial, err := selectDialect(cfg.Database.Driver)
	if err != nil {
		return nil, err
	}

	writerSQL, err := openSQLDB(cfg.Database.Driver, cfg.Database.WriterDSN)
	if err != nil {
		return nil, fmt.Errorf("open writer: %w", err)
	}

	applyPoolSettings(writerSQL, cfg.Database)

	writer := bun.NewDB(writerSQL, dial)

	var reader *bun.DB
	if cfg.Database.ReaderDSN != cfg.Database.WriterDSN {
		readerSQL, err := openSQLDB(cfg.Database.Driver, cfg.Database.ReaderDSN)
		if err != nil {
			return nil, fmt.Errorf("open reader: %w", err)
		}
		applyPoolSettings(readerSQL, cfg.Database)
		reader = bun.NewDB(readerSQL, dial)
	} else {
		reader = writer
	}

	conns := &Connections{Writer: writer, Reader: reader}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pingContext(ctx, writer); err != nil {
				return fmt.Errorf("ping writer: %w", err)
			}
			if reader != writer {
				if err := pingContext(ctx, reader); err != nil {
					return fmt.Errorf("ping reader: %w", err)
				}
			}
			logger.Info("database connected", zap.String("driver", cfg.Database.Driver))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			var closeErr error
			if err := writer.Close(); err != nil {
				closeErr = fmt.Errorf("close writer: %w", err)
			}
			if reader != writer {
				if err := reader.Close(); err != nil && closeErr == nil {
					closeErr = fmt.Errorf("close reader: %w", err)
				}
			}
			return closeErr
		},
	})

	return conns, nil
}

func selectDialect(driver string) (schema.Dialect, error) {
	switch driver {
	case "postgres":
		return pgdialect.New(), nil
	case "mysql":
		return mysqldialect.New(), nil
	case "sqlite":
		return sqlitedialect.New(), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

func openSQLDB(driver, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("empty DSN")
	}

	switch driver {
	case "postgres":
		connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
		return sql.OpenDB(connector), nil
	case "mysql":
		return sql.Open("mysql", dsn)
	case "sqlite":
		return sql.Open("sqlite3", dsn)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}

func applyPoolSettings(db *sql.DB, cfg config.Database) {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxConnLifetime > 0 {
		db.SetConnMaxLifetime(cfg.MaxConnLifetime)
	}
}

func pingContext(ctx context.Context, db *bun.DB) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.DB.PingContext(pingCtx)
}
