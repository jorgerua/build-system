# Image Service

Biblioteca para construção de imagens OCI usando buildah.

## Funcionalidades

- Construção de imagens OCI com buildah
- Localização automática de Dockerfile em locais comuns
- Validação de Dockerfile
- Aplicação de tags baseadas em commit hash e branch
- Suporte a build args
- Logging estruturado com Zap

## Interface

```go
type ImageService interface {
    BuildImage(ctx context.Context, config ImageConfig) (*ImageResult, error)
    TagImage(imageID string, tags []string) error
}
```

## Uso

```go
import (
    imageservice "github.com/oci-build-system/libs/image-service"
    "go.uber.org/zap"
)

// Criar serviço
logger, _ := zap.NewProduction()
service := imageservice.NewImageService(logger)

// Gerar tags
tags := imageservice.GenerateImageTags("myapp", "abc123def456", "main")

// Construir imagem
config := imageservice.ImageConfig{
    ContextPath: "/path/to/repo",
    Tags:        tags,
    BuildArgs: map[string]string{
        "VERSION": "1.0.0",
    },
}

result, err := service.BuildImage(ctx, config)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Image built: %s\n", result.ImageID)
```

## Localização de Dockerfile

O serviço procura Dockerfile nos seguintes locais (em ordem):

1. Path especificado em `DockerfilePath` (se fornecido)
2. `Dockerfile` (raiz do repositório)
3. `dockerfile` (raiz do repositório)
4. `docker/Dockerfile`
5. `build/Dockerfile`
6. `.docker/Dockerfile`
7. `deployment/Dockerfile`

## Validação

O serviço valida:

- Existência do context path
- Existência do Dockerfile
- Dockerfile não vazio
- Dockerfile contém pelo menos uma instrução FROM
- Pelo menos uma tag é fornecida

## Tags

A função `GenerateImageTags` gera automaticamente tags baseadas em:

- Commit hash completo: `myapp:abc123def456`
- Branch: `myapp:main`
- Latest (se branch for main/master): `myapp:latest`

## Requisitos

- buildah instalado no sistema
- Permissões para executar buildah

## Dependências

- go.uber.org/zap - Logging estruturado
