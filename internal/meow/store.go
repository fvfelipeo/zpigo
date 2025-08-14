package meow

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"

	"zpigo/internal/config"
	"zpigo/internal/logger"
)

func NewWhatsAppStore(cfg *config.Config) (*sqlstore.Container, error) {
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão SQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao testar conexão SQL: %w", err)
	}

	waLogger := logger.NewWhatsAppLogger("store", "INFO")

	container := sqlstore.NewWithDB(db, "postgres", waLogger)

	if err := container.Upgrade(context.Background()); err != nil {
		return nil, fmt.Errorf("erro ao executar upgrade do whatsmeow: %w", err)
	}

	return container, nil
}
