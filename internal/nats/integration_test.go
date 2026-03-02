package nats_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jorgerua/build-system/container-build-service/internal/config"
	natspkg "github.com/jorgerua/build-system/container-build-service/internal/nats"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TestNATSPublishSubscribe tests the full publish/subscribe round trip.
// Requires a running NATS server with JetStream enabled.
// Set NATS_URL env var to enable (e.g., NATS_URL=nats://localhost:4222).
func TestNATSPublishSubscribe(t *testing.T) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Skip("NATS_URL not set — skipping integration test")
	}

	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:            natsURL,
			StreamName:     "TEST_BUILDS",
			Subject:        "test.builds.jobs",
			ConsumerName:   "test-worker",
			AckWaitSeconds: 30,
			MaxDelivers:    3,
		},
		Worker: config.WorkerConfig{
			HeartbeatSeconds: 5,
		},
	}

	nc, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}

	ctx := context.Background()

	// Create stream.
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     cfg.NATS.StreamName,
		Subjects: []string{cfg.NATS.Subject},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer js.DeleteStream(ctx, cfg.NATS.StreamName) //nolint:errcheck

	// Publish a job.
	pub := natspkg.NewPublisher(js, cfg)
	job := natspkg.BuildJob{
		RepoURL:        "https://github.com/test/repo",
		SHA:            "abc123def456abc123def456abc123def456abc1",
		CommitMessages: []string{"feat: test feature"},
		InstallationID: 12345,
	}
	if err := pub.Publish(ctx, job); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Consume and verify.
	consumer, err := js.CreateOrUpdateConsumer(ctx, cfg.NATS.StreamName, jetstream.ConsumerConfig{
		Durable:       cfg.NATS.ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: cfg.NATS.Subject,
	})
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}

	received := make(chan natspkg.BuildJob, 1)
	go func() {
		msgs, _ := consumer.Messages()
		defer msgs.Stop()
		msg, _ := msgs.Next()
		if msg != nil {
			_ = msg.Ack()
		}
	}()

	select {
	case got := <-received:
		if got.SHA != job.SHA {
			t.Errorf("sha: got %q, want %q", got.SHA, job.SHA)
		}
	case <-time.After(5 * time.Second):
		t.Error("timed out waiting for message")
	}
}
