package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
	"go.mau.fi/whatsmeow/store/sqlstore"

	"zpigo/internal/api/dto"
	"zpigo/internal/db/models"
	"zpigo/internal/meow"
	"zpigo/internal/repository"
)

type SessionHandler struct {
	*BaseHandler
	sessionRepo    repository.SessionRepositoryInterface
	sessionManager *meow.SessionManager
	authManager    *meow.AuthManager
}

func NewSessionHandler(sessionRepo repository.SessionRepositoryInterface, container *sqlstore.Container, db *bun.DB) *SessionHandler {
	sessionManager := meow.NewSessionManager(container, db, sessionRepo)

	// Reconectar sessões que estavam conectadas antes do restart
	go func() {
		if err := sessionManager.ConnectOnStartup(); err != nil {
			// Log do erro mas não falha a inicialização
			fmt.Printf("Erro ao reconectar sessões na inicialização: %v\n", err)
		}
	}()

	return &SessionHandler{
		BaseHandler:    NewBaseHandler("SessionHandler"),
		sessionRepo:    sessionRepo,
		sessionManager: sessionManager,
		authManager:    meow.NewAuthManager(db, sessionRepo),
	}
}

// @Summary      Criar nova sessão WhatsApp
// @Description  Cria uma nova sessão WhatsApp com o nome especificado
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateSessionRequest  true  "Dados da sessão"
// @Success      201      {object}  dto.CreateSessionResponse
// @Failure      400      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]interface{}
// @Router       /sessions/add [post]
func (h *SessionHandler) AddSession(c *gin.Context) {
	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Erro ao decodificar request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Dados inválidos",
			"details": err.Error(),
		})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Nome da sessão é obrigatório",
		})
		return
	}

	h.logger.Info("Criando nova sessão", "name", req.Name)

	session := &models.Session{
		Name:   req.Name,
		Status: models.StatusDisconnected,
	}

	if err := h.sessionRepo.Create(c.Request.Context(), session); err != nil {
		h.logger.Error("Erro ao criar sessão no banco", "error", err, "name", req.Name)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Erro ao criar sessão",
			"details": err.Error(),
		})
		return
	}

	_, err := h.sessionManager.CreateSession(session.ID)
	if err != nil {
		h.logger.Error("Erro ao inicializar sessão no manager", "error", err, "sessionID", session.ID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Erro ao inicializar sessão",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Sessão criada com sucesso", "sessionID", session.ID, "name", session.Name)

	response := &dto.CreateSessionResponse{
		Session: dto.ToSessionResponse(session),
		Message: "Sessão criada com sucesso",
	}

	c.JSON(http.StatusCreated, response)
}

