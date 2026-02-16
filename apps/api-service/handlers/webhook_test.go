package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/oci-build-system/libs/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockNATSClient é um mock do cliente NATS
type MockNATSClient struct {
	mock.Mock
}

func (m *MockNATSClient) Connect(url string) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockNATSClient) Publish(subject string, data []byte) error {
	args := m.Called(subject, data)
	return args.Error(0)
}

func (m *MockNATSClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	args := m.Called(subject, handler)
	return args.Get(0).(*nats.Subscription), args.Error(1)
}

func (m *MockNATSClient) Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	args := m.Called(subject, data, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nats.Msg), args.Error(1)
}

func (m *MockNATSClient) Close() {
	m.Called()
}

func (m *MockNATSClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestWebhookHandler_ValidPayload(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	secret := "test-secret"
	
	mockNATS := new(MockNATSClient)
	handler := NewWebhookHandler(mockNATS, logger, secret)
	
	// Criar payload válido
	payload := WebhookPayload{
		Ref:   "refs/heads/main",
		After: "abc123def456",
		Repository: struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			CloneURL string `json:"clone_url"`
			Owner    struct {
				Login string `json:"login"`
			} `json:"owner"`
		}{
			Name:     "test-repo",
			FullName: "testuser/test-repo",
			CloneURL: "https://github.com/testuser/test-repo.git",
			Owner: struct {
				Login string `json:"login"`
			}{
				Login: "testuser",
			},
		},
		HeadCommit: struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		}{
			ID:      "abc123def456",
			Message: "Test commit",
			Author: struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				Name:  "Test User",
				Email: "test@example.com",
			},
		},
	}
	
	payloadBytes, _ := json.Marshal(payload)
	
	// Calcular assinatura HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	
	// Configurar mock para esperar publicação no NATS
	mockNATS.On("Publish", "builds.webhook", mock.MatchedBy(func(data []byte) bool {
		var job shared.BuildJob
		err := json.Unmarshal(data, &job)
		if err != nil {
			return false
		}
		return job.Repository.Name == "test-repo" &&
			job.CommitHash == "abc123def456" &&
			job.Branch == "main"
	})).Return(nil)
	
	// Criar requisição HTTP
	router := gin.New()
	router.POST("/webhook", handler.Handle)
	
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", signature)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusAccepted, w.Code)
	
	var response WebhookResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "accepted", response.Status)
	assert.NotEmpty(t, response.JobID)
	
	mockNATS.AssertExpectations(t)
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	secret := "test-secret"
	
	mockNATS := new(MockNATSClient)
	handler := NewWebhookHandler(mockNATS, logger, secret)
	
	// Criar payload
	payload := map[string]string{"test": "data"}
	payloadBytes, _ := json.Marshal(payload)
	
	// Usar assinatura inválida
	invalidSignature := "sha256=invalid"
	
	// Criar requisição HTTP
	router := gin.New()
	router.POST("/webhook", handler.Handle)
	
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", invalidSignature)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid signature", response["error"])
	
	// NATS não deve ser chamado
	mockNATS.AssertNotCalled(t, "Publish")
}

func TestWebhookHandler_MissingSignature(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	secret := "test-secret"
	
	mockNATS := new(MockNATSClient)
	handler := NewWebhookHandler(mockNATS, logger, secret)
	
	// Criar payload
	payload := map[string]string{"test": "data"}
	payloadBytes, _ := json.Marshal(payload)
	
	// Criar requisição HTTP sem assinatura
	router := gin.New()
	router.POST("/webhook", handler.Handle)
	
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// NATS não deve ser chamado
	mockNATS.AssertNotCalled(t, "Publish")
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	secret := "test-secret"
	
	mockNATS := new(MockNATSClient)
	handler := NewWebhookHandler(mockNATS, logger, secret)
	
	// Criar payload inválido
	payloadBytes := []byte("invalid json")
	
	// Calcular assinatura
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	
	// Criar requisição HTTP
	router := gin.New()
	router.POST("/webhook", handler.Handle)
	
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", signature)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid JSON payload", response["error"])
	
	// NATS não deve ser chamado
	mockNATS.AssertNotCalled(t, "Publish")
}

func TestExtractBuildJob_ValidPayload(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	secret := "test-secret"
	mockNATS := new(MockNATSClient)
	handler := NewWebhookHandler(mockNATS, logger, secret)
	
	// Criar payload válido
	payload := &WebhookPayload{
		Ref:   "refs/heads/develop",
		After: "xyz789",
		Repository: struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			CloneURL string `json:"clone_url"`
			Owner    struct {
				Login string `json:"login"`
			} `json:"owner"`
		}{
			Name:     "my-repo",
			FullName: "myorg/my-repo",
			CloneURL: "https://github.com/myorg/my-repo.git",
			Owner: struct {
				Login string `json:"login"`
			}{
				Login: "myorg",
			},
		},
		HeadCommit: struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		}{
			ID:      "xyz789",
			Message: "Fix bug",
			Author: struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				Name:  "Developer",
				Email: "dev@example.com",
			},
		},
	}
	
	// Extrair BuildJob
	job, err := handler.extractBuildJob(payload)
	
	// Verificar resultado
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.NotEmpty(t, job.ID)
	assert.Equal(t, "my-repo", job.Repository.Name)
	assert.Equal(t, "myorg", job.Repository.Owner)
	assert.Equal(t, "https://github.com/myorg/my-repo.git", job.Repository.URL)
	assert.Equal(t, "develop", job.Branch)
	assert.Equal(t, "xyz789", job.CommitHash)
	assert.Equal(t, "Developer", job.CommitAuthor)
	assert.Equal(t, "Fix bug", job.CommitMsg)
	assert.Equal(t, shared.JobStatusPending, job.Status)
}

func TestExtractBuildJob_MissingRequiredFields(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	secret := "test-secret"
	mockNATS := new(MockNATSClient)
	handler := NewWebhookHandler(mockNATS, logger, secret)
	
	// Testar payload sem clone URL
	payload := &WebhookPayload{
		Ref:   "refs/heads/main",
		After: "abc123",
	}
	
	job, err := handler.extractBuildJob(payload)
	assert.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "clone_url")
}
