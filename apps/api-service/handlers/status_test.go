package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jorgerua/build-system/libs/shared"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestStatusHandler_GetBuildStatus_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	mockNATS := new(MockNATSClient)
	handler := NewStatusHandler(mockNATS, logger)

	// Criar job de teste
	now := time.Now()
	testJob := &shared.BuildJob{
		ID: "test-job-123",
		Repository: shared.RepositoryInfo{
			URL:    "https://github.com/test/repo.git",
			Name:   "repo",
			Owner:  "test",
			Branch: "main",
		},
		CommitHash:   "abc123",
		CommitAuthor: "Test User",
		CommitMsg:    "Test commit",
		Branch:       "main",
		Status:       shared.JobStatusSuccess,
		CreatedAt:    now,
		StartedAt:    &now,
		CompletedAt:  &now,
		Duration:     5 * time.Minute,
	}

	// Criar resposta mock
	response := StatusResponse{
		Job: testJob,
	}
	responseData, _ := json.Marshal(response)

	// Configurar mock
	mockNATS.On("Request", "builds.status.get", mock.Anything, 5*time.Second).Return(
		&nats.Msg{Data: responseData},
		nil,
	)

	// Criar requisição HTTP
	router := gin.New()
	router.GET("/builds/:id", handler.GetBuildStatus)

	req, _ := http.NewRequest("GET", "/builds/test-job-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verificar resposta
	assert.Equal(t, http.StatusOK, w.Code)

	var job shared.BuildJob
	err := json.Unmarshal(w.Body.Bytes(), &job)
	assert.NoError(t, err)
	assert.Equal(t, "test-job-123", job.ID)
	assert.Equal(t, shared.JobStatusSuccess, job.Status)

	mockNATS.AssertExpectations(t)
}

func TestStatusHandler_GetBuildStatus_NotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	mockNATS := new(MockNATSClient)
	handler := NewStatusHandler(mockNATS, logger)

	// Criar resposta mock com erro
	response := StatusResponse{
		Error: "build not found",
	}
	responseData, _ := json.Marshal(response)

	// Configurar mock
	mockNATS.On("Request", "builds.status.get", mock.Anything, 5*time.Second).Return(
		&nats.Msg{Data: responseData},
		nil,
	)

	// Criar requisição HTTP
	router := gin.New()
	router.GET("/builds/:id", handler.GetBuildStatus)

	req, _ := http.NewRequest("GET", "/builds/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verificar resposta
	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "build not found", errorResponse["error"])

	mockNATS.AssertExpectations(t)
}

func TestStatusHandler_GetBuildStatus_MissingID(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	mockNATS := new(MockNATSClient)
	handler := NewStatusHandler(mockNATS, logger)

	// Criar requisição HTTP sem ID
	router := gin.New()
	router.GET("/builds/:id", handler.GetBuildStatus)

	req, _ := http.NewRequest("GET", "/builds/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verificar resposta
	assert.Equal(t, http.StatusNotFound, w.Code) // Gin retorna 404 para rota não encontrada

	// NATS não deve ser chamado
	mockNATS.AssertNotCalled(t, "Request")
}

func TestStatusHandler_ListBuilds_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	mockNATS := new(MockNATSClient)
	handler := NewStatusHandler(mockNATS, logger)

	// Criar lista de jobs de teste
	now := time.Now()
	testJobs := []shared.BuildJob{
		{
			ID:         "job-1",
			Repository: shared.RepositoryInfo{Name: "repo1", Owner: "test"},
			CommitHash: "abc123",
			Status:     shared.JobStatusSuccess,
			CreatedAt:  now,
		},
		{
			ID:         "job-2",
			Repository: shared.RepositoryInfo{Name: "repo2", Owner: "test"},
			CommitHash: "def456",
			Status:     shared.JobStatusRunning,
			CreatedAt:  now,
		},
	}

	// Criar resposta mock
	response := BuildsListResponse{
		Builds: testJobs,
		Total:  2,
	}
	responseData, _ := json.Marshal(response)

	// Configurar mock
	mockNATS.On("Request", "builds.status.list", mock.Anything, 5*time.Second).Return(
		&nats.Msg{Data: responseData},
		nil,
	)

	// Criar requisição HTTP
	router := gin.New()
	router.GET("/builds", handler.ListBuilds)

	req, _ := http.NewRequest("GET", "/builds", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verificar resposta
	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse BuildsListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.NoError(t, err)
	assert.Equal(t, 2, listResponse.Total)
	assert.Len(t, listResponse.Builds, 2)

	mockNATS.AssertExpectations(t)
}

func TestStatusHandler_ListBuilds_WithFilters(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	mockNATS := new(MockNATSClient)
	handler := NewStatusHandler(mockNATS, logger)

	// Criar resposta mock vazia
	response := BuildsListResponse{
		Builds: []shared.BuildJob{},
		Total:  0,
	}
	responseData, _ := json.Marshal(response)

	// Configurar mock
	mockNATS.On("Request", "builds.status.list", mock.Anything, 5*time.Second).Return(
		&nats.Msg{Data: responseData},
		nil,
	)

	// Criar requisição HTTP com filtros
	router := gin.New()
	router.GET("/builds", handler.ListBuilds)

	req, _ := http.NewRequest("GET", "/builds?repository=test/repo&status=success&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verificar resposta
	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse BuildsListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.NoError(t, err)
	assert.Equal(t, 0, listResponse.Total)

	mockNATS.AssertExpectations(t)
}
