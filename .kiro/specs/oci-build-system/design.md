# Design Document: OCI Build System

## Overview

O OCI Build System é um conjunto de serviços escritos em Go que automatiza o processo de build e containerização de aplicações. O sistema recebe webhooks do GitHub, gerencia repositórios localmente, executa builds usando NX, e cria imagens OCI usando buildah. A arquitetura é baseada em microserviços que se comunicam via NATS, com cada componente em sua própria pasta e gerenciado por NX.

### Principais Características

- API REST usando Gin para recepção de webhooks e consultas
- NATS como message broker para comunicação entre serviços
- Sistema de filas distribuído para processamento assíncrono de builds
- Cache inteligente de código e dependências
- Suporte para Java, .NET e Go
- Logging estruturado com Zap e métricas de performance
- Isolamento de builds para segurança
- Dependency injection com Uber FX
- Monorepo gerenciado por NX
- Docker Compose para desenvolvimento e testes locais
- Testes integrados com Robot Framework

## Architecture

### Visão Geral da Arquitetura

```
┌─────────────────┐
│  GitHub Webhook │
└────────┬────────┘
         │ HTTP POST
         ▼
┌─────────────────────────────────────────┐
│      API Service (Gin + FX)             │
│  - Webhook Handler                      │
│  - Status Query Handler                 │
│  - Authentication Middleware            │
│  - Zap Logger                           │
└────────┬────────────────────────────────┘
         │ Publish to NATS
         ▼
┌─────────────────────────────────────────┐
│            NATS Message Broker          │
│  - Subject: builds.webhook              │
│  - Subject: builds.status               │
│  - Subject: builds.complete             │
└────────┬────────────────────────────────┘
         │ Subscribe
         ▼
┌─────────────────────────────────────────┐
│      Worker Service (FX)                │
│  - NATS Subscriber                      │
│  - Build Orchestrator                   │
│  - Worker Pool                          │
└────────┬────────────────────────────────┘
         │
         ├──────────────┬──────────────┬──────────────┐
         ▼              ▼              ▼              ▼
┌──────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│Git Service   │ │NX Service│ │  Image   │ │  Cache   │
│   (FX)       │ │   (FX)   │ │ Service  │ │ Service  │
│              │ │          │ │  (FX)    │ │  (FX)    │
└──────────────┘ └──────────┘ └──────────┘ └──────────┘
         │              │              │              │
         ▼              ▼              ▼              ▼
┌──────────────────────────────────────────────────────┐
│              File System Storage                     │
│  - Code Cache (/var/cache/oci-build/repos)           │
│  - Build Cache (/var/cache/oci-build/deps)           │
│  - Logs (/var/log/oci-build)                         │
└──────────────────────────────────────────────────────┘
```

### Estrutura do Monorepo

```
oci-build-system/
├── nx.json
├── package.json
├── docker-compose.yml
├── apps/
│   ├── api-service/          # Serviço de API REST
│   │   ├── main.go
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── *_test.go
│   └── worker-service/       # Serviço de processamento
│       ├── main.go
│       ├── orchestrator/
│       └── *_test.go
├── libs/
│   ├── git-service/          # Biblioteca de operações Git
│   │   ├── manager.go
│   │   └── *_test.go
│   ├── nx-service/           # Biblioteca de builds NX
│   │   ├── builder.go
│   │   └── *_test.go
│   ├── image-service/        # Biblioteca de builds OCI
│   │   ├── buildah.go
│   │   └── *_test.go
│   ├── cache-service/        # Biblioteca de cache
│   │   ├── manager.go
│   │   └── *_test.go
│   ├── nats-client/          # Cliente NATS compartilhado
│   │   ├── client.go
│   │   └── *_test.go
│   └── shared/               # Tipos e utilitários compartilhados
│       ├── types.go
│       ├── config.go
│       └── *_test.go
├── tests/
│   └── integration/          # Testes integrados Robot Framework
│       ├── webhook.robot
│       ├── build.robot
│       └── api.robot
└── tools/
    └── scripts/
```

### Fluxo de Processamento

1. **Recepção**: Webhook chega no API Service (Gin)
2. **Validação**: Autenticação e validação da assinatura
3. **Publicação**: Job é publicado no NATS (subject: builds.webhook)
4. **Subscrição**: Worker Service recebe mensagem do NATS
5. **Processamento**: Worker executa build usando serviços auxiliares
6. **Git Operations**: Git Service faz clone ou pull do repositório
7. **Build**: NX Service executa o build
8. **Containerização**: Image Service cria imagem OCI com buildah
9. **Notificação**: Status é publicado no NATS (subject: builds.complete)
10. **Finalização**: API Service atualiza status e logging

