package shared

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
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
	URL string `mapstructure:"url"`
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
	PoolSize   int `mapstructure:"pool_size"`
	Timeout    int `mapstructure:"timeout"`
	MaxRetries int `mapstructure:"max_retries"`
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
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// validateConfig valida os campos obrigatórios da configuração
func validateConfig(config *Config) error {
	// Validar NATS
	if config.NATS.URL == "" {
		return fmt.Errorf("nats.url is required")
	}

	// Validar Server (apenas se configurado)
	if config.Server.Port != 0 {
		if config.Server.Port < 1 || config.Server.Port > 65535 {
			return fmt.Errorf("server.port must be between 1 and 65535")
		}
		if config.Server.ReadTimeout < 0 {
			return fmt.Errorf("server.read_timeout must be non-negative")
		}
		if config.Server.WriteTimeout < 0 {
			return fmt.Errorf("server.write_timeout must be non-negative")
		}
		if config.Server.ShutdownTimeout < 0 {
			return fmt.Errorf("server.shutdown_timeout must be non-negative")
		}
	}

	// Validar Worker (apenas se configurado)
	if config.Worker.PoolSize != 0 {
		if config.Worker.PoolSize < 1 {
			return fmt.Errorf("worker.pool_size must be at least 1")
		}
		if config.Worker.Timeout < 0 {
			return fmt.Errorf("worker.timeout must be non-negative")
		}
		if config.Worker.MaxRetries < 0 {
			return fmt.Errorf("worker.max_retries must be non-negative")
		}
	}

	// Validar Build (apenas se configurado)
	if config.Build.CodeCachePath != "" || config.Build.BuildCachePath != "" {
		if config.Build.CodeCachePath == "" {
			return fmt.Errorf("build.code_cache_path is required when build config is present")
		}
		if config.Build.BuildCachePath == "" {
			return fmt.Errorf("build.build_cache_path is required when build config is present")
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
			return fmt.Errorf("logging.level must be one of: debug, info, warn, error, fatal")
		}
	}

	if config.Logging.Format != "" {
		if config.Logging.Format != "json" && config.Logging.Format != "text" {
			return fmt.Errorf("logging.format must be either 'json' or 'text'")
		}
	}

	return nil
}
