package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	githubpkg "github.com/jorgerua/build-system/container-build-service/internal/github"
	natspkg "github.com/jorgerua/build-system/container-build-service/internal/nats"
	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"go.uber.org/zap"
)

// pushPayload represents the relevant fields of a GitHub push webhook.
type pushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Commits []struct {
		Message string `json:"message"`
	} `json:"commits"`
}

// Handler handles incoming GitHub webhook requests.
type Handler struct {
	cfg       *config.Config
	publisher *natspkg.Publisher
	logger    *zap.Logger
}

// NewHandler creates a webhook Handler.
func NewHandler(cfg *config.Config, publisher *natspkg.Publisher, logger *zap.Logger) *Handler {
	return &Handler{cfg: cfg, publisher: publisher, logger: logger}
}

// ServeHTTP handles POST /webhook.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Warn("read body failed", zap.Error(err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Validate HMAC-SHA256 signature.
	sig := r.Header.Get("X-Hub-Signature-256")
	if err := githubpkg.ValidateWebhookSignature(h.cfg.GitHub.WebhookSecret, sig, body); err != nil {
		h.logger.Warn("webhook signature invalid", zap.Error(err))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Only process push events.
	if r.Header.Get("X-GitHub-Event") != "push" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload pushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Warn("unmarshal payload failed", zap.Error(err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Filter to main branch only.
	if payload.Ref != "refs/heads/main" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Collect commit messages.
	messages := make([]string, 0, len(payload.Commits))
	for _, c := range payload.Commits {
		messages = append(messages, c.Message)
	}

	// Publish build job; worker will generate the installation token.
	job := natspkg.BuildJob{
		RepoURL:        payload.Repository.CloneURL,
		SHA:            payload.After,
		CommitMessages: messages,
		InstallationID: payload.Installation.ID,
		PublishedAt:    time.Now().UTC(),
	}

	if err := h.publisher.Publish(context.Background(), job); err != nil {
		h.logger.Error("publish build job failed", zap.Error(err), zap.String("sha", job.SHA))
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	h.logger.Info("build job published",
		zap.String("repo", job.RepoURL),
		zap.String("sha", job.SHA),
		zap.Int64("installation_id", job.InstallationID),
	)
	w.WriteHeader(http.StatusAccepted)
}
