package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	natsclient "github.com/jorgerua/build-system/libs/nats-client"
	"github.com/jorgerua/build-system/libs/shared"
	"go.uber.org/zap"
)

// StatusHandler gerencia consultas de status de builds
type StatusHandler struct {
	natsClient natsclient.NATSClient
	logger     *zap.Logger
}

// NewStatusHandler cria uma nova instância do StatusHandler
func NewStatusHandler(natsClient natsclient.NATSClient, logger *zap.Logger) *StatusHandler {
	return &StatusHandler{
		natsClient: natsClient,
		logger:     logger,
	}
}

// StatusRequest representa uma requisição de status
type StatusRequest struct {
	JobID string `json:"job_id"`
}

// StatusResponse representa a resposta de status
type StatusResponse struct {
	Job   *shared.BuildJob `json:"job,omitempty"`
	Error string           `json:"error,omitempty"`
}

// BuildsListResponse representa a resposta de listagem de builds
type BuildsListResponse struct {
	Builds []shared.BuildJob `json:"builds"`
	Total  int               `json:"total"`
}

// GetBuildStatus retorna o status de um build específico
func (h *StatusHandler) GetBuildStatus(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		h.logger.Warn("missing job ID in request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job ID is required",
		})
		return
	}

	h.logger.Debug("querying build status",
		zap.String("job_id", jobID),
	)

	// Criar requisição de status
	request := StatusRequest{
		JobID: jobID,
	}
	requestData, err := json.Marshal(request)
	if err != nil {
		h.logger.Error("failed to marshal status request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	// Usar NATS request/reply para consultar status
	timeout := 5 * time.Second
	msg, err := h.natsClient.Request("builds.status.get", requestData, timeout)
	if err != nil {
		h.logger.Error("failed to query build status",
			zap.Error(err),
			zap.String("job_id", jobID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to query build status",
		})
		return
	}

	// Parse da resposta
	var response StatusResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		h.logger.Error("failed to parse status response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to parse response",
		})
		return
	}

	// Verificar se houve erro na resposta
	if response.Error != "" {
		h.logger.Warn("build not found",
			zap.String("job_id", jobID),
			zap.String("error", response.Error),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": response.Error,
		})
		return
	}

	// Verificar se o job foi encontrado
	if response.Job == nil {
		h.logger.Warn("build not found",
			zap.String("job_id", jobID),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("build with ID %s not found", jobID),
		})
		return
	}

	h.logger.Debug("build status retrieved successfully",
		zap.String("job_id", jobID),
		zap.String("status", string(response.Job.Status)),
	)

	// Retornar informações do build
	c.JSON(http.StatusOK, response.Job)
}

// ListBuilds retorna o histórico de builds
func (h *StatusHandler) ListBuilds(c *gin.Context) {
	// Obter parâmetros de query opcionais
	repository := c.Query("repository")
	status := c.Query("status")
	limit := c.DefaultQuery("limit", "50")

	h.logger.Debug("listing builds",
		zap.String("repository", repository),
		zap.String("status", status),
		zap.String("limit", limit),
	)

	// Criar requisição de listagem
	request := map[string]string{
		"repository": repository,
		"status":     status,
		"limit":      limit,
	}
	requestData, err := json.Marshal(request)
	if err != nil {
		h.logger.Error("failed to marshal list request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	// Usar NATS request/reply para listar builds
	timeout := 5 * time.Second
	msg, err := h.natsClient.Request("builds.status.list", requestData, timeout)
	if err != nil {
		h.logger.Error("failed to list builds", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list builds",
		})
		return
	}

	// Parse da resposta
	var response BuildsListResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		h.logger.Error("failed to parse list response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to parse response",
		})
		return
	}

	h.logger.Debug("builds listed successfully",
		zap.Int("total", response.Total),
	)

	// Retornar lista de builds
	c.JSON(http.StatusOK, response)
}
