package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Feature: oci-build-system, Property 23: Autenticação em endpoints de consulta
// Para qualquer requisição aos endpoints de consulta sem token de autenticação válido,
// o sistema deve retornar HTTP 401.
// Valida: Requisitos 8.4
func TestProperty_AuthenticationOnQueryEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	properties := gopter.NewProperties(nil)
	
	properties.Property("invalid token returns 401", prop.ForAll(
		func(validToken string, invalidToken string) bool {
			// Garantir que os tokens sejam diferentes
			if validToken == invalidToken || validToken == "" {
				return true // Skip este caso
			}
			
			// Setup
			logger := zap.NewNop()
			router := gin.New()
			router.Use(AuthMiddleware(logger, validToken))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			// Criar requisição com token inválido
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+invalidToken)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Verificar que retorna 401
			if w.Code != http.StatusUnauthorized {
				t.Logf("Expected 401, got %d", w.Code)
				return false
			}
			
			return true
		},
		gen.AnyString(),  // validToken
		gen.AnyString(),  // invalidToken
	))
	
	properties.Property("valid token allows access", prop.ForAll(
		func(token string) bool {
			// Skip tokens vazios
			if token == "" {
				return true
			}
			
			// Setup
			logger := zap.NewNop()
			router := gin.New()
			router.Use(AuthMiddleware(logger, token))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			// Criar requisição com token válido
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Verificar que NÃO retorna 401
			if w.Code == http.StatusUnauthorized {
				t.Logf("Valid token returned 401")
				return false
			}
			
			// Deve retornar 200
			if w.Code != http.StatusOK {
				t.Logf("Expected 200, got %d", w.Code)
				return false
			}
			
			return true
		},
		gen.AnyString(),  // token
	))
	
	properties.Property("missing authorization header returns 401", prop.ForAll(
		func(token string) bool {
			// Setup
			logger := zap.NewNop()
			router := gin.New()
			router.Use(AuthMiddleware(logger, token))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			// Criar requisição sem header de autorização
			req, _ := http.NewRequest("GET", "/test", nil)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Verificar que retorna 401
			if w.Code != http.StatusUnauthorized {
				t.Logf("Missing auth header returned %d instead of 401", w.Code)
				return false
			}
			
			return true
		},
		gen.AnyString(),  // token
	))
	
	properties.Property("malformed authorization header returns 401", prop.ForAll(
		func(token string, malformedHeader string) bool {
			// Skip headers que começam com "Bearer " (formato correto)
			if len(malformedHeader) >= 7 && malformedHeader[:7] == "Bearer " {
				return true
			}
			
			// Setup
			logger := zap.NewNop()
			router := gin.New()
			router.Use(AuthMiddleware(logger, token))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			// Criar requisição com header malformado
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", malformedHeader)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Verificar que retorna 401
			if w.Code != http.StatusUnauthorized {
				t.Logf("Malformed auth header returned %d instead of 401", w.Code)
				return false
			}
			
			return true
		},
		gen.AnyString(),  // token
		gen.AnyString(),  // malformedHeader
	))
	
	properties.Property("empty token in header returns 401", prop.ForAll(
		func(token string) bool {
			// Skip tokens vazios (não há como diferenciar)
			if token == "" {
				return true
			}
			
			// Setup
			logger := zap.NewNop()
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
			
			// Verificar que retorna 401
			if w.Code != http.StatusUnauthorized {
				t.Logf("Empty token returned %d instead of 401", w.Code)
				return false
			}
			
			return true
		},
		gen.AnyString(),  // token
	))
	
	properties.Property("case sensitive token validation", prop.ForAll(
		func(token string) bool {
			// Skip tokens vazios ou que não têm letras
			if token == "" || token == token[:0] {
				return true
			}
			
			// Criar versão com case diferente
			var differentCase string
			for _, c := range token {
				if c >= 'a' && c <= 'z' {
					differentCase += string(c - 32) // Uppercase
				} else if c >= 'A' && c <= 'Z' {
					differentCase += string(c + 32) // Lowercase
				} else {
					differentCase += string(c)
				}
			}
			
			// Se ficou igual, skip
			if differentCase == token {
				return true
			}
			
			// Setup
			logger := zap.NewNop()
			router := gin.New()
			router.Use(AuthMiddleware(logger, token))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			// Criar requisição com token em case diferente
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+differentCase)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Verificar que retorna 401 (validação é case-sensitive)
			if w.Code != http.StatusUnauthorized {
				t.Logf("Different case token returned %d instead of 401", w.Code)
				return false
			}
			
			return true
		},
		gen.AlphaString(),  // token com letras
	))
	
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
