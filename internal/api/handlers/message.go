package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"zpigo/internal/api/dto"
	"zpigo/internal/meow"
	"zpigo/internal/repository"
)

type MessageHandler struct {
	*BaseHandler
	sessionRepo    repository.SessionRepositoryInterface
	sessionManager *meow.SessionManager
	authManager    *meow.AuthManager
}

func NewMessageHandler(sessionRepo repository.SessionRepositoryInterface, container *sqlstore.Container, db *bun.DB) *MessageHandler {
	sessionManager := meow.NewSessionManager(container, db, sessionRepo)

	return &MessageHandler{
		BaseHandler:    NewBaseHandler("MessageHandler"),
		sessionRepo:    sessionRepo,
		sessionManager: sessionManager,
		authManager:    meow.NewAuthManager(db, sessionRepo),
	}
}

// NewMessageHandlerWithManager cria um MessageHandler com um SessionManager compartilhado
func NewMessageHandlerWithManager(sessionRepo repository.SessionRepositoryInterface, sessionManager *meow.SessionManager) *MessageHandler {
	return &MessageHandler{
		BaseHandler:    NewBaseHandler("MessageHandler"),
		sessionRepo:    sessionRepo,
		sessionManager: sessionManager,
		authManager:    meow.NewAuthManager(sessionManager.GetDB(), sessionRepo),
	}
}

// SendTextMessage godoc
// @Summary      Enviar mensagem de texto via WhatsApp
// @Description  Envia uma mensagem de texto para um número específico através da sessão WhatsApp
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string                        true  "ID da sessão"
// @Param        request    body      dto.SendTextMessageRequest    true  "Dados da mensagem"
// @Success      200        {object}  dto.SendTextMessageResponse
// @Failure      400        {object}  dto.MessageErrorResponse
// @Failure      404        {object}  dto.MessageErrorResponse
// @Failure      500        {object}  dto.MessageErrorResponse
// @Router       /sessions/{sessionID}/message/send/text [post]
// @Security     ApiKeyAuth
func (h *MessageHandler) SendTextMessage(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		h.logger.Error("ID da sessão não fornecido")
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"ID da sessão é obrigatório",
			"O parâmetro sessionID deve ser fornecido na URL",
		))
		return
	}

	h.logger.Info("Iniciando envio de mensagem de texto", "sessionID", sessionID)

	// Validar e decodificar request
	var req dto.SendTextMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Erro ao decodificar request", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Dados inválidos",
			err.Error(),
		))
		return
	}

	// Validar campos obrigatórios
	if req.Phone == "" {
		h.logger.Error("Número de telefone não fornecido", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Número de telefone é obrigatório",
			"O campo 'phone' deve ser fornecido",
		))
		return
	}

	if req.Message == "" {
		h.logger.Error("Mensagem não fornecida", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Mensagem é obrigatória",
			"O campo 'message' deve ser fornecido",
		))
		return
	}

	// Validar formato do telefone
	if !req.ValidatePhoneNumber() {
		h.logger.Error("Formato de telefone inválido", "sessionID", sessionID, "phone", req.Phone)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Formato de telefone inválido",
			"O número deve conter entre 10 e 20 dígitos",
		))
		return
	}

	// Verificar se a sessão existe
	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Sessão não encontrada", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, dto.ToMessageErrorResponse(
			http.StatusNotFound,
			"Sessão não encontrada",
			err.Error(),
		))
		return
	}

	// Verificar se a sessão está conectada
	if !session.IsConnected() {
		h.logger.Error("Sessão não está conectada", "sessionID", sessionID, "status", session.Status)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Sessão não conectada",
			"A sessão precisa estar conectada para enviar mensagens",
		))
		return
	}

	// Debug: Listar todas as sessões ativas
	activeSessions := h.sessionManager.ListSessions()
	h.logger.Info("Sessões ativas no SessionManager", "sessionID", sessionID, "activeSessions", activeSessions, "totalSessions", len(activeSessions))

	// Obter cliente WhatsApp
	client, exists := h.sessionManager.GetSession(sessionID)
	if !exists {
		h.logger.Error("Cliente WhatsApp não encontrado", "sessionID", sessionID, "activeSessions", activeSessions)
		c.JSON(http.StatusInternalServerError, dto.ToMessageErrorResponse(
			http.StatusInternalServerError,
			"Cliente WhatsApp não encontrado",
			"Sessão não está ativa no gerenciador",
		))
		return
	}

	h.logger.Info("Cliente WhatsApp encontrado", "sessionID", sessionID, "clientConnected", client.IsConnected())

	// Verificar se o cliente está conectado
	if !client.IsConnected() {
		h.logger.Error("Cliente WhatsApp não está conectado", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Cliente WhatsApp não conectado",
			"O cliente WhatsApp precisa estar conectado",
		))
		return
	}

	// Validar ContextInfo se fornecido (para replies)
	if err := h.validateContextInfo(req.ContextInfo); err != nil {
		h.logger.Error("ContextInfo inválido", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"ContextInfo inválido",
			err.Error(),
		))
		return
	}

	// Validar e parsear JID do destinatário
	recipient, err := h.parseJID(req.Phone)
	if err != nil {
		h.logger.Error("Erro ao parsear número de telefone", "sessionID", sessionID, "phone", req.Phone, "error", err)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Número de telefone inválido",
			err.Error(),
		))
		return
	}

	// Gerar ID da mensagem
	messageID := req.ID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Criar mensagem WhatsApp
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(req.Message),
		},
	}

	// Adicionar ContextInfo se fornecido (para replies e mentions)
	if req.ContextInfo != nil {
		msg.ExtendedTextMessage.ContextInfo = req.ContextInfo
		h.logger.Info("ContextInfo adicionado à mensagem", "sessionID", sessionID, "messageID", messageID)
	}

	h.logger.Info("Enviando mensagem", "sessionID", sessionID, "phone", req.Phone, "messageID", messageID)

	// Enviar mensagem
	resp, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		h.logger.Error("Erro ao enviar mensagem", "sessionID", sessionID, "phone", req.Phone, "messageID", messageID, "error", err)
		c.JSON(http.StatusInternalServerError, dto.ToMessageErrorResponse(
			http.StatusInternalServerError,
			"Erro ao enviar mensagem",
			err.Error(),
		))
		return
	}

	h.logger.Info("Mensagem enviada com sucesso", "sessionID", sessionID, "phone", req.Phone, "messageID", messageID, "timestamp", resp.Timestamp)

	// Criar resposta de sucesso
	response := dto.ToMessageSuccessResponse(messageID, req.Phone)
	response.Timestamp = resp.Timestamp.Unix()

	c.JSON(http.StatusOK, response)
}

