package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware cria um middleware de logging estruturado com Zap
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Registrar timestamp de início
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method

		// Processar requisição
		c.Next()

		// Calcular duração
		duration := time.Since(start)

		// Obter informações da resposta
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Construir campos de log
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		}

		// Adicionar erro se houver
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		// Log baseado no status code
		switch {
		case statusCode >= 500:
			logger.Error("HTTP request completed with server error", fields...)
		case statusCode >= 400:
			logger.Warn("HTTP request completed with client error", fields...)
		case statusCode >= 300:
			logger.Info("HTTP request completed with redirect", fields...)
		default:
			logger.Info("HTTP request completed successfully", fields...)
		}
	}
}
