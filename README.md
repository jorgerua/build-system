# OCI Build System

Sistema automatizado de build OCI distribuído usando Go, NATS, NX, e buildah.

## Estrutura do Projeto

```
oci-build-system/
├── apps/                    # Aplicações principais
│   ├── api-service/        # Serviço de API REST
│   └── worker-service/     # Serviço de processamento de builds
├── libs/                    # Bibliotecas compartilhadas
│   ├── git-service/        # Operações Git
│   ├── nx-service/         # Builds NX
│   ├── image-service/      # Construção de imagens OCI
│   ├── cache-service/      # Gerenciamento de cache
│   ├── nats-client/        # Cliente NATS
│   └── shared/             # Tipos e utilitários compartilhados
├── tests/                   # Testes de integração
│   └── integration/        # Testes Robot Framework
└── tools/                   # Scripts e ferramentas
    └── scripts/
```

## Pré-requisitos

- Go 1.21+
- Node.js 18+ (para NX)
- Docker e Docker Compose
- Buildah (para construção de imagens OCI)

## Instalação

1. Clone o repositório
2. Instale as dependências Go:
   ```bash
   go mod download
   ```
3. Instale as dependências Node.js:
   ```bash
   npm install
   ```

## Desenvolvimento

### Iniciar serviços localmente

```bash
docker-compose up
```

### Build com NX

```bash
# Build de um serviço específico
nx build api-service

# Build de todos os projetos
nx run-many --target=build --all

# Build apenas dos projetos afetados
nx affected:build
```

### Testes

```bash
# Testes de um serviço específico
nx test api-service

# Testes de todos os projetos
nx run-many --target=test --all
```

## Configuração

As configurações são carregadas de arquivos YAML e podem ser sobrescritas por variáveis de ambiente.

Exemplo de variáveis de ambiente:
```bash
export GITHUB_WEBHOOK_SECRET=your-secret-here
export NATS_URL=nats://localhost:4222
export LOG_LEVEL=debug
```

## Arquitetura

O sistema é composto por:

- **API Service**: Recebe webhooks do GitHub e expõe API REST
- **Worker Service**: Processa builds de forma assíncrona
- **NATS**: Message broker para comunicação entre serviços
- **Bibliotecas compartilhadas**: Lógica reutilizável para Git, NX, imagens OCI, e cache

## Licença

MIT
