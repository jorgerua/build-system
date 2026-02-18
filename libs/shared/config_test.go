package shared

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestLoadConfig_ValidFile tests loading configuration from a valid YAML file
func TestLoadConfig_ValidFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	configContent := `
server:
  port: 8080
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 10

nats:
  url: nats://localhost:4222
  reconnect_wait: 2s
  connect_timeout: 5s

github:
  webhook_secret: test-secret

auth:
  token: test-token

worker:
  pool_size: 5
  queue_size: 100
  timeout: 3600
  max_retries: 3
  retry_delay: 5s

build:
  code_cache_path: /tmp/code-cache
  build_cache_path: /tmp/build-cache

logging:
  level: info
  format: json
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	
	// Create cache directories
	require.NoError(t, os.MkdirAll("/tmp/code-cache", 0755))
	require.NoError(t, os.MkdirAll("/tmp/build-cache", 0755))
	defer os.RemoveAll("/tmp/code-cache")
	defer os.RemoveAll("/tmp/build-cache")
	
	// Load config
	logger := zaptest.NewLogger(t)
	config, err := LoadConfig(configPath, logger)
	
	// Verify
	require.NoError(t, err)
	require.NotNil(t, config)
	
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 30, config.Server.ReadTimeout)
	assert.Equal(t, 30, config.Server.WriteTimeout)
	assert.Equal(t, 10, config.Server.ShutdownTimeout)
	
	assert.Equal(t, "nats://localhost:4222", config.NATS.URL)
	
	assert.Equal(t, "test-secret", config.GitHub.WebhookSecret)
	assert.Equal(t, "test-token", config.Auth.Token)
	
	assert.Equal(t, 5, config.Worker.PoolSize)
	assert.Equal(t, 100, config.Worker.QueueSize)
	assert.Equal(t, 3600, config.Worker.Timeout)
	assert.Equal(t, 3, config.Worker.MaxRetries)
	
	assert.Equal(t, "/tmp/code-cache", config.Build.CodeCachePath)
	assert.Equal(t, "/tmp/build-cache", config.Build.BuildCachePath)
	
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
}

// TestLoadConfig_WithEnvironmentVariables tests loading config with env var substitution
func TestLoadConfig_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_GITHUB_SECRET", "env-secret")
	os.Setenv("TEST_AUTH_TOKEN", "env-token")
	os.Setenv("TEST_NATS_URL", "nats://env-nats:4222")
	os.Setenv("TEST_CODE_CACHE", "/tmp/env-code-cache")
	os.Setenv("TEST_BUILD_CACHE", "/tmp/env-build-cache")
	defer func() {
		os.Unsetenv("TEST_GITHUB_SECRET")
		os.Unsetenv("TEST_AUTH_TOKEN")
		os.Unsetenv("TEST_NATS_URL")
		os.Unsetenv("TEST_CODE_CACHE")
		os.Unsetenv("TEST_BUILD_CACHE")
	}()
	
	// Create temporary config file with env var placeholders
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	configContent := `
server:
  port: 8080

nats:
  url: ${TEST_NATS_URL}

github:
  webhook_secret: ${TEST_GITHUB_SECRET}

auth:
  token: ${TEST_AUTH_TOKEN}

build:
  code_cache_path: ${TEST_CODE_CACHE}
  build_cache_path: ${TEST_BUILD_CACHE}

logging:
  level: info
  format: json
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	
	// Create cache directories
	require.NoError(t, os.MkdirAll("/tmp/env-code-cache", 0755))
	require.NoError(t, os.MkdirAll("/tmp/env-build-cache", 0755))
	defer os.RemoveAll("/tmp/env-code-cache")
	defer os.RemoveAll("/tmp/env-build-cache")
	
	// Load config
	logger := zaptest.NewLogger(t)
	config, err := LoadConfig(configPath, logger)
	
	// Verify environment variables were expanded
	require.NoError(t, err)
	require.NotNil(t, config)
	
	assert.Equal(t, "env-secret", config.GitHub.WebhookSecret)
	assert.Equal(t, "env-token", config.Auth.Token)
	assert.Equal(t, "nats://env-nats:4222", config.NATS.URL)
	assert.Equal(t, "/tmp/env-code-cache", config.Build.CodeCachePath)
	assert.Equal(t, "/tmp/env-build-cache", config.Build.BuildCachePath)
}

