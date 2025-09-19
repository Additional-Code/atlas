package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/Additional-Code/atlas/internal/config"
)

// Module exposes the gRPC server and lifecycle hooks to Fx.
var Module = fx.Module("grpc_server",
	fx.Provide(NewServer),
	fx.Invoke(Run),
)

// NewServer builds a gRPC server with basic unary/stream logging interceptors.
func NewServer(logger *zap.Logger) *grpc.Server {
	unary := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)
		if err != nil {
			logger.Warn("grpc unary call finished", zap.String("method", info.FullMethod), zap.Duration("duration", duration), zap.Error(err))
		} else {
			logger.Info("grpc unary call finished", zap.String("method", info.FullMethod), zap.Duration("duration", duration))
		}
		return resp, err
	}

	stream := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		duration := time.Since(start)
		if err != nil {
			logger.Warn("grpc stream call finished", zap.String("method", info.FullMethod), zap.Duration("duration", duration), zap.Error(err))
		} else {
			logger.Info("grpc stream call finished", zap.String("method", info.FullMethod), zap.Duration("duration", duration))
		}
		return err
	}

	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(unary),
		grpc.ChainStreamInterceptor(stream),
	)
}

// Run binds the gRPC server to the configured host/port and manages lifecycle.
func Run(lc fx.Lifecycle, cfg config.Config, server *grpc.Server, logger *zap.Logger) {
	addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	var listener net.Listener

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("listen grpc: %w", err)
			}
			listener = ln
			logger.Info("starting gRPC server", zap.String("addr", addr))
			go func() {
				if err := server.Serve(listener); err != nil {
					logger.Fatal("grpc server failed", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping gRPC server")
			stopped := make(chan struct{})
			go func() {
				server.GracefulStop()
				close(stopped)
			}()

			select {
			case <-ctx.Done():
				server.Stop()
				return ctx.Err()
			case <-stopped:
				if listener != nil {
					_ = listener.Close()
				}
				return nil
			}
		},
	})
}
