# API Service

O API Service é o ponto de entrada HTTP do OCI Build System. Ele recebe webhooks do GitHub, expõe endpoints de consulta de status, e publica jobs de build no NATS para processamento assíncrono.

## Funcionalidades

- **Recepção de Webhooks**: Recebe e valida webhooks do GitHub com assinatura HMAC-SHA256
- **Consulta de Status**: Endpoints REST para consultar status de builds
- **Health Check**: Endpoint para verificar saúde do serviço
- **Autenticação**: Middleware de autenticação baseado em token Bearer
- **Logging Estruturado**: Logs em formato JSON com Zap

## Endpoints

### POST /webhook
Recebe webhooks do GitHub.

**Headers:**
- `X-Hub-Signature-256`: Assinatura HMAC-SHA256 do payload
- `Content-Type`: application/json

**Response:**
- `202 Accepted`: Webhook aceito e job enfileirado
- `401 Unauthorized`: Assinatura inválida
- `400 Bad Request`: Payload inválido

**Exemplo:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "accepted",
  "message": "build job enqueued successfully"
}
```

### GET /builds/:id
Consulta o status de um build específico. Requer autenticação.

**Headers:**
- `Authorization`: Bearer <token>

**Response:**
- `200 OK`: Build encontrado
- `404 Not Found`: Build não encontrado
- `401 Unauthorized`: Token inválido

**Exemplo:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "repository": {
    "url": "https://github.com/owner/repo.git",
    "name": "repo",
    "owner": "owner",
    "branch": "main"
  },
  "commit_hash": "abc123def456",
  "commit_author": "John Doe",
  "commit_message": "Fix bug",
  "branch": "main",
  "status": "success",
  "created_at": "2024-01-01T10:00:00Z",
  "started_at": "2024-01-01T10:00:05Z",
  "completed_at": "2024-01-01T10:05:00Z",
  "duration": 295000000000,
  "phases": [...]
}
```

### GET /builds
Lista histórico de builds. Requer autenticação.

**Headers:**
- `Authorization`: Bearer <token>

**Query Parameters:**
- `repository`: Filtrar por repositório (opcional)
- `status`: Filtrar por status (opcional)
- `limit`: Número máximo de resultados (padrão: 50)

**Response:**
- `200 OK`: Lista de builds

**Exemplo:**
```json
{
  "builds": [...],
  "total": 42
}
```

### GET /health
Verifica a saúde do serviço.

**Response:**
- `200 OK`: Serviço saudável
- `503 Service Unavailable`: Serviço com problemas

**Exemplo:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T10:00:00Z",
  "uptime": "2h30m15s",
  "checks": {
    "nats": "connected"
  }
}
```

## Configuração

O serviço é configurado via arquivo `config.yaml` ou variáveis de ambiente.

### config.yaml

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

auth:
  token: "${API_AUTH_TOKEN}"
  
logging:
  level: "info"
  format: "json"
```

### Variáveis de Ambiente

- `CONFIG_PATH`: Caminho para o arquivo de configuração (padrão: config.yaml)
- `GITHUB_WEBHOOK_SECRET`: Secret para validação de webhooks do GitHub
- `API_AUTH_TOKEN`: Token para autenticação nos endpoints de consulta
- `SERVER_PORT`: Porta do servidor HTTP
- `NATS_URL`: URL do servidor NATS
- `LOGGING_LEVEL`: Nível de log (debug, info, warn, error, fatal)

## Executando

### Desenvolvimento

```bash
# Instalar dependências
go mod tidy

# Executar testes
go test -v ./...

# Executar serviço
go run main.go
```

### Produção

```bash
# Build
go build -o api-service

# Executar
./api-service
```

### Docker

```bash
# Build da imagem
docker build -t api-service .

# Executar container
docker run -p 8080:8080 \
  -e GITHUB_WEBHOOK_SECRET=your-secret \
  -e API_AUTH_TOKEN=your-token \
  -e NATS_URL=nats://nats:4222 \
  api-service
```

## Testes

O serviço possui três tipos de testes:

### Testes Unitários
Testam componentes individuais com casos específicos.

```bash
go test -v ./handlers
go test -v ./middleware
```

### Testes de Propriedade
Testam propriedades universais com entradas geradas automaticamente.

```bash
go test -v -run TestProperty ./...
```

### Cobertura

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Arquitetura

O serviço utiliza:

- **Gin**: Framework HTTP
- **Uber FX**: Dependency injection e lifecycle management
- **Zap**: Logging estruturado
- **Viper**: Gerenciamento de configuração
- **NATS**: Message broker para comunicação assíncrona

### Fluxo de Requisição

1. Requisição HTTP chega no Gin router
2. Middleware de logging registra a requisição
3. Middleware de autenticação valida token (se aplicável)
4. Handler processa a requisição
5. Para webhooks: publica job no NATS
6. Para consultas: faz request/reply no NATS
7. Resposta é enviada ao cliente
8. Middleware de logging registra a resposta

## Dependências

- `github.com/gin-gonic/gin`: Framework HTTP
- `go.uber.org/fx`: Dependency injection
- `go.uber.org/zap`: Logging estruturado
- `github.com/spf13/viper`: Configuração
- `github.com/nats-io/nats.go`: Cliente NATS
- `github.com/google/uuid`: Geração de UUIDs
- `github.com/stretchr/testify`: Assertions para testes
- `github.com/leanovate/gopter`: Property-based testing
