# OCI Build System

Sistema automatizado de build distribuído que recebe notificações de commits via webhook, realiza git pull de repositórios GitHub, executa builds utilizando NX, e gera imagens OCI compatíveis usando buildah.

## 🏗️ Arquitetura

O sistema é composto por microserviços escritos em Go que se comunicam via NATS:

```
GitHub Webhook → API Service → NATS → Worker Service → Build Pipeline
                                           ├─ Git Service
                                           ├─ NX Service
                                           ├─ Image Service
                                           └─ Cache Service
```

### Componentes Principais

- **API Service**: Recebe webhooks do GitHub e expõe API REST para consultas
- **Worker Service**: Processa builds de forma assíncrona usando pool de workers
- **Git Service**: Gerencia operações Git (clone, pull, checkout)
- **NX Service**: Executa builds usando NX com cache inteligente
- **Image Service**: Constrói imagens OCI usando buildah
- **Cache Service**: Gerencia cache de código e dependências
- **NATS**: Message broker para comunicação entre serviços

### Linguagens Suportadas

- ☕ Java (Maven/Gradle)
- 🔷 .NET (NuGet)
- 🐹 Go (Go modules)

## 🚀 Executando Localmente

### Pré-requisitos

- Docker e Docker Compose
- Go 1.21+
- Node.js 18+ (para NX)
- Buildah (para construção de imagens OCI)

### Configuração Inicial

1. Clone o repositório:
```bash
git clone <repository-url>
cd build-system
```

2. Copie o arquivo env de exemplo e check as variáveis de ambiente:
```bash
cp .env.example .env
cat .env
# Edite .env se necessário
```

3. Inicie os serviços com Docker Compose:
```bash
docker-compose up -d
```

4. Verifique se os serviços estão rodando:
```bash
docker-compose ps
curl http://localhost:8080/health
```

### Executando sem Docker

1. Inicie o NATS:
```bash
docker run -d -p 4222:4222 -p 8222:8222 nats:latest -js -m 8222
```

2. Build dos serviços:
```bash
nx build api-service
nx build worker-service
```

3. Execute os serviços:
```bash
# Terminal 1
./dist/apps/api-service/api-service

# Terminal 2
./dist/apps/worker-service/worker-service
```

## 🧪 Executando Testes

### Testes Unitários

Execute todos os testes unitários:
```bash
nx run-many --target=test --all
```

Execute testes de um componente específico:
```bash
nx test api-service
nx test worker-service
nx test git-service
```

### Testes de Propriedade (PBT)

Os testes de propriedade validam propriedades universais do sistema:
```bash
# Executar todos os testes incluindo PBT
nx run-many --target=test --all

# Executar apenas testes de propriedade de um componente
cd libs/git-service && go test -run Property
```

### Testes de Integração

Os testes de integração usam Robot Framework:
```bash
# Instalar dependências
pip install robotframework robotframework-requests

# Executar todos os testes de integração
cd tests/integration
robot .

# Executar suite específica
robot webhook.robot
robot build.robot
robot api.robot
```

### Cobertura de Código

```bash
# Gerar relatório de cobertura
nx run-many --target=test --all --codeCoverage

# Visualizar cobertura
go tool cover -html=coverage.out
```

## 🔗 Configuração de Webhooks GitHub

### 1. Gerar Secret

Gere um secret aleatório para validação de webhooks:
```bash
openssl rand -hex 32
```

### 2. Configurar no GitHub

1. Acesse as configurações do repositório
2. Vá em **Settings → Webhooks → Add webhook**
3. Configure:
   - **Payload URL**: `https://seu-dominio.com/webhook`
   - **Content type**: `application/json`
   - **Secret**: Cole o secret gerado
   - **Events**: Selecione "Just the push event"
4. Clique em **Add webhook**

### 3. Configurar no Sistema

Adicione o secret no arquivo de configuração ou variável de ambiente:

**Via arquivo** (`apps/api-service/config.yaml`):
```yaml
github:
  webhook_secret: "seu-secret-aqui"
```