// SendMedia godoc
// @Summary      Enviar mídia via WhatsApp
// @Description  Envia mídia (imagem, áudio, vídeo, documento) para um número específico através da sessão WhatsApp
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string                     true  "ID da sessão"
// @Param        request    body      dto.SendMediaRequest       true  "Dados da mídia"
// @Success      200        {object}  dto.SendMediaResponse
// @Failure      400        {object}  dto.MessageErrorResponse
// @Failure      404        {object}  dto.MessageErrorResponse
// @Failure      500        {object}  dto.MessageErrorResponse
// @Router       /sessions/{sessionID}/message/send/media [post]
// @Security     ApiKeyAuth
func (h *MessageHandler) SendMedia(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		h.logger.Error("ID da sessão não fornecido")
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"ID da sessão é obrigatório",
			"O parâmetro sessionID deve ser fornecido na URL",
		))
		return
	}

	h.logger.Info("Iniciando envio de mídia", "sessionID", sessionID)

	// Validar e decodificar request
	var req dto.SendMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Erro ao decodificar request", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Dados inválidos",
			err.Error(),
		))
		return
	}

	// Validar campos obrigatórios
	if req.Phone == "" {
		h.logger.Error("Número de telefone não fornecido", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Número de telefone é obrigatório",
			"O campo 'phone' deve ser fornecido",
		))
		return
	}

	if req.MediaType == "" {
		h.logger.Error("Tipo de mídia não fornecido", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Tipo de mídia é obrigatório",
			"O campo 'mediaType' deve ser fornecido",
		))
		return
	}

	if req.MediaData == "" {
		h.logger.Error("Dados da mídia não fornecidos", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Dados da mídia são obrigatórios",
			"O campo 'mediaData' deve ser fornecido",
		))
		return
	}

	// Validar formato do telefone
	if !req.ValidatePhoneNumber() {
		h.logger.Error("Formato de telefone inválido", "sessionID", sessionID, "phone", req.Phone)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Formato de telefone inválido",
			"O número deve conter entre 8 e 15 dígitos",
		))
		return
	}

	// Validar tipo de mídia
	if !req.ValidateMediaType() {
		h.logger.Error("Tipo de mídia inválido", "sessionID", sessionID, "mediaType", req.MediaType)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Tipo de mídia inválido",
			"Tipos suportados: image, audio, video, document",
		))
		return
	}

	// Validar dados da mídia (base64)
	if !req.ValidateMediaData() {
		h.logger.Error("Dados da mídia inválidos", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Dados da mídia inválidos",
			"Os dados devem estar em formato base64 válido",
		))
		return
	}

	// Verificar se a sessão existe
	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Sessão não encontrada", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusNotFound, dto.ToMessageErrorResponse(
			http.StatusNotFound,
			"Sessão não encontrada",
			err.Error(),
		))
		return
	}

	// Verificar se a sessão está conectada
	if !session.IsConnected() {
		h.logger.Error("Sessão não está conectada", "sessionID", sessionID, "status", session.Status)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Sessão não conectada",
			"A sessão precisa estar conectada para enviar mídia",
		))
		return
	}

	// Obter cliente WhatsApp
	client, exists := h.sessionManager.GetSession(sessionID)
	if !exists {
		h.logger.Error("Cliente WhatsApp não encontrado", "sessionID", sessionID)
		c.JSON(http.StatusInternalServerError, dto.ToMessageErrorResponse(
			http.StatusInternalServerError,
			"Cliente WhatsApp não encontrado",
			"Sessão não está ativa no gerenciador",
		))
		return
	}

	// Verificar se o cliente está conectado
	if !client.IsConnected() {
		h.logger.Error("Cliente WhatsApp não está conectado", "sessionID", sessionID)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Cliente WhatsApp não conectado",
			"O cliente WhatsApp precisa estar conectado",
		))
		return
	}

	// Validar ContextInfo se fornecido
	if err := h.validateContextInfo(req.ContextInfo); err != nil {
		h.logger.Error("ContextInfo inválido", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"ContextInfo inválido",
			err.Error(),
		))
		return
	}

	// Validar e parsear JID do destinatário
	recipient, err := h.parseJID(req.Phone)
	if err != nil {
		h.logger.Error("Erro ao parsear número de telefone", "sessionID", sessionID, "phone", req.Phone, "error", err)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Número de telefone inválido",
			err.Error(),
		))
		return
	}

	// Decodificar dados da mídia
	mediaBytes, err := base64.StdEncoding.DecodeString(req.MediaData)
	if err != nil {
		h.logger.Error("Erro ao decodificar dados da mídia", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Erro ao decodificar mídia",
			err.Error(),
		))
		return
	}

	// Gerar ID da mensagem
	messageID := req.ID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Preparar dados para upload
	fileName := req.GetFileName()
	mimeType := req.GetMimeType()

	h.logger.Info("Preparando upload de mídia",
		"sessionID", sessionID,
		"mediaType", req.MediaType,
		"fileName", fileName,
		"mimeType", mimeType,
		"size", len(mediaBytes))

	// Mapear tipo de mídia para whatsmeow.MediaType
	var mediaType whatsmeow.MediaType
	switch strings.ToLower(req.MediaType) {
	case "image":
		mediaType = whatsmeow.MediaImage
	case "audio":
		mediaType = whatsmeow.MediaAudio
	case "video":
		mediaType = whatsmeow.MediaVideo
	case "document":
		mediaType = whatsmeow.MediaDocument
	default:
		h.logger.Error("Tipo de mídia não suportado para upload", "sessionID", sessionID, "mediaType", req.MediaType)
		c.JSON(http.StatusBadRequest, dto.ToMessageErrorResponse(
			http.StatusBadRequest,
			"Tipo de mídia não suportado",
			fmt.Sprintf("Tipo '%s' não é suportado para upload", req.MediaType),
		))
		return
	}

	// Fazer upload da mídia para WhatsApp
	uploadResp, err := client.Upload(context.Background(), mediaBytes, mediaType)
	if err != nil {
		h.logger.Error("Erro ao fazer upload da mídia", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, dto.ToMessageErrorResponse(
			http.StatusInternalServerError,
			"Erro ao fazer upload da mídia",
			err.Error(),
		))
		return
	}

	// Criar mensagem baseada no tipo de mídia
	msg, err := h.createMediaMessage(req.MediaType, uploadResp, fileName, mimeType, req.Caption, req.ContextInfo)
	if err != nil {
		h.logger.Error("Erro ao criar mensagem de mídia", "sessionID", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, dto.ToMessageErrorResponse(
			http.StatusInternalServerError,
			"Erro ao criar mensagem de mídia",
			err.Error(),
		))
		return
	}

	h.logger.Info("Enviando mídia", "sessionID", sessionID, "phone", req.Phone, "messageID", messageID, "mediaType", req.MediaType)

	// Enviar mensagem
	resp, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		h.logger.Error("Erro ao enviar mídia", "sessionID", sessionID, "phone", req.Phone, "messageID", messageID, "error", err)
		c.JSON(http.StatusInternalServerError, dto.ToMessageErrorResponse(
			http.StatusInternalServerError,
			"Erro ao enviar mídia",
			err.Error(),
		))
		return
	}

	h.logger.Info("Mídia enviada com sucesso", "sessionID", sessionID, "phone", req.Phone, "messageID", messageID, "timestamp", resp.Timestamp, "mediaType", req.MediaType)

	// Criar resposta de sucesso
	response := dto.ToMediaSuccessResponse(messageID, req.Phone, req.MediaType, fileName)
	response.Timestamp = resp.Timestamp.Unix()

	c.JSON(http.StatusOK, response)
}

