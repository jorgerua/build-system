# Git Service

Biblioteca para gerenciamento de operações Git no OCI Build System.

## Funcionalidades

- **SyncRepository**: Sincroniza repositórios (clone ou pull) e faz checkout de commits específicos
- **RepositoryExists**: Verifica se um repositório existe no cache local
- **GetLocalPath**: Calcula o path local para um repositório baseado na URL

## Características

- Retry automático com backoff exponencial para operações de rede
- Fallback para cache local em caso de falha de rede no pull
- Logging detalhado de todas as operações
- Validação de repositórios Git
- Suporte a context para cancelamento de operações

## Uso

```go
import (
    gitservice "github.com/oci-build-system/libs/git-service"
    "github.com/oci-build-system/libs/shared"
    "go.uber.org/zap"
)

// Criar logger
logger, _ := zap.NewProduction()

// Configurar serviço
config := gitservice.Config{
    CodeCachePath: "/var/cache/oci-build/repos",
    MaxRetries:    3,
    RetryDelay:    time.Second,
}

// Criar instância
gitSvc := gitservice.NewGitService(config, logger)

// Sincronizar repositório
repo := shared.RepositoryInfo{
    URL:   "https://github.com/owner/repo.git",
    Name:  "repo",
    Owner: "owner",
    Branch: "main",
}

localPath, err := gitSvc.SyncRepository(ctx, repo, "abc123...")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Repository synced to:", localPath)
```

## Estrutura de Cache

Os repositórios são armazenados em:
```
/var/cache/oci-build/repos/
├── owner1/
│   ├── repo1/
│   └── repo2/
└── owner2/
    └── repo3/
```

## Retry e Resiliência

- Operações de clone e pull são retentadas até 3 vezes (configurável)
- Backoff exponencial: 1s, 2s, 4s entre tentativas
- Pull com falha usa cache existente e registra aviso (não falha o build)
- Suporte a context.Context para cancelamento e timeout

## Validações

- Valida informações do repositório antes de sincronizar
- Verifica se commit hash foi fornecido
- Valida integridade do repositório Git local
- Cria diretórios de cache automaticamente

## Logging

Todos os eventos são registrados com Zap:
- Info: Operações bem-sucedidas
- Debug: Detalhes de tentativas e progresso
- Warn: Falhas de retry e fallback para cache
- Error: Erros irrecuperáveis

## Requisitos Atendidos

- **2.1**: Verificação de repositório em cache
- **2.2**: Clone de repositório novo
- **2.3**: Pull de repositório existente
- **9.1**: Retry com backoff exponencial
