# NX Service

Biblioteca para execução de builds NX com suporte a múltiplas linguagens.

## Funcionalidades

- Execução de builds NX com captura de stdout/stderr
- Detecção automática de linguagem (Java, .NET, Go)
- Configuração de cache por linguagem
- Descoberta de projetos NX no workspace
- Timeout configurável
- Logging estruturado com Zap

## Interface

```go
type NXService interface {
    Build(ctx context.Context, repoPath string, config BuildConfig) (*BuildResult, error)
    DetectProjects(repoPath string) ([]string, error)
}
```

## Uso

```go
import (
    "context"
    "time"
    
    nxservice "github.com/oci-build-system/libs/nx-service"
    "github.com/oci-build-system/libs/shared"
    "go.uber.org/zap"
)

// Criar logger
logger, _ := zap.NewProduction()

// Criar serviço
service := nxservice.NewNXService(logger)

// Configurar build
config := nxservice.BuildConfig{
    CachePath: "/var/cache/oci-build/deps",
    Language: shared.LanguageJava,
    Environment: map[string]string{
        "NODE_ENV": "production",
    },
}

// Executar build com timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer cancel()

result, err := service.Build(ctx, "/path/to/repo", config)
if err != nil {
    logger.Error("build failed", zap.Error(err))
    return
}

logger.Info("build completed",
    zap.Bool("success", result.Success),
    zap.Duration("duration", result.Duration),
)
```

## Detecção de Linguagem

O serviço detecta automaticamente a linguagem baseado em arquivos de configuração:

- **Java**: `pom.xml`, `build.gradle`, `build.gradle.kts`
- **.NET**: `*.csproj`
- **Go**: `go.mod`

A detecção busca no diretório raiz e em subdiretórios comuns (`apps`, `libs`, `packages`, `src`).

## Configuração de Cache

O serviço configura automaticamente variáveis de ambiente para cache por linguagem:

### Java
- `MAVEN_OPTS=-Dmaven.repo.local={cachePath}/maven`
- `GRADLE_USER_HOME={cachePath}/gradle`

### .NET
- `NUGET_PACKAGES={cachePath}/nuget`

### Go
- `GOMODCACHE={cachePath}/go`
- `GOCACHE={cachePath}/go/build-cache`

## Descoberta de Projetos

O método `DetectProjects` encontra projetos NX no workspace:

- Analisa `workspace.json` (NX clássico)
- Busca arquivos `project.json` (NX moderno)
- Ignora diretórios comuns (`node_modules`, `.git`, `dist`, `.nx`)

## Requisitos Validados

- **3.1**: Execução de comando NX build
- **3.2**: Captura de stdout e stderr
- **3.5**: Configuração de NX para utilizar cache local
- **6.4**: Detecção automática de linguagem

## Dependências

- `go.uber.org/zap` - Logging estruturado
- `github.com/oci-build-system/libs/shared` - Tipos compartilhados
