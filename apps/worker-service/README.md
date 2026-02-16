# Worker Service

O Worker Service é responsável por processar jobs de build consumindo mensagens do NATS. Ele coordena a execução de todas as fases do build: sincronização Git, build NX, e construção de imagens OCI.

## Funcionalidades

- **Subscriber NATS**: Consome mensagens do subject `builds.webhook`
- **Pool de Workers**: Processa múltiplos builds simultaneamente usando goroutines
- **Build Orchestrator**: Coordena as três fases do build:
  1. Git Sync: Clona ou atualiza o repositório
  2. NX Build: Executa o build usando NX
  3. Image Build: Constrói a imagem OCI com buildah
- **Retry com Backoff**: Tenta novamente operações que falham temporariamente
- **Logging Estruturado**: Registra métricas de duração para cada fase
- **Graceful Shutdown**: Para workers de forma ordenada

## Configuração

O serviço é configurado via arquivo `config.yaml`:

```yaml
nats:
  url: "nats://localhost:4222"
  reconnect_wait: 2s
  connect_timeout: 5s

worker:
  pool_size: 5          # Número de workers simultâneos
  queue_size: 100       # Tamanho da fila de jobs
  timeout: 3600         # Timeout em segundos para cada build
  max_retries: 3        # Número máximo de tentativas
  retry_delay: 1s       # Delay inicial entre tentativas

build:
  code_cache_path: "/var/cache/oci-build/repos"
  build_cache_path: "/var/cache/oci-build/deps"

logging:
  level: "info"
  format: "json"
```

Variáveis de ambiente podem sobrescrever valores do arquivo (ex: `NATS_URL`, `WORKER_POOL_SIZE`).

## Execução

### Local

```bash
# Compilar
go build -o worker-service

# Executar
./worker-service

# Ou com configuração customizada
CONFIG_PATH=/path/to/config.yaml ./worker-service
```

### Com Docker

```bash
docker build -t worker-service .
docker run -v /var/cache/oci-build:/var/cache/oci-build worker-service
```

## Testes

```bash
# Testes unitários
go test -v ./...

# Testes com cobertura
go test -v -cover ./...

# Testes de propriedade
go test -v -run TestProperty
```

## Arquitetura

### Fluxo de Processamento

1. **Recepção**: Mensagem chega do NATS no subject `builds.webhook`
2. **Validação**: Job é validado e adicionado à fila
3. **Processamento**: Worker pega job da fila e executa orchestrator
4. **Git Sync**: Repositório é clonado ou atualizado
5. **NX Build**: Build é executado com cache configurado
6. **Image Build**: Imagem OCI é construída com buildah
7. **Publicação**: Status é publicado no NATS durante e após execução

### Componentes

- **WorkerService**: Gerencia pool de workers e fila de jobs
- **BuildOrchestrator**: Coordena execução das fases do build
- **Services**: Git, NX, Image, e Cache services injetados via FX

### Tratamento de Erros

- **Retry com Backoff Exponencial**: Operações de rede são tentadas até 3 vezes
- **Interrupção em Falha**: Se uma fase falha, as seguintes não são executadas
- **Preservação de Estado**: Logs e métricas são preservados mesmo em falhas
- **Timeout**: Builds que excedem o timeout são cancelados

## Dependências

- **Uber FX**: Dependency injection
- **Zap**: Logging estruturado
- **Viper**: Gerenciamento de configuração
- **NATS**: Message broker
- **go-git**: Operações Git
- **buildah**: Construção de imagens OCI (via exec)

## Integração

O Worker Service se comunica com:

- **NATS**: Consome jobs e publica status
- **API Service**: Recebe jobs via NATS
- **File System**: Acessa caches de código e dependências
- **Buildah**: Executa comandos para construir imagens

## Monitoramento

Logs estruturados em JSON incluem:

- Job ID e informações do repositório
- Duração de cada fase
- Erros e stack traces
- Métricas de performance

Use ferramentas como ELK Stack ou Grafana Loki para análise de logs.
