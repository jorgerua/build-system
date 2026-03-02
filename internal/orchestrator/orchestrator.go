package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	buildahpkg "github.com/jorgerua/build-system/container-build-service/internal/buildah"
	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"github.com/jorgerua/build-system/container-build-service/internal/detection"
	githubpkg "github.com/jorgerua/build-system/container-build-service/internal/github"
	metricspkg "github.com/jorgerua/build-system/container-build-service/internal/metrics"
	natspkg "github.com/jorgerua/build-system/container-build-service/internal/nats"
	"github.com/jorgerua/build-system/container-build-service/internal/semver"
	"github.com/jorgerua/build-system/container-build-service/internal/templates"
	"github.com/jorgerua/build-system/container-build-service/internal/tidb"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// Orchestrator processes build jobs from NATS.
type Orchestrator struct {
	cfg        *config.Config
	gh         *githubpkg.Client
	builder    *buildahpkg.Builder
	versions   *tidb.VersionRepository
	buildState *tidb.BuildStateRepository
	buildRec   *tidb.BuildRecordRepository
	subscriber *natspkg.Subscriber
	bm         *metricspkg.BuildMetrics
	logger     *zap.Logger
}

// New creates an Orchestrator.
func New(
	cfg *config.Config,
	gh *githubpkg.Client,
	builder *buildahpkg.Builder,
	versions *tidb.VersionRepository,
	buildState *tidb.BuildStateRepository,
	buildRec *tidb.BuildRecordRepository,
	subscriber *natspkg.Subscriber,
	bm *metricspkg.BuildMetrics,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		cfg:        cfg,
		gh:         gh,
		builder:    builder,
		versions:   versions,
		buildState: buildState,
		buildRec:   buildRec,
		subscriber: subscriber,
		bm:         bm,
		logger:     logger,
	}
}

// Run starts consuming build jobs until ctx is cancelled.
func (o *Orchestrator) Run(ctx context.Context) error {
	return o.subscriber.Subscribe(ctx, o.handleJob)
}

// handleJob is the NATS message handler. It processes a single build job.
// Returning an error causes the message to be nacked (used only for clone failures).
func (o *Orchestrator) handleJob(ctx context.Context, msg jetstream.Msg, job natspkg.BuildJob) error {
	log := o.logger.With(
		zap.String("sha", job.SHA),
		zap.String("repo", job.RepoURL),
	)
	log.Info("job received",
		zap.Time("published_at", job.PublishedAt),
		zap.Duration("queue_wait", time.Since(job.PublishedAt)),
	)

	jobID := job.SHA[:8] // short ID for temp paths
	repoDir := fmt.Sprintf("/tmp/repo-%s", jobID)
	defer os.RemoveAll(repoDir)

	// Clone repository. On failure: nack the message for retry.
	log.Info("clone started")
	if _, err := cloneRepo(ctx, o.gh, job.RepoURL, job.InstallationID, job.SHA, jobID); err != nil {
		log.Error("clone failed", zap.Error(err))
		return err // causes nack in subscriber
	}
	log.Info("clone complete", zap.String("repo_dir", repoDir))

	// Resolve base SHA for nx affected.
	baseSHA, err := o.buildState.GetLastSHA(ctx, job.RepoURL)
	if err != nil {
		log.Error("get last sha failed", zap.Error(err))
		return err
	}
	if baseSHA == "" {
		// First run: use the repository's initial commit.
		initial, err := initialCommitSHA(ctx, repoDir)
		if err != nil {
			log.Error("get initial sha failed", zap.Error(err))
			return err
		}
		baseSHA = initial
		log.Info("first run: using initial commit as base", zap.String("base_sha", baseSHA))
	}

	// Detect affected projects under apps/.
	projects, err := affectedProjects(ctx, repoDir, baseSHA, job.SHA)
	if err != nil {
		log.Error("nx affected failed", zap.Error(err))
		return err
	}
	log.Info("nx affected result",
		zap.Strings("projects", projects),
		zap.Int("count", len(projects)),
	)
	o.bm.QueueWaitTime(job.PublishedAt)
	o.bm.ProjectsAffected(len(projects))

	if len(projects) == 0 {
		log.Info("no affected projects, updating sha and acking")
		return o.finish(ctx, job.RepoURL, job.SHA, log)
	}

	// Dispatch parallel builds with concurrency semaphore.
	sem := make(chan struct{}, o.cfg.Worker.Concurrency)
	var wg sync.WaitGroup
	for _, project := range projects {
		wg.Add(1)
		sem <- struct{}{}
		go func(proj string) {
			defer wg.Done()
			defer func() { <-sem }()
			o.buildProject(ctx, job, jobID, repoDir, proj)
		}(project)
	}
	wg.Wait()

	log.Info("job completed", zap.String("sha", job.SHA))
	return o.finish(ctx, job.RepoURL, job.SHA, log)
}

// finish updates the last processed SHA and returns nil (triggering ack).
func (o *Orchestrator) finish(ctx context.Context, repo, sha string, log *zap.Logger) error {
	if err := o.buildState.UpdateLastSHA(ctx, repo, sha); err != nil {
		log.Error("update last sha failed", zap.Error(err))
		// Non-fatal: ack the message anyway to prevent reprocessing.
	}
	return nil
}

