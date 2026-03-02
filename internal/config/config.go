package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all service configuration.
type Config struct {
	NATS     NATSConfig
	TiDB     TiDBConfig
	GitHub   GitHubConfig
	Registry RegistryConfig
	Worker   WorkerConfig
	Buildah  BuildahConfig
	Metrics  MetricsConfig
}

type NATSConfig struct {
	URL          string `mapstructure:"url"`
	StreamName   string `mapstructure:"stream_name"`
	Subject      string `mapstructure:"subject"`
	ConsumerName string `mapstructure:"consumer_name"`
	// AckWait in seconds
	AckWaitSeconds int `mapstructure:"ack_wait_seconds"`
	MaxDelivers    int `mapstructure:"max_delivers"`
}

type TiDBConfig struct {
	DSN string `mapstructure:"dsn"`
}

type GitHubConfig struct {
	AppID          int64  `mapstructure:"app_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
}

type RegistryConfig struct {
	URL      string `mapstructure:"url"`
	AuthFile string `mapstructure:"auth_file"`
}

type WorkerConfig struct {
	Concurrency        int `mapstructure:"concurrency"`
	MaxBuildRetries    int `mapstructure:"max_build_retries"`
	StaleClaimMinutes  int `mapstructure:"stale_claim_minutes"`
	HeartbeatSeconds   int `mapstructure:"heartbeat_seconds"`
}

type BuildahConfig struct {
	StorageRoot   string `mapstructure:"storage_root"`
	StorageDriver string `mapstructure:"storage_driver"` // set at startup by detection
}

type MetricsConfig struct {
	DogStatsDAddr string `mapstructure:"dogstatsd_addr"`
}

// New loads configuration from file + environment variables.
func New() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/container-build-service")

	setDefaults(v)

	v.SetEnvPrefix("CBS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.ReadInConfig() // missing file is acceptable; env vars take precedence

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.stream_name", "BUILDS")
	v.SetDefault("nats.subject", "builds.jobs")
	v.SetDefault("nats.consumer_name", "build-worker")
	v.SetDefault("nats.ack_wait_seconds", 300)  // 5 minutes
	v.SetDefault("nats.max_delivers", 3)
	v.SetDefault("worker.concurrency", 3)
	v.SetDefault("worker.max_build_retries", 3)
	v.SetDefault("worker.stale_claim_minutes", 30)
	v.SetDefault("worker.heartbeat_seconds", 120) // 2 minutes
	v.SetDefault("buildah.storage_root", "/var/lib/buildah")
	v.SetDefault("metrics.dogstatsd_addr", "localhost:8125")
}
