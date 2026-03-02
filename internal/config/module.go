package config

import "go.uber.org/fx"

// Module provides Config via fx.
var Module = fx.Module("config",
	fx.Provide(New),
)
