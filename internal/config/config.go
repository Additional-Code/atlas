package config

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

// HTTP holds HTTP server configuration.
type HTTP struct {
	Host string
	Port int
}

// GRPC holds gRPC server configuration.
type GRPC struct {
	Host string
	Port int
}

// Cache configures caching behavior and backend selection.
type Cache struct {
	Enabled    bool
	Driver     string
	DefaultTTL time.Duration
	Redis      Redis
}

// Redis contains redis-specific connection settings.
type Redis struct {
	Addr     string
	Password string
	DB       int
}

// Messaging configures the message bus used by the application.
type Messaging struct {
	Driver        string
	Enabled       bool
	Kafka         Kafka
	ConsumerGroup string
	Workers       Worker
}

// Kafka holds Kafka connection details.
type Kafka struct {
	Brokers        []string
	ClientID       string
	Topic          string
	CommitInterval time.Duration
	MinBytes       int
	MaxBytes       int
	ConnectTimeout time.Duration
}

// Worker configures background worker concurrency and polling.
type Worker struct {
	Enabled      bool
	PollInterval time.Duration
	Concurrency  int
}

// Database holds primary and read replica connection settings.
type Database struct {
	Driver          string
	WriterDSN       string
	ReaderDSN       string
	MaxOpenConns    int
	MaxIdleConns    int
	MaxConnLifetime time.Duration
}

// Observability contains logging, tracing, and metrics configuration.
type Observability struct {
	ServiceName     string
	Environment     string
	LogLevel        string
	LogEncoding     string
	EnableTracing   bool
	TraceExporter   string
	TraceEndpoint   string
	TraceInsecure   bool
	EnableMetrics   bool
	MetricsExporter string
	PrometheusPath  string
}

// Config wraps all application configuration knobs.
type Config struct {
	HTTP          HTTP
	GRPC          GRPC
	Cache         Cache
	Messaging     Messaging
	Database      Database
	Observability Observability
}

// Module wires the configuration loader into the Fx graph.
var Module = fx.Provide(New)

var loadEnvOnce sync.Once

