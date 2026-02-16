# Guia de Desenvolvimento

Este documento fornece informações detalhadas sobre a estrutura do projeto e como desenvolver novos recursos.

## 📁 Estrutura do Monorepo

O projeto utiliza NX para gerenciar um monorepo com múltiplos serviços e bibliotecas:

```
oci-build-system/
├── apps/                      # Aplicações executáveis
│   ├── api-service/          # Serviço de API REST
│   │   ├── main.go           # Entry point
│   │   ├── config.yaml       # Configuração
│   │   ├── handlers/         # HTTP handlers
│   │   │   ├── webhook.go
│   │   │   ├── status.go
│   │   │   └── health.go
│   │   ├── middleware/       # HTTP middleware
│   │   │   ├── auth.go
│   │   │   └── logging.go
│   │   ├── Dockerfile        # Container image
│   │   ├── project.json      # Configuração NX
│   │   └── *_test.go         # Testes
│   │
│   └── worker-service/       # Serviço de processamento
│       ├── main.go
│       ├── config.yaml
│       ├── orchestrator.go   # Coordenação de builds
│       ├── worker.go         # Pool de workers
│       ├── Dockerfile
│       ├── project.json
│       └── *_test.go
│
├── libs/                      # Bibliotecas compartilhadas
│   ├── shared/               # Tipos e utilitários
│   │   ├── types.go          # Structs compartilhados
│   │   ├── config.go         # Carregamento de config
│   │   ├── project.json
│   │   └── *_test.go
│   │
│   ├── nats-client/          # Cliente NATS
│   │   ├── client.go
│   │   ├── project.json
│   │   └── *_test.go
│   │
│   ├── git-service/          # Operações Git
│   │   ├── manager.go
│   │   ├── project.json
│   │   └── *_test.go
│   │
│   ├── nx-service/           # Builds NX
│   │   ├── builder.go
│   │   ├── project.json
│   │   └── *_test.go
│   │
│   ├── image-service/        # Builds OCI
│   │   ├── service.go
│   │   ├── project.json
│   │   └── *_test.go
│   │
│   └── cache-service/        # Gerenciamento de cache
│       ├── manager.go
│       ├── project.json
│       └── *_test.go
│
├── tests/                     # Testes de integração
│   └── integration/
│       ├── webhook.robot     # Testes de webhook
│       ├── build.robot       # Testes de build
│       ├── api.robot         # Testes de API
│       └── fixtures/         # Repositórios de teste
│
├── nx.json                    # Configuração global do NX
├── go.mod                     # Dependências Go (raiz)
├── docker-compose.yml         # Ambiente de desenvolvimento
└── README.md                  # Documentação principal
```

## 🔧 Configuração do Ambiente de Desenvolvimento

### 1. Instalar Dependências

```bash
# Go
go version  # Requer 1.21+

# Node.js e NX
node --version  # Requer 18+
npm install -g nx

# Buildah (Linux)
sudo apt-get install buildah

# Buildah (macOS)
brew install buildah
```

### 2. Instalar Dependências do Projeto

```bash
# Dependências Go
go mod download

# Dependências Node (para NX)
npm install
```

### 3. Configurar IDE

#### VS Code

Instale as extensões recomendadas:
- Go (golang.go)
- Nx Console (nrwl.angular-console)

Configuração recomendada (`.vscode/settings.json`):
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v"],
  "editor.formatOnSave": true
}
```

#### GoLand

Configure o Go SDK e habilite o suporte a módulos Go.

## 🏗️ Adicionando Novos Serviços

### 1. Criar Estrutura do Serviço

```bash
# Criar diretório
mkdir -p apps/novo-service

# Criar arquivos base
touch apps/novo-service/main.go
touch apps/novo-service/config.yaml
touch apps/novo-service/Dockerfile
touch apps/novo-service/project.json
```

### 2. Implementar o Serviço

**main.go**:
```go
package main

import (
    "context"
    "log"
    
    "go.uber.org/fx"
    "go.uber.org/zap"
    "github.com/spf13/viper"
)