// @Summary      Listar todas as sessões
// @Description  Retorna uma lista com todas as sessões WhatsApp criadas
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.SessionListResponse
// @Failure      500  {object}  map[string]interface{}
// @Router       /sessions/list [get]
func (h *SessionHandler) ListSessions(c *gin.Context) {
	h.logger.Debug("Listando sessões")

	sessions, err := h.sessionRepo.List(c.Request.Context())
	if err != nil {
		h.logger.Error("Erro ao listar sessões", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Erro ao listar sessões",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Sessões listadas com sucesso", "total", len(sessions))

	response := &dto.SessionListResponse{
		Sessions: dto.ToSessionResponseList(sessions),
		Total:    len(sessions),
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      Obter informações da sessão
// @Description  Retorna informações detalhadas de uma sessão específica
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string  true  "ID da sessão"
// @Success      200        {object}  dto.SessionInfoResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Router       /sessions/{sessionID}/info [get]
// @Security     ApiKeyAuth
func (h *SessionHandler) GetSessionInfo(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	h.logger.Debug("Buscando informações da sessão", "sessionID", sessionID)

	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Sessão não encontrada", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Informações da sessão obtidas", "sessionID", sessionID, "status", session.Status)

	response := &dto.SessionInfoResponse{
		Session:     dto.ToSessionResponse(session),
		IsConnected: session.IsConnected(),
		HasProxy:    session.HasProxy(),
	}

	c.JSON(http.StatusOK, response)
}

// GetSessionStatus godoc
// @Summary      Verificar status da sessão
// @Description  Verifica o status atual de conexão de uma sessão WhatsApp
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string  true  "ID da sessão"
// @Success      200        {object}  dto.SessionStatusResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Router       /sessions/{sessionID}/status [get]
// @Security     ApiKeyAuth
func (h *SessionHandler) GetSessionStatus(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	h.logger.Debug("Verificando status da sessão", "sessionID", sessionID)

	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Sessão não encontrada para verificar status", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	isConnected, isLoggedIn, err := h.sessionManager.GetSessionStatus(sessionID)
	if err != nil {
		h.logger.Warn("Erro ao verificar status no manager", "sessionID", sessionID, "error", err)
		isConnected = session.IsConnected()
		isLoggedIn = session.Status == models.StatusConnected
	}

	h.logger.Info("Status da sessão verificado", "sessionID", sessionID, "connected", isConnected, "loggedIn", isLoggedIn)

	response := &dto.SessionStatusResponse{
		SessionID: sessionID,
		Connected: isConnected,
		LoggedIn:  isLoggedIn,
		Status:    session.Status,
		Phone:     session.Phone,
		HasProxy:  session.HasProxy(),
		Timestamp: session.UpdatedAt.Unix(),
	}

	c.JSON(http.StatusOK, response)
}

// DeleteSession godoc
// @Summary      Deletar sessão
// @Description  Remove uma sessão WhatsApp e todos os seus dados
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string  true  "ID da sessão"
// @Success      200        {object}  dto.DeleteSessionResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Router       /sessions/{sessionID} [delete]
// @Security     ApiKeyAuth
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	h.logger.Info("Removendo sessão", "sessionID", sessionID)

	if err := h.sessionManager.DeleteSession(sessionID); err != nil {
		h.logger.Warn("Erro ao remover sessão do manager", "sessionID", sessionID, "error", err)
	}

	if err := h.sessionRepo.Delete(c.Request.Context(), sessionID); err != nil {
		h.logger.Error("Erro ao remover sessão do banco", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Sessão removida com sucesso", "sessionID", sessionID)

	response := &dto.DeleteSessionResponse{
		Message: "Sessão removida com sucesso",
		Success: true,
	}

	c.JSON(http.StatusOK, response)
}

// ConnectSession godoc
// @Summary      Conectar sessão WhatsApp
// @Description  Inicia a conexão de uma sessão WhatsApp
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string  true  "ID da sessão"
// @Success      200        {object}  dto.ConnectSessionResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /sessions/{sessionID}/connect [post]
// @Security     ApiKeyAuth
func (h *SessionHandler) ConnectSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	h.logger.Info("Iniciando conexão da sessão", "sessionID", sessionID)

	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Sessão não encontrada para conexão", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	if err := h.sessionManager.ConnectSession(sessionID); err != nil {
		h.logger.Error("Erro ao conectar sessão", "sessionID", sessionID, "error", err)

		if updateErr := h.sessionRepo.UpdateStatus(c.Request.Context(), sessionID, models.StatusDisconnected); updateErr != nil {
			h.logger.Error("Erro ao atualizar status para disconnected após falha de conexão", "sessionID", sessionID, "error", updateErr)
		} else {
			h.logger.Info("Status da sessão voltou para disconnected após erro de conexão", "sessionID", sessionID)
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Erro ao conectar sessão",
			"details": err.Error(),
		})
		return
	}

	if err := h.sessionRepo.UpdateStatus(c.Request.Context(), sessionID, models.StatusConnecting); err != nil {
		h.logger.Warn("Erro ao atualizar status da sessão", "sessionID", sessionID, "error", err)
	}

	h.logger.Info("Conexão da sessão iniciada", "sessionID", sessionID)

	response := &dto.ConnectSessionResponse{
		Session: dto.ToSessionResponse(session),
		Message: "Conexão iniciada com sucesso",
	}

	c.JSON(http.StatusOK, response)
}

// LogoutSession godoc
// @Summary      Fazer logout da sessão
// @Description  Desconecta uma sessão WhatsApp ativa
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string  true  "ID da sessão"
// @Success      200        {object}  dto.LogoutSessionResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /sessions/{sessionID}/logout [post]
// @Security     ApiKeyAuth
func (h *SessionHandler) LogoutSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	h.logger.Info("Fazendo logout da sessão", "sessionID", sessionID)

	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Sessão não encontrada para logout", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	if err := h.sessionManager.LogoutSession(sessionID); err != nil {
		h.logger.Error("Erro ao fazer logout da sessão", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Erro ao fazer logout",
			"details": err.Error(),
		})
		return
	}

	if err := h.sessionRepo.SetDisconnected(c.Request.Context(), sessionID); err != nil {
		h.logger.Warn("Erro ao atualizar status da sessão", "sessionID", sessionID, "error", err)
	}

	h.logger.Info("Logout da sessão realizado", "sessionID", sessionID)

	response := &dto.LogoutSessionResponse{
		Session: dto.ToSessionResponse(session),
		Message: "Logout realizado com sucesso",
	}

	c.JSON(http.StatusOK, response)
}

