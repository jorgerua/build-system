package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jorgerua/build-system/libs/shared"
)

func TestMockNATSClient_Publish(t *testing.T) {
	client := NewMockNATSClient()

	err := client.Publish("test.subject", []byte("test data"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(client.PublishedMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(client.PublishedMessages))
	}

	msg := client.PublishedMessages[0]
	if msg.Subject != "test.subject" {
		t.Errorf("expected subject 'test.subject', got '%s'", msg.Subject)
	}

	if string(msg.Data) != "test data" {
		t.Errorf("expected data 'test data', got '%s'", string(msg.Data))
	}
}

func TestMockNATSClient_NotConnected(t *testing.T) {
	client := NewMockNATSClient()
	client.Connected = false

	err := client.Publish("test.subject", []byte("test data"))
	if err == nil {
		t.Fatal("expected error when not connected, got nil")
	}
}

func TestMockGitService_SyncRepository(t *testing.T) {
	service := NewMockGitService()

	repo := shared.RepositoryInfo{
		URL:   "https://github.com/test/repo.git",
		Name:  "repo",
		Owner: "test",
	}

	path, err := service.SyncRepository(context.Background(), repo, "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if path != "/tmp/test-repo" {
		t.Errorf("expected path '/tmp/test-repo', got '%s'", path)
	}
}

func TestMockCacheService_GetCachePath(t *testing.T) {
	service := NewMockCacheService()

	path := service.GetCachePath(shared.LanguageJava)
	if path != "/tmp/cache/java" {
		t.Errorf("expected path '/tmp/cache/java', got '%s'", path)
	}
}

func TestMockCacheService_GetCacheSize(t *testing.T) {
	service := NewMockCacheService()

	size, err := service.GetCacheSize(shared.LanguageGo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedSize := int64(1024 * 1024 * 100) // 100MB
	if size != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, size)
	}
}

func TestMockNXService_Build(t *testing.T) {
	service := NewMockNXService()

	result, err := service.Build(context.Background(), "/tmp/repo", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestMockImageService_BuildImage(t *testing.T) {
	service := NewMockImageService()

	result, err := service.BuildImage(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestMockNATSClient_GetPublishedMessagesBySubject(t *testing.T) {
	client := NewMockNATSClient()

	client.Publish("test.subject1", []byte("data1"))
	client.Publish("test.subject2", []byte("data2"))
	client.Publish("test.subject1", []byte("data3"))

	messages := client.GetPublishedMessagesBySubject("test.subject1")
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages for subject1, got %d", len(messages))
	}

	if string(messages[0].Data) != "data1" {
		t.Errorf("expected first message 'data1', got '%s'", string(messages[0].Data))
	}

	if string(messages[1].Data) != "data3" {
		t.Errorf("expected second message 'data3', got '%s'", string(messages[1].Data))
	}
}

func TestMockNATSClient_ClearPublishedMessages(t *testing.T) {
	client := NewMockNATSClient()

	client.Publish("test.subject", []byte("data"))
	if len(client.PublishedMessages) != 1 {
		t.Fatalf("expected 1 message before clear, got %d", len(client.PublishedMessages))
	}

	client.ClearPublishedMessages()
	if len(client.PublishedMessages) != 0 {
		t.Fatalf("expected 0 messages after clear, got %d", len(client.PublishedMessages))
	}
}

func TestMockGitService_CustomFunction(t *testing.T) {
	service := NewMockGitService()

	// Test with custom function
	service.SyncRepositoryFunc = func(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
		return "/custom/path", nil
	}

	repo := shared.RepositoryInfo{
		URL:   "https://github.com/test/repo.git",
		Name:  "repo",
		Owner: "test",
	}

	path, err := service.SyncRepository(context.Background(), repo, "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if path != "/custom/path" {
		t.Errorf("expected custom path '/custom/path', got '%s'", path)
	}
}

func TestMockCacheService_CustomFunction(t *testing.T) {
	service := NewMockCacheService()

	// Test with custom function
	service.GetCacheSizeFunc = func(language shared.Language) (int64, error) {
		return 999, nil
	}

	size, err := service.GetCacheSize(shared.LanguageJava)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if size != 999 {
		t.Errorf("expected custom size 999, got %d", size)
	}
}

func TestMockImageService_TagImage(t *testing.T) {
	service := NewMockImageService()

	err := service.TagImage("sha256:abc123", []string{"v1.0.0", "latest"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMockCacheService_InitializeCache(t *testing.T) {
	service := NewMockCacheService()

	err := service.InitializeCache(shared.LanguageDotNet)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMockCacheService_CleanCache(t *testing.T) {
	service := NewMockCacheService()

	err := service.CleanCache(shared.LanguageGo, 24*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
