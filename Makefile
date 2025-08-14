##############################################################################
# ZPigo - Makefile
##############################################################################

# Carrega variáveis do arquivo .env se existir
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help dev prod build test clean setup run

##############################################################################
# Ajuda
##############################################################################
help: ## Mostra esta ajuda
	@echo "Comandos disponíveis:"
	@echo "  \033[36mbuild\033[0m        Compila a aplicação"
	@echo "  \033[36mclean\033[0m        Remove volumes e para ambiente"
	@echo "  \033[36mdeps\033[0m         Instala/atualiza dependências"
	@echo "  \033[36mdev\033[0m          Inicia ambiente de desenvolvimento"
	@echo "  \033[36mfmt\033[0m          Formata código Go"
	@echo "  \033[36mhelp\033[0m         Mostra esta ajuda"
	@echo "  \033[36mprod\033[0m         Inicia ambiente de produção"
	@echo "  \033[36mrun\033[0m          Executa aplicação localmente"
	@echo "  \033[36msetup\033[0m        Configuração inicial do projeto"
	@echo "  \033[36mstop\033[0m         Para ambiente atual"
	@echo "  \033[36mtest\033[0m         Executa testes"

##############################################################################
# Ambiente
##############################################################################
dev: setup ## Inicia ambiente de desenvolvimento
	docker-compose -f docker-compose.dev.yml up -d

prod: setup ## Inicia ambiente de produção
	docker-compose up -d

stop: ## Para ambiente atual
	docker-compose -f docker-compose.dev.yml down --remove-orphans 2>/dev/null || docker-compose down --remove-orphans

clean: ## Remove volumes e para ambiente
	docker-compose -f docker-compose.dev.yml down -v --remove-orphans 2>/dev/null || docker-compose down -v --remove-orphans

##############################################################################
# Aplicação
##############################################################################
run: ## Executa aplicação localmente
	go run ./cmd/server

build: ## Compila a aplicação
	go build -o bin/zpigo ./cmd/server

test: ## Executa testes
	go test -v ./...

##############################################################################
# Utilitários
##############################################################################
setup: ## Configuração inicial do projeto
	@if [ ! -f .env ]; then cp .env.example .env; echo "📝 Arquivo .env criado"; fi

fmt: ## Formata código Go
	go fmt ./...

deps: ## Instala/atualiza dependências
	go mod tidy

##############################################################################
# Default
##############################################################################
.DEFAULT_GOAL := help
