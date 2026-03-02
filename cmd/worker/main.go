package main

import (
	"context"

	buildahpkg "github.com/jorgerua/build-system/container-build-service/internal/buildah"
	"github.com/jorgerua/build-system/container-build-service/internal/config"
	githubpkg "github.com/jorgerua/build-system/container-build-service/internal/github"
	"github.com/jorgerua/build-system/container-build-service/internal/logging"
	"github.com/jorgerua/build-system/container-build-service/internal/metrics"
	natspkg "github.com/jorgerua/build-system/container-build-service/internal/nats"
	"github.com/jorgerua/build-system/container-build-service/internal/orchestrator"
	"github.com/jorgerua/build-system/container-build-service/internal/tidb"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		config.Module,
		logging.Module,
		metrics.Module,
		natspkg.Module,
		githubpkg.Module,
		tidb.Module,
		fx.Provide(
			tidb.NewVersionRepository,
			tidb.NewBuildStateRepository,
			tidb.NewBuildRecordRepository,
			natspkg.NewSubscriber,
			buildahpkg.New,
			orchestrator.New,
		),
		fx.Invoke(func(lc fx.Lifecycle, orch *orchestrator.Orchestrator, logger *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						if err := orch.Run(context.Background()); err != nil {
							logger.Error("orchestrator stopped", zap.Error(err))
						}
					}()
					return nil
				},
			})
		}),
	).Run()
}