// GetQRCode godoc
// @Summary      Gerar QR Code para conexão
// @Description  Gera um QR Code para conectar o WhatsApp Web
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string  true  "ID da sessão"
// @Success      200        {object}  dto.QRCodeResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /sessions/{sessionID}/qr [get]
// @Security     ApiKeyAuth
func (h *SessionHandler) GetQRCode(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	h.logger.Info("Gerando QR Code para sessão", "sessionID", sessionID)

	qrCode, err := h.sessionManager.GenerateQRCode(sessionID)
	if err != nil {
		h.logger.Error("Erro ao gerar QR code", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Erro ao gerar QR code",
			"details": err.Error(),
		})
		return
	}

	if err := h.sessionRepo.UpdateQRCode(c.Request.Context(), sessionID, qrCode); err != nil {
		h.logger.Warn("Erro ao salvar QR code no banco", "sessionID", sessionID, "error", err)
	}

	h.logger.Info("QR Code gerado com sucesso", "sessionID", sessionID)

	response := &dto.QRCodeResponse{
		SessionID: sessionID,
		QRCode:    qrCode,
		ExpiresIn: 60,
	}

	c.JSON(http.StatusOK, response)
}

// PairPhone godoc
// @Summary      Emparelhar telefone
// @Description  Emparelha um número de telefone com a sessão WhatsApp
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string                 true  "ID da sessão"
// @Param        request    body      dto.PairPhoneRequest   true  "Dados do telefone"
// @Success      200        {object}  dto.PairPhoneResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /sessions/{sessionID}/pairphone [post]
// @Security     ApiKeyAuth
func (h *SessionHandler) PairPhone(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	var req dto.PairPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Erro ao decodificar request de emparelhamento", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Dados inválidos",
			"details": err.Error(),
		})
		return
	}

	if req.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Número do telefone é obrigatório",
		})
		return
	}

	h.logger.Info("Iniciando emparelhamento de telefone", "sessionID", sessionID, "phone", req.PhoneNumber)

	linkingCode, err := h.sessionManager.PairPhone(sessionID, req.PhoneNumber)
	if err != nil {
		h.logger.Error("Erro ao emparelhar telefone", "sessionID", sessionID, "phone", req.PhoneNumber, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Erro ao emparelhar telefone",
			"details": err.Error(),
		})
		return
	}

	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Sessão não encontrada após emparelhamento", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Emparelhamento iniciado com sucesso", "sessionID", sessionID, "linkingCode", linkingCode)

	response := &dto.PairPhoneResponse{
		Session: dto.ToSessionResponse(session),
		Message: fmt.Sprintf("Código de emparelhamento: %s", linkingCode),
		Success: true,
	}

	c.JSON(http.StatusOK, response)
}

// SetProxy godoc
// @Summary      Configurar proxy
// @Description  Configura um proxy para a sessão WhatsApp
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string                true  "ID da sessão"
// @Param        request    body      dto.SetProxyRequest   true  "Dados do proxy"
// @Success      200        {object}  dto.SetProxyResponse
// @Failure      400        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Router       /sessions/{sessionID}/proxy/set [post]
// @Security     ApiKeyAuth
func (h *SessionHandler) SetProxy(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "ID da sessão é obrigatório",
		})
		return
	}

	var req dto.SetProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Erro ao decodificar request de proxy", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Dados inválidos",
			"details": err.Error(),
		})
		return
	}

	if req.Host == "" || req.Port == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Host e porta são obrigatórios",
		})
		return
	}

	h.logger.Info("Configurando proxy para sessão", "sessionID", sessionID, "host", req.Host, "port", req.Port, "type", req.Type)

	err := h.sessionRepo.UpdateProxy(c.Request.Context(), sessionID, req.Host, req.Port, req.Type, req.Username, req.Password)
	if err != nil {
		h.logger.Error("Erro ao atualizar proxy no banco", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Erro ao buscar sessão após configurar proxy", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   true,
			"message": "Sessão não encontrada",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Proxy configurado com sucesso", "sessionID", sessionID)

	response := &dto.SetProxyResponse{
		Session: dto.ToSessionResponse(session),
		Message: "Proxy configurado com sucesso",
	}

	c.JSON(http.StatusOK, response)
}