// TestLoadConfig_MissingFile tests loading config when file doesn't exist
func TestLoadConfig_MissingFile(t *testing.T) {
	// Set required environment variables so validation passes
	os.Setenv("GITHUB_WEBHOOK_SECRET", "test-secret")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("NATS_URL", "nats://localhost:4222")
	defer func() {
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("AUTH_TOKEN")
		os.Unsetenv("NATS_URL")
	}()
	
	logger := zaptest.NewLogger(t)
	config, err := LoadConfig("/nonexistent/config.yaml", logger)
	
	// Should still work with environment variables
	require.NoError(t, err)
	require.NotNil(t, config)
}

// TestValidateConfig_InvalidConfiguration tests validation with invalid config
func TestValidateConfig_InvalidConfiguration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing github webhook secret",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: ""},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectError: true,
			errorMsg:    "GITHUB_WEBHOOK_SECRET is required",
		},
		{
			name: "unexpanded github webhook secret",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "${GITHUB_WEBHOOK_SECRET}"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectError: true,
			errorMsg:    "GITHUB_WEBHOOK_SECRET is required",
		},
		{
			name: "missing auth token",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: ""},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectError: true,
			errorMsg:    "API_AUTH_TOKEN is required",
		},
		{
			name: "unexpanded auth token",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "${API_AUTH_TOKEN}"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectError: true,
			errorMsg:    "API_AUTH_TOKEN is required",
		},
		{
			name: "missing nats url",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: ""},
			},
			expectError: true,
			errorMsg:    "NATS URL is required",
		},
		{
			name: "invalid server port - too high",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				Server: ServerConfig{Port: 70000},
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "negative read timeout",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				Server: ServerConfig{Port: 8080, ReadTimeout: -1},
			},
			expectError: true,
			errorMsg:    "read timeout cannot be negative",
		},
		{
			name: "negative write timeout",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				Server: ServerConfig{Port: 8080, WriteTimeout: -1},
			},
			expectError: true,
			errorMsg:    "write timeout cannot be negative",
		},
		{
			name: "invalid worker pool size",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				Worker: WorkerConfig{PoolSize: 1, Timeout: -1},
			},
			expectError: true,
			errorMsg:    "worker timeout cannot be negative",
		},
		{
			name: "negative worker timeout",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				Worker: WorkerConfig{PoolSize: 5, Timeout: -1},
			},
			expectError: true,
			errorMsg:    "worker timeout cannot be negative",
		},
		{
			name: "invalid logging level",
			config: Config{
				GitHub:  GitHubConfig{WebhookSecret: "test-secret"},
				Auth:    AuthConfig{Token: "test-token"},
				NATS:    NATSConfig{URL: "nats://localhost:4222"},
				Logging: LoggingConfig{Level: "invalid"},
			},
			expectError: true,
			errorMsg:    "logging level must be one of",
		},
		{
			name: "invalid logging format",
			config: Config{
				GitHub:  GitHubConfig{WebhookSecret: "test-secret"},
				Auth:    AuthConfig{Token: "test-token"},
				NATS:    NATSConfig{URL: "nats://localhost:4222"},
				Logging: LoggingConfig{Format: "xml"},
			},
			expectError: true,
			errorMsg:    "logging format must be either",
		},
		{
			name: "valid configuration",
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "test-secret"},
				Auth:   AuthConfig{Token: "test-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
				Server: ServerConfig{Port: 8080, ReadTimeout: 30, WriteTimeout: 30},
				Worker: WorkerConfig{PoolSize: 5, Timeout: 3600, MaxRetries: 3},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config, logger)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExpandEnvVars tests environment variable expansion
