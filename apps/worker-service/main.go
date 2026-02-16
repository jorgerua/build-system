package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	cacheservice "github.com/oci-build-system/libs/cache-service"
	gitservice "github.com/oci-build-system/libs/git-service"
	imageservice "github.com/oci-build-system/libs/image-service"
	natsclient "github.com/oci-build-system/libs/nats-client"
	nxservice "github.com/oci-build-system/libs/nx-service"
	"github.com/oci-build-system/libs/shared"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		// Provide logger
		fx.Provide(NewLogger),
		
		// Provide configuration
		fx.Provide(NewConfig),
		
		// Provide NATS client
		fx.Provide(NewNATSClient),
		
		// Provide services
		fx.Provide(NewGitService),
		fx.Provide(NewNXService),
		fx.Provide(NewImageService),
		fx.Provide(NewCacheService),
		
		// Provide worker service
		fx.Provide(NewWorkerService),
		
		// Invoke worker service to start it
		fx.Invoke(func(ws *WorkerService) {}),
	)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
		app.Stop(context.Background())
	}()

	// Start the application
	if err := app.Start(ctx); err != nil {
		panic(err)
	}

	// Wait for shutdown signal
	<-app.Done()
}

// NewLogger creates a new Zap logger
func NewLogger(config *shared.Config) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if config.Logging.Level == "debug" {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewConfig loads configuration from file
func NewConfig() (*shared.Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	config, err := shared.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// NewNATSClient creates a new NATS client
func NewNATSClient(config *shared.Config, logger *zap.Logger, lc fx.Lifecycle) (*natsclient.Client, error) {
	natsConfig := &natsclient.Config{
		URL:            config.NATS.URL,
		MaxReconnects:  10,
		ReconnectWait:  config.NATS.ReconnectWait,
		ConnectTimeout: config.NATS.ConnectTimeout,
	}

	client, err := natsclient.NewClientWithConfig(logger, natsConfig)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			client.Close()
			return nil
		},
	})

	return client, nil
}

// NewGitService creates a new Git service
func NewGitService(config *shared.Config, logger *zap.Logger) gitservice.GitService {
	gitConfig := gitservice.Config{
		CodeCachePath: config.Build.CodeCachePath,
		MaxRetries:    config.Worker.MaxRetries,
		RetryDelay:    config.Worker.RetryDelay,
	}

	return gitservice.NewGitService(gitConfig, logger)
}

// NewNXService creates a new NX service
func NewNXService(logger *zap.Logger) nxservice.NXService {
	return nxservice.NewNXService(logger)
}

// NewImageService creates a new Image service
func NewImageService(logger *zap.Logger) imageservice.ImageService {
	return imageservice.NewImageService(logger)
}

// NewCacheService creates a new Cache service
func NewCacheService(config *shared.Config, logger *zap.Logger) cacheservice.CacheService {
	return cacheservice.NewCacheService(config.Build.BuildCachePath, logger)
}

// NewWorkerService creates a new Worker service
func NewWorkerService(
	config *shared.Config,
	logger *zap.Logger,
	natsClient *natsclient.Client,
	gitService gitservice.GitService,
	nxService nxservice.NXService,
	imageService imageservice.ImageService,
	cacheService cacheservice.CacheService,
	lc fx.Lifecycle,
) *WorkerService {
	ws := &WorkerService{
		config:       config,
		logger:       logger,
		natsClient:   natsClient,
		gitService:   gitService,
		nxService:    nxService,
		imageService: imageService,
		cacheService: cacheService,
		jobQueue:     make(chan *shared.BuildJob, config.Worker.QueueSize),
		stopChan:     make(chan struct{}),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return ws.Start()
		},
		OnStop: func(ctx context.Context) error {
			return ws.Shutdown(ctx)
		},
	})

	return ws
}
