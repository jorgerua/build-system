package tidb

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"go.uber.org/fx"
)

// New opens a TiDB (MySQL-compatible) connection pool.
func New(cfg *config.Config, lc fx.Lifecycle) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.TiDB.DSN)
	if err != nil {
		return nil, fmt.Errorf("tidb open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("tidb ping: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return db.Close()
		},
	})

	return db, nil
}

// Module provides *sql.DB via fx.
var Module = fx.Module("tidb",
	fx.Provide(New),
)
