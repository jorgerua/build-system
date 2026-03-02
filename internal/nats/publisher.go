package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"github.com/nats-io/nats.go/jetstream"
)

// BuildJob is the message published by the webhook-server and consumed by the worker.
type BuildJob struct {
	RepoURL        string    `json:"repo_url"`
	SHA            string    `json:"sha"`
	CommitMessages []string  `json:"commit_messages"`
	InstallationID int64     `json:"installation_id"`
	PublishedAt    time.Time `json:"published_at"`
}

// Publisher publishes build job messages to NATS JetStream.
type Publisher struct {
	js      jetstream.JetStream
	subject string
}

// NewPublisher creates a Publisher.
func NewPublisher(js jetstream.JetStream, cfg *config.Config) *Publisher {
	return &Publisher{js: js, subject: cfg.NATS.Subject}
}

// Publish serializes and publishes a BuildJob.
func (p *Publisher) Publish(ctx context.Context, job BuildJob) error {
	if job.PublishedAt.IsZero() {
		job.PublishedAt = time.Now().UTC()
	}
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal build job: %w", err)
	}
	if _, err := p.js.Publish(ctx, p.subject, data); err != nil {
		return fmt.Errorf("nats publish: %w", err)
	}
	return nil
}
