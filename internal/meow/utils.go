package meow

import (
	"crypto/tls"
	"time"

	"github.com/go-resty/resty/v2"
	waLog "go.mau.fi/whatsmeow/util/log"

	"zpigo/internal/db/models"
	"zpigo/internal/logger"
)


func BuildCacheKey(apiKey, sessionID string) string {
	return apiKey + ":" + sessionID
}

func NewSessionInfoFromModel(session *models.Session, apiKey string) *SessionInfo {
	return &SessionInfo{
		ID:      session.ID,
		Name:    session.Name,
		Phone:   session.Phone,
		Status:  string(session.Status),
		QRCode:  session.QRCode,
		APIKey:  apiKey,
		JID:     "",
		Events:  "",
		Webhook: "",
		Proxy:   "",
	}
}

func (s *SessionInfo) ToModelSession() *models.Session {
	return &models.Session{
		ID:     s.ID,
		Name:   s.Name,
		Phone:  s.Phone,
		Status: models.SessionStatus(s.Status),
		QRCode: s.QRCode,
	}
}

func NewHTTPClient() *resty.Client {
	client := resty.New()
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(15))
	client.SetTimeout(30 * time.Second)
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	client.OnError(func(req *resty.Request, err error) {
		if v, ok := err.(*resty.ResponseError); ok {
			_ = v
		}
	})

	return client
}

func NewLoggerForComponent(component string) logger.Logger {
	return logger.NewForComponent(component)
}

func NewWhatsAppLogger(component, level string) waLog.Logger {
	return logger.NewWhatsAppLogger(component, level)
}


const (
	DefaultHTTPTimeout  = 30 * time.Second
	DefaultQRTimeout    = 30 * time.Second
	DefaultCacheExpiry  = 24 * time.Hour
	DefaultCacheCleanup = 1 * time.Hour

	DefaultMaxRetries = 3
	DefaultRetryDelay = 5 * time.Second

	DefaultWebhookTimeout = 10 * time.Second

	DefaultLogLevel      = "INFO"
	DefaultDebugLogLevel = "DEBUG"
)


type contextKey string

const (
	AuthContextKey contextKey = "auth"
)


func ValidateSessionID(sessionID string) bool {
	return sessionID != "" && len(sessionID) > 0
}

func ValidateAPIKey(apiKey string) bool {
	return apiKey != "" && len(apiKey) >= 8
}

func ValidateWebhookURL(url string) bool {
	return url != "" && (len(url) > 7) && (url[:7] == "http://" || url[:8] == "https://")
}


func StringPtr(s string) *string {
	return &s
}

func TimePtr(t time.Time) *time.Time {
	return &t
}

func BoolPtr(b bool) *bool {
	return &b
}


func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		return d.Round(time.Minute).String()
	}
	return d.Round(time.Hour).String()
}


func SafeClose(ch chan bool) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func SafeSend(ch chan bool, value bool) bool {
	select {
	case ch <- value:
		return true
	default:
		return false
	}
}
