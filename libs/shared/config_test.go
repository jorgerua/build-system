package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// TestLoadConfig_ValidYAML testa o parsing de configuração YAML válida
func TestLoadConfig_ValidYAML(t *testing.T) {
	// Criar arquivo de configuração temporário
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 10

nats:
  url: "nats://localhost:4222"

github:
  webhook_secret: "test-secret"

auth:
  token: "test-token"

worker:
  pool_size: 5
  timeout: 3600
  max_retries: 3

build:
  code_cache_path: "/var/cache/oci-build/repos"
  build_cache_path: "/var/cache/oci-build/deps"

logging:
  level: "info"
  format: "json"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Carregar configuração
	logger := zap.NewNop() // Use no-op logger for tests
	config, err := LoadConfig(configPath, logger)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verificar valores
	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.Port)
	}
	if config.NATS.URL != "nats://localhost:4222" {
		t.Errorf("Expected NATS URL 'nats://localhost:4222', got '%s'", config.NATS.URL)
	}
	if config.GitHub.WebhookSecret != "test-secret" {
		t.Errorf("Expected webhook secret 'test-secret', got '%s'", config.GitHub.WebhookSecret)
	}
	if config.Worker.PoolSize != 5 {
		t.Errorf("Expected pool size 5, got %d", config.Worker.PoolSize)
	}
	if config.Build.CodeCachePath != "/var/cache/oci-build/repos" {
		t.Errorf("Expected code cache path '/var/cache/oci-build/repos', got '%s'", config.Build.CodeCachePath)
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Logging.Level)
	}
}

// TestLoadConfig_EnvironmentOverride testa override com variáveis de ambiente
func TestLoadConfig_EnvironmentOverride(t *testing.T) {
	// Criar arquivo de configuração temporário
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
nats:
  url: "nats://localhost:4222"

github:
  webhook_secret: "default-secret"

auth:
  token: "default-token"

logging:
  level: "info"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Definir variáveis de ambiente
	os.Setenv("GITHUB_WEBHOOK_SECRET", "env-secret")
	os.Setenv("LOGGING_LEVEL", "debug")
	defer func() {
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("LOGGING_LEVEL")
	}()

	// Carregar configuração
	logger := zap.NewNop() // Use no-op logger for tests
	config, err := LoadConfig(configPath, logger)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verificar que variáveis de ambiente sobrescreveram valores do YAML
	if config.GitHub.WebhookSecret != "env-secret" {
		t.Errorf("Expected webhook secret 'env-secret' from env, got '%s'", config.GitHub.WebhookSecret)
	}
	if config.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug' from env, got '%s'", config.Logging.Level)
	}
}

// TestLoadConfig_MissingFile testa que arquivo ausente usa apenas variáveis de ambiente
func TestLoadConfig_MissingFile(t *testing.T) {
	logger := zap.NewNop()
	
	// Limpar variáveis de ambiente que podem interferir
	oldGithubSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	oldAuthToken := os.Getenv("AUTH_TOKEN")
	oldNatsURL := os.Getenv("NATS_URL")
	
	os.Unsetenv("GITHUB_WEBHOOK_SECRET")
	os.Unsetenv("AUTH_TOKEN")
	os.Unsetenv("NATS_URL")
	
	defer func() {
		if oldGithubSecret != "" {
			os.Setenv("GITHUB_WEBHOOK_SECRET", oldGithubSecret)
		}
		if oldAuthToken != "" {
			os.Setenv("AUTH_TOKEN", oldAuthToken)
		}
		if oldNatsURL != "" {
			os.Setenv("NATS_URL", oldNatsURL)
		}
	}()
	
	// Sem definir variáveis de ambiente obrigatórias, deve falhar na validação
	_, err := LoadConfig("/nonexistent/config.yaml", logger)
	if err == nil {
		t.Error("Expected error for missing config file without env vars, got nil")
	}
	// Deve conter erro de validação
	if err != nil && !strings.Contains(err.Error(), "configuration validation failed") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

// TestLoadConfig_InvalidYAML testa que YAML inválido usa apenas variáveis de ambiente
func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
nats:
  url: "nats://localhost:4222"
  invalid yaml syntax here: [
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	logger := zap.NewNop()
	
	// Limpar variáveis de ambiente que podem interferir
	oldGithubSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	oldAuthToken := os.Getenv("AUTH_TOKEN")
	oldNatsURL := os.Getenv("NATS_URL")
	
	os.Unsetenv("GITHUB_WEBHOOK_SECRET")
	os.Unsetenv("AUTH_TOKEN")
	os.Unsetenv("NATS_URL")
	
	defer func() {
		if oldGithubSecret != "" {
			os.Setenv("GITHUB_WEBHOOK_SECRET", oldGithubSecret)
		}
		if oldAuthToken != "" {
			os.Setenv("AUTH_TOKEN", oldAuthToken)
		}
		if oldNatsURL != "" {
			os.Setenv("NATS_URL", oldNatsURL)
		}
	}()
	
	// Sem variáveis de ambiente obrigatórias, deve falhar na validação
	_, err := LoadConfig(configPath, logger)
	if err == nil {
		t.Error("Expected error for invalid YAML without env vars, got nil")
	}
	// Deve conter erro de validação
	if err != nil && !strings.Contains(err.Error(), "configuration validation failed") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

// TestValidateConfig_MissingNATSURL testa validação com NATS URL ausente
func TestValidateConfig_MissingNATSURL(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		NATS: NATSConfig{
			URL: "",
		},
		GitHub: GitHubConfig{
			WebhookSecret: "test-secret",
		},
		Auth: AuthConfig{
			Token: "test-token",
		},
	}

	err := validateConfig(config, logger)
	if err == nil {
		t.Error("Expected error for missing NATS URL, got nil")
	}
}

// TestValidateConfig_InvalidPort testa validação com porta inválida
func TestValidateConfig_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"port negative", -1},
		{"port too high", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			config := &Config{
				NATS: NATSConfig{URL: "nats://localhost:4222"},
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth: AuthConfig{Token: "test-token"},
				Server: ServerConfig{
					Port: tt.port,
				},
			}

			err := validateConfig(config, logger)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

// TestValidateConfig_PortZeroSkipsValidation testa que porta 0 pula validação (não configurado)
func TestValidateConfig_PortZeroSkipsValidation(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		NATS: NATSConfig{URL: "nats://localhost:4222"},
		GitHub: GitHubConfig{WebhookSecret: "test-secret"},
		Auth: AuthConfig{Token: "test-token"},
		Server: ServerConfig{
			Port: 0, // Port 0 significa "não configurado"
		},
	}

	err := validateConfig(config, logger)
	if err != nil {
		t.Errorf("Expected no error for port 0 (not configured), got %v", err)
	}
}

