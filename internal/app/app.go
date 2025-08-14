package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mau.fi/whatsmeow/store/sqlstore"

	"zpigo/internal/api/router"
	"zpigo/internal/config"
	"zpigo/internal/db"
	"zpigo/internal/logger"
	"zpigo/internal/meow"
	"zpigo/internal/repository"
)

type App struct {
	config    *config.Config
	db        *db.DB
	repos     *repository.Repositories
	server    *http.Server
	container *sqlstore.Container
}

func New() (*App, error) {
	logger.Init(logger.Config{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	})

	log := logger.WithComponent("app")
	log.Info("🚀 Iniciando aplicação ZPigo")

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	database, err := db.NewConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar ao banco: %w", err)
	}

	repos := repository.NewRepositories(database)

	log.Info("🔄 Executando migrações do banco de dados")
	if err := repos.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("erro ao executar migrações: %w", err)
	}
	log.Info("✅ Migrações executadas com sucesso")

	container, err := meow.NewWhatsAppStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar container WhatsApp: %w", err)
	}

	handler := router.NewRouter(repos, container)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &App{
		config:    cfg,
		db:        database,
		repos:     repos,
		server:    server,
		container: container,
	}, nil
}

func (a *App) Run() error {
	appLogger := logger.WithComponent("server")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		appLogger.Info("🚀 Servidor iniciado", "porta", a.config.Server.Port)
		appLogger.Info("📖 Documentação da API", "url", fmt.Sprintf("http://localhost:%d/health", a.config.Server.Port))

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

	log.Println("✅ Servidor parado com sucesso")
	return nil
}

func (a *App) Close() error {
	if a.repos != nil {
		if err := a.repos.Close(); err != nil {
			log.Printf("Erro ao fechar repositories: %v", err)
		}
	}

	if a.container != nil {
		if err := a.container.Close(); err != nil {
			log.Printf("Erro ao fechar container WhatsApp: %v", err)
		}
	}

	return nil
}
