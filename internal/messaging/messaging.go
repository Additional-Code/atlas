package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
)

// Message represents a message consumed from the bus.
type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string]string
	Offset  int64
	Time    time.Time
}

// Handler processes an inbound message.
type Handler func(context.Context, Message) error

// Client is the pluggable messaging abstraction.
type Client interface {
	Publish(ctx context.Context, key []byte, value []byte) error
	Consume(ctx context.Context, handler Handler) error
	Topic() string
}

// Module wires the messaging client.
var Module = fx.Provide(NewClient)

// noopClient is used when messaging is disabled.
type noopClient struct {
	topic string
}

func (n noopClient) Publish(context.Context, []byte, []byte) error { return nil }
func (n noopClient) Consume(ctx context.Context, handler Handler) error {
	<-ctx.Done()
	return ctx.Err()
}
func (n noopClient) Topic() string { return n.topic }

// kafkaClient implements the Client via kafka-go.
type kafkaClient struct {
	writer *kafka.Writer
	reader *kafka.Reader
	topic  string
	logger *zap.Logger
}

func (k *kafkaClient) Publish(ctx context.Context, key []byte, value []byte) error {
	msg := kafka.Message{Topic: k.topic, Key: key, Value: value}
	return k.writer.WriteMessages(ctx, msg)
}

func (k *kafkaClient) Consume(ctx context.Context, handler Handler) error {
	for {
		msg, err := k.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			k.logger.Error("kafka fetch failed", zap.Error(err))

			time.Sleep(time.Second)
			continue
		}

		wrapped := Message{
			Topic:  msg.Topic,
			Key:    append([]byte(nil), msg.Key...),
			Value:  append([]byte(nil), msg.Value...),
			Offset: msg.Offset,
			Time:   msg.Time,
			Headers: func() map[string]string {
				if len(msg.Headers) == 0 {
					return nil
				}
				m := make(map[string]string, len(msg.Headers))
				for _, h := range msg.Headers {
					m[h.Key] = string(h.Value)
				}
				return m
			}(),
		}

		if err := handler(ctx, wrapped); err != nil {
			k.logger.Error("message handler failed", zap.Error(err), zap.Int64("offset", msg.Offset))

			// Handler signals failure; skip commit to allow retry.
			continue
		}

		if err := k.reader.CommitMessages(ctx, msg); err != nil {
			k.logger.Warn("commit failed", zap.Error(err))

		}
	}
}

func (k *kafkaClient) Topic() string { return k.topic }

// NewClient builds a messaging client based on configuration.
func NewClient(lc fx.Lifecycle, cfg config.Config, logger *zap.Logger) (Client, error) {
	if !cfg.Messaging.Enabled || cfg.Messaging.Driver == "noop" {
		logger.Info("messaging disabled; using noop client")

		return noopClient{topic: cfg.Messaging.Kafka.Topic}, nil
	}

	switch cfg.Messaging.Driver {
	case "kafka":
		return newKafkaClient(lc, cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported messaging driver: %s", cfg.Messaging.Driver)
	}
}

func newKafkaClient(lc fx.Lifecycle, cfg config.Config, logger *zap.Logger) (Client, error) {
	topic := cfg.Messaging.Kafka.Topic

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Messaging.Kafka.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		Logger:       kafkaLogger{logger: logger},
		ErrorLogger:  kafkaLogger{logger: logger},
	}

	readerConfig := kafka.ReaderConfig{
		Brokers:        cfg.Messaging.Kafka.Brokers,
		GroupID:        cfg.Messaging.ConsumerGroup,
		Topic:          topic,
		MinBytes:       cfg.Messaging.Kafka.MinBytes,
		MaxBytes:       cfg.Messaging.Kafka.MaxBytes,
		CommitInterval: cfg.Messaging.Kafka.CommitInterval,
		Dialer: &kafka.Dialer{
			Timeout:  cfg.Messaging.Kafka.ConnectTimeout,
			ClientID: cfg.Messaging.Kafka.ClientID,
		},
	}

	reader := kafka.NewReader(readerConfig)

	client := &kafkaClient{writer: writer, reader: reader, topic: topic, logger: logger}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing kafka client")

			if err := writer.Close(); err != nil {
				return err
			}
			return reader.Close()
		},
	})

	return client, nil
}

type kafkaLogger struct {
	logger *zap.Logger
}

func (k kafkaLogger) Printf(msg string, args ...interface{}) {
	k.logger.Sugar().Debugf(msg, args...)

}
