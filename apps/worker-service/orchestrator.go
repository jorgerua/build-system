package main

import (
	"context"
	"fmt"
	"math"
	"time"

	cacheservice "github.com/jorgerua/build-system/libs/cache-service"
	gitservice "github.com/jorgerua/build-system/libs/git-service"
	imageservice "github.com/jorgerua/build-system/libs/image-service"
	nxservice "github.com/jorgerua/build-system/libs/nx-service"
	"github.com/jorgerua/build-system/libs/shared"
	"go.uber.org/zap"
)

// BuildOrchestrator coordena as fases do build
type BuildOrchestrator struct {
	gitService   gitservice.GitService
	nxService    nxservice.NXService
	imageService imageservice.ImageService
	cacheService cacheservice.CacheService
	logger       *zap.Logger
}

// NewBuildOrchestrator cria uma nova instância do orchestrator
func NewBuildOrchestrator(
	gitService gitservice.GitService,
	nxService nxservice.NXService,
	imageService imageservice.ImageService,
	cacheService cacheservice.CacheService,
	logger *zap.Logger,
) *BuildOrchestrator {
	return &BuildOrchestrator{
		gitService:   gitService,
		nxService:    nxService,
		imageService: imageService,
		cacheService: cacheService,
		logger:       logger,
	}
}

// ExecuteBuild executa todas as fases do build
func (bo *BuildOrchestrator) ExecuteBuild(ctx context.Context, job *shared.BuildJob) error {
	bo.logger.Info("executing build",
		zap.String("job_id", job.ID),
		zap.String("repo", job.Repository.FullName()),
		zap.String("commit", job.CommitHash),
	)

	// Fase 1: Git Sync
	repoPath, err := bo.executeGitSync(ctx, job)
	if err != nil {
		return fmt.Errorf("git sync failed: %w", err)
	}

	// Fase 2: NX Build
	buildResult, err := bo.executeNXBuild(ctx, job, repoPath)
	if err != nil {
		return fmt.Errorf("nx build failed: %w", err)
	}

	// Fase 3: Image Build
	if err := bo.executeImageBuild(ctx, job, repoPath, buildResult); err != nil {
		return fmt.Errorf("image build failed: %w", err)
	}

	bo.logger.Info("build execution completed successfully",
		zap.String("job_id", job.ID),
	)

	return nil
}

// executeGitSync executa a fase de sincronização Git
func (bo *BuildOrchestrator) executeGitSync(ctx context.Context, job *shared.BuildJob) (string, error) {
	phase := shared.PhaseMetric{
		Phase:     shared.BuildPhaseGitSync,
		StartTime: time.Now(),
	}

	bo.logger.Info("starting git sync phase",
		zap.String("job_id", job.ID),
		zap.String("repo", job.Repository.FullName()),
		zap.String("commit", job.CommitHash),
	)

	// Execute git sync with retry
	var repoPath string
	var err error

	err = bo.retryWithBackoff(ctx, 3, time.Second, func() error {
		repoPath, err = bo.gitService.SyncRepository(ctx, job.Repository, job.CommitHash)
		return err
	})

	phase.EndTime = time.Now()
	phase.CalculateDuration()

	if err != nil {
		phase.Success = false
		phase.Error = err.Error()
		job.AddPhase(phase)

		bo.logger.Error("git sync phase failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
			zap.Duration("duration", phase.Duration),
		)
		return "", err
	}

	phase.Success = true
	job.AddPhase(phase)

	bo.logger.Info("git sync phase completed",
		zap.String("job_id", job.ID),
		zap.String("repo_path", repoPath),
		zap.Duration("duration", phase.Duration),
	)

	return repoPath, nil
}

