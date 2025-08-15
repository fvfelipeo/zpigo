package models

import (
	"time"
)

type SessionStatus string

const (
	StatusDisconnected SessionStatus = "disconnected"
	StatusConnecting   SessionStatus = "connecting"
	StatusConnected    SessionStatus = "connected"
)

type ProxyType string

const (
	ProxyHTTP   ProxyType = "http"
	ProxySOCKS5 ProxyType = "socks5"
)

type Session struct {
	ID        string        `json:"id" db:"id"`
	Name      string        `json:"name" db:"name"`
	Phone     string        `json:"phone,omitempty" db:"phone"`
	Status    SessionStatus `json:"status" db:"status"`
	QRCode    string        `json:"qrCode,omitempty" db:"qrcode"`
	DeviceJid string        `json:"deviceJid,omitempty" db:"devicejid"`

	ProxyHost string    `json:"proxyHost,omitempty" db:"proxyhost"`
	ProxyPort int       `json:"proxyPort,omitempty" db:"proxyport"`
	ProxyType ProxyType `json:"proxyType,omitempty" db:"proxytype"`
	ProxyUser string    `json:"proxyUser,omitempty" db:"proxyuser"`
	ProxyPass string    `json:"proxyPass,omitempty" db:"proxypass"`

	CreatedAt   time.Time  `json:"createdAt" db:"createdat"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updatedat"`
	ConnectedAt *time.Time `json:"connectedAt,omitempty" db:"connectedat"`

	Webhooks []*Webhook `json:"webhooks,omitempty"`
}

func (Session) TableName() string {
	return "sessions"
}

func (s *Session) IsConnected() bool {
	return s.Status == StatusConnected
}

func (s *Session) HasProxy() bool {
	return s.ProxyHost != "" && s.ProxyPort > 0
}

func (s *Session) GetProxyURL() string {
	if !s.HasProxy() {
		return ""
	}

	protocol := string(s.ProxyType)
	if protocol == "" {
		protocol = "http"
	}

	if s.ProxyUser != "" && s.ProxyPass != "" {
		return protocol + "://" + s.ProxyUser + ":" + s.ProxyPass + "@" + s.ProxyHost + ":" + string(rune(s.ProxyPort))
	}

	return protocol + "://" + s.ProxyHost + ":" + string(rune(s.ProxyPort))
}

func (s *Session) SetConnected() {
	s.Status = StatusConnected
	now := time.Now()
	s.ConnectedAt = &now
	s.UpdatedAt = now
}

func (s *Session) SetDisconnected() {
	s.Status = StatusDisconnected
	s.QRCode = ""
	s.UpdatedAt = time.Now()
}
