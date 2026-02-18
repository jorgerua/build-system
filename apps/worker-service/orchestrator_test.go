package main

import (
	"context"
	"errors"
	"testing"
	"time"

	imageservice "github.com/jorgerua/build-system/libs/image-service"
	nxservice "github.com/jorgerua/build-system/libs/nx-service"
	"github.com/jorgerua/build-system/libs/shared"
	"go.uber.org/zap"
)

// Mock services for testing
type mockGitService struct {
	syncFunc func(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error)
}

func (m *mockGitService) SyncRepository(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
	if m.syncFunc != nil {
		return m.syncFunc(ctx, repo, commitHash)
	}
	return "/tmp/repo", nil
}

func (m *mockGitService) RepositoryExists(repoURL string) bool {
	return false
}

func (m *mockGitService) GetLocalPath(repoURL string) string {
	return "/tmp/repo"
}

type mockNXService struct {
	buildFunc func(ctx context.Context, repoPath string, config nxservice.BuildConfig) (*nxservice.BuildResult, error)
}

func (m *mockNXService) Build(ctx context.Context, repoPath string, config nxservice.BuildConfig) (*nxservice.BuildResult, error) {
	if m.buildFunc != nil {
		return m.buildFunc(ctx, repoPath, config)
	}
	return &nxservice.BuildResult{
		Success:      true,
		Duration:     time.Second,
		Output:       "build output",
		ErrorOutput:  "",
		ArtifactPath: "/tmp/artifacts",
	}, nil
}

func (m *mockNXService) DetectProjects(repoPath string) ([]string, error) {
	return []string{"project1"}, nil
}

type mockImageService struct {
	buildFunc func(ctx context.Context, config imageservice.ImageConfig) (*imageservice.ImageResult, error)
}

func (m *mockImageService) BuildImage(ctx context.Context, config imageservice.ImageConfig) (*imageservice.ImageResult, error) {
	if m.buildFunc != nil {
		return m.buildFunc(ctx, config)
	}
	return &imageservice.ImageResult{
		ImageID:  "sha256:abc123",
		Tags:     config.Tags,
		Size:     1024,
		Duration: time.Second,
	}, nil
}

func (m *mockImageService) TagImage(imageID string, tags []string) error {
	return nil
}

type mockCacheService struct{}

func (m *mockCacheService) GetCachePath(language shared.Language) string {
	return "/tmp/cache"
}

func (m *mockCacheService) InitializeCache(language shared.Language) error {
	return nil
}

func (m *mockCacheService) CleanCache(language shared.Language, olderThan time.Duration) error {
	return nil
}

func (m *mockCacheService) GetCacheSize(language shared.Language) (int64, error) {
	return 0, nil
}

// Test successful build job processing
func TestOrchestrator_ExecuteBuild_Success(t *testing.T) {
	logger := zap.NewNop()

	gitService := &mockGitService{}
	nxService := &mockNXService{}
	imageService := &mockImageService{}
	cacheService := &mockCacheService{}

	orchestrator := NewBuildOrchestrator(
		gitService,
		nxService,
		imageService,
		cacheService,
		logger,
	)

	job := &shared.BuildJob{
		ID: "test-job-1",
		Repository: shared.RepositoryInfo{
			URL:   "https://github.com/test/repo",
			Name:  "repo",
			Owner: "test",
		},
		CommitHash: "abc123",
		Branch:     "main",
		Status:     shared.JobStatusPending,
	}

	ctx := context.Background()
	err := orchestrator.ExecuteBuild(ctx, job)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify phases were added
	if len(job.Phases) != 3 {
		t.Errorf("expected 3 phases, got %d", len(job.Phases))
	}

	// Verify all phases succeeded
	for _, phase := range job.Phases {
		if !phase.Success {
			t.Errorf("phase %s failed: %s", phase.Phase, phase.Error)
		}
	}
}

