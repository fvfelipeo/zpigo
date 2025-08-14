package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"zpigo/internal/db/models"
)

type WebhookRepositoryInterface interface {
	Create(ctx context.Context, webhook *models.Webhook) error
	GetByID(ctx context.Context, id string) (*models.Webhook, error)
	GetBySessionID(ctx context.Context, sessionID string) ([]*models.Webhook, error)
	List(ctx context.Context) ([]*models.Webhook, error)
	Update(ctx context.Context, webhook *models.Webhook) error
	Delete(ctx context.Context, id string) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
}

type WebhookRepository struct {
	db *bun.DB
}

func NewWebhookRepository(db *bun.DB) *WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	now := time.Now()
	webhook.CreatedAt = now
	webhook.UpdatedAt = now

	_, err := r.db.NewInsert().Model(webhook).Exec(ctx)
	return err
}

func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*models.Webhook, error) {
	webhook := &models.Webhook{}
	err := r.db.NewSelect().Model(webhook).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("webhook não encontrado")
		}
		return nil, err
	}
	return webhook, nil
}

func (r *WebhookRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*models.Webhook, error) {
	var webhooks []*models.Webhook
	err := r.db.NewSelect().
		Model(&webhooks).
		Where("sessionId = ?", sessionID).
		Order("createdAt DESC").
		Scan(ctx)
	return webhooks, err
}

func (r *WebhookRepository) List(ctx context.Context) ([]*models.Webhook, error) {
	var webhooks []*models.Webhook
	err := r.db.NewSelect().
		Model(&webhooks).
		Relation("Session").
		Order("createdAt DESC").
		Scan(ctx)
	return webhooks, err
}

func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	webhook.UpdatedAt = time.Now()

	result, err := r.db.NewUpdate().
		Model(webhook).
		Where("id = ?", webhook.ID).
		Exec(ctx)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("webhook não encontrado")
	}

	return nil
}

func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.NewDelete().
		Model((*models.Webhook)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("webhook não encontrado")
	}

	return nil
}

func (r *WebhookRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	_, err := r.db.NewDelete().
		Model((*models.Webhook)(nil)).
		Where("sessionId = ?", sessionID).
		Exec(ctx)

	return err
}
