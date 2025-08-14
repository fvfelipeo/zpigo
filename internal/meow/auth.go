package meow

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/uptrace/bun"

	"zpigo/internal/db/models"
	"zpigo/internal/logger"
	"zpigo/internal/repository"
)

type AuthManager struct {
	db           *bun.DB
	sessionRepo  repository.SessionRepositoryInterface
	cacheManager *CacheManager
	logger       logger.Logger
}

func NewAuthManager(db *bun.DB, sessionRepo repository.SessionRepositoryInterface) *AuthManager {
	return &AuthManager{
		db:           db,
		sessionRepo:  sessionRepo,
		cacheManager: GetGlobalCache(),
		logger:       NewLoggerForComponent("AuthManager"),
	}
}

type AuthContext struct {
	APIKey    string
	SessionID string
	Session   *models.Session
}

func (am *AuthManager) ValidateAPIKey(ctx context.Context, apiKey, sessionID string) (*AuthContext, error) {
	am.logger.Debug("Validando API Key", "sessionID", sessionID)

	if apiKey == "" {
		am.logger.Warn("API Key não fornecida")
		return nil, errors.New("API key is required")
	}

	if sessionID == "" {
		am.logger.Warn("Session ID não fornecido")
		return nil, errors.New("session ID is required")
	}

	cacheKey := BuildCacheKey(apiKey, sessionID)
	if sessionInfo, found := am.cacheManager.GetSessionInfo(cacheKey); found {
		am.logger.Debug("Sessão encontrada no cache", "sessionID", sessionID)

		return &AuthContext{
			APIKey:    apiKey,
			SessionID: sessionID,
			Session:   sessionInfo.ToModelSession(),
		}, nil
	}

	am.logger.Debug("Buscando sessão no banco de dados", "sessionID", sessionID)
	session, err := am.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		am.logger.Warn("Sessão não encontrada no banco", "sessionID", sessionID, "error", err)
		return nil, errors.New("session not found")
	}

	sessionInfo := NewSessionInfoFromModel(session, apiKey)
	am.cacheManager.SetSessionInfo(cacheKey, sessionInfo)
	am.logger.Info("Autenticação bem-sucedida", "sessionID", sessionID)

	return &AuthContext{
		APIKey:    apiKey,
		SessionID: sessionID,
		Session:   session,
	}, nil
}

func (am *AuthManager) ExtractAPIKeyFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	return r.URL.Query().Get("apikey")
}

func (am *AuthManager) ExtractSessionIDFromRequest(r *http.Request) string {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID != "" {
		return sessionID
	}

	sessionID = r.URL.Query().Get("sessionId")
	if sessionID != "" {
		return sessionID
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, part := range pathParts {
		if part == "sessions" && i+1 < len(pathParts) {
			return pathParts[i+1]
		}
	}

	return ""
}

func (am *AuthManager) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := am.ExtractAPIKeyFromRequest(r)
		sessionID := am.ExtractSessionIDFromRequest(r)

		am.logger.Debug("Tentativa de autenticação", "method", r.Method, "path", r.URL.Path, "sessionID", sessionID)

		authCtx, err := am.ValidateAPIKey(r.Context(), apiKey, sessionID)
		if err != nil {
			am.logger.Warn("Falha na autenticação", "error", err, "sessionID", sessionID, "path", r.URL.Path)
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx, ok
}

func (am *AuthManager) UpdateSessionCache(apiKey, sessionID, key, value string) {
	cacheKey := BuildCacheKey(apiKey, sessionID)
	am.cacheManager.UpdateSessionInfo(cacheKey, key, value)
}

func (am *AuthManager) InvalidateSessionCache(apiKey, sessionID string) {
	cacheKey := BuildCacheKey(apiKey, sessionID)
	am.cacheManager.DeleteSessionInfo(cacheKey)
}

func (am *AuthManager) GetSessionFromCache(apiKey, sessionID string) (*SessionInfo, bool) {
	cacheKey := BuildCacheKey(apiKey, sessionID)
	return am.cacheManager.GetSessionInfo(cacheKey)
}

func (am *AuthManager) RefreshSessionCache(ctx context.Context, apiKey, sessionID string) error {
	session, err := am.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	sessionInfo := NewSessionInfoFromModel(session, apiKey)
	cacheKey := BuildCacheKey(apiKey, sessionID)
	am.cacheManager.SetSessionInfo(cacheKey, sessionInfo)

	return nil
}
