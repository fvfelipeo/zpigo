package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"zpigo/internal/logger"
)

type Middleware struct {
	logger logger.Logger
}

func New() *Middleware {
	return &Middleware{
		logger: logger.NewForComponent("Middleware"),
	}
}

func (m *Middleware) Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		m.logger.Info("Request processado",
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency", param.Latency,
			"client_ip", param.ClientIP,
			"user_agent", param.Request.UserAgent(),
		)
		return ""
	})
}

func (m *Middleware) Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		m.logger.Error("Panic recuperado",
			"error", recovered,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"client_ip", c.ClientIP(),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     true,
			"message":   "Erro interno do servidor",
			"code":      http.StatusInternalServerError,
			"timestamp": time.Now().Unix(),
		})
	})
}

func (m *Middleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Permitir origens específicas ou localhost para desenvolvimento
		if origin == "" || strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Request-ID, Origin, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			m.logger.Debug("CORS preflight request", "path", c.Request.URL.Path, "origin", origin)
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

func (m *Middleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := generateRequestID()
		c.Header("X-Request-ID", requestID)
		c.Set("requestID", requestID)

		m.logger.Debug("Request ID gerado", "requestID", requestID, "path", c.Request.URL.Path)

		c.Next()
	}
}

func (m *Middleware) Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// CSP mais permissiva para Swagger UI
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			c.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data: https:; connect-src 'self' http: https:")
		} else {
			c.Header("Content-Security-Policy", "default-src 'self'; connect-src 'self' http: https:")
		}

		m.logger.Debug("Headers de segurança aplicados", "path", c.Request.URL.Path)

		c.Next()
	}
}

func (m *Middleware) Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan bool, 1)
		go func() {
			c.Next()
			done <- true
		}()

		select {
		case <-done:
		case <-ctx.Done():
			m.logger.Warn("Request timeout",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"timeout", timeout,
			)

			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":     true,
				"message":   "Request timeout",
				"code":      http.StatusRequestTimeout,
				"timeout":   timeout.String(),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
		}
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