// parseJID converte um número de telefone em JID do WhatsApp
// Segue exatamente o padrão da implementação de referência
func (h *MessageHandler) parseJID(phone string) (types.JID, error) {
	// Remove + se presente (como na referência)
	if len(phone) > 0 && phone[0] == '+' {
		phone = phone[1:]
	}

	// Se não contém @, adicionar o servidor padrão
	if !strings.ContainsRune(phone, '@') {
		return types.NewJID(phone, types.DefaultUserServer), nil
	}

	// Parsear JID completo
	recipient, err := types.ParseJID(phone)
	if err != nil {
		h.logger.Error("JID inválido", "phone", phone, "error", err)
		return types.JID{}, fmt.Errorf("JID inválido: %v", err)
	}

	if recipient.User == "" {
		h.logger.Error("JID inválido: nenhum servidor especificado", "phone", phone)
		return types.JID{}, fmt.Errorf("JID inválido: nenhum servidor especificado")
	}

	return recipient, nil
}

// validateContextInfo valida as informações de contexto para replies e mentions
// Segue o padrão da implementação de referência
func (h *MessageHandler) validateContextInfo(contextInfo *waE2E.ContextInfo) error {
	if contextInfo == nil {
		return nil // ContextInfo é opcional
	}

	// Validar regras para replies (StanzaID e Participant devem ser fornecidos juntos)
	if contextInfo.StanzaID != nil {
		if contextInfo.Participant == nil {
			return fmt.Errorf("participant é obrigatório quando StanzaID é fornecido")
		}
	}

	if contextInfo.Participant != nil {
		if contextInfo.StanzaID == nil {
			return fmt.Errorf("stanzaID é obrigatório quando Participant é fornecido")
		}
	}

	return nil
}

