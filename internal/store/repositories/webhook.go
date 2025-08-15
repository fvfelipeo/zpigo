package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"zpigo/internal/logger"
	"zpigo/internal/store/models"
)

type WebhookRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewWebhookRepository(db *sql.DB) *WebhookRepository {
	return &WebhookRepository{
		db:     db,
		logger: logger.NewForComponent("webhook-repo"),
	}
}

func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	now := time.Now()
	webhook.CreatedAt = now
	webhook.UpdatedAt = now

	query := `
		INSERT INTO webhooks (id, sessionid, url, events, createdat, updatedat)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		webhook.ID, webhook.SessionID, webhook.URL, webhook.Events,
		webhook.CreatedAt, webhook.UpdatedAt,
	)

	return err
}

func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*models.Webhook, error) {
	webhook := &models.Webhook{}
	query := `
		SELECT id, sessionid, url, events, createdat, updatedat
		FROM webhooks WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&webhook.ID, &webhook.SessionID, &webhook.URL, &webhook.Events,
		&webhook.CreatedAt, &webhook.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("webhook não encontrado")
		}
		return nil, err
	}

	return webhook, nil
}

func (r *WebhookRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*models.Webhook, error) {
	query := `
		SELECT id, sessionid, url, events, createdat, updatedat
		FROM webhooks WHERE sessionid = $1 ORDER BY createdat DESC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		webhook := &models.Webhook{}
		err := rows.Scan(
			&webhook.ID, &webhook.SessionID, &webhook.URL, &webhook.Events,
			&webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, rows.Err()
}

func (r *WebhookRepository) List(ctx context.Context) ([]*models.Webhook, error) {
	query := `
		SELECT id, sessionid, url, events, createdat, updatedat
		FROM webhooks ORDER BY createdat DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		webhook := &models.Webhook{}
		err := rows.Scan(
			&webhook.ID, &webhook.SessionID, &webhook.URL, &webhook.Events,
			&webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, rows.Err()
}

func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	webhook.UpdatedAt = time.Now()

	query := `
		UPDATE webhooks
		SET sessionid = $2, url = $3, events = $4, updatedat = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		webhook.ID, webhook.SessionID, webhook.URL, webhook.Events, webhook.UpdatedAt,
	)

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
	query := `DELETE FROM webhooks WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
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
	query := `DELETE FROM webhooks WHERE sessionid = $1`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	return err
}
