package logging

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// New creates a production zap logger (JSON to stdout).
func New() (*zap.Logger, error) {
	return zap.NewProduction()
}

// Module provides *zap.Logger via fx.
var Module = fx.Module("logging",
	fx.Provide(New),
)