## Components and Interfaces

### 1. API Service (apps/api-service)

**Responsabilidade**: Expor API REST e publicar eventos no NATS.

**Dependências**: Gin, FX, Zap, Viper, NATS client

**Interface Pública**:

```go
type APIService interface {
    Start() error
    Shutdown(ctx context.Context) error
}

type WebhookHandler struct {
    natsClient *nats.Conn
    logger     *zap.Logger
}

type StatusHandler struct {
    natsClient *nats.Conn
    logger     *zap.Logger
}
```

**Implementação**:
- Usa Gin para roteamento HTTP
- FX para dependency injection
- Zap para logging estruturado
- Viper para carregar configuração de arquivo YAML
- Middleware para autenticação via tokens
- Middleware para logging de requisições
- Validação de assinatura HMAC-SHA256 para webhooks GitHub
- Publica jobs no NATS subject `builds.webhook`
- Consulta status via request-reply no NATS

**Endpoints**:
- `POST /webhook` - Recebe webhooks do GitHub
- `GET /builds/:id` - Consulta status de um build
- `GET /builds` - Lista histórico de builds
- `GET /health` - Health check

**Configuração** (config.yaml):
```yaml
server:
  port: 8080
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 10

nats:
  url: "nats://localhost:4222"
  
github:
  webhook_secret: "${GITHUB_WEBHOOK_SECRET}"
  
logging:
  level: "info"
  format: "json"
```

### 2. Worker Service (apps/worker-service)

**Responsabilidade**: Processar builds consumindo mensagens do NATS.

**Dependências**: FX, Zap, Viper, NATS client, Git Service, NX Service, Image Service

**Interface Pública**:

```go
type WorkerService interface {
    Start() error
    Shutdown(ctx context.Context) error
}

type BuildOrchestrator struct {
    gitService   GitService
    nxService    NXService
    imageService ImageService
    cacheService CacheService
    logger       *zap.Logger
}
```

**Implementação**:
- Subscreve ao NATS subject `builds.webhook`
- Pool de workers (goroutines) configurável
- Executa fases sequencialmente com context para timeout
- Publica status no NATS subject `builds.status`
- Publica conclusão no NATS subject `builds.complete`
- Registra métricas de duração para cada fase
- Viper para carregar configuração de arquivo YAML

**Configuração** (config.yaml):
```yaml
nats:
  url: "nats://localhost:4222"
  
worker:
  pool_size: 5
  timeout: 3600
  max_retries: 3
  
build:
  code_cache_path: "/var/cache/oci-build/repos"
  build_cache_path: "/var/cache/oci-build/deps"
  
logging:
  level: "info"
  format: "json"
```

### 3. Git Service (libs/git-service)

**Responsabilidade**: Gerenciar operações Git (clone, pull, checkout).

**Dependências**: go-git, Zap, Viper

**Interface Pública**:

```go
type GitService interface {
    SyncRepository(ctx context.Context, repo RepositoryInfo, commitHash string) (string, error)
    RepositoryExists(repoURL string) bool
    GetLocalPath(repoURL string) string
}

type RepositoryInfo struct {
    URL       string
    Name      string
    Owner     string
    Branch    string
}
```

**Implementação**:
- Usa biblioteca `go-git` para operações Git
- Cache de repositórios em `/var/cache/oci-build/repos/{owner}/{name}`
- Retry com backoff exponencial para operações de rede
- Validação de integridade do repositório local
- Testes unitários para todas as operações

### 4. NX Service (libs/nx-service)

**Responsabilidade**: Executar builds usando NX.

**Dependências**: Zap, Viper, Cache Service

**Interface Pública**:

```go
type NXService interface {
    Build(ctx context.Context, repoPath string, config BuildConfig) (*BuildResult, error)
    DetectProjects(repoPath string) ([]string, error)
}

type BuildConfig struct {
    CachePath    string
    Language     Language
    Environment  map[string]string
}

type Language string

const (
    LanguageJava   Language = "java"
    LanguageDotNet Language = "dotnet"
    LanguageGo     Language = "go"
)

type BuildResult struct {
    Success      bool
    Duration     time.Duration
    Output       string
    ErrorOutput  string
    ArtifactPath string
}
```