// executeNXBuild executa a fase de build NX
func (bo *BuildOrchestrator) executeNXBuild(ctx context.Context, job *shared.BuildJob, repoPath string) (*nxservice.BuildResult, error) {
	phase := shared.PhaseMetric{
		Phase:     shared.BuildPhaseNXBuild,
		StartTime: time.Now(),
	}

	bo.logger.Info("starting nx build phase",
		zap.String("job_id", job.ID),
		zap.String("repo_path", repoPath),
	)

	// Detect language
	language := shared.LanguageUnknown
	// Language detection will be done by NXService

	// Initialize cache for detected language
	if language.IsSupported() {
		if err := bo.cacheService.InitializeCache(language); err != nil {
			bo.logger.Warn("failed to initialize cache",
				zap.String("language", string(language)),
				zap.Error(err),
			)
		}
	}

	// Get cache path
	cachePath := bo.cacheService.GetCachePath(language)

	// Build configuration
	buildConfig := nxservice.BuildConfig{
		CachePath:   cachePath,
		Language:    language,
		Environment: make(map[string]string),
	}

	// Execute build
	var buildResult *nxservice.BuildResult
	var err error

	err = bo.retryWithBackoff(ctx, 1, time.Second, func() error {
		buildResult, err = bo.nxService.Build(ctx, repoPath, buildConfig)
		return err
	})

	phase.EndTime = time.Now()
	phase.CalculateDuration()

	if err != nil {
		phase.Success = false
		phase.Error = err.Error()
		job.AddPhase(phase)

		bo.logger.Error("nx build phase failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
			zap.Duration("duration", phase.Duration),
		)
		return nil, err
	}

	phase.Success = true
	job.AddPhase(phase)

	bo.logger.Info("nx build phase completed",
		zap.String("job_id", job.ID),
		zap.Bool("success", buildResult.Success),
		zap.Duration("duration", phase.Duration),
	)

	return buildResult, nil
}

// executeImageBuild executa a fase de build de imagem OCI
func (bo *BuildOrchestrator) executeImageBuild(ctx context.Context, job *shared.BuildJob, repoPath string, buildResult *nxservice.BuildResult) error {
	phase := shared.PhaseMetric{
		Phase:     shared.BuildPhaseImageBuild,
		StartTime: time.Now(),
	}

	bo.logger.Info("starting image build phase",
		zap.String("job_id", job.ID),
		zap.String("repo_path", repoPath),
	)

	// Generate image tags
	tags := imageservice.GenerateImageTags(
		job.Repository.Name,
		job.CommitHash,
		job.Branch,
	)

	bo.logger.Debug("generated image tags",
		zap.String("job_id", job.ID),
		zap.Strings("tags", tags),
	)

	// Image configuration
	imageConfig := imageservice.ImageConfig{
		ContextPath:    repoPath,
		DockerfilePath: "", // Will be auto-detected
		Tags:           tags,
		BuildArgs:      make(map[string]string),
	}

	// Execute image build with retry
	var imageResult *imageservice.ImageResult
	var err error

	err = bo.retryWithBackoff(ctx, 2, time.Second, func() error {
		imageResult, err = bo.imageService.BuildImage(ctx, imageConfig)
		return err
	})

	phase.EndTime = time.Now()
	phase.CalculateDuration()

	if err != nil {
		phase.Success = false
		phase.Error = err.Error()
		job.AddPhase(phase)

		bo.logger.Error("image build phase failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
			zap.Duration("duration", phase.Duration),
		)
		return err
	}

	phase.Success = true
	job.AddPhase(phase)

	bo.logger.Info("image build phase completed",
		zap.String("job_id", job.ID),
		zap.String("image_id", imageResult.ImageID),
		zap.Strings("tags", imageResult.Tags),
		zap.Duration("duration", phase.Duration),
	)

	return nil
}

// retryWithBackoff executa uma função com retry e backoff exponencial
func (bo *BuildOrchestrator) retryWithBackoff(ctx context.Context, maxRetries int, initialDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		bo.logger.Debug("executing operation",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
		)

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		bo.logger.Warn("operation failed",
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		// If not the last retry, wait with backoff
		if attempt < maxRetries {
			bo.logger.Debug("waiting before retry",
				zap.Duration("delay", delay),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Exponential backoff with jitter
				delay = time.Duration(float64(delay) * math.Pow(2, float64(attempt-1)))
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, lastErr)
}