func main() {
    app := fx.New(
        fx.Provide(
            NewLogger,
            NewConfig,
            NewService,
        ),
        fx.Invoke(Run),
    )
    
    app.Run()
}

func NewLogger() (*zap.Logger, error) {
    return zap.NewProduction()
}

func NewConfig() (*Config, error) {
    viper.SetConfigFile("config.yaml")
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}

func NewService(logger *zap.Logger, config *Config) *Service {
    return &Service{
        logger: logger,
        config: config,
    }
}

func Run(lc fx.Lifecycle, service *Service) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return service.Start()
        },
        OnStop: func(ctx context.Context) error {
            return service.Shutdown(ctx)
        },
    })
}
```

### 3. Configurar NX

**project.json**:
```json
{
  "name": "novo-service",
  "targets": {
    "build": {
      "executor": "@nrwl/go:build",
      "options": {
        "outputPath": "dist/apps/novo-service",
        "main": "apps/novo-service/main.go"
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
        "buildTarget": "novo-service:build"
      }
    }
  }
}
```

### 4. Criar Dockerfile

**Dockerfile**:
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY apps/novo-service/ ./apps/novo-service/
COPY libs/ ./libs/

RUN CGO_ENABLED=0 GOOS=linux go build -o /novo-service ./apps/novo-service

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /novo-service .
COPY apps/novo-service/config.yaml .

EXPOSE 8080

CMD ["./novo-service"]
```

### 5. Adicionar ao Docker Compose

**docker-compose.yml**:
```yaml
services:
  novo-service:
    build:
      context: .
      dockerfile: apps/novo-service/Dockerfile
    ports:
      - "8081:8080"
    environment:
      - NATS_URL=nats://nats:4222
      - LOG_LEVEL=debug
    depends_on:
      - nats
```

## 📚 Adicionando Novas Bibliotecas

### 1. Criar Estrutura da Biblioteca

```bash
mkdir -p libs/nova-lib
touch libs/nova-lib/service.go
touch libs/nova-lib/service_test.go
touch libs/nova-lib/project.json
touch libs/nova-lib/README.md
```

### 2. Implementar a Interface

**service.go**:
```go
package novalib

import (
    "context"
    "go.uber.org/zap"
)

// Interface pública
type NovaLibService interface {
    DoSomething(ctx context.Context, input string) (string, error)
}

// Implementação
type service struct {
    logger *zap.Logger
}

func NewService(logger *zap.Logger) NovaLibService {
    return &service{
        logger: logger,
    }
}

func (s *service) DoSomething(ctx context.Context, input string) (string, error) {
    s.logger.Info("doing something", zap.String("input", input))
    // Implementação aqui
    return "result", nil
}
```

### 3. Escrever Testes

**service_test.go**:
```go
package novalib

import (
    "context"
    "testing"
    
    "go.uber.org/zap"
    "github.com/stretchr/testify/assert"
)

func TestDoSomething(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    service := NewService(logger)
    
    result, err := service.DoSomething(context.Background(), "test")
    
    assert.NoError(t, err)
    assert.Equal(t, "result", result)
}
```

### 4. Configurar NX

**project.json**:
```json
{
  "name": "nova-lib",
  "targets": {
    "build": {
      "executor": "@nrwl/go:build",
      "options": {
        "outputPath": "dist/libs/nova-lib"
      }
    },
    "test": {
      "executor": "@nrwl/go:test",
      "options": {
        "codeCoverage": true
      }
    }
  }
}
```

## 🧪 Adicionando Novos Testes

### Testes Unitários

Crie arquivos `*_test.go` ao lado do código:

```go
package mypackage

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyFunction(t *testing.T) {
    result := MyFunction("input")
    assert.Equal(t, "expected", result)
}
```

### Testes de Propriedade

Use `gopter` para testes baseados em propriedades:

```go
package mypackage

import (
    "testing"
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

func TestProperty_MyFunction(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("MyFunction always returns non-empty string", 
        prop.ForAll(
            func(input string) bool {
                result := MyFunction(input)
                return len(result) > 0
            },
            gen.AnyString(),
        ),
    )
    
    properties.TestingRun(t)
}
```

