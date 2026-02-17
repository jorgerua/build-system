# Design Document: Melhoria de Testes Unitários e Integrados

## Overview

Este documento detalha o design para melhorar a cobertura de testes unitários (meta de ~80%) e corrigir os testes integrados do OCI Build System. O design aborda correção de bugs de configuração (como `GITHUB_WEBHOOK_SECRET`), automação via Makefile, health checks adequados, e processo de validação de testes antes de correção de código.

### Principais Características

- Cobertura de testes unitários próxima a 80%
- Testes integrados executáveis via Makefile
- Correção de bugs de configuração do Docker Compose
- Health checks robustos para garantir ambiente pronto
- Property-based tests para propriedades críticas
- Relatórios HTML de cobertura
- Documentação completa de testes
- Processo de validação de testes antes de correção de código

## Architecture

### Estrutura de Testes

```
oci-build-system/
├── apps/
│   ├── api-service/
│   │   ├── handlers/
│   │   │   ├── webhook.go
│   │   │   ├── webhook_test.go              # Testes unitários
│   │   │   ├── webhook_property_test.go     # Property-based tests
│   │   │   ├── status.go
│   │   │   ├── status_test.go
│   │   │   ├── health.go
│   │   │   └── health_test.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── auth_test.go
│   │   │   ├── auth_property_test.go
│   │   │   ├── logging.go
│   │   │   └── logging_test.go
│   │   └── main.go
│   └── worker-service/
│       ├── orchestrator.go
│       ├── orchestrator_test.go
│       ├── worker.go
│       ├── worker_test.go
│       └── worker_property_test.go
├── libs/
│   ├── git-service/
│   │   ├── manager.go
│   │   ├── manager_test.go
│   │   └── manager_property_test.go
│   ├── cache-service/
│   │   ├── manager.go
│   │   ├── manager_test.go
│   │   └── manager_property_test.go
│   ├── image-service/
│   │   ├── service.go
│   │   ├── service_test.go
│   │   └── service_property_test.go
│   ├── nats-client/
│   │   ├── client.go
│   │   └── client_test.go
│   ├── nx-service/
│   │   ├── builder.go
│   │   ├── builder_test.go
│   │   └── builder_property_test.go
│   └── shared/
│       ├── types.go
│       ├── types_test.go
│       ├── config.go
│       └── config_test.go
├── tests/
│   ├── integration/
│   │   ├── webhook.robot
│   │   ├── build.robot
│   │   ├── api.robot
│   │   ├── resources/
│   │   │   ├── keywords.robot
│   │   │   └── variables.robot
│   │   ├── fixtures/
│   │   │   ├── sample-java-repo/
│   │   │   ├── sample-dotnet-repo/
│   │   │   └── sample-go-repo/
│   │   └── results/
│   └── testutil/                            # Helpers de teste
│       ├── mocks.go
│       ├── fixtures.go
│       └── assertions.go
├── test-reports/                            # Relatórios de teste
│   ├── coverage/
│   │   └── index.html
│   └── integration/
│       ├── report.html
│       └── log.html
├── .env.example                             # Exemplo de variáveis de ambiente
├── docker-compose.test.yml                  # Compose para testes
├── Makefile                                 # Comandos de teste
├── TESTING.md                               # Documentação de testes
└── scripts/
    ├── wait-for-services.sh                 # Script de health check
    └── run-integration-tests.sh             # Script de execução de testes
```

### Fluxo de Execução de Testes

```
┌─────────────────────────────────────────────────────────────┐
│                    make test-all                            │
└────────────────────┬────────────────────────────────────────┘
                     │
         ┌───────────┴───────────┐
         ▼                       ▼
┌──────────────────┐    ┌──────────────────┐
│  make test-unit  │    │ make test-       │
│                  │    │  integration     │
└────────┬─────────┘    └────────┬─────────┘
         │                       │
         ▼                       ▼
┌──────────────────┐    ┌──────────────────┐
│ go test ./...    │    │ docker-compose   │
│ -cover           │    │ up -d            │
│ -coverprofile    │    └────────┬─────────┘
└────────┬─────────┘             │
         │                       ▼
         ▼                ┌──────────────────┐
┌──────────────────┐     │ wait-for-        │
│ go tool cover    │     │ services.sh      │
│ -html            │     └────────┬─────────┘
└──────────────────┘              │
                                  ▼
                         ┌──────────────────┐
                         │ robot tests/     │
                         │ integration/     │
                         └────────┬─────────┘
                                  │
                                  ▼
                         ┌──────────────────┐
                         │ docker-compose   │
                         │ down             │
                         └──────────────────┘
```

## Components and Interfaces

### 1. Test Utilities (tests/testutil)

**Responsabilidade**: Fornecer helpers e utilitários reutilizáveis para testes.

**Conteúdo**:

```go
// tests/testutil/mocks.go
package testutil

import (
    "github.com/nats-io/nats.go"
    "go.uber.org/zap"
)

// MockNATSClient simula cliente NATS para testes
type MockNATSClient struct {
    PublishedMessages []MockMessage
    Subscriptions     map[string]nats.MsgHandler
    Connected         bool
}

type MockMessage struct {
    Subject string
    Data    []byte
}

func NewMockNATSClient() *MockNATSClient {
    return &MockNATSClient{
        PublishedMessages: make([]MockMessage, 0),
        Subscriptions:     make(map[string]nats.MsgHandler),
        Connected:         true,
    }
}

func (m *MockNATSClient) Publish(subject string, data []byte) error {
    if !m.Connected {
        return fmt.Errorf("not connected")
    }
    m.PublishedMessages = append(m.PublishedMessages, MockMessage{
        Subject: subject,
        Data:    data,
    })
    return nil
}

func (m *MockNATSClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
    if !m.Connected {
        return nil, fmt.Errorf("not connected")
    }
    m.Subscriptions[subject] = handler
    return &nats.Subscription{}, nil
}

// MockGitService simula operações Git para testes
type MockGitService struct {
    SyncRepositoryFunc func(ctx context.Context, repo RepositoryInfo, commitHash string) (string, error)
    RepositoryExistsFunc func(repoURL string) bool
}

func (m *MockGitService) SyncRepository(ctx context.Context, repo RepositoryInfo, commitHash string) (string, error) {
    if m.SyncRepositoryFunc != nil {
        return m.SyncRepositoryFunc(ctx, repo, commitHash)
    }
    return "/tmp/test-repo", nil
}

// MockNXService simula builds NX para testes
type MockNXService struct {
    BuildFunc func(ctx context.Context, repoPath string, config BuildConfig) (*BuildResult, error)
}

// MockImageService simula builds de imagem para testes
type MockImageService struct {
    BuildImageFunc func(ctx context.Context, config ImageConfig) (*ImageResult, error)
}
```

