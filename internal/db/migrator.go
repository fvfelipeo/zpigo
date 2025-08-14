package db

import (
	"context"
	"fmt"
	"log"

	"github.com/uptrace/bun"

	"zpigo/internal/db/models"
)

type Migrator struct {
	db *bun.DB
}

func NewMigrator(db *bun.DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) AutoMigrate(ctx context.Context) error {
	log.Println("🔄 Iniciando migrações automáticas com modelos Bun...")

	models := []interface{}{
		(*models.Session)(nil),
		(*models.Webhook)(nil),
	}

	for _, model := range models {
		if err := m.createTableFromModel(ctx, model); err != nil {
			return fmt.Errorf("erro ao migrar modelo %T: %w", model, err)
		}
	}

	log.Println("✅ Migrações automáticas concluídas com sucesso")
	return nil
}

func (m *Migrator) createTableFromModel(ctx context.Context, model interface{}) error {
	tableName := m.getTableName(model)
	log.Printf("📋 Criando/verificando tabela: %s", tableName)

	_, err := m.db.NewCreateTable().
		Model(model).
		IfNotExists().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("falha ao criar tabela %s: %w", tableName, err)
	}

	log.Printf("✅ Tabela %s criada/verificada automaticamente", tableName)
	return nil
}

func (m *Migrator) getTableName(model interface{}) string {
	switch model.(type) {
	case *models.Session:
		return "sessions"
	case *models.Webhook:
		return "webhooks"
	default:
		return "unknown"
	}
}

func (m *Migrator) DropAllTables(ctx context.Context) error {
	log.Println("🗑️  ATENÇÃO: Removendo todas as tabelas...")

	models := []interface{}{
		(*models.Webhook)(nil),
		(*models.Session)(nil),
	}

	for _, model := range models {
		tableName := m.getTableName(model)
		_, err := m.db.NewDropTable().
			Model(model).
			IfExists().
			Cascade().
			Exec(ctx)

		if err != nil {
			log.Printf("⚠️  Erro ao remover tabela %s: %v", tableName, err)
		} else {
			log.Printf("🗑️  Tabela %s removida", tableName)
		}
	}

	return nil
}