// Test build job with git sync failure
func TestOrchestrator_ExecuteBuild_GitSyncFailure(t *testing.T) {
	logger := zap.NewNop()

	gitService := &mockGitService{
		syncFunc: func(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
			return "", errors.New("git sync failed")
		},
	}
	nxService := &mockNXService{}
	imageService := &mockImageService{}
	cacheService := &mockCacheService{}

	orchestrator := NewBuildOrchestrator(
		gitService,
		nxService,
		imageService,
		cacheService,
		logger,
	)

	job := &shared.BuildJob{
		ID: "test-job-2",
		Repository: shared.RepositoryInfo{
			URL:   "https://github.com/test/repo",
			Name:  "repo",
			Owner: "test",
		},
		CommitHash: "abc123",
		Branch:     "main",
		Status:     shared.JobStatusPending,
	}

	ctx := context.Background()
	err := orchestrator.ExecuteBuild(ctx, job)

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Verify git sync phase was added and failed
	if len(job.Phases) != 1 {
		t.Errorf("expected 1 phase, got %d", len(job.Phases))
	}

	if job.Phases[0].Success {
		t.Error("expected git sync phase to fail")
	}
}

// Test build job with NX build failure
func TestOrchestrator_ExecuteBuild_NXBuildFailure(t *testing.T) {
	logger := zap.NewNop()

	gitService := &mockGitService{}
	nxService := &mockNXService{
		buildFunc: func(ctx context.Context, repoPath string, config nxservice.BuildConfig) (*nxservice.BuildResult, error) {
			return nil, errors.New("nx build failed")
		},
	}
	imageService := &mockImageService{}
	cacheService := &mockCacheService{}

	orchestrator := NewBuildOrchestrator(
		gitService,
		nxService,
		imageService,
		cacheService,
		logger,
	)

	job := &shared.BuildJob{
		ID: "test-job-3",
		Repository: shared.RepositoryInfo{
			URL:   "https://github.com/test/repo",
			Name:  "repo",
			Owner: "test",
		},
		CommitHash: "abc123",
		Branch:     "main",
		Status:     shared.JobStatusPending,
	}

	ctx := context.Background()
	err := orchestrator.ExecuteBuild(ctx, job)

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Verify git sync succeeded but nx build failed
	if len(job.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(job.Phases))
	}

	if !job.Phases[0].Success {
		t.Error("expected git sync phase to succeed")
	}

	if job.Phases[1].Success {
		t.Error("expected nx build phase to fail")
	}
}

// Test build job with timeout
func TestOrchestrator_ExecuteBuild_Timeout(t *testing.T) {
	logger := zap.NewNop()

	gitService := &mockGitService{
		syncFunc: func(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
			// Simulate long operation
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(2 * time.Second):
				return "/tmp/repo", nil
			}
		},
	}
	nxService := &mockNXService{}
	imageService := &mockImageService{}
	cacheService := &mockCacheService{}

	orchestrator := NewBuildOrchestrator(
		gitService,
		nxService,
		imageService,
		cacheService,
		logger,
	)

	job := &shared.BuildJob{
		ID: "test-job-4",
		Repository: shared.RepositoryInfo{
			URL:   "https://github.com/test/repo",
			Name:  "repo",
			Owner: "test",
		},
		CommitHash: "abc123",
		Branch:     "main",
		Status:     shared.JobStatusPending,
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := orchestrator.ExecuteBuild(ctx, job)

	if err == nil {
		t.Error("expected timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

// Test retry with backoff
func TestOrchestrator_RetryWithBackoff_Success(t *testing.T) {
	logger := zap.NewNop()
	orchestrator := NewBuildOrchestrator(nil, nil, nil, nil, logger)

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	ctx := context.Background()
	err := orchestrator.retryWithBackoff(ctx, 3, 10*time.Millisecond, fn)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// Test retry exhaustion
func TestOrchestrator_RetryWithBackoff_Exhausted(t *testing.T) {
	logger := zap.NewNop()
	orchestrator := NewBuildOrchestrator(nil, nil, nil, nil, logger)

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("persistent error")
	}

	ctx := context.Background()
	err := orchestrator.retryWithBackoff(ctx, 3, 10*time.Millisecond, fn)

	if err == nil {
		t.Error("expected error, got nil")
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
