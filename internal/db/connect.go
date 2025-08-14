package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"

	"zpigo/internal/config"
)

type DB struct {
	*bun.DB
	config *config.Config
}

func NewConnection(cfg *config.Config) (*DB, error) {
	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Database.DSN)))

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	db := bun.NewDB(sqlDB, pgdialect.New())

	if cfg.IsDevelopment() || cfg.App.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("✅ Successfully connected to PostgreSQL database")

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

func (db *DB) GetConfig() *config.Config {
	return db.config
}

func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

func (db *DB) Reset(ctx context.Context) error {
	if !db.config.IsDevelopment() {
		return fmt.Errorf("reset only allowed in development environment")
	}

	tables := []string{
		"sessions",
	}

	for _, table := range tables {
		_, err := db.NewTruncateTable().Table(table).Cascade().Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	fmt.Println("✅ Database reset completed")
	return nil
}

func (db *DB) GetStats() sql.DBStats {
	return db.DB.DB.Stats()
}

func (db *DB) Transaction(ctx context.Context, fn func(ctx context.Context, tx bun.Tx) error) error {
	return db.DB.RunInTx(ctx, nil, fn)
}

func (db *DB) NewMigrator(bunDB *bun.DB) *Migrator {
	return NewMigrator(bunDB)
}