// buildProject runs the two-phase claim + build pipeline for a single project,
// with application-level retry.
func (o *Orchestrator) buildProject(ctx context.Context, job natspkg.BuildJob, jobID, repoDir, project string) {
	log := o.logger.With(
		zap.String("project", project),
		zap.String("sha", job.SHA),
	)

	stale := time.Duration(o.cfg.Worker.StaleClaimMinutes) * time.Minute

	// Two-phase claim (task 10.5).
	claimed, err := o.buildRec.Claim(ctx, project, job.SHA, stale)
	if err != nil {
		log.Error("claim failed", zap.Error(err))
		return
	}
	if !claimed {
		log.Info("build skipped (already claimed or completed)")
		return
	}

	// Application-level retry (task 10.7).
	maxRetries := o.cfg.Worker.MaxBuildRetries
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log := log.With(zap.Int("attempt", attempt))
		log.Info("build started")

		start := time.Now()
		lastErr = o.runBuildPipeline(ctx, job, jobID, repoDir, project, log)
		elapsed := time.Since(start)
		if lastErr == nil {
			log.Info("build completed")
			_ = o.buildRec.SetStatus(ctx, project, job.SHA, tidb.BuildStatusSuccess)
			o.bm.BuildStatus(project, "success")
			return
		}

		o.bm.RetryCount(project, attempt)
		log.Warn("build attempt failed", zap.Error(lastErr))
		_ = elapsed // duration emitted on success only (failed durations tracked via retry count)
		if attempt < maxRetries {
			backoff := time.Duration(attempt*attempt) * 5 * time.Second
			log.Info("retrying after backoff", zap.Duration("backoff", backoff))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}

	// All attempts exhausted — mark as permanent failure.
	log.Error("build failed permanently", zap.Error(lastErr))
	_ = o.buildRec.SetStatus(ctx, project, job.SHA, tidb.BuildStatusFailure)
	o.bm.BuildStatus(project, "failure")
}

// runBuildPipeline executes the full per-project build pipeline:
// language detection → version calc → Dockerfile gen → buildah bud → buildah push → version update.
func (o *Orchestrator) runBuildPipeline(
	ctx context.Context,
	job natspkg.BuildJob,
	jobID, repoDir, project string,
	log *zap.Logger,
) error {
	projectDir := filepath.Join(repoDir, "apps", project)

	// Language detection — unknown language is a skip, not a build failure.
	result, err := detection.Detect(projectDir)
	if err == nil {
		defer pipelineTimer(o, project, string(result.Language))(&err)
	}
	if err != nil {
		var unknownErr *detection.ErrUnknownLanguage
		if errors.As(err, &unknownErr) {
			log.Warn("unknown language, skipping project", zap.String("project_dir", projectDir))
			// Mark claim as failure so it doesn't block future builds.
			_ = o.buildRec.SetStatus(ctx, project, job.SHA, tidb.BuildStatusFailure)
			return nil // not a retryable error
		}
		return fmt.Errorf("language detection: %w", err)
	}

	// Calculate version.
	currentVersion, err := o.versions.Get(ctx, project)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	bump := semver.HighestBump(job.CommitMessages)
	newVersion, err := semver.Increment(currentVersion, bump)
	if err != nil {
		return fmt.Errorf("semver increment: %w", err)
	}

	// Generate Dockerfile.
	dockerfileContent, err := templates.Render(result.BuildTool, templates.TemplateVars{
		ProjectName:    project,
		ProjectSubpath: "apps/" + project,
		ArtifactName:   project,
	})
	if err != nil {
		return fmt.Errorf("render dockerfile: %w", err)
	}

	// Build image.
	imageRef := buildahpkg.ImageRef(o.cfg.Registry.URL, project, newVersion)
	if err := o.builder.Build(ctx, jobID, project, imageRef, repoDir, dockerfileContent); err != nil {
		return fmt.Errorf("buildah build: %w", err)
	}

	// Push image.
	if err := o.builder.Push(ctx, project, imageRef); err != nil {
		return fmt.Errorf("buildah push: %w", err)
	}

	// Update version in TiDB on success.
	if err := o.versions.Update(ctx, project, newVersion); err != nil {
		log.Error("version update failed", zap.Error(err), zap.String("new_version", newVersion))
		// Non-fatal: image was pushed successfully.
	}

	log.Info("build pipeline complete",
		zap.String("language", string(result.Language)),
		zap.String("version", newVersion),
		zap.String("image", imageRef),
	)
	return nil
}

// pipelineStart marks the beginning of a timed build for metrics.
// Usage: defer pipelineStart(o, project, language)()
func pipelineTimer(o *Orchestrator, project, language string) func(err *error) {
	start := time.Now()
	return func(err *error) {
		status := "success"
		if err != nil && *err != nil {
			status = "failure"
		}
		o.bm.BuildDuration(project, language, status, time.Since(start))
	}
}
