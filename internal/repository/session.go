package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"zpigo/internal/db"
	"zpigo/internal/db/models"
)

type Repositories struct {
	Session SessionRepositoryInterface
	Webhook WebhookRepositoryInterface
	db      *db.DB
}

func NewRepositories(database *db.DB) *Repositories {
	return &Repositories{
		Session: NewSessionRepository(database.DB),
		Webhook: NewWebhookRepository(database.DB),
		db:      database,
	}
}

func (r *Repositories) GetDB() *bun.DB {
	return r.db.DB
}

func (r *Repositories) Migrate(ctx context.Context) error {
	migrator := r.db.NewMigrator(r.db.DB)

	return migrator.AutoMigrate(ctx)
}

func (r *Repositories) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

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

type SessionRepository struct {
	db *bun.DB
}

func NewSessionRepository(db *bun.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	if session.Status == "" {
		session.Status = models.StatusDisconnected
	}

	_, err := r.db.NewInsert().Model(session).Exec(ctx)
	return err
}

func (r *SessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	session := &models.Session{}
	err := r.db.NewSelect().Model(session).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sessão não encontrada")
		}
		return nil, err
	}
	return session, nil
}

func (r *SessionRepository) List(ctx context.Context) ([]*models.Session, error) {
	var sessions []*models.Session
	err := r.db.NewSelect().Model(&sessions).Order("createdAt DESC").Scan(ctx)
	return sessions, err
}

func (r *SessionRepository) Update(ctx context.Context, session *models.Session) error {
	session.UpdatedAt = time.Now()

	result, err := r.db.NewUpdate().
		Model(session).
		Where("id = ?", session.ID).
		Exec(ctx)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sessão não encontrada")
	}

	return nil
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.NewDelete().
		Model((*models.Session)(nil)).
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
		return fmt.Errorf("sessão não encontrada")
	}

	return nil
}

func (r *SessionRepository) UpdateStatus(ctx context.Context, id string, status models.SessionStatus) error {
	session := &models.Session{
		ID:        id,
		Status:    status,
		UpdatedAt: time.Now(),
	}

	result, err := r.db.NewUpdate().
		Model(session).
		Column("status", "updatedAt").
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
		return fmt.Errorf("sessão não encontrada")
	}

	return nil
}

func (r *SessionRepository) UpdateQRCode(ctx context.Context, id string, qrCode string) error {
	session := &models.Session{
		ID:        id,
		QRCode:    qrCode,
		UpdatedAt: time.Now(),
	}

	result, err := r.db.NewUpdate().
		Model(session).
		Column("qrCode", "updatedAt").
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
		return fmt.Errorf("sessão não encontrada")
	}

	return nil
}

func (r *SessionRepository) SetConnected(ctx context.Context, id string, phone string, deviceJid string) error {
	now := time.Now()
	session := &models.Session{
		ID:          id,
		Status:      models.StatusConnected,
		Phone:       phone,
		DeviceJid:   deviceJid,
		ConnectedAt: &now,
		UpdatedAt:   now,
	}

	result, err := r.db.NewUpdate().
		Model(session).
		Column("status", "phone", "deviceJid", "connectedAt", "updatedAt").
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
		return fmt.Errorf("sessão não encontrada")
	}

	return nil
}

func (r *SessionRepository) SetDisconnected(ctx context.Context, id string) error {
	session := &models.Session{
		ID:        id,
		Status:    models.StatusDisconnected,
		QRCode:    "",
		Phone:     "",
		DeviceJid: "",
		UpdatedAt: time.Now(),
	}

	result, err := r.db.NewUpdate().
		Model(session).
		Column("status", "qrCode", "phone", "deviceJid", "updatedAt").
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
		return fmt.Errorf("sessão não encontrada")
	}

	return nil
}

func (r *SessionRepository) UpdateProxy(ctx context.Context, id string, proxyHost string, proxyPort int, proxyType models.ProxyType, proxyUser, proxyPass string) error {
	session := &models.Session{
		ID:        id,
		ProxyHost: proxyHost,
		ProxyPort: proxyPort,
		ProxyType: proxyType,
		ProxyUser: proxyUser,
		ProxyPass: proxyPass,
		UpdatedAt: time.Now(),
	}

	result, err := r.db.NewUpdate().
		Model(session).
		Column("proxyHost", "proxyPort", "proxyType", "proxyUser", "proxyPass", "updatedAt").
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
		return fmt.Errorf("sessão não encontrada")
	}

	return nil
}

func (r *SessionRepository) UpdateDeviceJid(ctx context.Context, id string, deviceJid string) error {
	session := &models.Session{
		ID:        id,
		DeviceJid: deviceJid,
		UpdatedAt: time.Now(),
	}

	result, err := r.db.NewUpdate().
		Model(session).
		Column("deviceJid", "updatedAt").
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update device jid: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

func (r *SessionRepository) GetAll(ctx context.Context) ([]models.Session, error) {
	var sessions []models.Session

	err := r.db.NewSelect().
		Model(&sessions).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get all sessions: %w", err)
	}

	return sessions, nil
}
