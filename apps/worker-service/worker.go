package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	cacheservice "github.com/oci-build-system/libs/cache-service"
	gitservice "github.com/oci-build-system/libs/git-service"
	imageservice "github.com/oci-build-system/libs/image-service"
	natsclient "github.com/oci-build-system/libs/nats-client"
	nxservice "github.com/oci-build-system/libs/nx-service"
	"github.com/oci-build-system/libs/shared"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// WorkerService gerencia o processamento de builds
type WorkerService struct {
	config       *shared.Config
	logger       *zap.Logger
	natsClient   *natsclient.Client
	gitService   gitservice.GitService
	nxService    nxservice.NXService
	imageService imageservice.ImageService
	cacheService cacheservice.CacheService
	jobQueue     chan *shared.BuildJob
	stopChan     chan struct{}
	wg           sync.WaitGroup
	subscription *nats.Subscription
}

// Start inicia o worker service
func (ws *WorkerService) Start() error {
	ws.logger.Info("starting worker service",
		zap.Int("pool_size", ws.config.Worker.PoolSize),
		zap.Int("queue_size", ws.config.Worker.QueueSize),
	)

	// Subscribe to NATS subject
	sub, err := ws.natsClient.Subscribe("builds.webhook", ws.handleMessage)
	if err != nil {
		return fmt.Errorf("failed to subscribe to builds.webhook: %w", err)
	}
	ws.subscription = sub

	// Start worker pool
	for i := 0; i < ws.config.Worker.PoolSize; i++ {
		ws.wg.Add(1)
		go ws.worker(i)
	}

	ws.logger.Info("worker service started successfully",
		zap.Int("workers", ws.config.Worker.PoolSize),
	)

	return nil
}

// Shutdown para o worker service gracefully
func (ws *WorkerService) Shutdown(ctx context.Context) error {
	ws.logger.Info("shutting down worker service")

	// Unsubscribe from NATS
	if ws.subscription != nil {
		if err := ws.subscription.Unsubscribe(); err != nil {
			ws.logger.Warn("error unsubscribing from NATS", zap.Error(err))
		}
	}

	// Signal workers to stop
	close(ws.stopChan)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		ws.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		ws.logger.Info("all workers stopped gracefully")
	case <-ctx.Done():
		ws.logger.Warn("shutdown timeout exceeded, some workers may not have finished")
	}

	return nil
}

// handleMessage processa mensagens recebidas do NATS
func (ws *WorkerService) handleMessage(msg *nats.Msg) {
	ws.logger.Debug("received message from NATS",
		zap.String("subject", msg.Subject),
		zap.Int("size", len(msg.Data)),
	)

	// Parse build job
	var job shared.BuildJob
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		ws.logger.Error("failed to unmarshal build job",
			zap.Error(err),
			zap.String("data", string(msg.Data)),
		)
		return
	}

	// Validate job
	if !job.IsValid() {
		ws.logger.Error("received invalid build job",
			zap.String("job_id", job.ID),
		)
		return
	}

	ws.logger.Info("enqueueing build job",
		zap.String("job_id", job.ID),
		zap.String("repo", job.Repository.FullName()),
		zap.String("commit", job.CommitHash),
	)

	// Try to enqueue job (non-blocking)
	select {
	case ws.jobQueue <- &job:
		ws.logger.Debug("job enqueued successfully", zap.String("job_id", job.ID))
	default:
		ws.logger.Error("job queue is full, rejecting job",
			zap.String("job_id", job.ID),
			zap.Int("queue_size", ws.config.Worker.QueueSize),
		)
		// Publish failure status
		job.MarkFailed("queue is full")
		ws.publishJobStatus(&job)
	}
}

// worker processa jobs da fila
func (ws *WorkerService) worker(id int) {
	defer ws.wg.Done()

	ws.logger.Info("worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-ws.stopChan:
			ws.logger.Info("worker stopping", zap.Int("worker_id", id))
			return

		case job := <-ws.jobQueue:
			ws.logger.Info("worker processing job",
				zap.Int("worker_id", id),
				zap.String("job_id", job.ID),
				zap.String("repo", job.Repository.FullName()),
			)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ws.config.Worker.Timeout)*time.Second)

			// Process job
			ws.processJob(ctx, job)

			cancel()
		}
	}
}

// processJob processa um build job
func (ws *WorkerService) processJob(ctx context.Context, job *shared.BuildJob) {
	// Mark job as started
	job.MarkStarted()
	ws.publishJobStatus(job)

	ws.logger.Info("starting build job",
		zap.String("job_id", job.ID),
		zap.String("repo", job.Repository.FullName()),
		zap.String("commit", job.CommitHash),
		zap.String("branch", job.Branch),
	)

	// Create orchestrator
	orchestrator := NewBuildOrchestrator(
		ws.gitService,
		ws.nxService,
		ws.imageService,
		ws.cacheService,
		ws.logger,
	)

	// Execute build
	err := orchestrator.ExecuteBuild(ctx, job)

	if err != nil {
		ws.logger.Error("build job failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
		job.MarkFailed(err.Error())
	} else {
		ws.logger.Info("build job completed successfully",
			zap.String("job_id", job.ID),
			zap.Duration("duration", job.Duration),
		)
		job.MarkCompleted()
	}

	// Publish final status
	ws.publishJobStatus(job)
	ws.publishJobComplete(job)
}

// publishJobStatus publica o status do job no NATS
func (ws *WorkerService) publishJobStatus(job *shared.BuildJob) {
	data, err := json.Marshal(job)
	if err != nil {
		ws.logger.Error("failed to marshal job status",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
		return
	}

	if err := ws.natsClient.Publish("builds.status", data); err != nil {
		ws.logger.Error("failed to publish job status",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
	}
}

// publishJobComplete publica a conclusão do job no NATS
func (ws *WorkerService) publishJobComplete(job *shared.BuildJob) {
	data, err := json.Marshal(job)
	if err != nil {
		ws.logger.Error("failed to marshal job completion",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
		return
	}

	if err := ws.natsClient.Publish("builds.complete", data); err != nil {
		ws.logger.Error("failed to publish job completion",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
	}
}
