package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewServer creates and registers an HTTP server with health check and webhook endpoint.
func NewServer(cfg *config.Config, handler *Handler, logger *zap.Logger, lc fx.Lifecycle) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/webhook", handler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", 8080),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				logger.Info("webhook server starting", zap.String("addr", srv.Addr))
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("webhook server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

// Module provides the webhook HTTP server via fx.
var Module = fx.Module("webhook",
	fx.Provide(NewHandler, NewServer),
)
