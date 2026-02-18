package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/jorgerua/build-system/libs/shared"
	"github.com/nats-io/nats.go"
)

// MockMessage representa uma mensagem publicada no mock NATS
type MockMessage struct {
	Subject string
	Data    []byte
}

// MockNATSClient simula cliente NATS para testes
type MockNATSClient struct {
	PublishedMessages []MockMessage
	Subscriptions     map[string]nats.MsgHandler
	Connected         bool
	PublishFunc       func(subject string, data []byte) error
	SubscribeFunc     func(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
	RequestFunc       func(subject string, data []byte, timeout time.Duration) (*nats.Msg, error)
}

// NewMockNATSClient cria uma nova instância de MockNATSClient
func NewMockNATSClient() *MockNATSClient {
	return &MockNATSClient{
		PublishedMessages: make([]MockMessage, 0),
		Subscriptions:     make(map[string]nats.MsgHandler),
		Connected:         true,
	}
}

// Connect simula conexão ao NATS
func (m *MockNATSClient) Connect(url string) error {
	if !m.Connected {
		return fmt.Errorf("connection failed")
	}
	return nil
}

// Publish simula publicação de mensagem
func (m *MockNATSClient) Publish(subject string, data []byte) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(subject, data)
	}

	if !m.Connected {
		return fmt.Errorf("not connected")
	}

	m.PublishedMessages = append(m.PublishedMessages, MockMessage{
		Subject: subject,
		Data:    data,
	})
	return nil
}

// Subscribe simula subscrição a um subject
func (m *MockNATSClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(subject, handler)
	}

	if !m.Connected {
		return nil, fmt.Errorf("not connected")
	}

	m.Subscriptions[subject] = handler
	return &nats.Subscription{}, nil
}

// Request simula requisição com resposta
func (m *MockNATSClient) Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	if m.RequestFunc != nil {
		return m.RequestFunc(subject, data, timeout)
	}

	if !m.Connected {
		return nil, fmt.Errorf("not connected")
	}

	// Simular resposta
	return &nats.Msg{
		Subject: subject,
		Data:    []byte(`{"status":"ok"}`),
	}, nil
}

// Close simula fechamento da conexão
func (m *MockNATSClient) Close() {
	m.Connected = false
}

// IsConnected retorna o status da conexão
func (m *MockNATSClient) IsConnected() bool {
	return m.Connected
}

// GetPublishedMessage retorna uma mensagem publicada por índice
func (m *MockNATSClient) GetPublishedMessage(index int) *MockMessage {
	if index < 0 || index >= len(m.PublishedMessages) {
		return nil
	}
	return &m.PublishedMessages[index]
}

// GetPublishedMessagesBySubject retorna todas as mensagens publicadas em um subject
func (m *MockNATSClient) GetPublishedMessagesBySubject(subject string) []MockMessage {
	messages := make([]MockMessage, 0)
	for _, msg := range m.PublishedMessages {
		if msg.Subject == subject {
			messages = append(messages, msg)
		}
	}
	return messages
}

// ClearPublishedMessages limpa todas as mensagens publicadas
func (m *MockNATSClient) ClearPublishedMessages() {
	m.PublishedMessages = make([]MockMessage, 0)
}

// MockGitService simula operações Git para testes
type MockGitService struct {
	SyncRepositoryFunc   func(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error)
	RepositoryExistsFunc func(repoURL string) bool
	GetLocalPathFunc     func(repoURL string) string
}

// NewMockGitService cria uma nova instância de MockGitService
func NewMockGitService() *MockGitService {
	return &MockGitService{}
}

// SyncRepository simula sincronização de repositório
func (m *MockGitService) SyncRepository(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
	if m.SyncRepositoryFunc != nil {
		return m.SyncRepositoryFunc(ctx, repo, commitHash)
	}
	return "/tmp/test-repo", nil
}

// RepositoryExists simula verificação de existência de repositório
func (m *MockGitService) RepositoryExists(repoURL string) bool {
	if m.RepositoryExistsFunc != nil {
		return m.RepositoryExistsFunc(repoURL)
	}
	return false
}

