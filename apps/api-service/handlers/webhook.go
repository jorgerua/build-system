package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	natsclient "github.com/oci-build-system/libs/nats-client"
	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// WebhookHandler gerencia requisições de webhook do GitHub
type WebhookHandler struct {
	natsClient natsclient.NATSClient
	logger     *zap.Logger
	secret     string
}

// NewWebhookHandler cria uma nova instância do WebhookHandler
func NewWebhookHandler(natsClient natsclient.NATSClient, logger *zap.Logger, secret string) *WebhookHandler {
	return &WebhookHandler{
		natsClient: natsClient,
		logger:     logger,
		secret:     secret,
	}
}

// WebhookPayload representa o payload recebido do GitHub
type WebhookPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"head_commit"`
}

// WebhookResponse representa a resposta enviada ao GitHub
type WebhookResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Handle processa requisições de webhook do GitHub
func (h *WebhookHandler) Handle(c *gin.Context) {
	// Ler o corpo da requisição
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to read request body",
		})
		return
	}

	// Validar assinatura HMAC-SHA256
	signature := c.GetHeader("X-Hub-Signature-256")
	if !h.validateSignature(body, signature) {
		h.logger.Warn("invalid webhook signature",
			zap.String("signature", signature),
			zap.String("remote_addr", c.ClientIP()),
		)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid signature",
		})
		return
	}

	// Parse do payload JSON
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	// Extrair informações do repositório e commit
	buildJob, err := h.extractBuildJob(&payload)
	if err != nil {
		h.logger.Error("failed to extract build job from payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid payload: %s", err.Error()),
		})
		return
	}

	// Serializar BuildJob para JSON
	jobData, err := json.Marshal(buildJob)
	if err != nil {
		h.logger.Error("failed to marshal build job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create build job",
		})
		return
	}

	// Publicar no NATS subject builds.webhook
	if err := h.natsClient.Publish("builds.webhook", jobData); err != nil {
		h.logger.Error("failed to publish build job to NATS",
			zap.Error(err),
			zap.String("job_id", buildJob.ID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to enqueue build job",
		})
		return
	}

	h.logger.Info("webhook processed successfully",
		zap.String("job_id", buildJob.ID),
		zap.String("repository", buildJob.Repository.FullName()),
		zap.String("commit", buildJob.CommitHash),
		zap.String("branch", buildJob.Branch),
	)

	// Retornar HTTP 202 Accepted com job ID
	c.JSON(http.StatusAccepted, WebhookResponse{
		ID:      buildJob.ID,
		Status:  "accepted",
		Message: "build job enqueued successfully",
	})
}

// validateSignature valida a assinatura HMAC-SHA256 do webhook
func (h *WebhookHandler) validateSignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Remover prefixo "sha256=" da assinatura
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calcular HMAC-SHA256 do payload
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)

	// Comparar assinaturas usando hmac.Equal para evitar timing attacks
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	return hmac.Equal(signatureBytes, expectedMAC) || signature == expectedSignature
}

// extractBuildJob extrai informações do payload e cria um BuildJob
func (h *WebhookHandler) extractBuildJob(payload *WebhookPayload) (*shared.BuildJob, error) {
	// Validar campos obrigatórios
	if payload.Repository.CloneURL == "" {
		return nil, fmt.Errorf("repository clone_url is required")
	}
	if payload.Repository.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	if payload.Repository.Owner.Login == "" {
		return nil, fmt.Errorf("repository owner is required")
	}
	if payload.After == "" && payload.HeadCommit.ID == "" {
		return nil, fmt.Errorf("commit hash is required")
	}

	// Extrair branch do ref (refs/heads/main -> main)
	branch := payload.Ref
	if strings.HasPrefix(branch, "refs/heads/") {
		branch = strings.TrimPrefix(branch, "refs/heads/")
	}

	// Usar After ou HeadCommit.ID como commit hash
	commitHash := payload.After
	if commitHash == "" {
		commitHash = payload.HeadCommit.ID
	}

	// Criar BuildJob
	buildJob := &shared.BuildJob{
		ID: uuid.New().String(),
		Repository: shared.RepositoryInfo{
			URL:   payload.Repository.CloneURL,
			Name:  payload.Repository.Name,
			Owner: payload.Repository.Owner.Login,
			Branch: branch,
		},
		CommitHash:   commitHash,
		CommitAuthor: payload.HeadCommit.Author.Name,
		CommitMsg:    payload.HeadCommit.Message,
		Branch:       branch,
		Status:       shared.JobStatusPending,
		CreatedAt:    time.Now(),
		Phases:       []shared.PhaseMetric{},
	}

	// Validar BuildJob
	if !buildJob.IsValid() {
		return nil, fmt.Errorf("invalid build job created")
	}

	return buildJob, nil
}
