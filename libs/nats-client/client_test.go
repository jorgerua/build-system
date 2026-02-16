package natsclient

import (
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// startTestServer inicia um servidor NATS de teste
func startTestServer(t *testing.T) (*server.Server, string) {
	opts := test.DefaultTestOptions
	opts.Port = -1 // Porta aleatória
	s := test.RunServer(&opts)
	if s == nil {
		t.Fatal("failed to start test server")
	}
	return s, s.ClientURL()
}

// TestNewClient testa a criação de um novo cliente
func TestNewClient(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	
	if client == nil {
		t.Fatal("expected client to be created")
	}
	
	if client.logger == nil {
		t.Error("expected logger to be set")
	}
	
	if client.conn != nil {
		t.Error("expected connection to be nil before Connect")
	}
}

// TestNewClientWithNilLogger testa criação de cliente com logger nil
func TestNewClientWithNilLogger(t *testing.T) {
	client := NewClient(nil)
	
	if client == nil {
		t.Fatal("expected client to be created")
	}
	
	if client.logger == nil {
		t.Error("expected default logger to be set")
	}
}

// TestConnect testa conexão básica ao servidor NATS
func TestConnect(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}
}

// TestConnectWithInvalidURL testa conexão com URL inválida
func TestConnectWithInvalidURL(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect("nats://invalid:9999")
	if err == nil {
		t.Error("expected error when connecting to invalid URL")
	}
	
	if client.IsConnected() {
		t.Error("expected client to not be connected")
	}
}

// TestConnectWithConfig testa conexão com configuração customizada
func TestConnectWithConfig(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	config := &Config{
		URL:            url,
		MaxReconnects:  5,
		ReconnectWait:  1 * time.Second,
		ConnectTimeout: 3 * time.Second,
	}
	
	client, err := NewClientWithConfig(logger, config)
	if err != nil {
		t.Fatalf("failed to create client with config: %v", err)
	}
	defer client.Close()
	
	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}
}

// TestReconnect testa que o cliente está configurado para reconexão automática
func TestReconnect(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	config := &Config{
		URL:            url,
		MaxReconnects:  3,
		ReconnectWait:  100 * time.Millisecond,
		ConnectTimeout: 2 * time.Second,
	}
	
	client, err := NewClientWithConfig(logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()
	
	if !client.IsConnected() {
		t.Fatal("expected client to be connected")
	}
	
	// Verificar que o cliente tem configuração de reconexão
	stats := client.Stats()
	if stats.Reconnects < 0 {
		t.Error("expected reconnects stat to be available")
	}
}

// TestPublish testa publicação de mensagens
func TestPublish(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	subject := "test.publish"
	data := []byte("test message")
	
	err = client.Publish(subject, data)
	if err != nil {
		t.Errorf("failed to publish: %v", err)
	}
	
	// Flush para garantir que a mensagem foi enviada
	err = client.Flush()
	if err != nil {
		t.Errorf("failed to flush: %v", err)
	}
}

// TestPublishWithoutConnection testa publicação sem conexão
func TestPublishWithoutConnection(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Publish("test.subject", []byte("data"))
	if err == nil {
		t.Error("expected error when publishing without connection")
	}
}

// TestSubscribe testa subscrição a um subject
func TestSubscribe(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	subject := "test.subscribe"
	received := make(chan []byte, 1)
	
	handler := func(msg *nats.Msg) {
		received <- msg.Data
	}
	
	sub, err := client.Subscribe(subject, handler)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	
	// Publicar mensagem
	testData := []byte("test message")
	err = client.Publish(subject, testData)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	
	// Aguardar recebimento
	select {
	case data := <-received:
		if string(data) != string(testData) {
			t.Errorf("expected %s, got %s", testData, data)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for message")
	}
}

// TestSubscribeWithoutConnection testa subscrição sem conexão
func TestSubscribeWithoutConnection(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	
	handler := func(msg *nats.Msg) {}
	
	_, err := client.Subscribe("test.subject", handler)
	if err == nil {
		t.Error("expected error when subscribing without connection")
	}
}

// TestPublishSubscribeMultipleMessages testa múltiplas mensagens
func TestPublishSubscribeMultipleMessages(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	subject := "test.multiple"
	messageCount := 10
	received := make(chan []byte, messageCount)
	
	handler := func(msg *nats.Msg) {
		received <- msg.Data
	}
	
	sub, err := client.Subscribe(subject, handler)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	
	// Publicar múltiplas mensagens
	for i := 0; i < messageCount; i++ {
		data := []byte("message " + string(rune('0'+i)))
		err = client.Publish(subject, data)
		if err != nil {
			t.Fatalf("failed to publish message %d: %v", i, err)
		}
	}
	
	// Aguardar recebimento de todas as mensagens
	receivedCount := 0
	timeout := time.After(3 * time.Second)
	
	for receivedCount < messageCount {
		select {
		case <-received:
			receivedCount++
		case <-timeout:
			t.Fatalf("timeout: received %d/%d messages", receivedCount, messageCount)
		}
	}
}

// TestRequest testa o padrão request/reply
func TestRequest(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	
	// Cliente que responde
	responder := NewClient(logger)
	err := responder.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect responder: %v", err)
	}
	defer responder.Close()
	
	subject := "test.request"
	
	// Configurar responder
	handler := func(msg *nats.Msg) {
		response := []byte("response: " + string(msg.Data))
		msg.Respond(response)
	}
	
	sub, err := responder.Subscribe(subject, handler)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	
	// Cliente que faz request
	requester := NewClient(logger)
	err = requester.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect requester: %v", err)
	}
	defer requester.Close()
	
	// Fazer request
	requestData := []byte("hello")
	msg, err := requester.Request(subject, requestData, 2*time.Second)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	
	expectedResponse := "response: hello"
	if string(msg.Data) != expectedResponse {
		t.Errorf("expected %s, got %s", expectedResponse, msg.Data)
	}
}

