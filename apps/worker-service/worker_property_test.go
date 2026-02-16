package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	imageservice "github.com/oci-build-system/libs/image-service"
	natsclient "github.com/oci-build-system/libs/nats-client"
	nxservice "github.com/oci-build-system/libs/nx-service"
	"github.com/oci-build-system/libs/shared"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Feature: oci-build-system, Property 3: Enfileiramento de webhooks simultâneos
// Valida: Requisitos 1.5
//
// Para qualquer conjunto de webhooks válidos recebidos simultaneamente,
// todos devem ser adicionados à fila de builds e nenhum deve ser perdido.
func TestProperty_WebhookEnqueueing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("simultaneous webhooks are all enqueued", prop.ForAll(
		func(jobCount int) bool {
			// Create worker service with sufficient queue size
			logger := zap.NewNop()
			config := &shared.Config{
				Worker: shared.WorkerConfig{
					PoolSize:  1,
					QueueSize: jobCount + 10, // Ensure queue is large enough
				},
			}

			// Create a real NATS client (will be used only for type compatibility)
			natsClient := natsclient.NewClient(logger)

			ws := &WorkerService{
				config:       config,
				logger:       logger,
				natsClient:   natsClient,
				gitService:   &mockGitService{},
				nxService:    &mockNXService{},
				imageService: &mockImageService{},
				cacheService: &mockCacheService{},
				jobQueue:     make(chan *shared.BuildJob, config.Worker.QueueSize),
				stopChan:     make(chan struct{}),
			}

			// Generate jobs
			jobs := make([]*shared.BuildJob, jobCount)
			for i := 0; i < jobCount; i++ {
				jobs[i] = &shared.BuildJob{
					ID: generateJobID(i),
					Repository: shared.RepositoryInfo{
						URL:   "https://github.com/test/repo",
						Name:  "repo",
						Owner: "test",
					},
					CommitHash: generateCommitHash(i),
					Branch:     "main",
					Status:     shared.JobStatusPending,
					CreatedAt:  time.Now(),
				}
			}

			// Send all jobs simultaneously
			var wg sync.WaitGroup
			for _, job := range jobs {
				wg.Add(1)
				go func(j *shared.BuildJob) {
					defer wg.Done()
					data, _ := json.Marshal(j)
					msg := &nats.Msg{
						Subject: "builds.webhook",
						Data:    data,
					}
					ws.handleMessage(msg)
				}(job)
			}

			wg.Wait()

			// Verify all jobs were enqueued
			enqueuedCount := len(ws.jobQueue)
			return enqueuedCount == jobCount
		},
		gen.IntRange(1, 50), // Test with 1 to 50 simultaneous webhooks
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper functions
func generateJobID(index int) string {
	return "job-" + string(rune('a'+index%26)) + string(rune('0'+index/26))
}

func generateCommitHash(index int) string {
	// Generate a simple but unique commit hash
	hash := ""
	for i := 0; i < 40; i++ {
		hash += string(rune('0' + (index+i)%10))
	}
	return hash
}

// Feature: oci-build-system, Property 7: Interrupção em falha de build
// Valida: Requisitos 3.3
//
// Para qualquer build NX que retorne código de saída diferente de zero,
// o sistema deve marcar o BuildJob como failed, registrar o erro,
// e não prosseguir para construção de imagem.
func TestProperty_BuildFailureInterruption(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("build failure prevents image build", prop.ForAll(
		func(errorMsg string) bool {
			logger := zap.NewNop()

			// Track if image build was called
			imageBuildCalled := false

			gitService := &mockGitService{}
			nxService := &mockNXService{
				buildFunc: func(ctx context.Context, repoPath string, config nxservice.BuildConfig) (*nxservice.BuildResult, error) {
					// Simulate build failure
					return nil, errors.New(errorMsg)
				},
			}
			imageService := &mockImageService{
				buildFunc: func(ctx context.Context, config imageservice.ImageConfig) (*imageservice.ImageResult, error) {
					imageBuildCalled = true
					return nil, errors.New("should not be called")
				},
			}
			cacheService := &mockCacheService{}

			orchestrator := NewBuildOrchestrator(
				gitService,
				nxService,
				imageService,
				cacheService,
				logger,
			)

			job := &shared.BuildJob{
				ID: "test-job",
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

			// Verify build failed
			if err == nil {
				return false
			}

			// Verify image build was not called
			if imageBuildCalled {
				return false
			}

			// Verify git sync succeeded but nx build failed
			if len(job.Phases) != 2 {
				return false
			}

			if !job.Phases[0].Success {
				return false
			}

			if job.Phases[1].Success {
				return false
			}

			// Verify error was recorded
			if job.Phases[1].Error == "" {
				return false
			}

			return true
		},
		gen.AnyString().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: oci-build-system, Property 8: Progressão após build bem-sucedido
// Valida: Requisitos 3.4
//
// Para qualquer build NX que retorne código de saída zero,
// o sistema deve prosseguir para a fase de construção de imagem OCI.
func TestProperty_BuildSuccessProgression(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("successful build proceeds to image build", prop.ForAll(
		func(commitHash string, branch string) bool {
			logger := zap.NewNop()

			// Track if image build was called
			imageBuildCalled := false

			gitService := &mockGitService{}
			nxService := &mockNXService{
				buildFunc: func(ctx context.Context, repoPath string, config nxservice.BuildConfig) (*nxservice.BuildResult, error) {
					// Simulate successful build
					return &nxservice.BuildResult{
						Success:      true,
						Duration:     time.Second,
						Output:       "build successful",
						ErrorOutput:  "",
						ArtifactPath: "/tmp/artifacts",
					}, nil
				},
			}
			imageService := &mockImageService{
				buildFunc: func(ctx context.Context, config imageservice.ImageConfig) (*imageservice.ImageResult, error) {
					imageBuildCalled = true
					return &imageservice.ImageResult{
						ImageID:  "sha256:abc123",
						Tags:     config.Tags,
						Size:     1024,
						Duration: time.Second,
					}, nil
				},
			}
			cacheService := &mockCacheService{}

			orchestrator := NewBuildOrchestrator(
				gitService,
				nxService,
				imageService,
				cacheService,
				logger,
			)

			job := &shared.BuildJob{
				ID: "test-job",
				Repository: shared.RepositoryInfo{
					URL:   "https://github.com/test/repo",
					Name:  "repo",
					Owner: "test",
				},
				CommitHash: commitHash,
				Branch:     branch,
				Status:     shared.JobStatusPending,
			}

			ctx := context.Background()
			err := orchestrator.ExecuteBuild(ctx, job)

			// Verify build succeeded
			if err != nil {
				return false
			}

			// Verify image build was called
			if !imageBuildCalled {
				return false
			}

			// Verify all three phases completed successfully
			if len(job.Phases) != 3 {
				return false
			}

			for _, phase := range job.Phases {
				if !phase.Success {
					return false
				}
			}

			return true
		},
		gen.Identifier().SuchThat(func(s string) bool { return len(s) >= 7 }),
		gen.OneConstOf("main", "develop", "feature/test", "refs/heads/main"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: oci-build-system, Property 17: Logging de início e fim de job
// Valida: Requisitos 7.1
//
// Para qualquer BuildJob processado, deve existir uma entrada de log
// marcando o início (com timestamp) e outra marcando o fim (com timestamp e duração).
func TestProperty_JobLogging(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("job has start and completion timestamps", prop.ForAll(
		func(jobID string, repoName string) bool {
			// Skip invalid inputs
			if len(jobID) < 3 || len(repoName) < 3 {
				return true
			}

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
				ID: jobID,
				Repository: shared.RepositoryInfo{
					URL:   "https://github.com/test/" + repoName,
					Name:  repoName,
					Owner: "test",
				},
				CommitHash: "abc123def456",
				Branch:     "main",
				Status:     shared.JobStatusPending,
				CreatedAt:  time.Now(),
			}

			// Mark job as started (simulating worker behavior)
			job.MarkStarted()

			// Verify StartedAt is set
			if job.StartedAt == nil {
				return false
			}

			// Verify status is running
			if job.Status != shared.JobStatusRunning {
				return false
			}

			ctx := context.Background()
			err := orchestrator.ExecuteBuild(ctx, job)

			// After execution, mark as completed or failed
			if err != nil {
				job.MarkFailed(err.Error())
			} else {
				job.MarkCompleted()
			}

			// Verify CompletedAt is set
			if job.CompletedAt == nil {
				return false
			}

			// Verify duration is calculated (allow 0 for very fast mock execution)
			// Duration should be non-negative and reasonable
			if job.Duration < 0 || job.Duration > 1*time.Hour {
				return false
			}

			// Verify CompletedAt is after or equal to StartedAt (can be equal for very fast execution)
			if job.CompletedAt.Before(*job.StartedAt) {
				return false
			}

			// Verify status is terminal
			if !job.Status.IsTerminal() {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: oci-build-system, Property 19: Métricas de duração por fase
// Valida: Requisitos 7.3
//
// Para qualquer BuildJob, cada fase (git_sync, nx_build, image_build)
// deve ter sua duração registrada individualmente nos logs.
func TestProperty_PhaseMetrics(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("each phase has duration metrics", prop.ForAll(
		func(repoURL string) bool {
			// Skip invalid inputs
			if len(repoURL) < 10 {
				return true
			}

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
				ID: "test-job",
				Repository: shared.RepositoryInfo{
					URL:   repoURL,
					Name:  "repo",
					Owner: "test",
				},
				CommitHash: "abc123",
				Branch:     "main",
				Status:     shared.JobStatusPending,
			}

			ctx := context.Background()
			_ = orchestrator.ExecuteBuild(ctx, job)

			// Verify all three phases are present
			if len(job.Phases) != 3 {
				return false
			}

			// Verify each phase has the correct phase type
			expectedPhases := []shared.BuildPhase{
				shared.BuildPhaseGitSync,
				shared.BuildPhaseNXBuild,
				shared.BuildPhaseImageBuild,
			}

			for i, expectedPhase := range expectedPhases {
				if job.Phases[i].Phase != expectedPhase {
					return false
				}

				// Verify phase has start and end times
				if job.Phases[i].StartTime.IsZero() {
					return false
				}

				if job.Phases[i].EndTime.IsZero() {
					return false
				}

				// Verify duration is calculated (allow 0 for very fast mock execution)
				// Duration should be non-negative
				if job.Phases[i].Duration < 0 {
					return false
				}

				// Verify end time is after or equal to start time (can be equal for very fast execution)
				if job.Phases[i].EndTime.Before(job.Phases[i].StartTime) {
					return false
				}

				// Verify duration matches the time difference
				expectedDuration := job.Phases[i].EndTime.Sub(job.Phases[i].StartTime)
				if job.Phases[i].Duration != expectedDuration {
					return false
				}
			}

			return true
		},
		gen.Identifier().Map(func(id string) string {
			return "https://github.com/test/" + id
		}),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
