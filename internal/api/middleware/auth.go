package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"zpigo/internal/logger"
	"zpigo/internal/meow"
)

type AuthContextKey string

const (
	AuthContextKeyValue AuthContextKey = "auth"
)

type AuthContext struct {
	APIKey    string
	SessionID string
	UserID    string
}

func AuthMiddleware(authManager *meow.AuthManager) gin.HandlerFunc {
	authLogger := logger.NewForComponent("AuthMiddleware")

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authLogger.Warn("API Key não fornecida", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     true,
				"message":   "API Key é obrigatória",
				"code":      http.StatusUnauthorized,
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		apiKey = strings.TrimSpace(apiKey)

		if apiKey == "" {
			authLogger.Warn("API Key vazia", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     true,
				"message":   "API Key inválida",
				"code":      http.StatusUnauthorized,
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		authCtxResult, err := authManager.ValidateAPIKey(c.Request.Context(), apiKey, "")
		if err != nil || authCtxResult == nil {
			authLogger.Warn("API Key inválida", "apiKey", maskAPIKey(apiKey), "path", c.Request.URL.Path, "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     true,
				"message":   "API Key inválida",
				"code":      http.StatusUnauthorized,
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		authCtx := &AuthContext{
			APIKey: apiKey,
			UserID: getUserIDFromAPIKey(apiKey),
		}

		c.Set(string(AuthContextKeyValue), authCtx)

		authLogger.Debug("Autenticação bem-sucedida",
			"apiKey", maskAPIKey(apiKey),
			"path", c.Request.URL.Path,
			"method", c.Request.Method)

		c.Next()
	}
}

func OptionalAuthMiddleware(authManager *meow.AuthManager) gin.HandlerFunc {
	authLogger := logger.NewForComponent("OptionalAuthMiddleware")

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader != "" {
			apiKey := strings.TrimPrefix(authHeader, "Bearer ")
			apiKey = strings.TrimSpace(apiKey)

			authCtxResult, err := authManager.ValidateAPIKey(c.Request.Context(), apiKey, "")
			if apiKey != "" && err == nil && authCtxResult != nil {
				authCtx := &AuthContext{
					APIKey: apiKey,
					UserID: getUserIDFromAPIKey(apiKey),
				}

				c.Set(string(AuthContextKeyValue), authCtx)
				authLogger.Debug("Autenticação opcional bem-sucedida", "apiKey", maskAPIKey(apiKey))
			}
		}

		c.Next()
	}
}

func GetAuthContext(c *gin.Context) (*AuthContext, bool) {
	authCtx, exists := c.Get(string(AuthContextKeyValue))
	if !exists {
		return nil, false
	}
	auth, ok := authCtx.(*AuthContext)
	return auth, ok
}

func GetAuthContextFromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(AuthContextKeyValue).(*AuthContext)
	return authCtx, ok
}

func RequireAuth(c *gin.Context) (*AuthContext, error) {
	authCtx, ok := GetAuthContext(c)
	if !ok {
		return nil, &AuthError{Message: "Autenticação necessária"}
	}
	return authCtx, nil
}

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}


func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}

func getUserIDFromAPIKey(apiKey string) string {
	return "user_" + apiKey[:8]
}

func SessionAuthMiddleware() gin.HandlerFunc {
	authLogger := logger.NewForComponent("SessionAuthMiddleware")

	return func(c *gin.Context) {
		authCtx, ok := GetAuthContext(c)
		if !ok {
			authLogger.Warn("Contexto de autenticação não encontrado")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     true,
				"message":   "Autenticação necessária",
				"code":      http.StatusUnauthorized,
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		authLogger.Debug("Validação de sessão bem-sucedida",
			"apiKey", maskAPIKey(authCtx.APIKey),
			"path", c.Request.URL.Path)

		c.Next()
	}
}