**Implementação**:
- Executa comando `nx build` via `os/exec`
- Captura stdout e stderr
- Configura variáveis de ambiente para cache
- Detecta linguagem baseado em arquivos (pom.xml, *.csproj, go.mod)
- Timeout configurável por build
- Testes unitários com mocks de execução

### 5. Image Service (libs/image-service)

**Responsabilidade**: Construir imagens OCI usando buildah.

**Dependências**: Zap, Viper

**Interface Pública**:

```go
type ImageService interface {
    BuildImage(ctx context.Context, config ImageConfig) (*ImageResult, error)
    TagImage(imageID string, tags []string) error
}

type ImageConfig struct {
    ContextPath    string
    DockerfilePath string
    Tags           []string
    BuildArgs      map[string]string
}

type ImageResult struct {
    ImageID   string
    Tags      []string
    Size      int64
    Duration  time.Duration
}
```

**Implementação**:
- Executa comandos buildah via `os/exec`
- Suporta build args e multi-stage builds
- Aplica tags baseadas em commit hash e branch
- Validação de Dockerfile antes do build
- Testes unitários com mocks de buildah

### 6. Cache Service (libs/cache-service)

**Responsabilidade**: Gerenciar caches de dependências por linguagem.

**Dependências**: Zap, Viper

**Interface Pública**:

```go
type CacheService interface {
    GetCachePath(language Language) string
    InitializeCache(language Language) error
    CleanCache(language Language, olderThan time.Duration) error
    GetCacheSize(language Language) (int64, error)
}
```

**Implementação**:
- Estrutura de diretórios:
  - `/var/cache/oci-build/deps/maven` (Java/Maven)
  - `/var/cache/oci-build/deps/gradle` (Java/Gradle)
  - `/var/cache/oci-build/deps/nuget` (.NET)
  - `/var/cache/oci-build/deps/go` (Go modules)
- Limpeza periódica de cache antigo
- Monitoramento de uso de disco
- Testes unitários para todas as operações

### 7. NATS Client (libs/nats-client)

**Responsabilidade**: Cliente NATS compartilhado entre serviços.

**Dependências**: NATS.go, Zap, Viper

**Interface Pública**:

```go
type NATSClient interface {
    Connect(url string) error
    Publish(subject string, data []byte) error
    Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
    Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error)
    Close()
}
```

**Implementação**:
- Wrapper sobre nats.Conn
- Reconexão automática
- Logging de mensagens
- Testes unitários com NATS test server

### 8. Shared Library (libs/shared)

**Responsabilidade**: Tipos e utilitários compartilhados.

**Dependências**: Viper, Zap

**Conteúdo**:
- Tipos de dados (BuildJob, JobStatus, PhaseMetric)
- Configuração (Config struct com Viper)
- Utilitários (validação, parsing)
- Constantes compartilhadas

**Configuração Compartilhada**:

Todos os serviços utilizam Viper para carregar configuração de arquivos YAML com suporte a:
- Variáveis de ambiente (override de valores)
- Valores padrão
- Validação de configuração obrigatória
- Hot reload de configuração (opcional)

**Exemplo de uso do Viper**:

```go
func LoadConfig(configPath string) (*Config, error) {
    viper.SetConfigFile(configPath)
    viper.SetConfigType("yaml")
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    return &config, nil
}
```

## Data Models

### BuildJob

Representa um job de build na fila.

```go
type BuildJob struct {
    ID           string          `json:"id"`
    Repository   RepositoryInfo  `json:"repository"`
    CommitHash   string          `json:"commit_hash"`
    CommitAuthor string          `json:"commit_author"`
    CommitMsg    string          `json:"commit_message"`
    Branch       string          `json:"branch"`
    Status       JobStatus       `json:"status"`
    CreatedAt    time.Time       `json:"created_at"`
    StartedAt    *time.Time      `json:"started_at,omitempty"`
    CompletedAt  *time.Time      `json:"completed_at,omitempty"`
    Duration     time.Duration   `json:"duration"`
    Error        string          `json:"error,omitempty"`
    Phases       []PhaseMetric   `json:"phases"`
}
```

### PhaseMetric

Métricas de uma fase do build.

```go
type PhaseMetric struct {
    Phase     BuildPhase    `json:"phase"`
    StartTime time.Time     `json:"start_time"`
    EndTime   time.Time     `json:"end_time"`
    Duration  time.Duration `json:"duration"`
    Success   bool          `json:"success"`
    Error     string        `json:"error,omitempty"`
}
```

