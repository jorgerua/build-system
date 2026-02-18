.PHONY: help build test run clean install deps lint format check-deps

# Variáveis
NPM := npm
NX := $(NPM) exec nx
GO := go
DOCKER_COMPOSE := docker-compose

# Cores para output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Mostra esta mensagem de ajuda
	@echo "$(GREEN)OCI Build System - Comandos disponíveis:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

# Instalação e Dependências
install: ## Instala todas as dependências (npm e go)
	@echo "$(GREEN)Instalando dependências npm...$(NC)"
	$(NPM) install
	@echo "$(GREEN)Instalando dependências Go...$(NC)"
	cd libs/shared && $(GO) mod download
	@echo "$(GREEN)Dependências instaladas com sucesso!$(NC)"

deps: install ## Alias para install

check-deps: ## Verifica se as dependências estão instaladas
	@command -v node >/dev/null 2>&1 || { echo "$(RED)Node.js não está instalado!$(NC)"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go não está instalado!$(NC)"; exit 1; }
	@echo "$(GREEN)Todas as dependências necessárias estão instaladas!$(NC)"

# Build
build: ## Executa build de todos os projetos usando NX
	@echo "$(GREEN)Executando build de todos os projetos...$(NC)"
	$(NX) run-many --target=build --all --parallel=3
	@echo "$(GREEN)Build concluído com sucesso!$(NC)"

build-affected: ## Executa build apenas dos projetos afetados
	@echo "$(GREEN)Executando build dos projetos afetados...$(NC)"
	$(NX) affected --target=build --parallel=3
	@echo "$(GREEN)Build dos projetos afetados concluído!$(NC)"

build-shared: ## Executa build da biblioteca shared
	@echo "$(GREEN)Executando build da biblioteca shared...$(NC)"
	$(NX) run shared:build
	@echo "$(GREEN)Build da shared concluído!$(NC)"

build-api: ## Executa build do api-service
	@echo "$(GREEN)Executando build do api-service...$(NC)"
	$(NX) run api-service:build
	@echo "$(GREEN)Build do api-service concluído!$(NC)"

build-worker: ## Executa build do worker-service
	@echo "$(GREEN)Executando build do worker-service...$(NC)"
	$(NX) run worker-service:build
	@echo "$(GREEN)Build do worker-service concluído!$(NC)"

# Testes
test: ## Executa testes de todos os projetos usando NX
	@echo "$(GREEN)Executando testes de todos os projetos...$(NC)"
	$(NX) run-many --target=test --all --parallel=3
	@echo "$(GREEN)Testes concluídos!$(NC)"

test-affected: ## Executa testes apenas dos projetos afetados
	@echo "$(GREEN)Executando testes dos projetos afetados...$(NC)"
	$(NX) affected --target=test --parallel=3
	@echo "$(GREEN)Testes dos projetos afetados concluídos!$(NC)"

test-shared: ## Executa testes da biblioteca shared
	@echo "$(GREEN)Executando testes da biblioteca shared...$(NC)"
	$(NX) run shared:test
	@echo "$(GREEN)Testes da shared concluídos!$(NC)"

test-api: ## Executa testes do api-service
	@echo "$(GREEN)Executando testes do api-service...$(NC)"
	$(NX) run api-service:test
	@echo "$(GREEN)Testes do api-service concluídos!$(NC)"

test-worker: ## Executa testes do worker-service
	@echo "$(GREEN)Executando testes do worker-service...$(NC)"
	$(NX) run worker-service:test
	@echo "$(GREEN)Testes do worker-service concluídos!$(NC)"

test-coverage: ## Executa testes com cobertura de código
	@echo "$(GREEN)Executando testes com cobertura...$(NC)"
	$(NX) run-many --target=test --all --parallel=3 --coverage
	@echo "$(GREEN)Testes com cobertura concluídos!$(NC)"

# Validação
validate-env: ## Valida que docker-compose falha sem variáveis obrigatórias
	@echo "$(GREEN)Validando configuração de variáveis de ambiente...$(NC)"
	@bash tests/validate-compose-env.sh
	@echo "$(GREEN)Validação concluída com sucesso!$(NC)"

# Execução
run: ## Inicia todos os serviços usando docker-compose
	@echo "$(GREEN)Iniciando todos os serviços...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Serviços iniciados! Use 'make logs' para ver os logs.$(NC)"

run-api: ## Executa o api-service localmente
	@echo "$(GREEN)Executando api-service...$(NC)"
	$(NX) run api-service:serve

run-worker: ## Executa o worker-service localmente
	@echo "$(GREEN)Executando worker-service...$(NC)"
	$(NX) run worker-service:serve

run-nats: ## Inicia apenas o NATS usando docker-compose
	@echo "$(GREEN)Iniciando NATS...$(NC)"
	$(DOCKER_COMPOSE) up -d nats
	@echo "$(GREEN)NATS iniciado!$(NC)"

stop: ## Para todos os serviços docker-compose
	@echo "$(YELLOW)Parando todos os serviços...$(NC)"
	$(DOCKER_COMPOSE) down
	@echo "$(GREEN)Serviços parados!$(NC)"

restart: stop run ## Reinicia todos os serviços

logs: ## Mostra logs de todos os serviços
	$(DOCKER_COMPOSE) logs -f

logs-api: ## Mostra logs do api-service
	$(DOCKER_COMPOSE) logs -f api-service

logs-worker: ## Mostra logs do worker-service
	$(DOCKER_COMPOSE) logs -f worker-service

logs-nats: ## Mostra logs do NATS
	$(DOCKER_COMPOSE) logs -f nats

# Linting e Formatação
lint: ## Executa linting em todos os projetos
	@echo "$(GREEN)Executando linting...$(NC)"
	$(NX) run-many --target=lint --all --parallel=3
	@echo "$(GREEN)Linting concluído!$(NC)"

format: ## Formata o código Go
	@echo "$(GREEN)Formatando código Go...$(NC)"
	@find . -name "*.go" -not -path "./node_modules/*" -exec gofmt -w {} \;
	@echo "$(GREEN)Código formatado!$(NC)"

# Limpeza
clean: ## Remove arquivos de build e cache
	@echo "$(YELLOW)Limpando arquivos de build e cache...$(NC)"
	$(NX) reset
	rm -rf dist
	rm -rf .nx/cache
	@find . -name "*.test" -type f -delete
	@echo "$(GREEN)Limpeza concluída!$(NC)"

clean-all: clean ## Remove tudo incluindo node_modules e go cache
	@echo "$(YELLOW)Limpando tudo...$(NC)"
	rm -rf node_modules
	rm -rf libs/shared/go.sum
	@echo "$(GREEN)Limpeza completa concluída!$(NC)"

# Desenvolvimento
dev: run-nats ## Inicia ambiente de desenvolvimento (NATS + serviços locais)
	@echo "$(GREEN)Ambiente de desenvolvimento pronto!$(NC)"
	@echo "$(YELLOW)Execute 'make run-api' e 'make run-worker' em terminais separados$(NC)"

graph: ## Mostra o gráfico de dependências do NX
	@echo "$(GREEN)Abrindo gráfico de dependências...$(NC)"
	$(NX) graph

affected-graph: ## Mostra o gráfico de projetos afetados
	@echo "$(GREEN)Abrindo gráfico de projetos afetados...$(NC)"
	$(NX) affected:graph

# CI/CD
ci: check-deps install build test ## Executa pipeline completo de CI
	@echo "$(GREEN)Pipeline de CI concluído com sucesso!$(NC)"

# Status
status: ## Mostra status dos serviços docker-compose
	$(DOCKER_COMPOSE) ps

# Default target
.DEFAULT_GOAL := help
