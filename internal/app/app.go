package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zpigo/internal/api/router"
	"zpigo/internal/config"
	"zpigo/internal/logger"
	"zpigo/internal/store"
)

type App struct {
	config *config.Config
	store  *store.UnifiedStore
	server *http.Server
}

func New() (*App, error) {
	logger.Init(logger.Config{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	})

	log := logger.WithComponent("app")
	log.Info("Iniciando aplicação")

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	unifiedStore, err := store.NewUnifiedStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar store unificado: %w", err)
	}

	migrator := store.NewMigrator(unifiedStore.GetDB())
	ctx := context.Background()
	migrationsDir := "internal/store/migrations"

	if err := migrator.RunMigrations(ctx, migrationsDir); err != nil {
		return nil, fmt.Errorf("erro ao executar migrações: %w", err)
	}

	handler := router.NewRouter(unifiedStore)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &App{
		config: cfg,
		store:  unifiedStore,
		server: server,
	}, nil
}

func (a *App) Run() error {
	appLogger := logger.WithComponent("server")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		appLogger.Info("Servidor iniciado", "porta", a.config.Server.Port, "health", fmt.Sprintf("http://localhost:%d/health", a.config.Server.Port))

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Erro ao iniciar servidor", "error", err)
		}
	}()

	<-quit
	appLogger.Info("🛑 Parando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		appLogger.Error("Erro ao parar servidor", "error", err)
		return err
	}

	logger.Info("Servidor parado")
	return nil
}

func (a *App) Close() error {
	if a.store != nil {
		if err := a.store.Close(); err != nil {
			logger.Error("Erro ao fechar store unificado", "error", err)
		}
	}

	return nil
}
