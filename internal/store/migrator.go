package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"zpigo/internal/logger"
)

type Migrator struct {
	db     *sql.DB
	logger logger.Logger
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger.NewForComponent("migrator"),
	}
}

func (m *Migrator) RunMigrations(ctx context.Context, migrationsDir string) error {

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("erro ao criar tabela de migrations: %w", err)
	}

	upFiles, err := m.findMigrationFiles(migrationsDir, ".up.sql")
	if err != nil {
		return fmt.Errorf("erro ao buscar arquivos de migration: %w", err)
	}

	for _, file := range upFiles {
		migrationName := m.extractMigrationName(file)

		if executed, err := m.isMigrationExecuted(ctx, migrationName); err != nil {
			return fmt.Errorf("erro ao verificar migration %s: %w", migrationName, err)
		} else if executed {
			m.logger.Debug("Migration já executada", "migration", migrationName)
			continue
		}

		if err := m.executeMigrationFile(ctx, file, migrationName); err != nil {
			return fmt.Errorf("erro ao executar migration %s: %w", migrationName, err)
		}

		m.logger.Info("Migration aplicada", "migration", migrationName)
	}

	return nil
}

func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

func (m *Migrator) findMigrationFiles(migrationsDir, suffix string) ([]string, error) {
	var files []string

	err := filepath.Walk(migrationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, suffix) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func (m *Migrator) extractMigrationName(filePath string) string {
	fileName := filepath.Base(filePath)
	name := strings.TrimSuffix(fileName, ".up.sql")
	name = strings.TrimSuffix(name, ".down.sql")
	return name
}

func (m *Migrator) isMigrationExecuted(ctx context.Context, migrationName string) (bool, error) {
	query := `SELECT COUNT(*) FROM schema_migrations WHERE version = $1`

	var count int
	err := m.db.QueryRowContext(ctx, query, migrationName).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *Migrator) executeMigrationFile(ctx context.Context, filePath, migrationName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo %s: %w", filePath, err)
	}

	_, err = m.db.ExecContext(ctx, string(content))
	if err != nil {
		return fmt.Errorf("erro ao executar SQL do arquivo %s: %w", filePath, err)
	}

	insertQuery := `INSERT INTO schema_migrations (version) VALUES ($1)`
	_, err = m.db.ExecContext(ctx, insertQuery, migrationName)
	if err != nil {
		return fmt.Errorf("erro ao marcar migration como executada: %w", err)
	}

	return nil
}
