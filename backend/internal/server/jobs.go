package server

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/internal/services"
	"github.com/rera/property-platform/backend/pkg/logger"
	"gorm.io/gorm"
)

// StartExpirySweep runs the stale-rental sweep on a timer and returns a function
// that stops it. Rental stock turns over in weeks: a listing nobody has
// confirmed in two months is the fastest way for a rental site to lose a
// renter's trust, so retiring them cannot wait for someone to press a button.
//
// This is a ticker rather than a scheduler because it is the only recurring job
// the platform has. A second one would be the moment to introduce a real
// scheduler instead of a second goroutine here.
func StartExpirySweep(cfg *config.Config, db *gorm.DB) func() {
	interval := time.Duration(cfg.Rental.SweepIntervalHours) * time.Hour
	if interval <= 0 {
		logger.Log.Info("rental expiry sweep is disabled (RENTAL_SWEEP_INTERVAL_HOURS=0)")
		return func() {}
	}

	expiry := services.NewExpiryService(
		repositories.NewPropertyRepository(db),
		repositories.NewAccountRepository(db),
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			// Run once at startup so a server that has been down over a weekend
			// does not serve stale listings until the first tick.
			result, err := expiry.Sweep(ctx)
			if err != nil {
				logger.Log.Error("rental expiry sweep failed", "error", err)
			} else if result.Expired > 0 || result.Warned > 0 {
				logger.Log.Info("rental expiry sweep",
					"expired", result.Expired, "warned", result.Warned)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return cancel
}