```go
// tests/testutil/fixtures.go
package testutil

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// CreateTempRepo cria repositório temporário para testes
func CreateTempRepo(t *testing.T, language string) string {
    tmpDir := t.TempDir()
    
    switch language {
    case "java":
        createFile(t, filepath.Join(tmpDir, "pom.xml"), javaPomXML)
    case "dotnet":
        createFile(t, filepath.Join(tmpDir, "project.csproj"), dotnetCsproj)
    case "go":
        createFile(t, filepath.Join(tmpDir, "go.mod"), goMod)
    }
    
    createFile(t, filepath.Join(tmpDir, "Dockerfile"), dockerfile)
    return tmpDir
}

// LoadWebhookPayload carrega payload de webhook de exemplo
func LoadWebhookPayload(t *testing.T, repoName string) map[string]interface{} {
    payload := map[string]interface{}{
        "ref": "refs/heads/main",
        "after": "abc123def456",
        "repository": map[string]interface{}{
            "name": repoName,
            "full_name": "test-owner/" + repoName,
            "clone_url": "https://github.com/test-owner/" + repoName + ".git",
            "owner": map[string]interface{}{
                "login": "test-owner",
            },
        },
        "head_commit": map[string]interface{}{
            "id": "abc123def456",
            "message": "Test commit",
            "author": map[string]interface{}{
                "name": "Test Author",
                "email": "test@example.com",
            },
        },
    }
    return payload
}

// GenerateHMACSignature gera assinatura HMAC para webhook
func GenerateHMACSignature(payload []byte, secret string) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
```

```go
// tests/testutil/assertions.go
package testutil

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

// AssertBuildJobValid verifica se BuildJob tem todos os campos obrigatórios
func AssertBuildJobValid(t *testing.T, job *BuildJob) {
    assert.NotEmpty(t, job.ID, "Job ID should not be empty")
    assert.NotEmpty(t, job.Repository.Name, "Repository name should not be empty")
    assert.NotEmpty(t, job.CommitHash, "Commit hash should not be empty")
    assert.NotEmpty(t, job.Branch, "Branch should not be empty")
    assert.NotZero(t, job.CreatedAt, "CreatedAt should be set")
}

// AssertHTTPStatus verifica código de status HTTP
func AssertHTTPStatus(t *testing.T, expected, actual int, msgAndArgs ...interface{}) {
    assert.Equal(t, expected, actual, msgAndArgs...)
}

// AssertJSONResponse verifica se resposta é JSON válido
func AssertJSONResponse(t *testing.T, body []byte) {
    var js map[string]interface{}
    err := json.Unmarshal(body, &js)
    assert.NoError(t, err, "Response should be valid JSON")
}
```

### 2. Configuration Fixes

**Problema Identificado**: `GITHUB_WEBHOOK_SECRET` não está sendo carregado corretamente.

**Solução**:

**Arquivo: .env.example**
```bash
# GitHub Configuration
GITHUB_WEBHOOK_SECRET=your-webhook-secret-here

# API Configuration
API_AUTH_TOKEN=your-api-token-here

# NATS Configuration
NATS_URL=nats://nats:4222

# Server Configuration
SERVER_PORT=8080
LOG_LEVEL=info
LOG_FORMAT=json

# Worker Configuration
WORKER_POOL_SIZE=5
WORKER_TIMEOUT=3600
WORKER_MAX_RETRIES=3

# Build Configuration
BUILD_CODE_CACHE_PATH=/var/cache/oci-build/repos
BUILD_BUILD_CACHE_PATH=/var/cache/oci-build/deps
```

**Arquivo: .env (para desenvolvimento local)**
```bash
GITHUB_WEBHOOK_SECRET=test-secret-key-for-development
API_AUTH_TOKEN=test-auth-token-for-development
NATS_URL=nats://localhost:4222
LOG_LEVEL=debug
```

**Correção no docker-compose.yml**:
```yaml
version: '3.8'

services:
  nats:
    image: nats:latest
    ports:
      - "4222:4222"
      - "8222:8222"
    command: "-js -m 8222"
    networks:
      - oci-build-network
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8222/healthz"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 5s
    restart: unless-stopped
    
  api-service:
    build:
      context: .
      dockerfile: apps/api-service/Dockerfile
    ports:
      - "8080:8080"
    environment:
      # NATS Configuration
      - NATS_URL=${NATS_URL:-nats://nats:4222}
      
      # GitHub Configuration (OBRIGATÓRIO)
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET:?GITHUB_WEBHOOK_SECRET is required}
      
      # Auth Configuration (OBRIGATÓRIO)
      - API_AUTH_TOKEN=${API_AUTH_TOKEN:?API_AUTH_TOKEN is required}

      # Server Configuration
      - SERVER_PORT=${SERVER_PORT:-8080}
      - SERVER_READ_TIMEOUT=${SERVER_READ_TIMEOUT:-30}
      - SERVER_WRITE_TIMEOUT=${SERVER_WRITE_TIMEOUT:-30}
      - SERVER_SHUTDOWN_TIMEOUT=${SERVER_SHUTDOWN_TIMEOUT:-10}
      
      # Logging Configuration
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - LOG_FORMAT=${LOG_FORMAT:-json}
    volumes:
      - cache-repos:/var/cache/oci-build/repos
      - cache-deps:/var/cache/oci-build/deps
      - logs-api:/var/log/oci-build
      - ./apps/api-service/config.yaml:/app/config.yaml:ro
    depends_on:
      nats:
        condition: service_healthy
    networks:
      - oci-build-network
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped
      
  worker-service:
    build:
      context: .
      dockerfile: apps/worker-service/Dockerfile
    environment:
      - NATS_URL=${NATS_URL:-nats://nats:4222}
      - WORKER_POOL_SIZE=${WORKER_POOL_SIZE:-5}
      - WORKER_TIMEOUT=${WORKER_TIMEOUT:-3600}
      - WORKER_MAX_RETRIES=${WORKER_MAX_RETRIES:-3}
      - BUILD_CODE_CACHE_PATH=/var/cache/oci-build/repos
      - BUILD_BUILD_CACHE_PATH=/var/cache/oci-build/deps
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - LOG_FORMAT=${LOG_FORMAT:-json}
      - MAVEN_CACHE=/var/cache/oci-build/deps/maven
      - GRADLE_CACHE=/var/cache/oci-build/deps/gradle
      - NUGET_CACHE=/var/cache/oci-build/deps/nuget
      - GOCACHE=/var/cache/oci-build/deps/go
      - GOMODCACHE=/var/cache/oci-build/deps/go/mod
    volumes:
      - cache-repos:/var/cache/oci-build/repos
      - cache-deps:/var/cache/oci-build/deps
      - logs-worker:/var/log/oci-build
      - /var/run/docker.sock:/var/run/docker.sock
      - ./apps/worker-service/config.yaml:/app/config.yaml:ro
    depends_on:
      nats:
        condition: service_healthy
      api-service:
        condition: service_healthy
    networks:
      - oci-build-network
    restart: unless-stopped

networks:
  oci-build-network:
    driver: bridge

volumes:
  cache-repos:
    driver: local
  cache-deps:
    driver: local
  logs-api:
    driver: local
  logs-worker:
    driver: local
```

**Nota**: A sintaxe `${VAR:?message}` faz com que o Docker Compose falhe se a variável não estiver definida.

**Correção no código de carregamento de configuração (libs/shared/config.go)**:

```go
package shared

import (
    "fmt"
    "os"
    "strings"
    
    "github.com/spf13/viper"
    "go.uber.org/zap"
)

// LoadConfig carrega configuração de arquivo YAML com suporte a variáveis de ambiente
func LoadConfig(configPath string, logger *zap.Logger) (*Config, error) {
    viper.SetConfigFile(configPath)
    viper.SetConfigType("yaml")
    
    // Permitir override via variáveis de ambiente
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    // Ler arquivo de configuração
    if err := viper.ReadInConfig(); err != nil {
        logger.Warn("Failed to read config file, using environment variables only",
            zap.String("config_path", configPath),
            zap.Error(err))
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    // Expandir variáveis de ambiente em strings de configuração
    expandEnvVars(&config)
    
    // Validar configuração obrigatória
    if err := validateConfig(&config, logger); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    // Log configuração carregada (sem secrets)
    logConfig(&config, logger)
    
    return &config, nil
}

// expandEnvVars expande variáveis de ambiente em formato ${VAR_NAME}
func expandEnvVars(config *Config) {
    config.GitHub.WebhookSecret = os.ExpandEnv(config.GitHub.WebhookSecret)
    config.Auth.Token = os.ExpandEnv(config.Auth.Token)
    config.NATS.URL = os.ExpandEnv(config.NATS.URL)
}

// validateConfig valida que configurações obrigatórias estão presentes
func validateConfig(config *Config, logger *zap.Logger) error {
    var errors []string
    
    // Validar GitHub webhook secret
    if config.GitHub.WebhookSecret == "" || config.GitHub.WebhookSecret == "${GITHUB_WEBHOOK_SECRET}" {
        errors = append(errors, "GITHUB_WEBHOOK_SECRET is required but not set")
    }
    
    // Validar auth token
    if config.Auth.Token == "" || config.Auth.Token == "${API_AUTH_TOKEN}" {
        errors = append(errors, "API_AUTH_TOKEN is required but not set")
    }
    
    // Validar NATS URL
    if config.NATS.URL == "" {
        errors = append(errors, "NATS URL is required but not set")
    }
    
    // Validar porta do servidor
    if config.Server.Port < 1 || config.Server.Port > 65535 {
        errors = append(errors, fmt.Sprintf("invalid server port: %d", config.Server.Port))
    }
    
    // Validar timeouts
    if config.Server.ReadTimeout < 0 {
        errors = append(errors, "server read timeout cannot be negative")
    }
    if config.Server.WriteTimeout < 0 {
        errors = append(errors, "server write timeout cannot be negative")
    }
    
    // Validar paths de cache (se configurados)
    if config.Build.CodeCachePath != "" {
        if err := validateCachePath(config.Build.CodeCachePath); err != nil {
            errors = append(errors, fmt.Sprintf("invalid code cache path: %v", err))
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
    }
    
    return nil
}

// validateCachePath verifica se path de cache existe e é gravável
func validateCachePath(path string) error {
    info, err := os.Stat(path)
    if err != nil {
        if os.IsNotExist(err) {
            // Tentar criar o diretório
            if err := os.MkdirAll(path, 0755); err != nil {
                return fmt.Errorf("cache path does not exist and cannot be created: %w", err)
            }
            return nil
        }
        return err
    }
    
    if !info.IsDir() {
        return fmt.Errorf("cache path is not a directory")
    }
    
    // Testar se é gravável
    testFile := filepath.Join(path, ".write-test")
    if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
        return fmt.Errorf("cache path is not writable: %w", err)
    }
    os.Remove(testFile)
    
    return nil
}

// logConfig registra configuração carregada (sem expor secrets)
func logConfig(config *Config, logger *zap.Logger) {
    logger.Info("Configuration loaded",
        zap.Int("server_port", config.Server.Port),
        zap.String("nats_url", config.NATS.URL),
        zap.String("log_level", config.Logging.Level),
        zap.String("log_format", config.Logging.Format),
        zap.Bool("github_secret_set", config.GitHub.WebhookSecret != ""),
        zap.Bool("auth_token_set", config.Auth.Token != ""),
    )
}
```

### 3. Health Check Implementation

**Arquivo: apps/api-service/handlers/health.go**

```go
package handlers

import (
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/nats-io/nats.go"
    "go.uber.org/zap"
)

type HealthHandler struct {
    natsClient *nats.Conn
    logger     *zap.Logger
    startTime  time.Time
}

func NewHealthHandler(natsClient *nats.Conn, logger *zap.Logger) *HealthHandler {
    return &HealthHandler{
        natsClient: natsClient,
        logger:     logger,
        startTime:  time.Now(),
    }
}

type HealthResponse struct {
    Status      string            `json:"status"`
    Timestamp   time.Time         `json:"timestamp"`
    Uptime      string            `json:"uptime"`
    Version     string            `json:"version"`
    Checks      map[string]string `json:"checks"`
}

func (h *HealthHandler) Health(c *gin.Context) {
    checks := make(map[string]string)
    allHealthy := true
    
    // Check NATS connection
    if h.natsClient == nil || !h.natsClient.IsConnected() {
        checks["nats"] = "unhealthy"
        allHealthy = false
        h.logger.Warn("NATS health check failed: not connected")
    } else {
        checks["nats"] = "healthy"
    }
    
    // Determine overall status
    status := "healthy"
    statusCode := http.StatusOK
    if !allHealthy {
        status = "unhealthy"
        statusCode = http.StatusServiceUnavailable
    }
    
    response := HealthResponse{
        Status:    status,
        Timestamp: time.Now(),
        Uptime:    time.Since(h.startTime).String(),
        Version:   "1.0.0", // TODO: Get from build info
        Checks:    checks,
    }
    
    c.JSON(statusCode, response)
}

// Readiness verifica se o serviço está pronto para receber tráfego
func (h *HealthHandler) Readiness(c *gin.Context) {
    // Verificar se NATS está conectado
    if h.natsClient == nil || !h.natsClient.IsConnected() {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "status": "not_ready",
            "reason": "nats_not_connected",
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "status": "ready",
    })
}

// Liveness verifica se o serviço está vivo (para Kubernetes)
func (h *HealthHandler) Liveness(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status": "alive",
    })
}
```

### 4. Wait for Services Script

**Arquivo: scripts/wait-for-services.sh**

```bash
#!/bin/bash
set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuração
MAX_RETRIES=${MAX_RETRIES:-30}
RETRY_INTERVAL=${RETRY_INTERVAL:-2}
API_URL=${API_URL:-http://localhost:8080}
NATS_URL=${NATS_URL:-http://localhost:8222}

echo -e "${YELLOW}Waiting for services to be ready...${NC}"

# Função para verificar se um serviço está pronto
check_service() {
    local service_name=$1
    local health_url=$2
    local retries=0
    
    echo -e "${YELLOW}Checking ${service_name}...${NC}"
    
    while [ $retries -lt $MAX_RETRIES ]; do
        if curl -f -s -o /dev/null "$health_url"; then
            echo -e "${GREEN}✓ ${service_name} is ready${NC}"
            return 0
        fi
        
        retries=$((retries + 1))
        echo -e "${YELLOW}  Attempt $retries/$MAX_RETRIES - ${service_name} not ready yet, waiting...${NC}"
        sleep $RETRY_INTERVAL
    done
    
    echo -e "${RED}✗ ${service_name} failed to become ready after $MAX_RETRIES attempts${NC}"
    return 1
}

# Verificar NATS
if ! check_service "NATS" "${NATS_URL}/healthz"; then
    echo -e "${RED}NATS is not ready. Exiting.${NC}"
    exit 1
fi

# Verificar API Service
if ! check_service "API Service" "${API_URL}/health"; then
    echo -e "${RED}API Service is not ready. Exiting.${NC}"
    exit 1
fi

echo -e "${GREEN}All services are ready!${NC}"
exit 0
```