// GetLocalPath simula obtenção do path local
func (m *MockGitService) GetLocalPath(repoURL string) string {
	if m.GetLocalPathFunc != nil {
		return m.GetLocalPathFunc(repoURL)
	}
	return "/tmp/test-repo"
}

// MockNXService simula builds NX para testes
type MockNXService struct {
	BuildFunc          func(ctx context.Context, repoPath string, config interface{}) (interface{}, error)
	DetectProjectsFunc func(repoPath string) ([]string, error)
}

// NewMockNXService cria uma nova instância de MockNXService
func NewMockNXService() *MockNXService {
	return &MockNXService{}
}

// Build simula execução de build NX
func (m *MockNXService) Build(ctx context.Context, repoPath string, config interface{}) (interface{}, error) {
	if m.BuildFunc != nil {
		return m.BuildFunc(ctx, repoPath, config)
	}

	// Retornar resultado de sucesso padrão
	return &struct {
		Success      bool
		Duration     time.Duration
		Output       string
		ErrorOutput  string
		ArtifactPath string
	}{
		Success:      true,
		Duration:     5 * time.Second,
		Output:       "Build successful",
		ErrorOutput:  "",
		ArtifactPath: "/tmp/test-repo/dist",
	}, nil
}

// DetectProjects simula detecção de projetos NX
func (m *MockNXService) DetectProjects(repoPath string) ([]string, error) {
	if m.DetectProjectsFunc != nil {
		return m.DetectProjectsFunc(repoPath)
	}
	return []string{"project1", "project2"}, nil
}

// MockImageService simula builds de imagem para testes
type MockImageService struct {
	BuildImageFunc func(ctx context.Context, config interface{}) (interface{}, error)
	TagImageFunc   func(imageID string, tags []string) error
}

// NewMockImageService cria uma nova instância de MockImageService
func NewMockImageService() *MockImageService {
	return &MockImageService{}
}

// BuildImage simula construção de imagem OCI
func (m *MockImageService) BuildImage(ctx context.Context, config interface{}) (interface{}, error) {
	if m.BuildImageFunc != nil {
		return m.BuildImageFunc(ctx, config)
	}

	// Retornar resultado de sucesso padrão
	return &struct {
		ImageID  string
		Tags     []string
		Size     int64
		Duration time.Duration
	}{
		ImageID:  "sha256:abc123def456",
		Tags:     []string{"test:latest"},
		Size:     100 * 1024 * 1024, // 100MB
		Duration: 30 * time.Second,
	}, nil
}

// TagImage simula aplicação de tags a uma imagem
func (m *MockImageService) TagImage(imageID string, tags []string) error {
	if m.TagImageFunc != nil {
		return m.TagImageFunc(imageID, tags)
	}
	return nil
}

// MockCacheService simula gerenciamento de cache para testes
type MockCacheService struct {
	GetCachePathFunc    func(language shared.Language) string
	InitializeCacheFunc func(language shared.Language) error
	CleanCacheFunc      func(language shared.Language, olderThan time.Duration) error
	GetCacheSizeFunc    func(language shared.Language) (int64, error)
}

// NewMockCacheService cria uma nova instância de MockCacheService
func NewMockCacheService() *MockCacheService {
	return &MockCacheService{}
}

// GetCachePath simula obtenção do path de cache
func (m *MockCacheService) GetCachePath(language shared.Language) string {
	if m.GetCachePathFunc != nil {
		return m.GetCachePathFunc(language)
	}
	return fmt.Sprintf("/tmp/cache/%s", language)
}

// InitializeCache simula inicialização de cache
func (m *MockCacheService) InitializeCache(language shared.Language) error {
	if m.InitializeCacheFunc != nil {
		return m.InitializeCacheFunc(language)
	}
	return nil
}

// CleanCache simula limpeza de cache
func (m *MockCacheService) CleanCache(language shared.Language, olderThan time.Duration) error {
	if m.CleanCacheFunc != nil {
		return m.CleanCacheFunc(language, olderThan)
	}
	return nil
}

// GetCacheSize simula obtenção do tamanho do cache
func (m *MockCacheService) GetCacheSize(language shared.Language) (int64, error) {
	if m.GetCacheSizeFunc != nil {
		return m.GetCacheSizeFunc(language)
	}
	return 1024 * 1024 * 100, nil // 100MB padrão
}
