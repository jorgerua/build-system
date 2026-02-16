package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	natsclient "github.com/oci-build-system/libs/nats-client"
	"go.uber.org/zap"
)

// HealthHandler gerencia requisições de health check
type HealthHandler struct {
	natsClient natsclient.NATSClient
	logger     *zap.Logger
	startTime  time.Time
}

// NewHealthHandler cria uma nova instância do HealthHandler
func NewHealthHandler(natsClient natsclient.NATSClient, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		natsClient: natsClient,
		logger:     logger,
		startTime:  time.Now(),
	}
}

// HealthResponse representa a resposta do health check
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
}

// Handle processa requisições de health check
func (h *HealthHandler) Handle(c *gin.Context) {
	checks := make(map[string]string)
	overallStatus := "healthy"

	// Verificar conectividade com NATS
	if h.natsClient.IsConnected() {
		checks["nats"] = "connected"
	} else {
		checks["nats"] = "disconnected"
		overallStatus = "unhealthy"
		h.logger.Warn("NATS connection is down")
	}

	// Calcular uptime
	uptime := time.Since(h.startTime)

	// Construir resposta
	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
		Checks:    checks,
	}

	// Determinar status code HTTP
	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	h.logger.Debug("health check completed",
		zap.String("status", overallStatus),
		zap.String("uptime", uptime.String()),
	)

	c.JSON(statusCode, response)
}