### 5. Makefile Updates

**Adições ao Makefile**:

```makefile
# Testes
test-unit: ## Executa testes unitários com cobertura
	@echo "$(GREEN)Executando testes unitários...$(NC)"
	@mkdir -p test-reports/coverage
	$(GO) test ./... -v -race -coverprofile=test-reports/coverage/coverage.out -covermode=atomic
	$(GO) tool cover -html=test-reports/coverage/coverage.out -o test-reports/coverage/index.html
	@echo "$(GREEN)Cobertura: $$($(GO) tool cover -func=test-reports/coverage/coverage.out | grep total | awk '{print $$3}')$(NC)"
	@echo "$(GREEN)Relatório disponível em: test-reports/coverage/index.html$(NC)"

test-unit-quick: ## Executa testes unitários sem cobertura (rápido)
	@echo "$(GREEN)Executando testes unitários (modo rápido)...$(NC)"
	$(GO) test ./... -v -short

test-property: ## Executa apenas property-based tests
	@echo "$(GREEN)Executando property-based tests...$(NC)"
	$(GO) test ./... -v -run "TestProperty"

test-coverage-report: ## Abre relatório de cobertura no browser
	@echo "$(GREEN)Abrindo relatório de cobertura...$(NC)"
	@open test-reports/coverage/index.html || xdg-open test-reports/coverage/index.html

test-integration-setup: ## Sobe ambiente para testes integrados
	@echo "$(GREEN)Subindo ambiente de testes integrados...$(NC)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Arquivo .env não encontrado. Copiando .env.example...$(NC)"; \
		cp .env.example .env; \
		echo "$(YELLOW)ATENÇÃO: Configure as variáveis em .env antes de continuar!$(NC)"; \
		exit 1; \
	fi
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Aguardando serviços ficarem prontos...$(NC)"
	@bash scripts/wait-for-services.sh
	@echo "$(GREEN)Ambiente pronto!$(NC)"

test-integration-teardown: ## Derruba ambiente de testes integrados
	@echo "$(YELLOW)Derrubando ambiente de testes integrados...$(NC)"
	$(DOCKER_COMPOSE) down -v
	@echo "$(GREEN)Ambiente removido!$(NC)"

test-integration: test-integration-setup ## Executa testes integrados completos
	@echo "$(GREEN)Executando testes integrados...$(NC)"
	@cd tests/integration && \
		python3 -m pip install -q -r requirements.txt && \
		robot --outputdir results .
	@echo "$(GREEN)Testes integrados concluídos!$(NC)"
	@echo "$(GREEN)Relatório disponível em: tests/integration/results/report.html$(NC)"
	@$(MAKE) test-integration-teardown

test-integration-keep: test-integration-setup ## Executa testes integrados e mantém ambiente
	@echo "$(GREEN)Executando testes integrados (mantendo ambiente)...$(NC)"
	@cd tests/integration && \
		python3 -m pip install -q -r requirements.txt && \
		robot --outputdir results .
	@echo "$(GREEN)Testes concluídos! Ambiente ainda está rodando.$(NC)"
	@echo "$(YELLOW)Use 'make test-integration-teardown' para derrubar o ambiente.$(NC)"

test-integration-logs: ## Mostra logs dos serviços durante testes
	@echo "$(GREEN)Logs dos serviços:$(NC)"
	$(DOCKER_COMPOSE) logs --tail=100

test-all: test-unit test-integration ## Executa todos os testes (unitários e integrados)
	@echo "$(GREEN)Todos os testes concluídos!$(NC)"

test-quick: test-unit-quick ## Executa apenas testes rápidos
	@echo "$(GREEN)Testes rápidos concluídos!$(NC)"

# Validação de ambiente
validate-env: ## Valida que variáveis de ambiente necessárias estão definidas
	@echo "$(GREEN)Validando variáveis de ambiente...$(NC)"
	@if [ -z "$$GITHUB_WEBHOOK_SECRET" ]; then \
		echo "$(RED)ERRO: GITHUB_WEBHOOK_SECRET não está definido$(NC)"; \
		exit 1; \
	fi
	@if [ -z "$$API_AUTH_TOKEN" ]; then \
		echo "$(RED)ERRO: API_AUTH_TOKEN não está definido$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Todas as variáveis obrigatórias estão definidas!$(NC)"

# Limpeza de relatórios
clean-test-reports: ## Remove relatórios de teste antigos
	@echo "$(YELLOW)Removendo relatórios de teste...$(NC)"
	rm -rf test-reports/
	rm -rf tests/integration/results/
	@echo "$(GREEN)Relatórios removidos!$(NC)"
```

### 6. Unit Test Coverage Improvements

**Estratégia para atingir 80% de cobertura**:

1. **Identificar gaps de cobertura**:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -v "100.0%" | sort -k3 -n
```

2. **Priorizar componentes críticos**:
   - Handlers HTTP (webhook, status, health)
   - Middlewares (auth, logging)
   - Orchestrator de builds
   - Serviços auxiliares (git, cache, image, nx)

3. **Adicionar testes faltantes**:

**Exemplo: Middleware de Logging (apps/api-service/middleware/logging_test.go)**

```go
package middleware

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func TestLoggingMiddleware_LogsRequest(t *testing.T) {
    // Setup
    gin.SetMode(gin.TestMode)
    
    // Criar logger que escreve em buffer para verificar logs
    var buf bytes.Buffer
    encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
    core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.InfoLevel)
    logger := zap.New(core)
    
    // Criar router com middleware
    router := gin.New()
    router.Use(LoggingMiddleware(logger))
    router.GET("/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "success"})
    })
    
    // Executar request
    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Verificar
    assert.Equal(t, http.StatusOK, w.Code)
    
    // Verificar que log foi gerado
    logOutput := buf.String()
    assert.Contains(t, logOutput, "GET")
    assert.Contains(t, logOutput, "/test")
    assert.Contains(t, logOutput, "200")
}

func TestLoggingMiddleware_LogsError(t *testing.T) {
    // Setup
    gin.SetMode(gin.TestMode)
    
    var buf bytes.Buffer
    encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
    core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.InfoLevel)
    logger := zap.New(core)
    
    router := gin.New()
    router.Use(LoggingMiddleware(logger))
    router.GET("/error", func(c *gin.Context) {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
    })
    
    // Executar request
    req := httptest.NewRequest(http.MethodGet, "/error", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Verificar
    assert.Equal(t, http.StatusInternalServerError, w.Code)
    
    // Verificar que erro foi logado
    logOutput := buf.String()
    assert.Contains(t, logOutput, "500")
}

func TestLoggingMiddleware_IncludesLatency(t *testing.T) {
    // Setup
    gin.SetMode(gin.TestMode)
    
    var buf bytes.Buffer
    encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
    core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.InfoLevel)
    logger := zap.New(core)
    
    router := gin.New()
    router.Use(LoggingMiddleware(logger))
    router.GET("/slow", func(c *gin.Context) {
        time.Sleep(10 * time.Millisecond)
        c.JSON(http.StatusOK, gin.H{"message": "done"})
    })
    
    // Executar request
    req := httptest.NewRequest(http.MethodGet, "/slow", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Verificar que latência foi registrada
    logOutput := buf.String()
    assert.Contains(t, logOutput, "latency")
}
```

**Exemplo: NX Service Tests (libs/nx-service/builder_test.go)**

```go
package nxservice

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap"
)