### Configuration

Configuração do sistema usando Viper e arquivos YAML.

```go
type Config struct {
    Server struct {
        Port            int    `mapstructure:"port"`
        ReadTimeout     int    `mapstructure:"read_timeout"`
        WriteTimeout    int    `mapstructure:"write_timeout"`
        ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
    } `mapstructure:"server"`
    
    NATS struct {
        URL string `mapstructure:"url"`
    } `mapstructure:"nats"`
    
    GitHub struct {
        WebhookSecret string `mapstructure:"webhook_secret"`
    } `mapstructure:"github"`
    
    Worker struct {
        PoolSize   int `mapstructure:"pool_size"`
        Timeout    int `mapstructure:"timeout"`
        MaxRetries int `mapstructure:"max_retries"`
    } `mapstructure:"worker"`
    
    Build struct {
        CodeCachePath  string `mapstructure:"code_cache_path"`
        BuildCachePath string `mapstructure:"build_cache_path"`
    } `mapstructure:"build"`
    
    Logging struct {
        Level  string `mapstructure:"level"`
        Format string `mapstructure:"format"`
    } `mapstructure:"logging"`
}
```

**Arquivos de Configuração**:

**apps/api-service/config.yaml**:
```yaml
server:
  port: 8080
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 10

nats:
  url: "nats://localhost:4222"
  
github:
  webhook_secret: "${GITHUB_WEBHOOK_SECRET}"
  
logging:
  level: "info"
  format: "json"
```

**apps/worker-service/config.yaml**:
```yaml
nats:
  url: "nats://localhost:4222"
  
worker:
  pool_size: 5
  timeout: 3600
  max_retries: 3
  
build:
  code_cache_path: "/var/cache/oci-build/repos"
  build_cache_path: "/var/cache/oci-build/deps"
  
logging:
  level: "info"
  format: "json"
```

### WebhookPayload

Payload recebido do GitHub.

```go
type WebhookPayload struct {
    Ref        string `json:"ref"`
    After      string `json:"after"`
    Repository struct {
        Name     string `json:"name"`
        FullName string `json:"full_name"`
        CloneURL string `json:"clone_url"`
        Owner    struct {
            Login string `json:"login"`
        } `json:"owner"`
    } `json:"repository"`
    HeadCommit struct {
        ID      string `json:"id"`
        Message string `json:"message"`
        Author  struct {
            Name  string `json:"name"`
            Email string `json:"email"`
        } `json:"author"`
    } `json:"head_commit"`
}
```

### Docker Compose Configuration

O sistema inclui um `docker-compose.yml` para desenvolvimento e testes locais:

```yaml
version: '3.8'

services:
  nats:
    image: nats:latest
    ports:
      - "4222:4222"
      - "8222:8222"
    command: "-js -m 8222"
    
  api-service:
    build:
      context: .
      dockerfile: apps/api-service/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - NATS_URL=nats://nats:4222
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}
      - LOG_LEVEL=debug
    volumes:
      - ./cache:/var/cache/oci-build
      - ./logs:/var/log/oci-build
    depends_on:
      - nats
      
  worker-service:
    build:
      context: .
      dockerfile: apps/worker-service/Dockerfile
    environment:
      - NATS_URL=nats://nats:4222
      - LOG_LEVEL=debug
    volumes:
      - ./cache:/var/cache/oci-build
      - ./logs:/var/log/oci-build
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      - nats
    deploy:
      replicas: 2
```

### NX Build Configuration

O monorepo utiliza NX para gerenciar builds e dependências entre componentes:

**nx.json**:
```json
{
  "tasksRunnerOptions": {
    "default": {
      "runner": "nx/tasks-runners/default",
      "options": {
        "cacheableOperations": ["build", "test", "lint"]
      }
    }
  },
  "targetDefaults": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["{projectRoot}/dist"]
    },
    "test": {
      "dependsOn": ["build"]
    }
  }
}
```

**project.json** (exemplo para api-service):
```json
{
  "name": "api-service",
  "targets": {
    "build": {
      "executor": "@nrwl/go:build",
      "options": {
        "outputPath": "dist/apps/api-service",
        "main": "apps/api-service/main.go"
      }
    },
    "test": {
      "executor": "@nrwl/go:test",
      "options": {
        "codeCoverage": true
      }
    },
    "serve": {
      "executor": "@nrwl/go:serve",
      "options": {
        "buildTarget": "api-service:build"
      }
    }
  }
}
```

