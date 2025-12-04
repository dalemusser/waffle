// app/app.go
package app

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/dalemusser/waffle/config"
	"github.com/dalemusser/waffle/logging"
	"github.com/dalemusser/waffle/metrics"
	"github.com/dalemusser/waffle/server"
	"go.uber.org/zap"
)

const defaultShutdownTimeout = 10 * time.Second

// Hooks defines the integration points an application must provide
// for WAFFLE to run it.
type Hooks[C any, D any] struct {
	// Name is used only for logging/diagnostics.
	Name string

	// LoadConfig must return both the core config (WAFFLE-level) and
	// the app-specific config. It typically calls waffle/config.Load
	// internally, plus any app-level config loading/validation.
	LoadConfig func(logger *zap.Logger) (*config.CoreConfig, C, error)

	// ValidateConfig can perform app-specific validation on the loaded
	// core and app config before any backends are connected. It may be
	// nil if the app doesn’t require extra validation. Returning an
	// error here will abort startup before any external resources are used.
	ValidateConfig func(core *config.CoreConfig, appCfg C, logger *zap.Logger) error

	// ConnectDB is responsible for connecting to any databases or backends
	// the app needs, using the core + app config. It should respect
	// cfg.DBConnectTimeout for its own timeouts.
	ConnectDB func(ctx context.Context, core *config.CoreConfig, appCfg C, logger *zap.Logger) (D, error)

	// EnsureSchema can run validators/index creation or other startup tasks
	// that depend on the DB being connected. It may be nil if the app
	// doesn’t need any schema bootstrapping.
	EnsureSchema func(ctx context.Context, core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) error

	// Startup runs one-time application initialization after DBs and schemas
	// are ready, but before the HTTP handler is built and requests are served.
	// It may be nil if the app doesn’t need any extra initialization
	// beyond config, DB, and schema setup.
	Startup func(ctx context.Context, core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) error

	// BuildHandler must construct the final http.Handler for the app:
	// this includes routers, Waffle middleware, app middleware, and routes.
	BuildHandler func(core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) (http.Handler, error)

	// Shutdown is called after the HTTP server has stopped and the
	// shutdown context has been canceled. It is the app's opportunity
	// to gracefully tear down any resources created in ConnectDB
	// (databases, caches, external clients, etc.). It may be nil if
	// the app doesn’t need explicit shutdown logic.
	Shutdown func(context.Context, *config.CoreConfig, C, D, *zap.Logger) error
}

// Run executes the standard Waffle startup sequence:
//
//  1. Bootstrap logger
//  2. Load core + app config (Hooks.LoadConfig)
//  3. Validate the loaded config (Hooks.ValidateConfig, if provided)
//  4. Build final logger based on core config
//  5. Register default metrics
//  6. Connect DB/backends (Hooks.ConnectDB)
//  7. Ensure schema/indexes (Hooks.EnsureSchema, if provided)
//  8. Wire shutdown signals to a context
//  9. Startup (Hooks.Startup, if provided)
//  10. Build the HTTP handler (Hooks.BuildHandler)
//  11. Start the HTTP(S) server and block until shutdown
//  12. Run the optional shutdown hook (Hooks.Shutdown) to clean up resources
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

	// 3) Optionally validate the loaded config before proceeding.
	if hooks.ValidateConfig != nil {
		if err := hooks.ValidateConfig(coreCfg, appCfg, bootstrap); err != nil {
			bootstrap.Error("config validation failed", zap.Error(err))
			os.Exit(1)
		}
	}

	// 4) Build final logger
	logger := logging.MustBuildLogger(coreCfg.LogLevel, coreCfg.Env)
	defer logger.Sync()
	logger.Info("logger initialized", zap.String("app", hooks.Name))

	// 5) Register default metrics (Go, process, HTTP histograms)
	metrics.RegisterDefault(logger)

	// 6) Connect DB/backends
	dbBundle, err := hooks.ConnectDB(ctx, coreCfg, appCfg, logger)
	if err != nil {
		logger.Error("DB connect failed", zap.Error(err))
		os.Exit(1)
	}

	// 7) Ensure schema/indexes (optional)
	if hooks.EnsureSchema != nil {
		schemaCtx, cancel := context.WithTimeout(context.Background(), coreCfg.IndexBootTimeout)
		defer cancel()

		if err := hooks.EnsureSchema(schemaCtx, coreCfg, appCfg, dbBundle, logger); err != nil {
			logger.Error("schema ensure failed", zap.Error(err))
			os.Exit(1)
		}
	}

	// 8) Wire shutdown signals → context
	ctx, cancel := server.WithShutdownSignals(ctx, logger)
	defer cancel()

	// 9) Startup (optional)
	if hooks.Startup != nil {
		if err := hooks.Startup(ctx, coreCfg, appCfg, dbBundle, logger); err != nil {
			logger.Error("startup failed", zap.Error(err))
			os.Exit(1)
		}
	}

	// 10) Build HTTP handler (router + middleware + routes)
	handler, err := hooks.BuildHandler(coreCfg, appCfg, dbBundle, logger)
	if err != nil {
		logger.Error("handler build failed", zap.Error(err))
		os.Exit(1)
	}

	// 11) Start HTTP server and wait for shutdown
	serverErr := server.ListenAndServeWithContext(ctx, coreCfg, handler, logger)
	if serverErr != nil {
		logger.Error("server exited with error", zap.Error(serverErr))
	} else {
		logger.Info("server stopped")
	}

	// 12) Run optional shutdown hook (cleanup)
	var shutdownErr error
	if hooks.Shutdown != nil {
		// defaultShutdownTimeout controls how long WAFFLE will wait
		// for the app's Shutdown hook to complete.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := hooks.Shutdown(shutdownCtx, coreCfg, appCfg, dbBundle, logger); err != nil {
			logger.Error("shutdown hook failed", zap.Error(err))
			shutdownErr = err
		}
	}

	// Prefer to return the server error if it exists,
	// otherwise return any shutdown error.
	if serverErr != nil {
		return serverErr
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	return nil
}