func TestNXService_DetectLanguage_Java(t *testing.T) {
    // Setup
    logger := zap.NewNop()
    service := NewNXService(logger)
    
    tmpDir := t.TempDir()
    pomXML := filepath.Join(tmpDir, "pom.xml")
    err := os.WriteFile(pomXML, []byte("<project></project>"), 0644)
    require.NoError(t, err)
    
    // Execute
    language, err := service.DetectLanguage(tmpDir)
    
    // Verify
    assert.NoError(t, err)
    assert.Equal(t, LanguageJava, language)
}

func TestNXService_DetectLanguage_DotNet(t *testing.T) {
    logger := zap.NewNop()
    service := NewNXService(logger)
    
    tmpDir := t.TempDir()
    csproj := filepath.Join(tmpDir, "project.csproj")
    err := os.WriteFile(csproj, []byte("<Project></Project>"), 0644)
    require.NoError(t, err)
    
    language, err := service.DetectLanguage(tmpDir)
    
    assert.NoError(t, err)
    assert.Equal(t, LanguageDotNet, language)
}

func TestNXService_DetectLanguage_Go(t *testing.T) {
    logger := zap.NewNop()
    service := NewNXService(logger)
    
    tmpDir := t.TempDir()
    goMod := filepath.Join(tmpDir, "go.mod")
    err := os.WriteFile(goMod, []byte("module test"), 0644)
    require.NoError(t, err)
    
    language, err := service.DetectLanguage(tmpDir)
    
    assert.NoError(t, err)
    assert.Equal(t, LanguageGo, language)
}

func TestNXService_DetectLanguage_Unknown(t *testing.T) {
    logger := zap.NewNop()
    service := NewNXService(logger)
    
    tmpDir := t.TempDir()
    
    language, err := service.DetectLanguage(tmpDir)
    
    assert.Error(t, err)
    assert.Equal(t, LanguageUnknown, language)
}

func TestNXService_Build_Timeout(t *testing.T) {
    logger := zap.NewNop()
    service := NewNXService(logger)
    
    tmpDir := t.TempDir()
    
    // Context com timeout muito curto
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()
    
    config := BuildConfig{
        CachePath: tmpDir,
        Language:  LanguageGo,
    }
    
    result, err := service.Build(ctx, tmpDir, config)
    
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "context deadline exceeded")
}
```

### 7. Property-Based Test Examples

**Exemplo: Image Service Tag Validation (libs/image-service/service_property_test.go)**

```go
package imageservice

import (
    "testing"
    
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
    "go.uber.org/zap"
)

