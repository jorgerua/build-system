package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Params groups fx dependencies for NATS setup.
type Params struct {
	fx.In
	Config *config.Config
	Logger *zap.Logger
}

// Result groups fx outputs for NATS.
type Result struct {
	fx.Out
	Conn      *nats.Conn
	JetStream jetstream.JetStream
	Consumer  jetstream.Consumer
}

// New establishes the NATS connection, creates/updates the stream and
// durable consumer, and returns them for injection.
func New(p Params, lc fx.Lifecycle) (Result, error) {
	nc, err := nats.Connect(p.Config.NATS.URL)
	if err != nil {
		return Result{}, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return Result{}, fmt.Errorf("jetstream init: %w", err)
	}

	ctx := context.Background()
	ackWait := time.Duration(p.Config.NATS.AckWaitSeconds) * time.Second

	// Create or update the stream.
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     p.Config.NATS.StreamName,
		Subjects: []string{p.Config.NATS.Subject},
	})
	if err != nil {
		nc.Close()
		return Result{}, fmt.Errorf("stream create/update: %w", err)
	}

	// Create or update the durable consumer.
	//  - AckWait: 5 min (workers send heartbeats every 2 min to prevent false redelivery)
	//  - MaxDelivers: 3  (crash-recovery only; build retries are application-level)
	consumer, err := js.CreateOrUpdateConsumer(ctx, p.Config.NATS.StreamName, jetstream.ConsumerConfig{
		Durable:       p.Config.NATS.ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       ackWait,
		MaxDeliver:    p.Config.NATS.MaxDelivers,
		FilterSubject: p.Config.NATS.Subject,
	})
	if err != nil {
		nc.Close()
		return Result{}, fmt.Errorf("consumer create/update: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			nc.Close()
			return nil
		},
	})

	p.Logger.Info("nats connected",
		zap.String("url", p.Config.NATS.URL),
		zap.String("stream", p.Config.NATS.StreamName),
		zap.String("consumer", p.Config.NATS.ConsumerName),
	)

	return Result{
		Conn:      nc,
		JetStream: js,
		Consumer:  consumer,
	}, nil
}

// Module provides NATS connection, JetStream, and Consumer via fx.
var Module = fx.Module("nats",
	fx.Provide(New),
)