**Comandos NX**:
- `nx build api-service` - Build do API service
- `nx build worker-service` - Build do Worker service
- `nx test git-service` - Testes do Git service
- `nx affected:build` - Build apenas dos projetos afetados
- `nx affected:test` - Testes apenas dos projetos afetados
- `nx run-many --target=build --all` - Build de todos os projetos


## Correctness Properties

*Uma propriedade é uma característica ou comportamento que deve ser verdadeiro em todas as execuções válidas de um sistema - essencialmente, uma declaração formal sobre o que o sistema deve fazer. Propriedades servem como ponte entre especificações legíveis por humanos e garantias de corretude verificáveis por máquina.*

### Propriedade 1: Validação de Assinatura de Webhook

*Para qualquer* requisição de webhook recebida, se a assinatura HMAC-SHA256 não corresponder ao secret configurado, então o sistema deve retornar HTTP 401 e não processar o webhook.

**Valida: Requisitos 1.1, 10.4**

### Propriedade 2: Extração Completa de Informações de Webhook

*Para qualquer* webhook válido do GitHub, o sistema deve extrair corretamente todas as informações necessárias (repository URL, owner, name, commit hash, branch, author, message) e criar um BuildJob com esses dados.

**Valida: Requisitos 1.2**

### Propriedade 3: Enfileiramento de Webhooks Simultâneos

*Para qualquer* conjunto de webhooks válidos recebidos simultaneamente, todos devem ser adicionados à fila de builds e nenhum deve ser perdido.

**Valida: Requisitos 1.5**

### Propriedade 4: Sincronização de Repositório

*Para qualquer* repositório, se ele não existe localmente, então git clone deve ser executado; se existe, então git pull deve ser executado; e em ambos os casos o código local deve refletir o commit especificado.

**Valida: Requisitos 2.1, 2.2, 2.3**

### Propriedade 5: Fallback para Cache em Falha de Rede

*Para qualquer* operação git pull que falhe devido a erro de rede, se o repositório existe em cache, então o sistema deve usar o código em cache e registrar um aviso sem falhar o build.

**Valida: Requisitos 2.4**

### Propriedade 6: Captura de Saída de Build

*Para qualquer* execução de build NX, tanto stdout quanto stderr devem ser capturados completamente e armazenados no BuildJob.

**Valida: Requisitos 3.2**

### Propriedade 7: Interrupção em Falha de Build

*Para qualquer* build NX que retorne código de saída diferente de zero, o sistema deve marcar o BuildJob como failed, registrar o erro, e não prosseguir para construção de imagem.

**Valida: Requisitos 3.3**

### Propriedade 8: Progressão após Build Bem-Sucedido

*Para qualquer* build NX que retorne código de saída zero, o sistema deve prosseguir para a fase de construção de imagem OCI.

**Valida: Requisitos 3.4**

### Propriedade 9: Configuração de Cache por Linguagem

*Para qualquer* projeto detectado como Java, .NET ou Go, o sistema deve configurar as variáveis de ambiente apropriadas para que as ferramentas de build (Maven/Gradle/NuGet/Go) utilizem o cache local correspondente.

**Valida: Requisitos 4.2, 6.1, 6.2, 6.3**

### Propriedade 10: Persistência de Dependências em Cache

*Para qualquer* build que baixe dependências, essas dependências devem ser armazenadas no diretório de cache apropriado para a linguagem e estar disponíveis para builds subsequentes.

**Valida: Requisitos 4.3, 4.5**

### Propriedade 11: Detecção Automática de Linguagem

*Para qualquer* repositório contendo arquivos de configuração de linguagem (pom.xml, build.gradle, *.csproj, go.mod), o sistema deve detectar corretamente a linguagem correspondente.

**Valida: Requisitos 6.4**

### Propriedade 12: Suporte a Projetos Polyglot

*Para qualquer* repositório contendo múltiplos arquivos de configuração de linguagens diferentes, o sistema deve detectar todas as linguagens e configurar caches para todas elas.

**Valida: Requisitos 6.5**

### Propriedade 13: Localização de Dockerfile

*Para qualquer* repositório, o sistema deve buscar Dockerfile no diretório raiz e em subdiretórios comuns (./docker, ./build, etc.).

**Valida: Requisitos 5.2**

### Propriedade 14: Falha em Dockerfile Ausente

*Para qualquer* repositório que não contenha Dockerfile em nenhum dos locais esperados, o sistema deve falhar o BuildJob com mensagem de erro descritiva.

