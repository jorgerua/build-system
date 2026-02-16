# Cache Service

Biblioteca para gerenciamento de cache de dependências por linguagem de programação.

## Funcionalidades

- **GetCachePath**: Retorna o caminho do cache para uma linguagem específica
- **InitializeCache**: Cria a estrutura de diretórios para o cache
- **CleanCache**: Remove arquivos de cache mais antigos que uma duração especificada
- **GetCacheSize**: Calcula o tamanho total do cache para uma linguagem

## Estrutura de Cache

O cache é organizado por linguagem:

```
/var/cache/oci-build/deps/
├── maven/      # Cache Maven (Java)
├── gradle/     # Cache Gradle (Java)
├── nuget/      # Cache NuGet (.NET)
└── go/         # Cache Go modules
```

## Uso

```go
import (
    "time"
    cacheservice "github.com/oci-build-system/libs/cache-service"
    "github.com/oci-build-system/libs/shared"
    "go.uber.org/zap"
)

// Criar instância do cache service
logger, _ := zap.NewProduction()
cacheService := cacheservice.NewCacheService("/var/cache/oci-build/deps", logger)

// Inicializar cache para Java
err := cacheService.InitializeCache(shared.LanguageJava)

// Obter caminho do cache
cachePath := cacheService.GetCachePath(shared.LanguageJava)

// Limpar cache antigo (mais de 30 dias)
err = cacheService.CleanCache(shared.LanguageJava, 30*24*time.Hour)

// Obter tamanho do cache
size, err := cacheService.GetCacheSize(shared.LanguageJava)
```

## Linguagens Suportadas

- **Java**: Maven e Gradle
- **.NET**: NuGet
- **Go**: Go modules

## Validações

- Valida se a linguagem é suportada antes de qualquer operação
- Cria diretórios automaticamente se não existirem
- Registra operações usando Zap logger estruturado
- Trata erros de permissão e I/O gracefully
