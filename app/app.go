// app/app.go
package app

import (
	"context"
	"net/http"
	"os"

	"github.com/dalemusser/waffle/config"
	"github.com/dalemusser/waffle/logging"
	"github.com/dalemusser/waffle/metrics"
	"github.com/dalemusser/waffle/server"
	"go.uber.org/zap"
)

// Hooks defines the integration points an application must provide
// for WAFFLE to run it.
type Hooks[C any, D any] struct {
	// Name is used only for logging/diagnostics.
	Name string

	// LoadConfig must return both the core config (WAFFLE-level) and
	// the app-specific config. It typically calls waffle/config.Load
	// internally, plus any app-level config loading/validation.
	LoadConfig func(logger *zap.Logger) (*config.CoreConfig, C, error)

	// ConnectDB is responsible for connecting to any databases or backends
	// the app needs, using the core + app config. It should respect
	// cfg.DBConnectTimeout for its own timeouts.
	ConnectDB func(ctx context.Context, core *config.CoreConfig, appCfg C, logger *zap.Logger) (D, error)

	// EnsureSchema can run validators/index creation or other startup tasks
	// that depend on the DB being connected. It may be nil if the app
	// doesn’t need any schema bootstrapping.
	EnsureSchema func(ctx context.Context, core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) error

	// BuildHandler must construct the final http.Handler for the app:
	// this includes routers, Waffle middleware, app middleware, and routes.
	BuildHandler func(core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) (http.Handler, error)
}

// Run executes the standard Waffle startup sequence:
//
//  1. Bootstrap logger
//  2. Load core + app config (Hooks.LoadConfig)
//  3. Build final logger based on core config
//  4. Register default metrics
//  5. Connect DB/backends (Hooks.ConnectDB)
//  6. Ensure schema/indexes (Hooks.EnsureSchema, if provided)
//  7. Wire shutdown signals to a context
//  8. Build the HTTP handler (Hooks.BuildHandler)
//  9. Start the HTTP(S) server and block until shutdown
func Run[C any, D any](ctx context.Context, hooks Hooks[C, D]) error {
	// 1) Bootstrap logger for early startup
	bootstrap := logging.BootstrapLogger()
	defer bootstrap.Sync()
	bootstrap.Info("bootstrap logger initialized", zap.String("app", hooks.Name))

	// 2) Load config (core + app-specific)
	coreCfg, appCfg, err := hooks.LoadConfig(bootstrap)
	if err != nil {
		bootstrap.Error("config load failed", zap.Error(err))
		// For a runner, exiting here is correct.
		os.Exit(1)
	}
	bootstrap.Info("config loaded",
		zap.String("env", coreCfg.Env),
		zap.String("log_level", coreCfg.LogLevel),
	)

	// 3) Build final logger
	logger := logging.MustBuildLogger(coreCfg.LogLevel, coreCfg.Env)
	defer logger.Sync()
	logger.Info("logger initialized", zap.String("app", hooks.Name))

	// 4) Register default metrics (Go, process, HTTP histograms)
	metrics.RegisterDefault(logger)

	// 5) Connect DB/backends
	dbBundle, err := hooks.ConnectDB(ctx, coreCfg, appCfg, logger)
	if err != nil {
		logger.Error("DB connect failed", zap.Error(err))
		os.Exit(1)
	}

	// 6) Ensure schema/indexes (optional)
	if hooks.EnsureSchema != nil {
		schemaCtx, cancel := context.WithTimeout(context.Background(), coreCfg.IndexBootTimeout)
		defer cancel()

		if err := hooks.EnsureSchema(schemaCtx, coreCfg, appCfg, dbBundle, logger); err != nil {
			logger.Error("schema ensure failed", zap.Error(err))
			os.Exit(1)
		}
	}

	// 7) Wire shutdown signals → context
	ctx, cancel := server.WithShutdownSignals(ctx, logger)
	defer cancel()

	// 8) Build HTTP handler (router + middleware + routes)
	handler, err := hooks.BuildHandler(coreCfg, appCfg, dbBundle, logger)
	if err != nil {
		logger.Error("handler build failed", zap.Error(err))
		os.Exit(1)
	}

	// 9) Start HTTP server
	if err := server.ListenAndServeWithContext(ctx, coreCfg, handler, logger); err != nil {
		logger.Error("server exited with error", zap.Error(err))
		return err
	}
	logger.Info("server stopped")
	return nil
}
