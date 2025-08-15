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

type SessionRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{
		db:     db,
		logger: logger.NewForComponent("session-repo"),
	}
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

	query := `
		INSERT INTO sessions (id, name, phone, status, qrcode, devicejid, 
			proxyhost, proxyport, proxytype, proxyuser, proxypass, 
			createdat, updatedat, connectedat)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.Name, session.Phone, session.Status, session.QRCode,
		session.DeviceJid, session.ProxyHost, session.ProxyPort, session.ProxyType,
		session.ProxyUser, session.ProxyPass, session.CreatedAt, session.UpdatedAt,
		session.ConnectedAt,
	)

	return err
}

func (r *SessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	session := &models.Session{}
	query := `
		SELECT id, name, phone, status, qrcode, devicejid, proxyhost, proxyport,
			proxytype, proxyuser, proxypass, createdat, updatedat, connectedat
		FROM sessions WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID, &session.Name, &session.Phone, &session.Status, &session.QRCode,
		&session.DeviceJid, &session.ProxyHost, &session.ProxyPort, &session.ProxyType,
		&session.ProxyUser, &session.ProxyPass, &session.CreatedAt, &session.UpdatedAt,
		&session.ConnectedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sessão não encontrada")
		}
		return nil, err
	}

	return session, nil
}

func (r *SessionRepository) List(ctx context.Context) ([]*models.Session, error) {
	query := `
		SELECT id, name, phone, status, qrcode, devicejid, proxyhost, proxyport,
			proxytype, proxyuser, proxypass, createdat, updatedat, connectedat
		FROM sessions ORDER BY createdat DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.Session
	for rows.Next() {
		session := &models.Session{}
		err := rows.Scan(
			&session.ID, &session.Name, &session.Phone, &session.Status, &session.QRCode,
			&session.DeviceJid, &session.ProxyHost, &session.ProxyPort, &session.ProxyType,
			&session.ProxyUser, &session.ProxyPass, &session.CreatedAt, &session.UpdatedAt,
			&session.ConnectedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (r *SessionRepository) Update(ctx context.Context, session *models.Session) error {
	session.UpdatedAt = time.Now()

	query := `
		UPDATE sessions
		SET name = $2, phone = $3, status = $4, qrcode = $5, devicejid = $6,
		    proxyhost = $7, proxyport = $8, proxytype = $9, proxyuser = $10, proxypass = $11,
		    updatedat = $12, connectedat = $13
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		session.ID, session.Name, session.Phone, session.Status, session.QRCode,
		session.DeviceJid, session.ProxyHost, session.ProxyPort, session.ProxyType,
		session.ProxyUser, session.ProxyPass, session.UpdatedAt, session.ConnectedAt,
	)

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
	query := `DELETE FROM sessions WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
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
	query := `UPDATE sessions SET status = $2, updatedat = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, status, time.Now())
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
	query := `UPDATE sessions SET qrcode = $2, updatedat = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, qrCode, time.Now())
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
	query := `
		UPDATE sessions
		SET status = $2, phone = $3, devicejid = $4, connectedat = $5, updatedat = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, models.StatusConnected, phone, deviceJid, now, now)
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
	query := `UPDATE sessions SET status = $2, updatedat = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, models.StatusDisconnected, time.Now())
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
	query := `
		UPDATE sessions
		SET proxyhost = $2, proxyport = $3, proxytype = $4, proxyuser = $5, proxypass = $6, updatedat = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, proxyHost, proxyPort, proxyType, proxyUser, proxyPass, time.Now())
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
	query := `UPDATE sessions SET devicejid = $2, updatedat = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, deviceJid, time.Now())
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

func (r *SessionRepository) GetAll(ctx context.Context) ([]models.Session, error) {
	sessions, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []models.Session
	for _, session := range sessions {
		result = append(result, *session)
	}

	return result, nil
}