**Via variável de ambiente**:
```bash
export GITHUB_WEBHOOK_SECRET="seu-secret-aqui"
```

## ⚙️ Variáveis de Ambiente

### API Service

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `SERVER_PORT` | Porta do servidor HTTP | `8080` |
| `NATS_URL` | URL do servidor NATS | `nats://localhost:4222` |
| `GITHUB_WEBHOOK_SECRET` | Secret para validação de webhooks | - |
| `LOG_LEVEL` | Nível de log (debug, info, warn, error) | `info` |
| `LOG_FORMAT` | Formato de log (json, console) | `json` |

### Worker Service

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `NATS_URL` | URL do servidor NATS | `nats://localhost:4222` |
| `WORKER_POOL_SIZE` | Número de workers simultâneos | `5` |
| `WORKER_TIMEOUT` | Timeout de build em segundos | `3600` |
| `WORKER_MAX_RETRIES` | Número máximo de tentativas | `3` |
| `BUILD_CODE_CACHE_PATH` | Caminho do cache de código | `/var/cache/oci-build/repos` |
| `BUILD_BUILD_CACHE_PATH` | Caminho do cache de dependências | `/var/cache/oci-build/deps` |
| `LOG_LEVEL` | Nível de log | `info` |
| `LOG_FORMAT` | Formato de log | `json` |

## 📡 API REST

### Endpoints

#### POST /webhook
Recebe webhooks do GitHub.

**Headers**:
- `X-Hub-Signature-256`: Assinatura HMAC-SHA256 do payload

**Response**:
```json
{
  "job_id": "uuid",
  "status": "queued"
}
```

#### GET /builds/:id
Consulta status de um build específico.

**Headers**:
- `Authorization`: Bearer token

**Response**:
```json
{
  "id": "uuid",
  "repository": {
    "url": "https://github.com/owner/repo",
    "name": "repo",
    "owner": "owner"
  },
  "commit_hash": "abc123",
  "branch": "main",
  "status": "completed",
  "created_at": "2024-01-01T00:00:00Z",
  "duration": 120000000000,
  "phases": [
    {
      "phase": "git_sync",
      "duration": 5000000000,
      "success": true
    }
  ]
}
```

#### GET /builds
Lista histórico de builds.

**Headers**:
- `Authorization`: Bearer token

**Query Parameters**:
- `limit`: Número máximo de resultados (padrão: 50)
- `offset`: Offset para paginação (padrão: 0)

#### GET /health
Health check do serviço.

**Response**:
```json
{
  "status": "healthy",
  "nats": "connected"
}
```

## 📊 Monitoramento

### Logs

Os logs são gerados em formato JSON estruturado:

```json
{
  "level": "info",
  "ts": "2024-01-01T00:00:00Z",
  "msg": "build completed",
  "job_id": "uuid",
  "repository": "owner/repo",
  "duration": 120
}
```

Visualizar logs em tempo real:
```bash
docker-compose logs -f api-service
docker-compose logs -f worker-service
```

### Métricas

O sistema registra métricas de:
- Duração de cada fase do build
- Taxa de sucesso/falha
- Tamanho do cache
- Número de builds simultâneos

## 🐛 Troubleshooting

### Build falha com "Dockerfile not found"

Verifique se o Dockerfile existe no repositório em um dos locais esperados:
- `./Dockerfile`
- `./docker/Dockerfile`
- `./build/Dockerfile`

### Erro de autenticação no webhook

Verifique se o secret configurado no GitHub corresponde ao configurado no sistema:
```bash
# Verificar secret no sistema
docker-compose exec api-service env | grep GITHUB_WEBHOOK_SECRET
```

### Worker não processa builds

Verifique a conectividade com NATS:
```bash
# Verificar logs do worker
docker-compose logs worker-service

# Verificar status do NATS
curl http://localhost:8222/varz
```

### Cache não está sendo utilizado

Verifique se os volumes estão montados corretamente:
```bash
docker-compose exec worker-service ls -la /var/cache/oci-build/
```

## 📝 Licença

[Adicione sua licença aqui]

## 🤝 Contribuindo

[Adicione guia de contribuição aqui]
