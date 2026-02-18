package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config representa a configuração completa do sistema
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	NATS    NATSConfig    `mapstructure:"nats"`
	GitHub  GitHubConfig  `mapstructure:"github"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Worker  WorkerConfig  `mapstructure:"worker"`
	Build   BuildConfig   `mapstructure:"build"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// ServerConfig contém configurações do servidor HTTP
type ServerConfig struct {
	Port            int `mapstructure:"port"`
	ReadTimeout     int `mapstructure:"read_timeout"`
	WriteTimeout    int `mapstructure:"write_timeout"`
	ShutdownTimeout int `mapstructure:"shutdown_timeout"`
}

// NATSConfig contém configurações do NATS
type NATSConfig struct {
	URL            string        `mapstructure:"url"`
	ReconnectWait  time.Duration `mapstructure:"reconnect_wait"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

// GitHubConfig contém configurações do GitHub
type GitHubConfig struct {
	WebhookSecret string `mapstructure:"webhook_secret"`
}

// AuthConfig contém configurações de autenticação
type AuthConfig struct {
	Token string `mapstructure:"token"`
}

// WorkerConfig contém configurações do worker
type WorkerConfig struct {
	PoolSize   int           `mapstructure:"pool_size"`
	QueueSize  int           `mapstructure:"queue_size"`
	Timeout    int           `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
	RetryDelay time.Duration `mapstructure:"retry_delay"`
}

// BuildConfig contém configurações de build
type BuildConfig struct {
	CodeCachePath  string `mapstructure:"code_cache_path"`
	BuildCachePath string `mapstructure:"build_cache_path"`
}

// LoggingConfig contém configurações de logging
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// LoadConfig carrega a configuração de um arquivo YAML com suporte a variáveis de ambiente
func LoadConfig(configPath string, logger *zap.Logger) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	
	// Permitir override via variáveis de ambiente
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Ler arquivo de configuração
	if err := viper.ReadInConfig(); err != nil {
		logger.Warn("Failed to read config file, using environment variables only",
			zap.String("config_path", configPath),
			zap.Error(err))
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expandir variáveis de ambiente em strings de configuração
	expandEnvVars(&config)

	// Validar configuração obrigatória
	if err := validateConfig(&config, logger); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Log configuração carregada (sem secrets)
	logConfig(&config, logger)

	return &config, nil
}

// expandEnvVars expande variáveis de ambiente em formato ${VAR_NAME}
func expandEnvVars(config *Config) {
	config.GitHub.WebhookSecret = os.ExpandEnv(config.GitHub.WebhookSecret)
	config.Auth.Token = os.ExpandEnv(config.Auth.Token)
	config.NATS.URL = os.ExpandEnv(config.NATS.URL)
	config.Build.CodeCachePath = os.ExpandEnv(config.Build.CodeCachePath)
	config.Build.BuildCachePath = os.ExpandEnv(config.Build.BuildCachePath)
}

// validateConfig valida os campos obrigatórios da configuração
func validateConfig(config *Config, logger *zap.Logger) error {
	var errors []string

	// Validar GitHub webhook secret
	if config.GitHub.WebhookSecret == "" || config.GitHub.WebhookSecret == "${GITHUB_WEBHOOK_SECRET}" {
		errors = append(errors, "GITHUB_WEBHOOK_SECRET is required but not set")
	}

	// Validar auth token
	if config.Auth.Token == "" || config.Auth.Token == "${API_AUTH_TOKEN}" {
		errors = append(errors, "API_AUTH_TOKEN is required but not set")
	}

	// Validar NATS
	if config.NATS.URL == "" {
		errors = append(errors, "NATS URL is required but not set")
	}

	// Validar Server (apenas se configurado)
	if config.Server.Port != 0 {
		if config.Server.Port < 1 || config.Server.Port > 65535 {
			errors = append(errors, fmt.Sprintf("invalid server port: %d", config.Server.Port))
		}
		if config.Server.ReadTimeout < 0 {
			errors = append(errors, "server read timeout cannot be negative")
		}
		if config.Server.WriteTimeout < 0 {
			errors = append(errors, "server write timeout cannot be negative")
		}
		if config.Server.ShutdownTimeout < 0 {
			errors = append(errors, "server shutdown timeout cannot be negative")
		}
	}

	// Validar Worker (apenas se configurado)
	if config.Worker.PoolSize != 0 {
		if config.Worker.PoolSize < 1 {
			errors = append(errors, "worker pool size must be at least 1")
		}
		if config.Worker.Timeout < 0 {
			errors = append(errors, "worker timeout cannot be negative")
		}
		if config.Worker.MaxRetries < 0 {
			errors = append(errors, "worker max retries cannot be negative")
		}
	}

	// Validar Build (apenas se configurado)
	if config.Build.CodeCachePath != "" || config.Build.BuildCachePath != "" {
		if config.Build.CodeCachePath == "" {
			errors = append(errors, "build code cache path is required when build config is present")
		}
		if config.Build.BuildCachePath == "" {
			errors = append(errors, "build build cache path is required when build config is present")
		}
	}

	// Validar paths de cache (se configurados)
	if config.Build.CodeCachePath != "" {
		if err := validateCachePath(config.Build.CodeCachePath); err != nil {
			errors = append(errors, fmt.Sprintf("invalid code cache path: %v", err))
		}
	}
	if config.Build.BuildCachePath != "" {
		if err := validateCachePath(config.Build.BuildCachePath); err != nil {
			errors = append(errors, fmt.Sprintf("invalid build cache path: %v", err))
		}
	}

	// Validar Logging
	if config.Logging.Level != "" {
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
			"fatal": true,
		}
		if !validLevels[config.Logging.Level] {
			errors = append(errors, "logging level must be one of: debug, info, warn, error, fatal")
		}
	}

	if config.Logging.Format != "" {
		if config.Logging.Format != "json" && config.Logging.Format != "text" {
			errors = append(errors, "logging format must be either 'json' or 'text'")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateCachePath verifica se path de cache existe e é gravável
func validateCachePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Tentar criar o diretório
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("cache path does not exist and cannot be created: %w", err)
			}
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("cache path is not a directory")
	}

	// Testar se é gravável
	testFile := filepath.Join(path, ".write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cache path is not writable: %w", err)
	}
	os.Remove(testFile)

	return nil
}

// logConfig registra configuração carregada (sem expor secrets)
func logConfig(config *Config, logger *zap.Logger) {
	logger.Info("Configuration loaded",
		zap.Int("server_port", config.Server.Port),
		zap.String("nats_url", config.NATS.URL),
		zap.String("log_level", config.Logging.Level),
		zap.String("log_format", config.Logging.Format),
		zap.Bool("github_secret_set", config.GitHub.WebhookSecret != ""),
		zap.Bool("auth_token_set", config.Auth.Token != ""),
		zap.String("code_cache_path", config.Build.CodeCachePath),
		zap.String("build_cache_path", config.Build.BuildCachePath),
	)
}