// New builds a Config from environment variables or defaults.
func New() (Config, error) {
	loadEnvOnce.Do(func() {
		_ = godotenv.Load()
	})

	cfg := Config{
		HTTP: HTTP{
			Host: getEnv("HTTP_HOST", "0.0.0.0"),
			Port: getEnvAsInt("HTTP_PORT", 8080),
		},
		GRPC: GRPC{
			Host: getEnv("GRPC_HOST", "0.0.0.0"),
			Port: getEnvAsInt("GRPC_PORT", 9090),
		},
		Cache: Cache{
			Enabled:    getEnvAsBool("CACHE_ENABLED", true),
			Driver:     getEnv("CACHE_DRIVER", "redis"),
			DefaultTTL: getEnvAsDuration("CACHE_DEFAULT_TTL", time.Minute*5),
			Redis: Redis{
				Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
				Password: getEnv("REDIS_PASSWORD", ""),
				DB:       getEnvAsInt("REDIS_DB", 0),
			},
		},
		Messaging: Messaging{
			Driver:  getEnv("MESSAGING_DRIVER", "kafka"),
			Enabled: getEnvAsBool("MESSAGING_ENABLED", true),
			Kafka: Kafka{
				Brokers:        getEnvAsStringSlice("KAFKA_BROKERS", []string{"127.0.0.1:9092"}),
				ClientID:       getEnv("KAFKA_CLIENT_ID", "atlas-service"),
				Topic:          getEnv("KAFKA_TOPIC", "orders.events"),
				CommitInterval: getEnvAsDuration("KAFKA_COMMIT_INTERVAL", time.Second),
				MinBytes:       getEnvAsInt("KAFKA_MIN_BYTES", 10e3),
				MaxBytes:       getEnvAsInt("KAFKA_MAX_BYTES", 10e6),
				ConnectTimeout: getEnvAsDuration("KAFKA_CONNECT_TIMEOUT", 5*time.Second),
			},
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "atlas-worker"),
			Workers: Worker{
				Enabled:      getEnvAsBool("WORKER_ENABLED", true),
				PollInterval: getEnvAsDuration("WORKER_POLL_INTERVAL", time.Second),
				Concurrency:  getEnvAsInt("WORKER_CONCURRENCY", 4),
			},
		},
		Database: Database{
			Driver:          getEnv("DB_DRIVER", "postgres"),
			WriterDSN:       getEnv("DB_WRITER_DSN", "postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable"),
			ReaderDSN:       getEnv("DB_READER_DSN", ""),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
			MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Minute*5),
		},
		Observability: Observability{
			ServiceName:     getEnv("OBS_SERVICE_NAME", "atlas"),
			Environment:     getEnv("OBS_ENVIRONMENT", "local"),
			LogLevel:        getEnv("OBS_LOG_LEVEL", "info"),
			LogEncoding:     getEnv("OBS_LOG_ENCODING", "json"),
			EnableTracing:   getEnvAsBool("OBS_ENABLE_TRACING", true),
			TraceExporter:   getEnv("OBS_TRACE_EXPORTER", "stdout"),
			TraceEndpoint:   getEnv("OBS_OTLP_ENDPOINT", "localhost:4317"),
			TraceInsecure:   getEnvAsBool("OBS_OTLP_INSECURE", true),
			EnableMetrics:   getEnvAsBool("OBS_ENABLE_METRICS", true),
			MetricsExporter: getEnv("OBS_METRICS_EXPORTER", "prometheus"),
			PrometheusPath:  getEnv("OBS_PROMETHEUS_PATH", "/metrics"),
		},
	}

	if cfg.HTTP.Port <= 0 {
		return Config{}, fmt.Errorf("invalid HTTP port: %d", cfg.HTTP.Port)
	}

	if cfg.GRPC.Port <= 0 {
		return Config{}, fmt.Errorf("invalid gRPC port: %d", cfg.GRPC.Port)
	}

	if !cfg.Cache.Enabled {
		cfg.Cache.Driver = "noop"
	}

	switch cfg.Cache.Driver {
	case "redis", "noop":
		// supported
	default:
		return Config{}, fmt.Errorf("unsupported cache driver: %s", cfg.Cache.Driver)
	}

	if cfg.Cache.Driver == "redis" && cfg.Cache.Redis.Addr == "" {
		return Config{}, fmt.Errorf("missing REDIS_ADDR for redis cache")
	}

	if cfg.Cache.DefaultTTL < 0 {
		cfg.Cache.DefaultTTL = time.Minute * 5
	}

	cfg.Observability.LogLevel = strings.ToLower(strings.TrimSpace(cfg.Observability.LogLevel))
	if cfg.Observability.LogLevel == "" {
		cfg.Observability.LogLevel = "info"
	}
	cfg.Observability.LogEncoding = strings.ToLower(strings.TrimSpace(cfg.Observability.LogEncoding))
	if cfg.Observability.LogEncoding == "" {
		cfg.Observability.LogEncoding = "json"
	}
	cfg.Observability.TraceExporter = strings.ToLower(strings.TrimSpace(cfg.Observability.TraceExporter))
	if cfg.Observability.TraceExporter == "" {
		cfg.Observability.TraceExporter = "stdout"
	}
	cfg.Observability.MetricsExporter = strings.ToLower(strings.TrimSpace(cfg.Observability.MetricsExporter))
	if cfg.Observability.MetricsExporter == "" {
		cfg.Observability.MetricsExporter = "prometheus"
	}

	if cfg.Observability.PrometheusPath == "" {
		cfg.Observability.PrometheusPath = "/metrics"
	} else if !strings.HasPrefix(cfg.Observability.PrometheusPath, "/") {
		cfg.Observability.PrometheusPath = "/" + cfg.Observability.PrometheusPath
	}

	if !cfg.Messaging.Enabled {
		cfg.Messaging.Driver = "noop"
	}

	switch cfg.Messaging.Driver {
	case "kafka", "noop":
		// supported
	default:
		return Config{}, fmt.Errorf("unsupported messaging driver: %s", cfg.Messaging.Driver)
	}

	if cfg.Messaging.Driver == "kafka" {
		if len(cfg.Messaging.Kafka.Brokers) == 0 {
			return Config{}, fmt.Errorf("KAFKA_BROKERS must be provided")
		}
		if cfg.Messaging.Kafka.Topic == "" {
			return Config{}, fmt.Errorf("KAFKA_TOPIC must be provided")
		}
		if cfg.Messaging.ConsumerGroup == "" {
			return Config{}, fmt.Errorf("KAFKA_CONSUMER_GROUP must be provided")
		}
	}

	if cfg.Messaging.Workers.Concurrency <= 0 {
		cfg.Messaging.Workers.Concurrency = 1
	}
	if cfg.Messaging.Workers.PollInterval <= 0 {
		cfg.Messaging.Workers.PollInterval = time.Second
	}

	if cfg.Database.WriterDSN == "" {
		return Config{}, fmt.Errorf("missing DB_WRITER_DSN")
	}

	if cfg.Database.ReaderDSN == "" {
		cfg.Database.ReaderDSN = cfg.Database.WriterDSN
	}

	return cfg, nil
}
