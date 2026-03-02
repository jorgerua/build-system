package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// HandlerFunc processes a deserialized BuildJob.
// Returning a non-nil error causes the message to be nacked.
type HandlerFunc func(ctx context.Context, msg jetstream.Msg, job BuildJob) error

// Subscriber consumes build job messages from NATS JetStream.
type Subscriber struct {
	consumer         jetstream.Consumer
	cfg              *config.Config
	logger           *zap.Logger
	heartbeatSeconds time.Duration
}

// NewSubscriber creates a Subscriber.
func NewSubscriber(consumer jetstream.Consumer, cfg *config.Config, logger *zap.Logger) *Subscriber {
	return &Subscriber{
		consumer:         consumer,
		cfg:              cfg,
		logger:           logger,
		heartbeatSeconds: time.Duration(cfg.Worker.HeartbeatSeconds) * time.Second,
	}
}

// Subscribe starts consuming messages, calling handler for each.
// It sends periodic msg.InProgress() heartbeats so NATS does not
// redeliver the message while the handler is running.
func (s *Subscriber) Subscribe(ctx context.Context, handler HandlerFunc) error {
	msgCh, err := s.consumer.Messages()
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			msgCh.Stop()
			return ctx.Err()
		default:
		}

		msg, err := msgCh.Next()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.logger.Error("fetch message error", zap.Error(err))
			continue
		}

		go s.handle(ctx, msg, handler)
	}
}

func (s *Subscriber) handle(ctx context.Context, msg jetstream.Msg, handler HandlerFunc) {
	var job BuildJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		s.logger.Error("unmarshal build job failed",
			zap.Error(err),
			zap.String("raw", string(msg.Data())),
		)
		_ = msg.Nak()
		return
	}

	// Start heartbeat goroutine: sends InProgress every heartbeatSeconds
	// to prevent false redelivery during long-running processing.
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go s.heartbeat(heartbeatCtx, msg)

	if err := handler(ctx, msg, job); err != nil {
		s.logger.Error("build job handler error",
			zap.Error(err),
			zap.String("sha", job.SHA),
			zap.String("repo", job.RepoURL),
		)
		_ = msg.Nak()
		return
	}

	if err := msg.Ack(); err != nil {
		s.logger.Error("ack failed", zap.Error(err), zap.String("sha", job.SHA))
	}
}

func (s *Subscriber) heartbeat(ctx context.Context, msg jetstream.Msg) {
	ticker := time.NewTicker(s.heartbeatSeconds)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := msg.InProgress(); err != nil {
				return
			}
		}
	}
}