### Testes de Integração (Robot Framework)

Crie arquivos `.robot` em `tests/integration/`:

```robot
*** Settings ***
Library    RequestsLibrary
Library    Collections

*** Variables ***
${API_URL}    http://localhost:8080

*** Test Cases ***
Test New Feature
    Create Session    api    ${API_URL}
    ${response}=    GET On Session    api    /new-endpoint
    Should Be Equal As Integers    ${response.status_code}    200
```

## 🔨 Comandos NX Úteis

### Build

```bash
# Build de um projeto específico
nx build api-service

# Build de todos os projetos
nx run-many --target=build --all

# Build apenas dos projetos afetados por mudanças
nx affected:build

# Build com cache limpo
nx build api-service --skip-nx-cache
```

### Test

```bash
# Testes de um projeto
nx test git-service

# Testes de todos os projetos
nx run-many --target=test --all

# Testes apenas dos projetos afetados
nx affected:test

# Testes com cobertura
nx test git-service --codeCoverage
```

### Lint

```bash
# Lint de um projeto
nx lint api-service

# Lint de todos os projetos
nx run-many --target=lint --all
```

### Visualização

```bash
# Visualizar grafo de dependências
nx graph

# Visualizar projetos afetados
nx affected:graph
```

### Cache

```bash
# Limpar cache do NX
nx reset

# Ver estatísticas de cache
nx show projects --with-target=build
```

## 🔍 Debugging

### Debugging Local

Use o debugger do Go:

```bash
# Instalar delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debugar um serviço
cd apps/api-service
dlv debug
```

### Debugging em Container

Adicione ao Dockerfile:

```dockerfile
# Instalar delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Expor porta do debugger
EXPOSE 2345

# Executar com delve
CMD ["dlv", "exec", "./api-service", "--headless", "--listen=:2345", "--api-version=2"]
```

### Logs Estruturados

Use Zap para logging:

```go
logger.Info("processing build",
    zap.String("job_id", jobID),
    zap.String("repository", repo),
    zap.Duration("duration", duration),
)
```

## 📊 Métricas e Observabilidade

### Adicionar Métricas

Use Prometheus para métricas:

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    buildsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "builds_total",
            Help: "Total number of builds",
        },
        []string{"status"},
    )
)

func init() {
    prometheus.MustRegister(buildsTotal)
}

func recordBuild(status string) {
    buildsTotal.WithLabelValues(status).Inc()
}
```

## 🎯 Boas Práticas

### Código

- Use interfaces para abstrações
- Implemente dependency injection com FX
- Escreva testes para todo código novo
- Use context para cancelamento e timeouts
- Valide entradas o mais cedo possível

### Git

- Use commits semânticos: `feat:`, `fix:`, `docs:`, `test:`
- Crie branches descritivas: `feature/nova-funcionalidade`
- Faça pull requests pequenos e focados
- Adicione testes antes de fazer merge

### Documentação

- Documente funções públicas com comentários Go
- Atualize README.md quando adicionar features
- Mantenha DEVELOPMENT.md atualizado
- Adicione exemplos de uso

## 🚀 Workflow de Desenvolvimento

1. **Criar branch**: `git checkout -b feature/minha-feature`
2. **Implementar**: Escrever código e testes
3. **Testar localmente**: `nx test meu-projeto`
4. **Build**: `nx build meu-projeto`
5. **Testar integração**: `docker-compose up`
6. **Commit**: `git commit -m "feat: adicionar nova feature"`
7. **Push**: `git push origin feature/minha-feature`
8. **Pull Request**: Criar PR no GitHub
9. **Review**: Aguardar aprovação
10. **Merge**: Fazer merge após aprovação

## 📞 Suporte

Para dúvidas ou problemas:
- Abra uma issue no GitHub
- Consulte a documentação do NX: https://nx.dev
- Consulte a documentação do Go: https://go.dev/doc
