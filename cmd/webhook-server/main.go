package main

import (
	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"github.com/jorgerua/build-system/container-build-service/internal/logging"
	"github.com/jorgerua/build-system/container-build-service/internal/metrics"
	natspkg "github.com/jorgerua/build-system/container-build-service/internal/nats"
	"github.com/jorgerua/build-system/container-build-service/internal/webhook"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		config.Module,
		logging.Module,
		metrics.Module,
		natspkg.Module,
		webhook.Module,
		fx.Provide(natspkg.NewPublisher),
	).Run()
}
