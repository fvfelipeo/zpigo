package dto

import (
	"encoding/base64"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// SendTextMessageRequest representa a requisição para envio de mensagem de texto
type SendTextMessageRequest struct {
	Phone       string             `json:"phone" validate:"required,min=10,max=20" example:"5511999999999" binding:"required"`           // Número do telefone destinatário
	Message     string             `json:"message" validate:"required,min=1,max=4096" example:"Olá, como você está?" binding:"required"` // Conteúdo da mensagem
	ID          string             `json:"id,omitempty" example:"custom-message-id"`                                                     // ID personalizado da mensagem (opcional)
	ContextInfo *waE2E.ContextInfo `json:"contextInfo,omitempty"`                                                                        // Informações de contexto para replies e mentions (opcional)
}

// SendTextMessageResponse representa a resposta do envio de mensagem de texto
type SendTextMessageResponse struct {
	Success   bool   `json:"success" example:"true"`                         // Indica se o envio foi bem-sucedido
	MessageID string `json:"messageId" example:"3EB0C431C26A1916EA9A_out"`   // ID da mensagem enviada
	Timestamp int64  `json:"timestamp" example:"1640995200"`                 // Timestamp do envio
	Details   string `json:"details" example:"Mensagem enviada com sucesso"` // Detalhes do envio
	Phone     string `json:"phone" example:"5511999999999"`                  // Número do telefone destinatário
}

// MessageErrorResponse representa uma resposta de erro específica para mensagens
type MessageErrorResponse struct {
	Error     bool   `json:"error" example:"true"`                                                               // Indica que houve erro
	Message   string `json:"message" example:"Sessão não conectada"`                                             // Mensagem de erro
	Code      int    `json:"code" example:"400"`                                                                 // Código HTTP do erro
	Details   string `json:"details,omitempty" example:"A sessão precisa estar conectada para enviar mensagens"` // Detalhes adicionais do erro
	Timestamp int64  `json:"timestamp" example:"1640995200"`                                                     // Timestamp do erro
}

// MessageStatusResponse representa o status de uma mensagem
type MessageStatusResponse struct {
	MessageID string `json:"messageId" example:"3EB0C431C26A1916EA9A_out"` // ID da mensagem
	Status    string `json:"status" example:"sent"`                        // Status da mensagem (sent, delivered, read)
	Timestamp int64  `json:"timestamp" example:"1640995200"`               // Timestamp da última atualização
	Phone     string `json:"phone" example:"5511999999999"`                // Número do telefone destinatário
}

// ValidatePhoneNumber valida se o número de telefone está no formato correto
// Segue o mesmo padrão da implementação de referência
func (req *SendTextMessageRequest) ValidatePhoneNumber() bool {
	phone := req.Phone
	if phone == "" {
		return false
	}

	// Remove + se presente (como na referência)
	phone = strings.TrimPrefix(phone, "+")

	// Verificar se contém apenas dígitos
	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}

	// Verificar comprimento (mínimo 8 dígitos, máximo 15 dígitos)
	if len(phone) < 8 || len(phone) > 15 {
		return false
	}

	return true
}

// SendMediaRequest representa a requisição para envio de mídia
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

// SendMediaResponse representa a resposta do envio de mídia
type SendMediaResponse struct {
	Success   bool   `json:"success" example:"true"`                       // Indica se o envio foi bem-sucedido
	MessageID string `json:"messageId" example:"3EB0C431C26A1916EA9A_out"` // ID da mensagem enviada
	Timestamp int64  `json:"timestamp" example:"1640995200"`               // Timestamp do envio
	Details   string `json:"details" example:"Mídia enviada com sucesso"`  // Detalhes da operação
	Phone     string `json:"phone" example:"5511999999999"`                // Número do telefone destinatário
	MediaType string `json:"mediaType" example:"image"`                    // Tipo de mídia enviada
	FileName  string `json:"fileName,omitempty" example:"imagem.jpg"`      // Nome do arquivo enviado
}

// ValidateMediaType valida se o tipo de mídia é suportado
func (req *SendMediaRequest) ValidateMediaType() bool {
	supportedTypes := map[string]bool{
		"image":    true,
		"audio":    true,
		"video":    true,
		"document": true,
	}
	return supportedTypes[strings.ToLower(req.MediaType)]
}

// ValidateMediaData valida se os dados da mídia estão em formato base64 válido
func (req *SendMediaRequest) ValidateMediaData() bool {
	if req.MediaData == "" {
		return false
	}

	// Verificar se é base64 válido
	_, err := base64.StdEncoding.DecodeString(req.MediaData)
	return err == nil
}

// GetMimeType retorna o tipo MIME da mídia
func (req *SendMediaRequest) GetMimeType() string {
	if req.MimeType != "" {
		return req.MimeType
	}

	// Detectar MIME type baseado no tipo de mídia e extensão do arquivo
	if req.FileName != "" {
		ext := strings.ToLower(filepath.Ext(req.FileName))
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	// MIME types padrão por tipo de mídia
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

// GetFileName retorna o nome do arquivo ou gera um padrão
func (req *SendMediaRequest) GetFileName() string {
	if req.FileName != "" {
		return req.FileName
	}

	// Gerar nome padrão baseado no tipo de mídia
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

// ValidatePhoneNumber valida se o número de telefone está no formato correto (reutiliza a validação existente)
func (req *SendMediaRequest) ValidatePhoneNumber() bool {
	phone := req.Phone
	if phone == "" {
		return false
	}

	// Remove + se presente (como na referência)
	phone = strings.TrimPrefix(phone, "+")

	// Verificar se contém apenas dígitos
	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}

	// Verificar comprimento (mínimo 8 dígitos, máximo 15 dígitos)
	if len(phone) < 8 || len(phone) > 15 {
		return false
	}

	return true
}

// ToErrorResponse converte um erro em MessageErrorResponse
func ToMessageErrorResponse(code int, message string, details string) *MessageErrorResponse {
	return &MessageErrorResponse{
		Error:     true,
		Message:   message,
		Code:      code,
		Details:   details,
		Timestamp: time.Now().Unix(),
	}
}

// ToSuccessResponse cria uma resposta de sucesso para envio de mensagem
func ToMessageSuccessResponse(messageID, phone string) *SendTextMessageResponse {
	return &SendTextMessageResponse{
		Success:   true,
		MessageID: messageID,
		Timestamp: time.Now().Unix(),
		Details:   "Mensagem enviada com sucesso",
		Phone:     phone,
	}
}

// ToMediaSuccessResponse cria uma resposta de sucesso para envio de mídia
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
