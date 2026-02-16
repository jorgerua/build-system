package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oci-build-system/apps/api-service/handlers"
	"github.com/oci-build-system/apps/api-service/middleware"
	natsclient "github.com/oci-build-system/libs/nats-client"
	"github.com/oci-build-system/libs/shared"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		fx.Provide(
			ProvideLogger,
			ProvideConfig,
			ProvideNATSClient,
			ProvideGinRouter,
			ProvideHTTPServer,
		),
		fx.Invoke(RegisterHandlers),
		fx.Invoke(RegisterLifecycleHooks),
	)

	app.Run()
}

// ProvideLogger cria e configura o logger Zap
func ProvideLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	
	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	
	return logger, nil
}

// ProvideConfig carrega a configuração do arquivo YAML
func ProvideConfig(logger *zap.Logger) (*shared.Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	
	logger.Info("loading configuration", zap.String("path", configPath))
	
	config, err := shared.LoadConfig(configPath)
	if err != nil {
		logger.Error("failed to load configuration", zap.Error(err))
		return nil, err
	}
	
	// Atualizar nível de log baseado na configuração
	if config.Logging.Level != "" {
		level, err := zap.ParseAtomicLevel(config.Logging.Level)
		if err == nil {
			logger = logger.WithOptions(zap.IncreaseLevel(level))
		}
	}
	
	logger.Info("configuration loaded successfully",
		zap.Int("server_port", config.Server.Port),
		zap.String("nats_url", config.NATS.URL),
		zap.String("log_level", config.Logging.Level),
	)
	
	return config, nil
}

// ProvideNATSClient cria e conecta o cliente NATS
func ProvideNATSClient(logger *zap.Logger, config *shared.Config) (natsclient.NATSClient, error) {
	logger.Info("connecting to NATS", zap.String("url", config.NATS.URL))
	
	client := natsclient.NewClient(logger)
	if err := client.Connect(config.NATS.URL); err != nil {
		logger.Error("failed to connect to NATS", zap.Error(err))
		return nil, err
	}
	
	logger.Info("successfully connected to NATS")
	return client, nil
}

// ProvideGinRouter cria e configura o router Gin
func ProvideGinRouter(logger *zap.Logger, config *shared.Config) *gin.Engine {
	// Configurar modo do Gin baseado no nível de log
	if config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.New()
	
	// Usar recovery middleware padrão do Gin
	router.Use(gin.Recovery())
	
	// Usar logging middleware customizado
	router.Use(middleware.LoggingMiddleware(logger))
	
	logger.Info("gin router configured")
	return router
}

// ProvideHTTPServer cria o servidor HTTP
func ProvideHTTPServer(config *shared.Config, router *gin.Engine) *http.Server {
	addr := fmt.Sprintf(":%d", config.Server.Port)
	
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Server.WriteTimeout) * time.Second,
	}
	
	return server
}

// RegisterHandlers registra os handlers HTTP
func RegisterHandlers(
	router *gin.Engine,
	logger *zap.Logger,
	natsClient natsclient.NATSClient,
	config *shared.Config,
) {
	logger.Info("registering HTTP handlers")
	
	// Criar e registrar health check handler (sem autenticação)
	healthHandler := handlers.NewHealthHandler(natsClient, logger)
	router.GET("/health", healthHandler.Handle)
	
	// Criar e registrar webhook handler (sem autenticação)
	webhookHandler := handlers.NewWebhookHandler(natsClient, logger, config.GitHub.WebhookSecret)
	router.POST("/webhook", webhookHandler.Handle)
	
	// Criar middleware de autenticação
	authMiddleware := middleware.AuthMiddleware(logger, config.Auth.Token)
	
	// Criar e registrar status handler (com autenticação)
	statusHandler := handlers.NewStatusHandler(natsClient, logger)
	router.GET("/builds/:id", authMiddleware, statusHandler.GetBuildStatus)
	router.GET("/builds", authMiddleware, statusHandler.ListBuilds)
	
	logger.Info("HTTP handlers registered successfully")
}

// RegisterLifecycleHooks registra os hooks de lifecycle do FX
func RegisterLifecycleHooks(
	lc fx.Lifecycle,
	server *http.Server,
	natsClient natsclient.NATSClient,
	config *shared.Config,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting API service",
				zap.String("address", server.Addr),
			)
			
			// Iniciar servidor HTTP em goroutine
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("failed to start HTTP server", zap.Error(err))
				}
			}()
			
			logger.Info("API service started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping API service")
			
			// Criar context com timeout para shutdown
			shutdownTimeout := time.Duration(config.Server.ShutdownTimeout) * time.Second
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			
			// Shutdown graceful do servidor HTTP
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error("error during HTTP server shutdown", zap.Error(err))
			}
			
			// Fechar conexão NATS
			natsClient.Close()
			
			logger.Info("API service stopped successfully")
			return nil
		},
	})
}