// Feature: oci-build-system, Property 15: Aplicação de tags de imagem
// Para qualquer imagem OCI construída com sucesso, o sistema deve aplicar
// pelo menos duas tags: uma com o commit hash completo e outra com o nome do branch.
func TestProperty_ImageTagging(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("image has commit hash and branch tags", prop.ForAll(
        func(commitHash string, branch string) bool {
            // Setup
            logger := zap.NewNop()
            service := NewImageService(logger)
            
            // Gerar tags esperadas
            expectedTags := []string{
                commitHash,
                branch,
            }
            
            // Simular resultado de build
            result := &ImageResult{
                ImageID: "test-image-id",
                Tags:    expectedTags,
            }
            
            // Verificar que ambas as tags estão presentes
            hasCommitTag := false
            hasBranchTag := false
            
            for _, tag := range result.Tags {
                if tag == commitHash {
                    hasCommitTag = true
                }
                if tag == branch {
                    hasBranchTag = true
                }
            }
            
            return hasCommitTag && hasBranchTag && len(result.Tags) >= 2
        },
        gen.RegexMatch("[a-f0-9]{40}"), // Commit hash SHA-1
        gen.RegexMatch("[a-zA-Z0-9_-]+"), // Branch name
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### 8. Integration Test Improvements

**Arquivo: tests/integration/resources/keywords.robot**

```robot
*** Settings ***
Library    RequestsLibrary
Library    Collections
Library    String
Library    OperatingSystem

*** Keywords ***
Setup Test Environment
    [Documentation]    Setup test environment before running tests
    Create Session    api    ${API_BASE_URL}    verify=False
    Set Suite Variable    ${API_SESSION}    api

Teardown Test Environment
    [Documentation]    Cleanup after tests
    Delete All Sessions

Send Webhook
    [Arguments]    ${payload}    ${signature}
    [Documentation]    Send webhook with signature
    ${headers}=    Create Dictionary
    ...    Content-Type=application/json
    ...    X-Hub-Signature-256=${signature}
    ${response}=    POST On Session    api    /webhook
    ...    json=${payload}
    ...    headers=${headers}
    ...    expected_status=any
    RETURN    ${response}

Send Webhook Without Signature
    [Arguments]    ${payload}
    [Documentation]    Send webhook without signature header
    ${headers}=    Create Dictionary    Content-Type=application/json
    ${response}=    POST On Session    api    /webhook
    ...    json=${payload}
    ...    headers=${headers}
    ...    expected_status=any
    RETURN    ${response}

Get Build Status
    [Arguments]    ${job_id}
    [Documentation]    Get build status by job ID
    ${headers}=    Create Dictionary
    ...    Authorization=Bearer ${AUTH_TOKEN}
    ${response}=    GET On Session    api    /builds/${job_id}
    ...    headers=${headers}
    ...    expected_status=any
    RETURN    ${response}

Wait Until Build Completes
    [Arguments]    ${job_id}    ${timeout}=300s
    [Documentation]    Wait until build reaches terminal state
    ${end_time}=    Evaluate    time.time() + ${timeout}    modules=time
    WHILE    True
        ${current_time}=    Evaluate    time.time()    modules=time
        IF    ${current_time} > ${end_time}
            Fail    Build did not complete within timeout
        END
        
        ${response}=    Get Build Status    ${job_id}
        ${status}=    Get From Dictionary    ${response.json()}    status
        
        IF    '${status}' in ['completed', 'failed']
            RETURN
        END
        
        Sleep    2s
    END

Generate Random Commit Hash
    [Documentation]    Generate random commit hash for testing
    ${hash}=    Generate Random String    40    [LOWER]abcdef0123456789
    RETURN    ${hash}

Create Webhook Payload
    [Arguments]    ${repo_name}    ${commit_hash}    ${branch}
    [Documentation]    Create webhook payload for testing
    ${payload}=    Create Dictionary
    ...    ref=refs/heads/${branch}
    ...    after=${commit_hash}
    ${repository}=    Create Dictionary
    ...    name=${repo_name}
    ...    full_name=test-owner/${repo_name}
    ...    clone_url=https://github.com/test-owner/${repo_name}.git
    ${owner}=    Create Dictionary    login=test-owner
    Set To Dictionary    ${repository}    owner=${owner}
    Set To Dictionary    ${payload}    repository=${repository}
    
    ${head_commit}=    Create Dictionary
    ...    id=${commit_hash}
    ...    message=Test commit message
    ${author}=    Create Dictionary
    ...    name=Test Author
    ...    email=test@example.com
    Set To Dictionary    ${head_commit}    author=${author}
    Set To Dictionary    ${payload}    head_commit=${head_commit}
    
    RETURN    ${payload}

Generate HMAC Signature
    [Arguments]    ${payload}    ${secret}
    [Documentation]    Generate HMAC-SHA256 signature for webhook
    ${json_payload}=    Evaluate    json.dumps($payload)    modules=json
    ${signature}=    Evaluate    hmac.new(b'${secret}', b'${json_payload}', hashlib.sha256).hexdigest()    modules=hmac,hashlib
    ${full_signature}=    Set Variable    sha256=${signature}
    RETURN    ${full_signature}

Verify Response Status
    [Arguments]    ${response}    ${expected_status}
    [Documentation]    Verify HTTP response status code
    Should Be Equal As Numbers    ${response.status_code}    ${expected_status}

Verify JSON Response
    [Arguments]    ${response}
    [Documentation]    Verify response is valid JSON
    ${json}=    Set Variable    ${response.json()}
    Should Not Be Empty    ${json}
```

**Arquivo: tests/integration/resources/variables.robot**

```robot
*** Variables ***
# API Configuration
${API_BASE_URL}         http://localhost:8080
${AUTH_TOKEN}           test-auth-token-for-development
${GITHUB_SECRET}        test-secret-key-for-development

# HTTP Status Codes
${HTTP_OK}              200
${HTTP_ACCEPTED}        202
${HTTP_BAD_REQUEST}     400
${HTTP_UNAUTHORIZED}    401
${HTTP_NOT_FOUND}       404
${HTTP_SERVICE_UNAVAILABLE}    503

# Test Repositories
${TEST_REPO_JAVA}       sample-java-repo
${TEST_REPO_DOTNET}     sample-dotnet-repo
${TEST_REPO_GO}         sample-go-repo

# Timeouts
${DEFAULT_TIMEOUT}      300s
${SHORT_TIMEOUT}        30s
```

### 9. Test Validation Process

**Processo para validar testes antes de corrigir código**:

1. **Revisar o teste que falhou**:
   - Ler o código do teste
   - Verificar se o teste está testando o comportamento correto
   - Verificar se as assertions fazem sentido

2. **Verificar configuração de mocks**:
   - Mocks estão retornando valores esperados?
   - Stubs estão configurados corretamente?
   - Dependências estão sendo injetadas corretamente?

3. **Executar teste isoladamente**:
```bash
# Executar apenas o teste que falhou
go test -v -run TestSpecificTest ./path/to/package
```

4. **Adicionar logs de debug**:
```go
func TestSomething(t *testing.T) {
    // Adicionar logs para entender o que está acontecendo
    t.Logf("Input: %+v", input)
    t.Logf("Expected: %+v", expected)
    t.Logf("Actual: %+v", actual)
    
    assert.Equal(t, expected, actual)
}
```

5. **Verificar ambiente de teste**:
   - Variáveis de ambiente estão definidas?
   - Arquivos temporários estão sendo criados corretamente?
   - Permissões de arquivo estão corretas?

6. **Comparar com testes similares**:
   - Outros testes no mesmo arquivo passam?
   - Testes similares em outros componentes passam?

7. **Decisão**:
   - Se o teste está incorreto → Corrigir o teste
   - Se o código está incorreto → Corrigir o código
   - Se ambos estão corretos mas há incompatibilidade → Ajustar ambos

**Checklist de Validação de Teste**:

```markdown
## Test Validation Checklist

- [ ] O teste tem um nome descritivo que explica o que está sendo testado?
- [ ] O teste segue o padrão Arrange-Act-Assert (Setup-Execute-Verify)?
- [ ] As assertions verificam o comportamento correto?
- [ ] Os mocks estão configurados corretamente?
- [ ] O teste é determinístico (não depende de timing ou ordem de execução)?
- [ ] O teste limpa recursos após execução (usando t.Cleanup ou defer)?
- [ ] O teste falha pelos motivos corretos?
- [ ] A mensagem de erro do teste é clara e útil?
- [ ] O teste não tem dependências externas desnecessárias?
- [ ] O teste pode ser executado isoladamente?
```

### 10. Documentation

**Arquivo: TESTING.md**

```markdown
# Testing Guide

Este documento descreve a estratégia de testes do OCI Build System e como executar e escrever testes.

## Visão Geral

O projeto utiliza três tipos de testes:

1. **Testes Unitários**: Testam componentes isolados
2. **Property-Based Tests**: Testam propriedades universais com entradas geradas
3. **Testes Integrados**: Testam o sistema completo end-to-end

## Meta de Cobertura

- Cobertura mínima: 75%
- Cobertura alvo: 80%
- Foco em lógica de negócio e tratamento de erros

## Executando Testes

### Testes Unitários

```bash
# Executar todos os testes unitários
make test-unit

# Executar testes rápidos (sem cobertura)
make test-unit-quick

# Executar apenas property-based tests
make test-property

# Ver relatório de cobertura
make test-coverage-report
```

### Testes Integrados

```bash
# Executar testes integrados completos (sobe e derruba ambiente)
make test-integration

# Executar testes mantendo ambiente rodando
make test-integration-keep

# Apenas subir ambiente
make test-integration-setup

# Apenas derrubar ambiente
make test-integration-teardown

# Ver logs dos serviços
make test-integration-logs
```

### Todos os Testes

```bash
# Executar todos os testes (unitários + integrados)
make test-all

# Executar apenas testes rápidos
make test-quick
```

## Escrevendo Testes

### Testes Unitários

Estrutura básica de um teste unitário:

```go
func TestComponentName_Behavior(t *testing.T) {
    // Arrange (Setup)
    logger := zap.NewNop()
    service := NewService(logger)
    input := "test-input"
    
    // Act (Execute)
    result, err := service.DoSomething(input)
    
    // Assert (Verify)
    assert.NoError(t, err)
    assert.Equal(t, "expected-output", result)
}
```

**Boas Práticas**:

- Use `t.TempDir()` para criar diretórios temporários
- Use `t.Cleanup()` para limpar recursos
- Use table-driven tests para múltiplos casos
- Teste casos de sucesso, erro e edge cases
- Use mocks para dependências externas

**Exemplo de Table-Driven Test**:

```go
func TestValidateInput(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid", false},
        {"empty input", "", true},
        {"too long", strings.Repeat("a", 1000), true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInput(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Property-Based Tests

Property-based tests verificam propriedades universais através de múltiplas entradas geradas:

```go
func TestProperty_SomeBehavior(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("description of property", prop.ForAll(
        func(input string) bool {
            // Test logic
            result := DoSomething(input)
            return result != ""
        },
        gen.AnyString(),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

**Generators Úteis**:

- `gen.AnyString()` - Qualquer string
- `gen.Int()` - Qualquer inteiro
- `gen.IntRange(min, max)` - Inteiro em range
- `gen.RegexMatch(pattern)` - String que match regex
- `gen.SliceOf(gen)` - Slice de elementos
- `gen.OneConstOf(values...)` - Um dos valores constantes

### Testes Integrados (Robot Framework)

Estrutura de um teste Robot:

```robot
*** Test Cases ***
Test Name
    [Documentation]    Description of what this test does
    [Tags]    tag1    tag2
    
    # Arrange
    ${data}=    Prepare Test Data
    
    # Act
    ${response}=    Call API    ${data}
    
    # Assert
    Should Be Equal    ${response.status_code}    200
```

**Keywords Disponíveis**:

- `Send Webhook` - Envia webhook com assinatura
- `Get Build Status` - Consulta status de build
- `Wait Until Build Completes` - Aguarda build terminar
- `Generate Random Commit Hash` - Gera hash aleatório
- `Create Webhook Payload` - Cria payload de teste
- `Generate HMAC Signature` - Gera assinatura HMAC

## Debugging Testes

### Teste Unitário Falhando

1. Execute o teste com verbose:
```bash
go test -v -run TestName ./path/to/package
```

2. Adicione logs de debug:
```go
t.Logf("Debug info: %+v", variable)
```

3. Use debugger:
```bash
dlv test ./path/to/package -- -test.run TestName
```

### Teste Integrado Falhando

1. Verifique logs dos serviços:
```bash
make test-integration-logs
```

2. Execute teste específico:
```bash
cd tests/integration
robot -t "Test Name" webhook.robot
```

3. Mantenha ambiente rodando para investigar:
```bash
make test-integration-keep
# Investigar manualmente
curl http://localhost:8080/health
make test-integration-teardown
```

## Ambiente de Teste

### Variáveis de Ambiente Necessárias

Copie `.env.example` para `.env` e configure:

```bash
cp .env.example .env
# Edite .env com suas configurações
```

Variáveis obrigatórias:
- `GITHUB_WEBHOOK_SECRET` - Secret para validação de webhooks
- `API_AUTH_TOKEN` - Token de autenticação da API

### Fixtures de Teste

Repositórios de exemplo em `tests/integration/fixtures/`:

- `sample-java-repo/` - Projeto Maven simples
- `sample-dotnet-repo/` - Projeto .NET simples
- `sample-go-repo/` - Projeto Go simples

Cada fixture contém:
- Arquivo de configuração da linguagem (pom.xml, *.csproj, go.mod)
- Dockerfile
- Código fonte mínimo

## CI/CD

### GitHub Actions

Os testes são executados automaticamente em cada PR:

```yaml
- name: Run unit tests
  run: make test-unit

- name: Run integration tests
  run: make test-integration
  env:
    GITHUB_WEBHOOK_SECRET: ${{ secrets.GITHUB_WEBHOOK_SECRET }}
    API_AUTH_TOKEN: ${{ secrets.API_AUTH_TOKEN }}
```

### Cobertura de Código

Relatórios de cobertura são gerados automaticamente e disponíveis em:
- `test-reports/coverage/index.html` - Relatório HTML
- `test-reports/coverage/coverage.out` - Dados brutos

## Troubleshooting

### "GITHUB_WEBHOOK_SECRET is required but not set"

Solução: Configure a variável no arquivo `.env`:
```bash
echo "GITHUB_WEBHOOK_SECRET=your-secret-here" >> .env
```

### "NATS is not ready"

Solução: Aguarde mais tempo ou verifique se NATS está rodando:
```bash
docker-compose ps nats
docker-compose logs nats
```

### "Test timeout"

Solução: Aumente o timeout no teste ou verifique se há deadlock:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### "Port already in use"

Solução: Pare serviços existentes:
```bash
make stop
# ou
docker-compose down
```

## Contribuindo

Ao adicionar novos testes:

1. Siga as convenções de nomenclatura
2. Adicione documentação no teste
3. Verifique que o teste passa localmente
4. Verifique que a cobertura não diminui
5. Adicione o teste ao CI se necessário

## Recursos

- [Testing in Go](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Gopter Documentation](https://github.com/leanovate/gopter)
- [Robot Framework User Guide](https://robotframework.org/robotframework/latest/RobotFrameworkUserGuide.html)
```

## Test Report Generation

### Coverage Report Structure

```
test-reports/
├── coverage/
│   ├── index.html              # HTML coverage report
│   ├── coverage.out            # Raw coverage data
│   └── coverage.txt            # Text summary
├── integration/
│   ├── report.html             # Robot Framework report
│   ├── log.html                # Detailed test log
│   └── output.xml              # Machine-readable results
└── junit/
    └── results.xml             # JUnit format for CI
```

### Generating Reports

**Coverage Report**:
```bash
# Generate HTML report
go test ./... -coverprofile=test-reports/coverage/coverage.out
go tool cover -html=test-reports/coverage/coverage.out -o test-reports/coverage/index.html

# Generate text summary
go tool cover -func=test-reports/coverage/coverage.out > test-reports/coverage/coverage.txt
```

**Integration Test Report**:
```bash
cd tests/integration
robot --outputdir ../../test-reports/integration .
```

**JUnit Report (for CI)**:
```bash
# Install go-junit-report
go install github.com/jstemmer/go-junit-report/v2@latest

# Generate JUnit XML
go test ./... -v 2>&1 | go-junit-report -set-exit-code > test-reports/junit/results.xml
```

## Correctness Properties

*As propriedades de corretude definem comportamentos que devem ser verdadeiros em todas as execuções válidas do sistema de testes.*

### Propriedade 1: Cobertura Mínima

*Para qualquer* execução de testes unitários com flag de cobertura, a cobertura reportada deve ser de pelo menos 75%.

**Valida: Requisito 1.1**

### Propriedade 2: Testes Determinísticos

*Para qualquer* teste unitário ou integrado, executar o teste múltiplas vezes com as mesmas entradas deve produzir o mesmo resultado.

**Valida: Requisito 5.3**

### Propriedade 3: Isolamento de Testes

*Para quaisquer* dois testes executados em sequência ou paralelo, o resultado de um teste não deve afetar o resultado do outro.

**Valida: Requisito 7.6**

### Propriedade 4: Validação de Configuração

*Para qualquer* tentativa de iniciar serviços sem variáveis de ambiente obrigatórias, o sistema deve falhar com mensagem de erro clara indicando qual variável está faltando.

**Valida: Requisitos 2.2, 2.3**

### Propriedade 5: Health Check Readiness

*Para qualquer* serviço que não consegue conectar ao NATS, o health check deve retornar status não-pronto.

**Valida: Requisitos 4.4**

### Propriedade 6: Timeout de Testes

*Para qualquer* teste unitário, o tempo de execução deve ser menor que 30 segundos; para testes integrados, menor que 5 minutos.

**Valida: Requisitos 9.1, 9.2**

### Propriedade 7: Limpeza de Recursos

*Para qualquer* teste que cria recursos temporários (arquivos, diretórios, conexões), esses recursos devem ser limpos após a execução do teste.

**Valida: Requisito 7.5**

### Propriedade 8: Mensagens de Erro Claras

*Para qualquer* teste que falha, a mensagem de erro deve incluir: nome do teste, o que era esperado, o que foi recebido, e contexto suficiente para debug.

**Valida: Requisitos 1.9, 8.4**

### Propriedade 9: Property-Based Test Iterations

*Para qualquer* property-based test, o teste deve executar pelo menos 100 iterações com entradas geradas.

**Valida: Requisito 6.6**

### Propriedade 10: Relatórios Preservados

*Para qualquer* execução de testes, os relatórios devem ser preservados em diretório com timestamp e não devem sobrescrever relatórios anteriores.

**Valida: Requisito 8.6**

## Implementation Plan

### Phase 1: Configuration Fixes (Priority: High)

1. Criar arquivo `.env.example` com todas as variáveis necessárias
2. Atualizar `docker-compose.yml` com validação de variáveis obrigatórias
3. Implementar função `LoadConfig` com expansão de variáveis de ambiente
4. Implementar função `validateConfig` com validação de configurações obrigatórias
5. Adicionar testes unitários para carregamento de configuração

**Entregáveis**:
- `.env.example`
- `docker-compose.yml` atualizado
- `libs/shared/config.go` com validação
- `libs/shared/config_test.go`

### Phase 2: Health Checks (Priority: High)

1. Implementar endpoint `/health` com verificação de NATS
2. Implementar endpoint `/readiness` 
3. Implementar endpoint `/liveness`
4. Adicionar health checks ao `docker-compose.yml`
5. Criar script `wait-for-services.sh`
6. Adicionar testes unitários para health handlers

**Entregáveis**:
- `apps/api-service/handlers/health.go` atualizado
- `apps/api-service/handlers/health_test.go`
- `scripts/wait-for-services.sh`
- `docker-compose.yml` com health checks

### Phase 3: Test Utilities (Priority: Medium)

1. Criar package `tests/testutil`
2. Implementar mocks (MockNATSClient, MockGitService, etc.)
3. Implementar fixtures helpers
4. Implementar assertion helpers
5. Adicionar documentação de uso

**Entregáveis**:
- `tests/testutil/mocks.go`
- `tests/testutil/fixtures.go`
- `tests/testutil/assertions.go`
- `tests/testutil/README.md`

### Phase 4: Unit Test Coverage (Priority: High)

1. Executar análise de cobertura atual
2. Identificar componentes com baixa cobertura
3. Adicionar testes unitários faltantes:
   - Middleware de logging
   - NX Service (detecção de linguagem, build)
   - Image Service (build, tag)
   - Git Service (clone, pull, sync)
   - Cache Service (init, clean, size)
   - NATS Client (connect, publish, subscribe)
4. Adicionar testes de erro e edge cases
5. Verificar meta de 80% de cobertura

**Entregáveis**:
- Testes unitários para todos os componentes
- Cobertura >= 75%
- Relatório de cobertura HTML

### Phase 5: Property-Based Tests (Priority: Medium)

1. Implementar property tests para Propriedade 1 (validação de assinatura)
2. Implementar property tests para Propriedade 2 (extração de webhook)
3. Implementar property tests para Propriedade 11 (detecção de linguagem)
4. Implementar property tests para Propriedade 15 (tags de imagem)
5. Verificar que todos executam >= 100 iterações

**Entregáveis**:
- Property tests anotados com propriedades do design
- Mínimo 100 iterações por teste

### Phase 6: Makefile Integration (Priority: High)

1. Adicionar comandos de teste ao Makefile:
   - `test-unit`
   - `test-unit-quick`
   - `test-property`
   - `test-coverage-report`
   - `test-integration-setup`
   - `test-integration-teardown`
   - `test-integration`
   - `test-integration-keep`
   - `test-all`
2. Adicionar comando `validate-env`
3. Adicionar comando `clean-test-reports`
4. Testar todos os comandos

**Entregáveis**:
- Makefile atualizado com comandos de teste
- Documentação dos comandos

### Phase 7: Integration Test Improvements (Priority: Medium)

1. Atualizar keywords Robot Framework
2. Atualizar variáveis Robot Framework
3. Adicionar testes integrados faltantes
4. Melhorar fixtures de teste
5. Adicionar validação de ambiente antes de executar testes

**Entregáveis**:
- `tests/integration/resources/keywords.robot` atualizado
- `tests/integration/resources/variables.robot` atualizado
- Fixtures atualizados em `tests/integration/fixtures/`

### Phase 8: Documentation (Priority: Medium)

1. Criar `TESTING.md` completo
2. Adicionar exemplos de testes
3. Adicionar guia de troubleshooting
4. Adicionar guia de debugging
5. Documentar processo de validação de testes

**Entregáveis**:
- `TESTING.md`
- Exemplos de código
- Guias de troubleshooting

### Phase 9: CI/CD Integration (Priority: Low)

1. Configurar execução de testes em CI
2. Configurar upload de relatórios de cobertura
3. Configurar notificações de falha
4. Adicionar badge de cobertura ao README

**Entregáveis**:
- `.github/workflows/test.yml`
- Integração com codecov ou similar
- Badge de cobertura

### Phase 10: Validation and Refinement (Priority: High)

1. Executar todos os testes localmente
2. Verificar que ambiente Docker Compose sobe corretamente
3. Verificar que testes integrados passam
4. Verificar cobertura >= 75%
5. Corrigir bugs encontrados
6. Documentar problemas conhecidos

**Entregáveis**:
- Todos os testes passando
- Cobertura >= 75%
- Ambiente funcional
- Documentação de problemas conhecidos

## Success Criteria

O projeto será considerado bem-sucedido quando:

1. ✅ Cobertura de testes unitários >= 75%
2. ✅ Todos os testes unitários passam
3. ✅ Todos os testes integrados passam
4. ✅ Ambiente Docker Compose sobe sem erros
5. ✅ Variáveis de ambiente são validadas corretamente
6. ✅ Health checks funcionam corretamente
7. ✅ Testes podem ser executados via Makefile
8. ✅ Relatórios de cobertura são gerados
9. ✅ Documentação está completa
10. ✅ Property-based tests executam >= 100 iterações

## Risks and Mitigations

### Risk 1: Baixa Cobertura Inicial

**Impacto**: Alto  
**Probabilidade**: Média  
**Mitigação**: Priorizar componentes críticos primeiro, adicionar testes incrementalmente

### Risk 2: Testes Flaky

**Impacto**: Médio  
**Probabilidade**: Média  
**Mitigação**: Usar mocks para dependências externas, evitar dependências de timing

### Risk 3: Ambiente Docker Não Sobe

**Impacto**: Alto  
**Probabilidade**: Baixa  
**Mitigação**: Validar configuração antes de subir, adicionar health checks robustos

### Risk 4: Testes Lentos

**Impacto**: Médio  
**Probabilidade**: Média  
**Mitigação**: Usar mocks em testes unitários, executar testes em paralelo

### Risk 5: Bugs no Código vs Bugs nos Testes

**Impacto**: Alto  
**Probabilidade**: Média  
**Mitigação**: Seguir processo de validação de testes, revisar testes antes de corrigir código

## Maintenance

### Adicionando Novos Testes

1. Identificar componente ou funcionalidade a testar
2. Escolher tipo de teste apropriado (unitário, property, integrado)
3. Escrever teste seguindo convenções
4. Verificar que teste passa
5. Verificar que cobertura não diminui
6. Adicionar documentação se necessário

### Atualizando Testes Existentes

1. Identificar teste que precisa atualização
2. Entender por que o teste está falhando
3. Validar se problema está no teste ou no código
4. Fazer correção apropriada
5. Verificar que teste passa
6. Atualizar documentação se necessário

### Monitorando Cobertura

```bash
# Verificar cobertura atual
make test-unit

# Ver componentes com baixa cobertura
go tool cover -func=test-reports/coverage/coverage.out | grep -v "100.0%" | sort -k3 -n

# Abrir relatório HTML
make test-coverage-report
```

### Debugging Testes Falhando

1. Executar teste isoladamente com verbose
2. Adicionar logs de debug
3. Verificar configuração de mocks
4. Verificar ambiente de teste
5. Comparar com testes similares
6. Seguir checklist de validação
