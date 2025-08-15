package store

import (
	"context"

	"zpigo/internal/store/models"
)

// SessionRepositoryInterface define as operações para sessões
type SessionRepositoryInterface interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, id string) (*models.Session, error)
	List(ctx context.Context) ([]*models.Session, error)
	Update(ctx context.Context, session *models.Session) error
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status models.SessionStatus) error
	UpdateQRCode(ctx context.Context, id string, qrCode string) error
	SetConnected(ctx context.Context, id string, phone string, deviceJid string) error
	SetDisconnected(ctx context.Context, id string) error
	UpdateProxy(ctx context.Context, id string, proxyHost string, proxyPort int, proxyType models.ProxyType, proxyUser, proxyPass string) error
	UpdateDeviceJid(ctx context.Context, id string, deviceJid string) error
	GetAll(ctx context.Context) ([]models.Session, error)
}

// WebhookRepositoryInterface define as operações para webhooks
type WebhookRepositoryInterface interface {
	Create(ctx context.Context, webhook *models.Webhook) error
	GetByID(ctx context.Context, id string) (*models.Webhook, error)
	GetBySessionID(ctx context.Context, sessionID string) ([]*models.Webhook, error)
	List(ctx context.Context) ([]*models.Webhook, error)
	Update(ctx context.Context, webhook *models.Webhook) error
	Delete(ctx context.Context, id string) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
}
