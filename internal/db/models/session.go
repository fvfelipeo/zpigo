package models

import (
	"time"

	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:sessions,alias:s"`

	ID        string        `json:"id" bun:"id,pk,type:varchar(255)"`
	Name      string        `json:"name" bun:"name,notnull,type:varchar(255)"`
	Phone     string        `json:"phone,omitempty" bun:"phone,type:varchar(20)"`
	Status    SessionStatus `json:"status" bun:"status,notnull,default:'disconnected',type:varchar(50)"`
	QRCode    string        `json:"qrCode,omitempty" bun:"qrCode,type:text"`
	DeviceJid string        `json:"deviceJid,omitempty" bun:"\"deviceJid\",type:varchar(255)"`

	ProxyHost string    `json:"proxyHost,omitempty" bun:"proxyHost,type:varchar(255)"`
	ProxyPort int       `json:"proxyPort,omitempty" bun:"proxyPort,type:integer"`
	ProxyType ProxyType `json:"proxyType,omitempty" bun:"proxyType,type:varchar(20)"`
	ProxyUser string    `json:"proxyUser,omitempty" bun:"proxyUser,type:varchar(255)"`
	ProxyPass string    `json:"proxyPass,omitempty" bun:"proxyPass,type:varchar(255)"`

	CreatedAt   time.Time  `json:"createdAt" bun:"createdAt,nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `json:"updatedAt" bun:"updatedAt,nullzero,notnull,default:current_timestamp"`
	ConnectedAt *time.Time `json:"connectedAt,omitempty" bun:"connectedAt,nullzero"`

	Webhooks []*Webhook `json:"webhooks,omitempty" bun:"rel:has-many,join:id=sessionId"`
}

func (Session) TableName() string {
	return "sessions"
}

func (s *Session) BeforeAppendModel(query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
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
