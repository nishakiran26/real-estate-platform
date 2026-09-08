// Command server is the entrypoint of the RERA property platform API. It loads
// configuration, opens the database and cache, and hands the dependency graph
// to internal/server. All behaviour lives in the packages under internal/.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/database"
	"github.com/rera/property-platform/backend/internal/server"
	"github.com/rera/property-platform/backend/internal/storage"
	"github.com/rera/property-platform/backend/pkg/logger"
	"gorm.io/gorm"
)

func main() {
	// A .env file is a convenience for local runs; real environment variables
	// (what Docker and any deployment target provide) always take precedence.
	config.LoadDotEnv()

	cfg := config.Load()
	logger.Init(cfg.App.Env)

	if err := cfg.Validate(); err != nil {
		logger.Log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	if err := database.EnsureDatabase(cfg); err != nil {
		logger.Log.Error("could not reach MySQL", "error", err)
		os.Exit(1)
	}

	db, err := database.NewMySQL(cfg.DSN())
	if err != nil {
		logger.Log.Error("could not connect to MySQL", "error", err)
		os.Exit(1)
	}
	if err := prepareSchema(db, cfg); err != nil {
		logger.Log.Error("could not prepare the database", "error", err)
		os.Exit(1)
	}

	// Redis is optional: a failed connection degrades caching and rate limiting
	// to their fallbacks rather than stopping the service.
	redis := cache.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	defer redis.Close()
	if !redis.Available() {
		logger.Log.Warn("redis is unavailable; caching and rate limiting fall back to in-process behaviour")
	}

	store, err := buildStorage(cfg)
	if err != nil {
		logger.Log.Error("could not initialise file storage", "error", err)
		os.Exit(1)
	}

	e := server.New(cfg, db, redis, store)

	// Stale rentals are retired on a timer as well as on demand.
	stopSweep := server.StartExpirySweep(cfg, db)
	defer stopSweep()

	go func() {
		addr := ":" + cfg.Server.Port
		logger.Log.Info("server starting", "addr", addr, "env", cfg.App.Env)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	waitForShutdown(e)
}

// prepareSchema applies migrations and seed data when enabled.
func prepareSchema(db *gorm.DB, cfg *config.Config) error {
	if cfg.Database.AutoMigrate {
		logger.Log.Info("applying database migrations")
		if err := database.AutoMigrate(db); err != nil {
			return err
		}
	}
	if cfg.Database.Seed {
		logger.Log.Info("seeding reference data")
		if err := database.Seed(db, cfg); err != nil {
			return err
		}
	}
	return nil
}

// buildStorage selects the file storage provider named in the configuration.
// Adding S3 or Cloudinary later means adding a case here and nothing else.
func buildStorage(cfg *config.Config) (storage.Storage, error) {
	switch cfg.Storage.Driver {
	case "local", "":
		return storage.NewLocal(cfg.Storage.LocalDir, cfg.Storage.BaseURL)
	default:
		logger.Log.Warn("unknown storage driver, falling back to local disk",
			"driver", cfg.Storage.Driver)
		return storage.NewLocal(cfg.Storage.LocalDir, cfg.Storage.BaseURL)
	}
}

// waitForShutdown drains in-flight requests on SIGINT or SIGTERM.
func waitForShutdown(e *echo.Echo) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Log.Error("graceful shutdown failed", "error", err)
	}
}
