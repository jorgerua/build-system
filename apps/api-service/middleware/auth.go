package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware cria um middleware de autenticação baseado em token
func AuthMiddleware(logger *zap.Logger, token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obter token do header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("missing authorization header",
				zap.String("path", c.Request.URL.Path),
				zap.String("remote_addr", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			c.Abort()
			return
		}

		// Verificar formato "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Warn("invalid authorization header format",
				zap.String("path", c.Request.URL.Path),
				zap.String("remote_addr", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format, expected 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		// Validar token
		providedToken := parts[1]
		if providedToken != token {
			logger.Warn("invalid authentication token",
				zap.String("path", c.Request.URL.Path),
				zap.String("remote_addr", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authentication token",
			})
			c.Abort()
			return
		}

		// Token válido, continuar processamento
		logger.Debug("authentication successful",
			zap.String("path", c.Request.URL.Path),
		)
		c.Next()
	}
}