// createMediaMessage cria uma mensagem de mídia baseada no tipo
func (h *MessageHandler) createMediaMessage(mediaType string, uploadResp whatsmeow.UploadResponse, fileName, mimeType, caption string, contextInfo *waE2E.ContextInfo) (*waE2E.Message, error) {
	switch strings.ToLower(mediaType) {
	case "image":
		return h.createImageMessage(uploadResp, fileName, mimeType, caption, contextInfo), nil
	case "audio":
		return h.createAudioMessage(uploadResp, fileName, mimeType, contextInfo), nil
	case "video":
		return h.createVideoMessage(uploadResp, fileName, mimeType, caption, contextInfo), nil
	case "document":
		return h.createDocumentMessage(uploadResp, fileName, mimeType, contextInfo), nil
	default:
		return nil, fmt.Errorf("tipo de mídia não suportado: %s", mediaType)
	}
}

// createImageMessage cria uma mensagem de imagem
func (h *MessageHandler) createImageMessage(uploadResp whatsmeow.UploadResponse, _ string, mimeType, caption string, contextInfo *waE2E.ContextInfo) *waE2E.Message {
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uploadResp.FileLength),
		},
	}

	if caption != "" {
		msg.ImageMessage.Caption = proto.String(caption)
	}

	if contextInfo != nil {
		msg.ImageMessage.ContextInfo = contextInfo
	}

	return msg
}

// createAudioMessage cria uma mensagem de áudio
func (h *MessageHandler) createAudioMessage(uploadResp whatsmeow.UploadResponse, _ string, mimeType string, contextInfo *waE2E.ContextInfo) *waE2E.Message {
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uploadResp.FileLength),
		},
	}

	if contextInfo != nil {
		msg.AudioMessage.ContextInfo = contextInfo
	}

	return msg
}

// createVideoMessage cria uma mensagem de vídeo
func (h *MessageHandler) createVideoMessage(uploadResp whatsmeow.UploadResponse, _ string, mimeType, caption string, contextInfo *waE2E.ContextInfo) *waE2E.Message {
	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uploadResp.FileLength),
		},
	}

	if caption != "" {
		msg.VideoMessage.Caption = proto.String(caption)
	}

	if contextInfo != nil {
		msg.VideoMessage.ContextInfo = contextInfo
	}

	return msg
}

// createDocumentMessage cria uma mensagem de documento
func (h *MessageHandler) createDocumentMessage(uploadResp whatsmeow.UploadResponse, fileName, mimeType string, contextInfo *waE2E.ContextInfo) *waE2E.Message {
	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uploadResp.FileLength),
			FileName:      proto.String(fileName),
		},
	}

	if contextInfo != nil {
		msg.DocumentMessage.ContextInfo = contextInfo
	}

	return msg
}