// TestValidateConfig_NegativeTimeouts testa validação com timeouts negativos
func TestValidateConfig_NegativeTimeouts(t *testing.T) {
	tests := []struct {
		name   string
		config ServerConfig
	}{
		{
			"negative read timeout",
			ServerConfig{Port: 8080, ReadTimeout: -1},
		},
		{
			"negative write timeout",
			ServerConfig{Port: 8080, WriteTimeout: -1},
		},
		{
			"negative shutdown timeout",
			ServerConfig{Port: 8080, ShutdownTimeout: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			config := &Config{
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				Server: tt.config,
			}

			err := validateConfig(config, logger)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

// TestValidateConfig_InvalidWorkerConfig testa validação de configuração de worker
func TestValidateConfig_InvalidWorkerConfig(t *testing.T) {
	tests := []struct {
		name   string
		config WorkerConfig
	}{
		{
			"negative timeout",
			WorkerConfig{PoolSize: 5, Timeout: -1},
		},
		{
			"negative max retries",
			WorkerConfig{PoolSize: 5, MaxRetries: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			config := &Config{
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				Worker: tt.config,
			}

			err := validateConfig(config, logger)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

// TestValidateConfig_PoolSizeZeroSkipsValidation testa que pool_size 0 pula validação (não configurado)
func TestValidateConfig_PoolSizeZeroSkipsValidation(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		NATS:   NATSConfig{URL: "nats://localhost:4222"},
		GitHub: GitHubConfig{WebhookSecret: "test-secret"},
		Auth:   AuthConfig{Token: "test-token"},
		Worker: WorkerConfig{
			PoolSize: 0, // PoolSize 0 significa "não configurado"
		},
	}

	err := validateConfig(config, logger)
	if err != nil {
		t.Errorf("Expected no error for pool_size 0 (not configured), got %v", err)
	}
}

// TestValidateConfig_IncompleteBuildConfig testa validação de configuração de build incompleta
func TestValidateConfig_IncompleteBuildConfig(t *testing.T) {
	tests := []struct {
		name   string
		config BuildConfig
	}{
		{
			"missing build cache path",
			BuildConfig{CodeCachePath: "/var/cache/repos"},
		},
		{
			"missing code cache path",
			BuildConfig{BuildCachePath: "/var/cache/deps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			config := &Config{
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				Build:  tt.config,
			}

			err := validateConfig(config, logger)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

// TestValidateConfig_InvalidLogLevel testa validação de nível de log inválido
func TestValidateConfig_InvalidLogLevel(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		NATS:   NATSConfig{URL: "nats://localhost:4222"},
		GitHub: GitHubConfig{WebhookSecret: "test-secret"},
		Auth:   AuthConfig{Token: "test-token"},
		Logging: LoggingConfig{
			Level: "invalid",
		},
	}

	err := validateConfig(config, logger)
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}
}

// TestValidateConfig_InvalidLogFormat testa validação de formato de log inválido
func TestValidateConfig_InvalidLogFormat(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		NATS:   NATSConfig{URL: "nats://localhost:4222"},
		GitHub: GitHubConfig{WebhookSecret: "test-secret"},
		Auth:   AuthConfig{Token: "test-token"},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "xml",
		},
	}

	err := validateConfig(config, logger)
	if err == nil {
		t.Error("Expected error for invalid log format, got nil")
	}
}

// TestValidateConfig_ValidConfig testa validação de configuração válida
func TestValidateConfig_ValidConfig(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		NATS:   NATSConfig{URL: "nats://localhost:4222"},
		GitHub: GitHubConfig{WebhookSecret: "test-secret"},
		Auth:   AuthConfig{Token: "test-token"},
		Server: ServerConfig{
			Port:            8080,
			ReadTimeout:     30,
			WriteTimeout:    30,
			ShutdownTimeout: 10,
		},
		Worker: WorkerConfig{
			PoolSize:   5,
			Timeout:    3600,
			MaxRetries: 3,
		},
		Build: BuildConfig{
			CodeCachePath:  "/var/cache/repos",
			BuildCachePath: "/var/cache/deps",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := validateConfig(config, logger)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}
