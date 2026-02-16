package gitservice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// TestGetLocalPath testa o cálculo de path local para repositórios
func TestGetLocalPath(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := Config{
		CodeCachePath: "/tmp/test-cache",
		MaxRetries:    3,
		RetryDelay:    time.Millisecond * 10,
	}
	svc := NewGitService(config, logger)

	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "GitHub URL with .git suffix",
			repoURL:  "https://github.com/owner/repo.git",
			expected: filepath.Join("/tmp/test-cache", "owner", "repo"),
		},
		{
			name:     "GitHub URL without .git suffix",
			repoURL:  "https://github.com/owner/repo",
			expected: filepath.Join("/tmp/test-cache", "owner", "repo"),
		},
		{
			name:     "GitLab URL",
			repoURL:  "https://gitlab.com/owner/repo.git",
			expected: filepath.Join("/tmp/test-cache", "owner", "repo"),
		},
		{
			name:     "URL with multiple path segments",
			repoURL:  "https://github.com/org/team/repo.git",
			expected: filepath.Join("/tmp/test-cache", "team", "repo"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.GetLocalPath(tt.repoURL)
			if result != tt.expected {
				t.Errorf("GetLocalPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestRepositoryExists testa a detecção de repositório em cache
func TestRepositoryExists(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	// Criar diretório temporário para testes
	tempDir := t.TempDir()
	
	config := Config{
		CodeCachePath: tempDir,
		MaxRetries:    3,
		RetryDelay:    time.Millisecond * 10,
	}
	svc := NewGitService(config, logger).(*gitServiceImpl)

	t.Run("repository does not exist", func(t *testing.T) {
		repoURL := "https://github.com/nonexistent/repo.git"
		exists := svc.RepositoryExists(repoURL)
		if exists {
			t.Error("RepositoryExists() = true, want false for non-existent repo")
		}
	})

	t.Run("directory exists but not a git repository", func(t *testing.T) {
		// Criar diretório que não é um repositório Git
		notGitDir := filepath.Join(tempDir, "owner", "notgit")
		if err := os.MkdirAll(notGitDir, 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Criar arquivo para simular diretório não-vazio
		testFile := filepath.Join(notGitDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		repoURL := "https://github.com/owner/notgit.git"
		exists := svc.RepositoryExists(repoURL)
		if exists {
			t.Error("RepositoryExists() = true, want false for non-git directory")
		}
	})
}

// TestNewGitService testa a criação de instância do serviço
func TestNewGitService(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("with default values", func(t *testing.T) {
		config := Config{
			CodeCachePath: "/tmp/cache",
		}
		svc := NewGitService(config, logger).(*gitServiceImpl)

		if svc.maxRetries != 3 {
			t.Errorf("maxRetries = %d, want 3", svc.maxRetries)
		}
		if svc.retryDelay != time.Second {
			t.Errorf("retryDelay = %v, want 1s", svc.retryDelay)
		}
		if svc.codeCachePath != "/tmp/cache" {
			t.Errorf("codeCachePath = %s, want /tmp/cache", svc.codeCachePath)
		}
	})

	t.Run("with custom values", func(t *testing.T) {
		config := Config{
			CodeCachePath: "/custom/path",
			MaxRetries:    5,
			RetryDelay:    time.Second * 2,
		}
		svc := NewGitService(config, logger).(*gitServiceImpl)

		if svc.maxRetries != 5 {
			t.Errorf("maxRetries = %d, want 5", svc.maxRetries)
		}
		if svc.retryDelay != time.Second*2 {
			t.Errorf("retryDelay = %v, want 2s", svc.retryDelay)
		}
		if svc.codeCachePath != "/custom/path" {
			t.Errorf("codeCachePath = %s, want /custom/path", svc.codeCachePath)
		}
	})
}

// TestSyncRepository_Validation testa validação de entrada
func TestSyncRepository_Validation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tempDir := t.TempDir()
	
	config := Config{
		CodeCachePath: tempDir,
		MaxRetries:    1,
		RetryDelay:    time.Millisecond * 10,
	}
	svc := NewGitService(config, logger)
	ctx := context.Background()

	t.Run("invalid repository info - empty URL", func(t *testing.T) {
		repo := shared.RepositoryInfo{
			URL:   "",
			Name:  "repo",
			Owner: "owner",
			Branch: "main",
		}
		_, err := svc.SyncRepository(ctx, repo, "abc123")
		if err == nil {
			t.Error("SyncRepository() error = nil, want error for invalid repo")
		}
		if err.Error() != "invalid repository info" {
			t.Errorf("SyncRepository() error = %v, want 'invalid repository info'", err)
		}
	})

	t.Run("invalid repository info - empty name", func(t *testing.T) {
		repo := shared.RepositoryInfo{
			URL:   "https://github.com/owner/repo.git",
			Name:  "",
			Owner: "owner",
			Branch: "main",
		}
		_, err := svc.SyncRepository(ctx, repo, "abc123")
		if err == nil {
			t.Error("SyncRepository() error = nil, want error for invalid repo")
		}
	})

	t.Run("invalid repository info - empty owner", func(t *testing.T) {
		repo := shared.RepositoryInfo{
			URL:   "https://github.com/owner/repo.git",
			Name:  "repo",
			Owner: "",
			Branch: "main",
		}
		_, err := svc.SyncRepository(ctx, repo, "abc123")
		if err == nil {
			t.Error("SyncRepository() error = nil, want error for invalid repo")
		}
	})

	t.Run("empty commit hash", func(t *testing.T) {
		repo := shared.RepositoryInfo{
			URL:   "https://github.com/owner/repo.git",
			Name:  "repo",
			Owner: "owner",
			Branch: "main",
		}
		_, err := svc.SyncRepository(ctx, repo, "")
		if err == nil {
			t.Error("SyncRepository() error = nil, want error for empty commit hash")
		}
		if err.Error() != "commit hash is required" {
			t.Errorf("SyncRepository() error = %v, want 'commit hash is required'", err)
		}
	})
}

// TestSyncRepository_ContextCancellation testa cancelamento via context
func TestSyncRepository_ContextCancellation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tempDir := t.TempDir()
	
	config := Config{
		CodeCachePath: tempDir,
		MaxRetries:    3,
		RetryDelay:    time.Second, // Delay longo para garantir cancelamento
	}
	svc := NewGitService(config, logger)

	t.Run("context cancelled before operation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancelar imediatamente

		repo := shared.RepositoryInfo{
			URL:   "https://github.com/invalid/nonexistent.git",
			Name:  "nonexistent",
			Owner: "invalid",
			Branch: "main",
		}

		_, err := svc.SyncRepository(ctx, repo, "abc123")
		if err == nil {
			t.Error("SyncRepository() error = nil, want error for cancelled context")
		}
	})

	t.Run("context with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		repo := shared.RepositoryInfo{
			URL:   "https://github.com/invalid/nonexistent.git",
			Name:  "nonexistent",
			Owner: "invalid",
			Branch: "main",
		}

		_, err := svc.SyncRepository(ctx, repo, "abc123")
		if err == nil {
			t.Error("SyncRepository() error = nil, want error for timeout")
		}
	})
}

// TestGetLocalPath_EdgeCases testa casos extremos de cálculo de path
func TestGetLocalPath_EdgeCases(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := Config{
		CodeCachePath: "/tmp/cache",
		MaxRetries:    3,
		RetryDelay:    time.Millisecond * 10,
	}
	svc := NewGitService(config, logger)

	t.Run("URL with trailing slash", func(t *testing.T) {
		repoURL := "https://github.com/owner/repo.git/"
		result := svc.GetLocalPath(repoURL)
		// Deve remover a barra final e processar corretamente
		if result == "" {
			t.Error("GetLocalPath() returned empty string")
		}
	})

	t.Run("URL with query parameters", func(t *testing.T) {
		repoURL := "https://github.com/owner/repo.git?ref=main"
		result := svc.GetLocalPath(repoURL)
		// Deve ignorar query parameters
		if result == "" {
			t.Error("GetLocalPath() returned empty string")
		}
	})

	t.Run("very short URL", func(t *testing.T) {
		repoURL := "repo"
		result := svc.GetLocalPath(repoURL)
		// Deve usar fallback com hash
		if result == "" {
			t.Error("GetLocalPath() returned empty string")
		}
	})
}

// TestRepositoryInfo_Integration testa integração com shared.RepositoryInfo
func TestRepositoryInfo_Integration(t *testing.T) {
	t.Run("valid repository info", func(t *testing.T) {
		repo := shared.RepositoryInfo{
			URL:   "https://github.com/owner/repo.git",
			Name:  "repo",
			Owner: "owner",
			Branch: "main",
		}

		if !repo.IsValid() {
			t.Error("IsValid() = false, want true for valid repo")
		}

		fullName := repo.FullName()
		if fullName != "owner/repo" {
			t.Errorf("FullName() = %s, want owner/repo", fullName)
		}
	})

	t.Run("invalid repository info", func(t *testing.T) {
		repo := shared.RepositoryInfo{
			URL:   "",
			Name:  "repo",
			Owner: "owner",
			Branch: "main",
		}

		if repo.IsValid() {
			t.Error("IsValid() = true, want false for invalid repo")
		}
	})
}

// TestConfig_Defaults testa valores padrão de configuração
func TestConfig_Defaults(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("zero values get defaults", func(t *testing.T) {
		config := Config{
			CodeCachePath: "/tmp/cache",
			MaxRetries:    0, // Deve usar padrão
			RetryDelay:    0, // Deve usar padrão
		}
		svc := NewGitService(config, logger).(*gitServiceImpl)

		if svc.maxRetries != 3 {
			t.Errorf("maxRetries = %d, want 3 (default)", svc.maxRetries)
		}
		if svc.retryDelay != time.Second {
			t.Errorf("retryDelay = %v, want 1s (default)", svc.retryDelay)
		}
	})
}
