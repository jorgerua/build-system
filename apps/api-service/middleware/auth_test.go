package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	token := "valid-token-123"
	
	router := gin.New()
	router.Use(AuthMiddleware(logger, token))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Criar requisição com token válido
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token-123")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	token := "valid-token-123"
	
	router := gin.New()
	router.Use(AuthMiddleware(logger, token))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Criar requisição com token inválido
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid authentication token")
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	token := "valid-token-123"
	
	router := gin.New()
	router.Use(AuthMiddleware(logger, token))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Criar requisição sem header de autorização
	req, _ := http.NewRequest("GET", "/test", nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "authorization header is required")
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	token := "valid-token-123"
	
	router := gin.New()
	router.Use(AuthMiddleware(logger, token))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Testar formato inválido (sem "Bearer")
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "valid-token-123")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid authorization header format")
}

func TestAuthMiddleware_EmptyToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	token := "valid-token-123"
	
	router := gin.New()
	router.Use(AuthMiddleware(logger, token))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Criar requisição com token vazio
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verificar resposta
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid authentication token")
}
