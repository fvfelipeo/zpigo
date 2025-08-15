package dto

import (
	"encoding/base64"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

type SendTextMessageRequest struct {
	Phone       string             `json:"phone" validate:"required,min=10,max=20" example:"5511999999999" binding:"required"`           // Número do telefone destinatário
	Message     string             `json:"message" validate:"required,min=1,max=4096" example:"Olá, como você está?" binding:"required"` // Conteúdo da mensagem
	ID          string             `json:"id,omitempty" example:"custom-message-id"`                                                     // ID personalizado da mensagem (opcional)
	ContextInfo *waE2E.ContextInfo `json:"contextInfo,omitempty"`                                                                        // Informações de contexto para replies e mentions (opcional)
}

type SendTextMessageResponse struct {
	Success   bool   `json:"success" example:"true"`                         // Indica se o envio foi bem-sucedido
	MessageID string `json:"messageId" example:"3EB0C431C26A1916EA9A_out"`   // ID da mensagem enviada
	Timestamp int64  `json:"timestamp" example:"1640995200"`                 // Timestamp do envio
	Details   string `json:"details" example:"Mensagem enviada com sucesso"` // Detalhes do envio
	Phone     string `json:"phone" example:"5511999999999"`                  // Número do telefone destinatário
}

type MessageErrorResponse struct {
	Error     bool   `json:"error" example:"true"`                                                               // Indica que houve erro
	Message   string `json:"message" example:"Sessão não conectada"`                                             // Mensagem de erro
	Code      int    `json:"code" example:"400"`                                                                 // Código HTTP do erro
	Details   string `json:"details,omitempty" example:"A sessão precisa estar conectada para enviar mensagens"` // Detalhes adicionais do erro
	Timestamp int64  `json:"timestamp" example:"1640995200"`                                                     // Timestamp do erro
}

type MessageStatusResponse struct {
	MessageID string `json:"messageId" example:"3EB0C431C26A1916EA9A_out"` // ID da mensagem
	Status    string `json:"status" example:"sent"`                        // Status da mensagem (sent, delivered, read)
	Timestamp int64  `json:"timestamp" example:"1640995200"`               // Timestamp da última atualização
	Phone     string `json:"phone" example:"5511999999999"`                // Número do telefone destinatário
}

func (req *SendTextMessageRequest) ValidatePhoneNumber() bool {
	phone := req.Phone
	if phone == "" {
		return false
	}

	phone = strings.TrimPrefix(phone, "+")

	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}

	if len(phone) < 8 || len(phone) > 15 {
		return false
	}

	return true
}

type SendMediaRequest struct {
	Phone       string             `json:"phone" validate:"required,min=10,max=20" example:"5511999999999" binding:"required"` // Número do telefone destinatário
	MediaType   string             `json:"mediaType" validate:"required" example:"image" binding:"required"`                   // Tipo de mídia: image, audio, video, document
	MediaData   string             `json:"mediaData" validate:"required" example:"base64_encoded_data" binding:"required"`     // Dados da mídia em base64
	FileName    string             `json:"fileName,omitempty" example:"documento.pdf"`                                         // Nome do arquivo (opcional)
	Caption     string             `json:"caption,omitempty" example:"Legenda da mídia"`                                       // Legenda da mídia (opcional)
	MimeType    string             `json:"mimeType,omitempty" example:"image/jpeg"`                                            // Tipo MIME (opcional, será detectado automaticamente)
	ID          string             `json:"id,omitempty" example:"custom-message-id"`                                           // ID personalizado da mensagem (opcional)
	ContextInfo *waE2E.ContextInfo `json:"contextInfo,omitempty"`                                                              // Informações de contexto para replies e mentions (opcional)
}

type SendMediaResponse struct {
	Success   bool   `json:"success" example:"true"`                       // Indica se o envio foi bem-sucedido
	MessageID string `json:"messageId" example:"3EB0C431C26A1916EA9A_out"` // ID da mensagem enviada
	Timestamp int64  `json:"timestamp" example:"1640995200"`               // Timestamp do envio
	Details   string `json:"details" example:"Mídia enviada com sucesso"`  // Detalhes da operação
	Phone     string `json:"phone" example:"5511999999999"`                // Número do telefone destinatário
	MediaType string `json:"mediaType" example:"image"`                    // Tipo de mídia enviada
	FileName  string `json:"fileName,omitempty" example:"imagem.jpg"`      // Nome do arquivo enviado
}

func (req *SendMediaRequest) ValidateMediaType() bool {
	supportedTypes := map[string]bool{
		"image":    true,
		"audio":    true,
		"video":    true,
		"document": true,
	}
	return supportedTypes[strings.ToLower(req.MediaType)]
}

func (req *SendMediaRequest) ValidateMediaData() bool {
	if req.MediaData == "" {
		return false
	}

	_, err := base64.StdEncoding.DecodeString(req.MediaData)
	return err == nil
}

func (req *SendMediaRequest) GetMimeType() string {
	if req.MimeType != "" {
		return req.MimeType
	}

	if req.FileName != "" {
		ext := strings.ToLower(filepath.Ext(req.FileName))
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	switch strings.ToLower(req.MediaType) {
	case "image":
		return "image/jpeg"
	case "audio":
		return "audio/mpeg"
	case "video":
		return "video/mp4"
	case "document":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func (req *SendMediaRequest) GetFileName() string {
	if req.FileName != "" {
		return req.FileName
	}

	switch strings.ToLower(req.MediaType) {
	case "image":
		return "image.jpg"
	case "audio":
		return "audio.mp3"
	case "video":
		return "video.mp4"
	case "document":
		return "document.pdf"
	default:
		return "file.bin"
	}
}

func (req *SendMediaRequest) ValidatePhoneNumber() bool {
	phone := req.Phone
	if phone == "" {
		return false
	}

	phone = strings.TrimPrefix(phone, "+")

	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}

	if len(phone) < 8 || len(phone) > 15 {
		return false
	}

	return true
}

func ToMessageErrorResponse(code int, message string, details string) *MessageErrorResponse {
	return &MessageErrorResponse{
		Error:     true,
		Message:   message,
		Code:      code,
		Details:   details,
		Timestamp: time.Now().Unix(),
	}
}

func ToMessageSuccessResponse(messageID, phone string) *SendTextMessageResponse {
	return &SendTextMessageResponse{
		Success:   true,
		MessageID: messageID,
		Timestamp: time.Now().Unix(),
		Details:   "Mensagem enviada com sucesso",
		Phone:     phone,
	}
}

func ToMediaSuccessResponse(messageID, phone, mediaType, fileName string) *SendMediaResponse {
	return &SendMediaResponse{
		Success:   true,
		MessageID: messageID,
		Timestamp: time.Now().Unix(),
		Details:   "Mídia enviada com sucesso",
		Phone:     phone,
		MediaType: mediaType,
		FileName:  fileName,
	}
}
