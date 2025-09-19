package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
)

// Store represents a generic cache backend.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// ErrCacheMiss indicates the key is absent from the cache.
var ErrCacheMiss = errors.New("cache miss")

// Module provides the cache store to the Fx graph.
var Module = fx.Provide(NewStore)

// NewStore initialises the configured cache store (redis or noop).
func NewStore(lc fx.Lifecycle, cfg config.Config, logger *zap.Logger) (Store, error) {
	switch cfg.Cache.Driver {
	case "noop":
		if logger != nil {
			logger.Info("cache disabled; using noop store")
		}
		return noopStore{}, nil
	case "redis":
		return newRedisStore(lc, cfg.Cache, logger)
	default:
		return nil, fmt.Errorf("unsupported cache driver: %s", cfg.Cache.Driver)
	}
}

type noopStore struct{}

func (noopStore) Get(context.Context, string) ([]byte, error) {
	return nil, ErrCacheMiss
}

func (noopStore) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (noopStore) Delete(context.Context, string) error {
	return nil
}

type redisStore struct {
	client     *goredis.Client
	defaultTTL time.Duration
}

func newRedisStore(lc fx.Lifecycle, cfg config.Cache, logger *zap.Logger) (Store, error) {
	opts := &goredis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := goredis.NewClient(opts)
	store := &redisStore{client: client, defaultTTL: cfg.DefaultTTL}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("ping redis: %w", err)
			}
			if logger != nil {
				logger.Info("redis cache connected", zap.String("addr", cfg.Redis.Addr))
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if logger != nil {
				logger.Info("closing redis cache")
			}
			return client.Close()
		},
	})

	return store, nil
}

func (s *redisStore) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, ErrCacheMiss
	}
	res, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *redisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return errors.New("cache key is required")
	}
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *redisStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return s.client.Del(ctx, key).Err()
}
