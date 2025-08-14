package dto

import (
	"time"

	"zpigo/internal/db/models"
)

type CreateSessionRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255" example:"Minha Sessão WhatsApp" binding:"required"` // Nome da sessão
}

type CreateSessionResponse struct {
	Session *SessionResponse `json:"session"`                                     // Dados da sessão criada
	Message string           `json:"message" example:"Sessão criada com sucesso"` // Mensagem de confirmação
}

type SessionResponse struct {
	ID          string               `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`           // ID único da sessão
	Name        string               `json:"name" example:"Minha Sessão WhatsApp"`                        // Nome da sessão
	Phone       string               `json:"phone,omitempty" example:"5511999999999"`                     // Número do telefone conectado
	Status      models.SessionStatus `json:"status" example:"disconnected"`                               // Status da sessão
	QRCode      string               `json:"qrCode,omitempty" example:"data:image/png;base64,iVBORw0..."` // QR Code em base64
	ProxyHost   string               `json:"proxyHost,omitempty" example:"proxy.example.com"`             // Host do proxy
	ProxyPort   int                  `json:"proxyPort,omitempty" example:"8080"`                          // Porta do proxy
	ProxyType   models.ProxyType     `json:"proxyType,omitempty" example:"http"`                          // Tipo do proxy
	ProxyUser   string               `json:"proxyUser,omitempty" example:"usuario"`                       // Usuário do proxy
	ProxyPass   string               `json:"proxyPass,omitempty" example:"senha"`                         // Senha do proxy
	CreatedAt   time.Time            `json:"createdAt" example:"2023-01-01T00:00:00Z"`                    // Data de criação
	UpdatedAt   time.Time            `json:"updatedAt" example:"2023-01-01T00:00:00Z"`                    // Data de atualização
	ConnectedAt *time.Time           `json:"connectedAt,omitempty" example:"2023-01-01T00:00:00Z"`        // Data de conexão
}

type SessionListResponse struct {
	Sessions []*SessionResponse `json:"sessions"`
	Total    int                `json:"total"`
}

type SessionInfoResponse struct {
	Session     *SessionResponse `json:"session"`
	IsConnected bool             `json:"isConnected"`
	HasProxy    bool             `json:"hasProxy"`
}

type ConnectSessionRequest struct {
	Timeout int `json:"timeout,omitempty"`
}

type ConnectSessionResponse struct {
	Session *SessionResponse `json:"session"`
	Message string           `json:"message"`
	QRCode  string           `json:"qrCode,omitempty"`
}

type LogoutSessionResponse struct {
	Session *SessionResponse `json:"session"`
	Message string           `json:"message"`
}

type QRCodeResponse struct {
	SessionID string `json:"sessionId"`
	QRCode    string `json:"qrCode"`
	ExpiresIn int    `json:"expiresIn"`
}

type PairPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber" validate:"required,min=10,max=20"`
	Code        string `json:"code" validate:"required,len=6"`
}

type PairPhoneResponse struct {
	Session *SessionResponse `json:"session"`
	Message string           `json:"message"`
	Success bool             `json:"success"`
}

type SetProxyRequest struct {
	Host     string           `json:"host" validate:"required"`
	Port     int              `json:"port" validate:"required,min=1,max=65535"`
	Type     models.ProxyType `json:"type" validate:"required"`
	Username string           `json:"username,omitempty"`
	Password string           `json:"password,omitempty"`
}

type SetProxyResponse struct {
	Session *SessionResponse `json:"session"`
	Message string           `json:"message"`
}

type DeleteSessionResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type SessionStatusResponse struct {
	SessionID string               `json:"sessionId"`
	Connected bool                 `json:"connected"`
	LoggedIn  bool                 `json:"loggedIn"`
	Status    models.SessionStatus `json:"status"`
	Phone     string               `json:"phone,omitempty"`
	HasProxy  bool                 `json:"hasProxy"`
	Timestamp int64                `json:"timestamp"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type APIResponse struct {
	Code    int  `json:"code"`
	Data    any  `json:"data"`
	Success bool `json:"success"`
}

type APIErrorResponse struct {
	Code    int    `json:"code"`
	Error   string `json:"error"`
	Success bool   `json:"success"`
}

type SessionConnectData struct {
	Details string `json:"details"`
	Events  string `json:"events,omitempty"`
	JID     string `json:"jid,omitempty"`
	Webhook string `json:"webhook,omitempty"`
}

type SessionDisconnectData struct {
	Details string `json:"Details"`
}

type QRCodeData struct {
	QRCode    string `json:"qrCode"`
	ExpiresIn int    `json:"expiresIn"`
}

func ToSessionResponse(session *models.Session) *SessionResponse {
	if session == nil {
		return nil
	}

	return &SessionResponse{
		ID:          session.ID,
		Name:        session.Name,
		Phone:       session.Phone,
		Status:      session.Status,
		QRCode:      session.QRCode,
		ProxyHost:   session.ProxyHost,
		ProxyPort:   session.ProxyPort,
		ProxyType:   session.ProxyType,
		ProxyUser:   session.ProxyUser,
		ProxyPass:   session.ProxyPass,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
		ConnectedAt: session.ConnectedAt,
	}
}

func ToSessionResponseList(sessions []*models.Session) []*SessionResponse {
	if len(sessions) == 0 {
		return []*SessionResponse{}
	}

	responses := make([]*SessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = ToSessionResponse(session)
	}
	return responses
}
