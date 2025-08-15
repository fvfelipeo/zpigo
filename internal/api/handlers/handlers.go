package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"zpigo/internal/logger"
)

type BaseHandler struct {
	logger logger.Logger
}

func NewBaseHandler(component string) *BaseHandler {
	return &BaseHandler{
		logger: logger.NewForComponent(component),
	}
}

func (h *BaseHandler) WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Erro ao codificar JSON", "error", err)
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
	}
}

func (h *BaseHandler) WriteErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	h.logger.Error("Erro HTTP", "status", statusCode, "message", message, "error", err)

	errorMsg := message
	if err != nil {
		errorMsg = message + ": " + err.Error()
	}

	response := map[string]interface{}{
		"error":     true,
		"message":   errorMsg,
		"code":      statusCode,
		"timestamp": time.Now().Unix(),
	}

	h.WriteJSONResponse(w, statusCode, response)
}

func (h *BaseHandler) WriteSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	response := map[string]interface{}{
		"success":   true,
		"message":   message,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	h.WriteJSONResponse(w, http.StatusOK, response)
}

// @Summary      Verificar saúde da API
// @Description  Endpoint para verificar se a API está funcionando
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /health [get]
func HealthCheck(c *gin.Context) {
	response := map[string]interface{}{
		"status":    "ok",
		"service":   "zpigo-api",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	}

	c.JSON(http.StatusOK, response)
}