// TestRequestTimeout testa timeout em request
func TestRequestTimeout(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	// Request para subject sem responder
	subject := "test.no.responder"
	_, err = client.Request(subject, []byte("data"), 500*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
}

// TestRequestWithoutConnection testa request sem conexão
func TestRequestWithoutConnection(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	
	_, err := client.Request("test.subject", []byte("data"), 1*time.Second)
	if err == nil {
		t.Error("expected error when requesting without connection")
	}
}

// TestClose testa fechamento da conexão
func TestClose(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	
	if !client.IsConnected() {
		t.Fatal("expected client to be connected")
	}
	
	client.Close()
	
	if client.IsConnected() {
		t.Error("expected client to be disconnected after Close")
	}
}

// TestCloseWithoutConnection testa Close sem conexão
func TestCloseWithoutConnection(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	
	// Não deve causar panic
	client.Close()
}

// TestMultipleClose testa múltiplas chamadas a Close
func TestMultipleClose(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	
	client.Close()
	client.Close() // Não deve causar panic
}

// TestIsConnected testa verificação de conexão
func TestIsConnected(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	if client.IsConnected() {
		t.Error("expected client to not be connected initially")
	}
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	
	if !client.IsConnected() {
		t.Error("expected client to be connected after Connect")
	}
	
	client.Close()
	
	if client.IsConnected() {
		t.Error("expected client to not be connected after Close")
	}
}

// TestStats testa obtenção de estatísticas
func TestStats(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	// Publicar algumas mensagens
	subject := "test.stats"
	for i := 0; i < 5; i++ {
		err = client.Publish(subject, []byte("test"))
		if err != nil {
			t.Fatalf("failed to publish: %v", err)
		}
	}
	
	client.Flush()
	
	stats := client.Stats()
	if stats.OutMsgs == 0 {
		t.Error("expected OutMsgs to be greater than 0")
	}
}

// TestFlush testa flush de mensagens pendentes
func TestFlush(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	err = client.Publish("test.flush", []byte("data"))
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	
	err = client.Flush()
	if err != nil {
		t.Errorf("flush failed: %v", err)
	}
}

// TestFlushTimeout testa flush com timeout
func TestFlushTimeout(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	err = client.Publish("test.flush", []byte("data"))
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	
	err = client.FlushTimeout(1 * time.Second)
	if err != nil {
		t.Errorf("flush timeout failed: %v", err)
	}
}

// TestConcurrentPublish testa publicação concorrente
func TestConcurrentPublish(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()
	
	logger := zap.NewNop()
	client := NewClient(logger)
	
	err := client.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()
	
	subject := "test.concurrent"
	goroutines := 10
	messagesPerGoroutine := 10
	
	var wg sync.WaitGroup
	wg.Add(goroutines)
	
	errors := make(chan error, goroutines*messagesPerGoroutine)
	
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				data := []byte("message from goroutine " + string(rune('0'+id)))
				if err := client.Publish(subject, data); err != nil {
					errors <- err
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	for err := range errors {
		t.Errorf("concurrent publish error: %v", err)
	}
}

// TestDefaultConfig testa configuração padrão
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.MaxReconnects != 10 {
		t.Errorf("expected MaxReconnects to be 10, got %d", config.MaxReconnects)
	}
	
	if config.ReconnectWait != 2*time.Second {
		t.Errorf("expected ReconnectWait to be 2s, got %v", config.ReconnectWait)
	}
	
	if config.ConnectTimeout != 5*time.Second {
		t.Errorf("expected ConnectTimeout to be 5s, got %v", config.ConnectTimeout)
	}
}