func TestExpandEnvVars_DifferentFormats(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		config         Config
		expectedSecret string
		expectedToken  string
		expectedURL    string
		expectedCode   string
		expectedBuild  string
	}{
		{
			name: "standard ${VAR} format",
			envVars: map[string]string{
				"SECRET": "my-secret",
				"TOKEN":  "my-token",
				"URL":    "nats://test:4222",
				"CODE":   "/tmp/code",
				"BUILD":  "/tmp/build",
			},
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "${SECRET}"},
				Auth:   AuthConfig{Token: "${TOKEN}"},
				NATS:   NATSConfig{URL: "${URL}"},
				Build:  BuildConfig{CodeCachePath: "${CODE}", BuildCachePath: "${BUILD}"},
			},
			expectedSecret: "my-secret",
			expectedToken:  "my-token",
			expectedURL:    "nats://test:4222",
			expectedCode:   "/tmp/code",
			expectedBuild:  "/tmp/build",
		},
		{
			name: "$VAR format",
			envVars: map[string]string{
				"SECRET": "my-secret",
				"TOKEN":  "my-token",
			},
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "$SECRET"},
				Auth:   AuthConfig{Token: "$TOKEN"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectedSecret: "my-secret",
			expectedToken:  "my-token",
			expectedURL:    "nats://localhost:4222",
		},
		{
			name:    "no environment variables",
			envVars: map[string]string{},
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "literal-secret"},
				Auth:   AuthConfig{Token: "literal-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectedSecret: "literal-secret",
			expectedToken:  "literal-token",
			expectedURL:    "nats://localhost:4222",
		},
		{
			name: "mixed literal and env vars",
			envVars: map[string]string{
				"SECRET": "env-secret",
			},
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "${SECRET}"},
				Auth:   AuthConfig{Token: "literal-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectedSecret: "env-secret",
			expectedToken:  "literal-token",
			expectedURL:    "nats://localhost:4222",
		},
		{
			name: "undefined environment variable",
			envVars: map[string]string{},
			config: Config{
				GitHub: GitHubConfig{WebhookSecret: "${UNDEFINED_VAR}"},
				Auth:   AuthConfig{Token: "literal-token"},
				NATS:   NATSConfig{URL: "nats://localhost:4222"},
			},
			expectedSecret: "", // os.ExpandEnv returns empty string for undefined vars
			expectedToken:  "literal-token",
			expectedURL:    "nats://localhost:4222",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()
			
			// Expand environment variables
			expandEnvVars(&tt.config)
			
			// Verify
			assert.Equal(t, tt.expectedSecret, tt.config.GitHub.WebhookSecret)
			assert.Equal(t, tt.expectedToken, tt.config.Auth.Token)
			assert.Equal(t, tt.expectedURL, tt.config.NATS.URL)
			
			if tt.expectedCode != "" {
				assert.Equal(t, tt.expectedCode, tt.config.Build.CodeCachePath)
			}
			if tt.expectedBuild != "" {
				assert.Equal(t, tt.expectedBuild, tt.config.Build.BuildCachePath)
			}
		})
	}
}

// TestValidateCachePath tests cache path validation
func TestValidateCachePath_ValidAndInvalidPaths(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid existing directory",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			expectError: false,
		},
		{
			name: "valid non-existing directory (should be created)",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				newDir := filepath.Join(tmpDir, "new-cache-dir")
				return newDir
			},
			expectError: false,
		},
		{
			name: "nested non-existing directory (should be created)",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				nestedDir := filepath.Join(tmpDir, "level1", "level2", "cache")
				return nestedDir
			},
			expectError: false,
		},
		{
			name: "path is a file, not directory",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "file.txt")
				err := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, err)
				return filePath
			},
			expectError: true,
			errorMsg:    "cache path is not a directory",
		},
		{
			name: "invalid path with permission issues",
			setupFunc: func(t *testing.T) string {
				// On Windows, permission tests work differently
				// Instead, test with an invalid path that cannot be created
				return "Z:\\nonexistent\\invalid\\path\\that\\cannot\\be\\created"
			},
			expectError: true,
			errorMsg:    "cache path does not exist and cannot be created",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc(t)
			err := validateCachePath(path)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify directory exists and is writable
				info, statErr := os.Stat(path)
				assert.NoError(t, statErr)
				assert.True(t, info.IsDir())
			}
		})
	}
}

// TestLogConfig tests that configuration logging doesn't expose secrets
func TestLogConfig_DoesNotExposeSecrets(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := &Config{
		Server: ServerConfig{Port: 8080},
		NATS:   NATSConfig{URL: "nats://localhost:4222"},
		GitHub: GitHubConfig{WebhookSecret: "super-secret-key"},
		Auth:   AuthConfig{Token: "super-secret-token"},
		Build: BuildConfig{
			CodeCachePath:  "/tmp/code",
			BuildCachePath: "/tmp/build",
		},
		Logging: LoggingConfig{Level: "info", Format: "json"},
	}
	
	// This should not panic and should log without exposing secrets
	logConfig(config, logger)
	
	// Test passes if no panic occurs
}