**Valida: Requisitos 5.3**

### Propriedade 15: Aplicação de Tags de Imagem

*Para qualquer* imagem OCI construída com sucesso, o sistema deve aplicar pelo menos duas tags: uma com o commit hash completo e outra com o nome do branch.

**Valida: Requisitos 5.4**

### Propriedade 16: Persistência de Imagem

*Para qualquer* imagem OCI construída com sucesso, a imagem deve estar disponível no storage local do buildah e ser listável via comando `buildah images`.

**Valida: Requisitos 5.5**

### Propriedade 17: Logging de Início e Fim de Job

*Para qualquer* BuildJob processado, deve existir uma entrada de log marcando o início (com timestamp) e outra marcando o fim (com timestamp e duração).

**Valida: Requisitos 7.1**

### Propriedade 18: Logging de Erros com Stack Trace

*Para qualquer* erro que ocorra durante o processamento de um BuildJob, o log deve conter o stack trace completo e contexto (job ID, fase, repositório).

**Valida: Requisitos 7.2**

### Propriedade 19: Métricas de Duração por Fase

*Para qualquer* BuildJob, cada fase (git_sync, nx_build, image_build) deve ter sua duração registrada individualmente nos logs.

**Valida: Requisitos 7.3**

### Propriedade 20: Logging de Informações de Commit

*Para qualquer* BuildJob, os logs devem conter commit hash, autor e mensagem do commit.

**Valida: Requisitos 7.4**

### Propriedade 21: Formato JSON de Logs

*Para qualquer* entrada de log gerada pelo sistema, ela deve ser um objeto JSON válido e parseável.

**Valida: Requisitos 7.5**

### Propriedade 22: Resposta JSON em Consultas de Status

*Para qualquer* requisição GET bem-sucedida aos endpoints de status, a resposta deve ter Content-Type application/json e conter JSON válido.

**Valida: Requisitos 8.3**

### Propriedade 23: Autenticação em Endpoints de Consulta

*Para qualquer* requisição aos endpoints de consulta sem token de autenticação válido, o sistema deve retornar HTTP 401.

**Valida: Requisitos 8.4**

### Propriedade 24: Códigos HTTP Apropriados

*Para qualquer* resposta da API, o código HTTP deve corresponder ao resultado: 200 para sucesso, 401 para não autorizado, 404 para não encontrado, 503 para sobrecarga, 500 para erro interno.

**Valida: Requisitos 8.5**

### Propriedade 25: Retry com Backoff Exponencial

*Para qualquer* operação de rede que falhe temporariamente, o sistema deve realizar até 3 tentativas com intervalos crescentes (1s, 2s, 4s).

**Valida: Requisitos 9.1**

### Propriedade 26: Preservação de Logs em Falha

*Para qualquer* BuildJob que falhe, todos os logs e estado do job devem ser preservados e acessíveis via API de consulta.

**Valida: Requisitos 9.2**

### Propriedade 27: Recuperação de Jobs em Reinicialização

*Para qualquer* BuildJob que estava em estado "running" quando o sistema foi encerrado, após reinicialização o job deve ser marcado como "failed" com mensagem indicando interrupção.

**Valida: Requisitos 9.3**

### Propriedade 28: Timeouts em Operações

*Para qualquer* operação de longa duração (git clone, build, image build), se ela exceder o timeout configurado, deve ser cancelada e o BuildJob deve falhar com erro de timeout.

**Valida: Requisitos 9.4**

### Propriedade 29: Rejeição em Sobrecarga

*Para qualquer* webhook recebido quando a fila de builds está cheia (tamanho máximo atingido), o sistema deve retornar HTTP 503 sem adicionar o job à fila.

**Valida: Requisitos 9.5**

### Propriedade 30: Validação de Configuração na Inicialização

*Para qualquer* configuração inválida (porta fora do range, timeout negativo, path inexistente), o sistema deve falhar na inicialização com mensagem de erro descritiva.

**Valida: Requisitos 10.2**

### Propriedade 31: Isolamento entre Builds

*Para quaisquer* dois BuildJobs executados simultaneamente, alterações feitas por um build (variáveis de ambiente, diretório de trabalho) não devem afetar o outro build.

**Valida: Requisitos 10.5**

## Error Handling

### Estratégia Geral

O sistema implementa tratamento de erros em múltiplas camadas:

