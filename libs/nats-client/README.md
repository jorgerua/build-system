# NATS Client Library

Cliente NATS compartilhado para o OCI Build System com suporte a retry automático, logging estruturado e graceful shutdown.

## Características

- **Retry Automático**: Reconexão automática em caso de falha de rede
- **Logging Estruturado**: Integração com Zap para logging detalhado
- **Graceful Shutdown**: Drain de mensagens pendentes antes de fechar
- **Handlers de Eventos**: Callbacks para disconnect, reconnect, close e error
- **Request/Reply Pattern**: Suporte completo para comunicação síncrona
- **Pub/Sub Pattern**: Suporte para publicação e subscrição de mensagens

## Uso

### Criação do Cliente

```go
import (
    "go.uber.org/zap"
    natsclient "github.com/oci-build-system/libs/nats-client"
)

// Com configuração padrão
logger, _ := zap.NewProduction()
client := natsclient.NewClient(logger)
err := client.Connect("nats://localhost:4222")

// Com configuração customizada
config := &natsclient.Config{
    URL:            "nats://localhost:4222",
    MaxReconnects:  10,
    ReconnectWait:  2 * time.Second,
    ConnectTimeout: 5 * time.Second,
}
client, err := natsclient.NewClientWithConfig(logger, config)
```

### Publicação de Mensagens

```go
data := []byte(`{"id": "123", "status": "pending"}`)
err := client.Publish("builds.webhook", data)
```

### Subscrição

```go
handler := func(msg *nats.Msg) {
    fmt.Printf("Received: %s\n", string(msg.Data))
}

sub, err := client.Subscribe("builds.webhook", handler)
defer sub.Unsubscribe()
```

### Request/Reply

```go
request := []byte(`{"job_id": "123"}`)
response, err := client.Request("builds.status", request, 5*time.Second)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Response: %s\n", string(response.Data))
```

### Graceful Shutdown

```go
defer client.Close() // Drain e fecha a conexão
```

## Interface

```go
type NATSClient interface {
    Connect(url string) error
    Publish(subject string, data []byte) error
    Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
    Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error)
    Close()
    IsConnected() bool
}
```

## Configuração

A struct `Config` permite customizar o comportamento do cliente:

- `URL`: URL do servidor NATS
- `MaxReconnects`: Número máximo de tentativas de reconexão (padrão: 10)
- `ReconnectWait`: Intervalo entre tentativas de reconexão (padrão: 2s)
- `ConnectTimeout`: Timeout para conexão inicial (padrão: 5s)

## Logging

O cliente registra os seguintes eventos:

- Conexão estabelecida
- Desconexão (com erro se aplicável)
- Reconexão bem-sucedida
- Publicação de mensagens (debug)
- Recebimento de mensagens (debug)
- Erros de subscrição

## Dependências

- `github.com/nats-io/nats.go`: Cliente NATS oficial
- `go.uber.org/zap`: Logger estruturado
