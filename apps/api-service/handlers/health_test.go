package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHealthHandler_Healthy(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	
	mockNATS := new(MockNATSClient)
	mockNATS.On("IsConnected").Return(true)
	
	handler := NewHealthHandler(mockNATS, logger)
	
	// Criar requisição HTTP
	router := gin.New()
	router.GET("/health", handler.Handle)
	
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "connected", response.Checks["nats"])
	assert.NotEmpty(t, response.Uptime)
	
	mockNATS.AssertExpectations(t)
}

func TestHealthHandler_Unhealthy_NATSDisconnected(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	
	mockNATS := new(MockNATSClient)
	mockNATS.On("IsConnected").Return(false)
	
	handler := NewHealthHandler(mockNATS, logger)
	
	// Criar requisição HTTP
	router := gin.New()
	router.GET("/health", handler.Handle)
	
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", response.Status)
	assert.Equal(t, "disconnected", response.Checks["nats"])
	
	mockNATS.AssertExpectations(t)
}

func TestHealthHandler_Readiness_Ready(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	
	mockNATS := new(MockNATSClient)
	mockNATS.On("IsConnected").Return(true)
	
	handler := NewHealthHandler(mockNATS, logger)
	
	// Criar requisição HTTP
	router := gin.New()
	router.GET("/readiness", handler.Readiness)
	
	req, _ := http.NewRequest("GET", "/readiness", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
	
	mockNATS.AssertExpectations(t)
}

func TestHealthHandler_Readiness_NotReady(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	
	mockNATS := new(MockNATSClient)
	mockNATS.On("IsConnected").Return(false)
	
	handler := NewHealthHandler(mockNATS, logger)
	
	// Criar requisição HTTP
	router := gin.New()
	router.GET("/readiness", handler.Readiness)
	
	req, _ := http.NewRequest("GET", "/readiness", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "not_ready", response["status"])
	assert.Equal(t, "nats_not_connected", response["reason"])
	
	mockNATS.AssertExpectations(t)
}

func TestHealthHandler_Liveness(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	
	mockNATS := new(MockNATSClient)
	
	handler := NewHealthHandler(mockNATS, logger)
	
	// Criar requisição HTTP
	router := gin.New()
	router.GET("/liveness", handler.Liveness)
	
	req, _ := http.NewRequest("GET", "/liveness", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "alive", response["status"])
}
