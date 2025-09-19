package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Additional-Code/atlas/internal/config"
	"github.com/Additional-Code/atlas/internal/messaging"
)

// HandlerRegistration binds message topics to handlers.
type HandlerRegistration struct {
	Topic   string
	Handler messaging.Handler
}

// Params collects dependencies via Fx.
type Params struct {
	fx.In

	Client        messaging.Client
	Logger        *zap.Logger
	Config        config.Config
	Registrations []HandlerRegistration `group:"worker.handlers"`
}

// Engine orchestrates background message consumption.
type Engine struct {
	client        messaging.Client
	logger        *zap.Logger
	cfg           config.Config
	registrations map[string]messaging.Handler
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
}

// NewEngine constructs the worker Engine.
func NewEngine(p Params) *Engine {
	reg := make(map[string]messaging.Handler, len(p.Registrations))
	for _, r := range p.Registrations {
		if r.Topic == "" || r.Handler == nil {
			continue
		}
		reg[r.Topic] = r.Handler
	}

	return &Engine{
		client:        p.Client,
		logger:        p.Logger,
		cfg:           p.Config,
		registrations: reg,
	}
}

// Module wires the engine into Fx lifecycle.
var Module = fx.Options(
	fx.Provide(NewEngine),
	fx.Invoke(func(lc fx.Lifecycle, engine *Engine) {
		lc.Append(fx.Hook{
			OnStart: engine.start,
			OnStop:  engine.stop,
		})
	}),
)

func (e *Engine) start(ctx context.Context) error {
	if !e.cfg.Messaging.Enabled || !e.cfg.Messaging.Workers.Enabled {
		e.logger.Info("worker engine disabled")

		return nil
	}
	if len(e.registrations) == 0 {
		e.logger.Info("worker engine has no handlers; skipping")

		return nil
	}

	concurrency := e.cfg.Messaging.Workers.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	runCtx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.wg = &sync.WaitGroup{}

	for i := 0; i < concurrency; i++ {
		workerID := i
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			e.consumeLoop(runCtx, workerID)
		}()
	}

	e.logger.Info("worker engine started", zap.Int("workers", concurrency))

	return nil
}

func (e *Engine) stop(ctx context.Context) error {
	if e.cancel == nil {
		return nil
	}
	e.cancel()
	done := make(chan struct{})
	go func() {
		if e.wg != nil {
			e.wg.Wait()
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		e.logger.Info("worker engine stopped")

		return nil
	}
}

func (e *Engine) consumeLoop(ctx context.Context, workerID int) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}

		err := e.client.Consume(ctx, func(msgCtx context.Context, msg messaging.Message) error {
			handler, ok := e.registrations[msg.Topic]
			if !ok {
				e.logger.Warn("no handler for topic", zap.String("topic", msg.Topic))

				return nil
			}

			e.logger.Debug("processing message", zap.String("topic", msg.Topic), zap.Int("worker", workerID))

			return handler(msgCtx, msg)
		})

		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}

		e.logger.Error("consume loop error", zap.Error(err))

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}

		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}
