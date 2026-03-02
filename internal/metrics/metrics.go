package metrics

import (
	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"go.uber.org/fx"
)

// New creates a DogStatsD client.
func New(cfg *config.Config) (statsd.ClientInterface, error) {
	return statsd.New(cfg.Metrics.DogStatsDAddr,
		statsd.WithNamespace("container_build_service."),
	)
}

// Module provides statsd.ClientInterface via fx.
var Module = fx.Module("metrics",
	fx.Provide(New),
)
