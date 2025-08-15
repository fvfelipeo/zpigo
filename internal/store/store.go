package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"

	"zpigo/internal/config"
	"zpigo/internal/logger"
	"zpigo/internal/store/repositories"
)

// Store é o store principal que gerencia conexões e repositórios
type Store struct {
	db        *sql.DB
	container *sqlstore.Container
	config    *config.Config
	logger    logger.Logger

	sessionRepo SessionRepositoryInterface
	webhookRepo WebhookRepositoryInterface
}

// NewStore cria uma nova instância do store
func NewStore(cfg *config.Config) (*Store, error) {
	log := logger.WithComponent("unified-store")

	// Conectar ao banco
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão SQL: %w", err)
	}

	// Configurar pool de conexões
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao testar conexão SQL: %w", err)
	}

	// Criar container WhatsApp
	waLogger := logger.ForWhatsApp("store")
	container := sqlstore.NewWithDB(db, "postgres", waLogger)

	if err := container.Upgrade(context.Background()); err != nil {
		return nil, fmt.Errorf("erro ao executar upgrade do whatsmeow: %w", err)
	}

	// Criar store
	store := &Store{
		db:          db,
		container:   container,
		config:      cfg,
		logger:      log,
		sessionRepo: repositories.NewSessionRepository(db),
		webhookRepo: repositories.NewWebhookRepository(db),
	}

	// Criar tabelas da aplicação
	if err := store.createAppTables(context.Background()); err != nil {
		return nil, fmt.Errorf("erro ao criar tabelas da aplicação: %w", err)
	}

	log.Info("Store inicializado")
	return store, nil
}

// GetDB retorna a conexão de banco
func (s *Store) GetDB() *sql.DB {
	return s.db
}

// GetContainer retorna o container do WhatsApp
func (s *Store) GetContainer() *sqlstore.Container {
	return s.container
}

// GetSessionRepository retorna o repositório de sessões
func (s *Store) GetSessionRepository() SessionRepositoryInterface {
	return s.sessionRepo
}

// GetWebhookRepository retorna o repositório de webhooks
func (s *Store) GetWebhookRepository() WebhookRepositoryInterface {
	return s.webhookRepo
}

// Close fecha as conexões
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// createAppTables cria as tabelas da aplicação
func (s *Store) createAppTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			phone VARCHAR(20),
			status VARCHAR(50) NOT NULL DEFAULT 'disconnected',
			qrcode TEXT,
			devicejid VARCHAR(255),
			proxyhost VARCHAR(255),
			proxyport INTEGER,
			proxytype VARCHAR(20),
			proxyuser VARCHAR(255),
			proxypass VARCHAR(255),
			createdat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updatedat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			connectedat TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS webhooks (
			id VARCHAR(255) PRIMARY KEY,
			sessionid VARCHAR(255) NOT NULL,
			url VARCHAR(500) NOT NULL,
			events TEXT,
			createdat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updatedat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (sessionid) REFERENCES sessions(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_devicejid ON sessions(devicejid)`,
		`CREATE INDEX IF NOT EXISTS idx_webhooks_sessionid ON webhooks(sessionid)`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("erro ao executar query: %s - %w", query, err)
		}
	}

	return nil
}