1. **Validação de Entrada**: Rejeitar dados inválidos o mais cedo possível
2. **Retry com Backoff**: Tentar novamente operações que podem falhar temporariamente
3. **Graceful Degradation**: Usar cache quando operações de rede falham
4. **Fail Fast**: Interromper processamento quando erros irrecuperáveis ocorrem
5. **Logging Completo**: Registrar todos os erros com contexto suficiente para diagnóstico

### Categorias de Erros

#### Erros de Validação

- **Webhook inválido**: Retornar HTTP 401, não processar
- **Configuração inválida**: Falhar na inicialização com mensagem clara
- **Payload malformado**: Retornar HTTP 400 com detalhes do erro

#### Erros de Rede

- **Git clone/pull falha**: Retry até 3 vezes, usar cache se disponível
- **Timeout de rede**: Cancelar operação, falhar job com erro de timeout
- **DNS resolution falha**: Retry com backoff, falhar após 3 tentativas

#### Erros de Build

- **Build NX falha**: Capturar stderr, marcar job como failed, não prosseguir
- **Dockerfile não encontrado**: Falhar job com mensagem descritiva
- **Buildah falha**: Capturar erro, marcar job como failed

#### Erros de Sistema

- **Disco cheio**: Retornar HTTP 503, rejeitar novos jobs
- **Memória insuficiente**: Limitar workers, rejeitar novos jobs
- **Permissões insuficientes**: Falhar na inicialização com mensagem clara

### Recuperação de Falhas

#### Reinicialização do Sistema

1. Carregar estado de jobs do disco (se persistido)
2. Marcar jobs "running" como "failed" com mensagem de interrupção
3. Manter jobs "pending" na fila
4. Reprocessar jobs "pending" após inicialização completa

#### Falha de Worker

1. Detectar worker travado via timeout
2. Cancelar context do worker
3. Marcar job como failed
4. Iniciar novo worker para substituir

#### Corrupção de Cache

1. Detectar repositório corrompido via git fsck
2. Remover repositório corrompido
3. Realizar clone limpo
4. Continuar processamento

## Testing Strategy

### Abordagem Dual de Testes

O sistema será testado usando duas abordagens complementares:

1. **Testes Unitários**: Verificam exemplos específicos, casos extremos e condições de erro
2. **Testes Baseados em Propriedades**: Verificam propriedades universais através de múltiplas entradas geradas

Ambas as abordagens são necessárias para cobertura abrangente. Testes unitários capturam bugs concretos, enquanto testes de propriedade verificam corretude geral.

### Configuração de Testes de Propriedade

- **Biblioteca**: Utilizaremos `gopter` (biblioteca de property-based testing para Go)
- **Iterações**: Mínimo de 100 iterações por teste de propriedade
- **Tagging**: Cada teste deve referenciar a propriedade do design
- **Formato de Tag**: `// Feature: oci-build-system, Property N: [texto da propriedade]`

### Testes Unitários

Todos os componentes devem ter testes unitários abrangentes:

**API Service**:
- Testes de handlers (webhook, status, health)
- Testes de middleware (autenticação, logging)
- Testes de validação de payload
- Mocks de NATS client

**Worker Service**:
- Testes de orchestrator
- Testes de processamento de mensagens NATS
- Testes de coordenação de fases
- Mocks de serviços auxiliares

**Git Service**:
- Testes de clone e pull
- Testes de detecção de repositório existente
- Testes de retry em falhas de rede
- Mocks de operações Git

**NX Service**:
- Testes de execução de build
- Testes de detecção de linguagem
- Testes de configuração de cache
- Mocks de execução de comandos

**Image Service**:
- Testes de build de imagem
- Testes de aplicação de tags
- Testes de validação de Dockerfile
- Mocks de buildah

**Cache Service**:
- Testes de inicialização de cache
- Testes de limpeza de cache
- Testes de cálculo de tamanho
- Testes de estrutura de diretórios

**NATS Client**:
- Testes de conexão
- Testes de publish/subscribe
- Testes de request/reply
- Testes de reconexão

Focar em:

- **Exemplos específicos**: Webhook válido do GitHub, configuração padrão
- **Casos extremos**: Repositório vazio, commit sem mensagem, branch com caracteres especiais
- **Condições de erro**: Rede indisponível, disco cheio, timeout
- **Integração entre componentes**: Comunicação via interfaces

Evitar escrever muitos testes unitários para cenários que podem ser cobertos por testes de propriedade.

### Testes de Propriedade

Cada propriedade de corretude listada acima deve ser implementada como um teste de propriedade único. Exemplos:

**Propriedade 1: Validação de Assinatura**
```go
// Feature: oci-build-system, Property 1: Validação de assinatura de webhook
func TestProperty_WebhookSignatureValidation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("invalid signature returns 401", prop.ForAll(
        func(payload []byte, wrongSecret string) bool {
            // Gerar webhook com assinatura usando wrongSecret
            // Configurar sistema com secret diferente
            // Verificar que retorna 401
        },
        gen.SliceOf(gen.UInt8()),
        gen.AnyString(),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

**Propriedade 11: Detecção de Linguagem**
```go
// Feature: oci-build-system, Property 11: Detecção automática de linguagem
func TestProperty_LanguageDetection(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("detects language from config files", prop.ForAll(
        func(configFile string) bool {
            // Gerar repositório temporário com configFile
            // Executar detecção de linguagem
            // Verificar que linguagem correta é detectada
        },
        gen.OneConstOf("pom.xml", "build.gradle", "project.csproj", "go.mod"),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Testes de Integração

Testes integrados usando Robot Framework:

**Estrutura de Testes**:

```
tests/integration/
├── webhook.robot          # Testes de recepção de webhooks
├── build.robot            # Testes de fluxo completo de build
├── api.robot              # Testes de API REST
├── resources/
│   ├── keywords.robot     # Keywords customizadas
│   └── variables.robot    # Variáveis de teste
└── fixtures/
    ├── sample-java-repo/
    ├── sample-dotnet-repo/
    └── sample-go-repo/
```

**Cenários de Teste**:

**webhook.robot**:
- Enviar webhook válido e verificar enfileiramento
- Enviar webhook com assinatura inválida e verificar rejeição
- Enviar múltiplos webhooks simultâneos
- Verificar parsing correto de payload

**build.robot**:
- Build completo de projeto Java
- Build completo de projeto .NET
- Build completo de projeto Go
- Build com falha (código não compila)
- Build com Dockerfile ausente
- Build com cache de dependências
- Build sem cache de dependências

**api.robot**:
- Consultar status de build existente
- Consultar status de build inexistente
- Listar histórico de builds
- Health check
- Autenticação com token válido
- Autenticação com token inválido

**Configuração**:
- Usar docker-compose para subir ambiente de teste
- Repositórios de teste em fixtures/
- Cleanup automático após cada teste
- Relatórios HTML gerados automaticamente

**Exemplo de Teste Robot**:

```robot
*** Settings ***
Library    RequestsLibrary
Library    Collections
Resource   resources/keywords.robot

*** Test Cases ***
Send Valid Webhook And Verify Build
    [Documentation]    Envia webhook válido e verifica que build é processado
    ${payload}=    Load Webhook Payload    sample-java-repo
    ${signature}=    Calculate HMAC Signature    ${payload}
    ${response}=    POST    ${API_URL}/webhook    
    ...    json=${payload}
    ...    headers=X-Hub-Signature-256=${signature}
    Should Be Equal As Numbers    ${response.status_code}    202
    ${job_id}=    Get From Dictionary    ${response.json()}    job_id
    Wait Until Build Completes    ${job_id}    timeout=300s
    ${status}=    Get Build Status    ${job_id}
    Should Be Equal    ${status}    success
```

- **End-to-end**: Webhook → Build completo → Imagem criada
- **Componentes**: Git Service + Cache Service + NX Service
- **API**: Todos os endpoints REST
- **Concorrência**: Múltiplos builds simultâneos
- **NATS**: Comunicação entre serviços

### Testes de Performance

- **Throughput**: Quantos builds por minuto o sistema suporta
- **Latência**: Tempo desde webhook até início do build
- **Uso de recursos**: Memória e CPU durante builds simultâneos
- **Cache hit rate**: Efetividade do cache de dependências

### Ambiente de Teste

- **Mocks**: GitHub API, buildah (para testes unitários rápidos)
- **NATS Test Server**: Para testes de integração de mensageria
- **Containers**: Ambiente isolado para testes de integração (docker-compose)
- **Repositórios de teste**: Projetos pequenos em Java, .NET e Go (em tests/integration/fixtures)
- **CI/CD**: Executar todos os testes em cada commit usando NX affected
- **Robot Framework**: Para testes end-to-end automatizados

### Cobertura de Código

- **Meta**: Mínimo 80% de cobertura de código
- **Foco**: Lógica de negócio e tratamento de erros
- **Exclusões**: Código gerado, structs de dados simples
