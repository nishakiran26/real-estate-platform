// Package server builds the HTTP application: it wires repositories, services
// and handlers together and returns a configured Echo instance. Keeping the
// dependency graph here rather than in main lets tests spin up the real
// application against a test database.
package server

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/handlers"
	"github.com/rera/property-platform/backend/internal/middleware"
	"github.com/rera/property-platform/backend/internal/notification"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/internal/routes"
	"github.com/rera/property-platform/backend/internal/services"
	"github.com/rera/property-platform/backend/internal/storage"
	"github.com/rera/property-platform/backend/internal/validators"
	"github.com/rera/property-platform/backend/pkg/auth"
	"github.com/rera/property-platform/backend/pkg/logger"
	"gorm.io/gorm"
)

// BuildNotifier registers one sender per channel. The log sender is the
// development default; swapping in a vendor is a change to this function only.
func BuildNotifier(cfg *config.Config) *notification.Dispatcher {
	var email notification.Sender = notification.NewLogSender(notification.ChannelEmail, logger.Log)
	if cfg.Mail.Driver == "smtp" {
		email = notification.NewSMTPSender(cfg.Mail.Host, cfg.Mail.Port, cfg.Mail.Username,
			cfg.Mail.Password, cfg.Mail.From, cfg.Mail.FromName)
	}
	return notification.NewDispatcher(
		email,
		notification.NewLogSender(notification.ChannelSMS, logger.Log),
		notification.NewLogSender(notification.ChannelWhatsApp, logger.Log),
	)
}

// New wires every layer together and returns the configured Echo instance.
// This is the only place where concrete implementations are chosen.
func New(cfg *config.Config, db *gorm.DB, redis *cache.Client, store storage.Storage) *echo.Echo {
	// Repositories: the only layer that talks to MySQL.
	// Repositories: the only layer that talks to MySQL.
	userRepo := repositories.NewUserRepository(db)
	authRepo := repositories.NewAuthRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	geoRepo := repositories.NewGeoRepository(db)
	propertyRepo := repositories.NewPropertyRepository(db)
	mediaRepo := repositories.NewMediaRepository(db)
	engagementRepo := repositories.NewEngagementRepository(db)
	analyticsRepo := repositories.NewAnalyticsRepository(db)
	cmsRepo := repositories.NewCMSRepository(db)
	visitRepo := repositories.NewVisitRepository(db)
	reportRepo := repositories.NewReportRepository(db)
	alertRepo := repositories.NewAlertRepository(db)
	statsRepo := repositories.NewStatsRepository(db)

	// Cross-cutting infrastructure.
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL, cfg.JWT.Issuer)
	notifier := BuildNotifier(cfg)

	// Services: all business logic.
	propertyService := services.NewPropertyService(propertyRepo, geoRepo, accountRepo, engagementRepo, redis, cfg)
	recommendationService := services.NewRecommendationService(engagementRepo, accountRepo, propertyRepo)
	authService := services.NewAuthService(userRepo, authRepo, accountRepo, jwtService, redis, notifier, cfg)
	accountService := services.NewAccountService(userRepo, accountRepo, engagementRepo, recommendationService)
	engagementService := services.NewEngagementService(engagementRepo, propertyRepo, accountRepo,
		userRepo, mediaRepo, notifier, cfg)
	geoService := services.NewGeoService(geoRepo, redis)
	cmsService := services.NewCMSService(cmsRepo, geoRepo, propertyService, redis)
	mediaService := services.NewMediaService(mediaRepo, propertyRepo, store, propertyService)
	alertService := services.NewAlertService(alertRepo, accountRepo, geoRepo, notifier)
	adminService := services.NewAdminService(propertyRepo, userRepo, accountRepo, engagementRepo,
		notifier, propertyService, alertService)
	analyticsService := services.NewAnalyticsService(analyticsRepo, propertyRepo, engagementRepo,
		userRepo, accountRepo, statsRepo, reportRepo)
	visitService := services.NewVisitService(visitRepo, propertyRepo, accountRepo,
		engagementRepo, notifier, cfg)
	reportService := services.NewReportService(reportRepo, propertyRepo, accountRepo, userRepo)
	expiryService := services.NewExpiryService(propertyRepo, accountRepo)
	seoService := services.NewSEOService(propertyRepo, geoRepo, cmsRepo, cfg)

	// Handlers: HTTP only.
	handlerSet := routes.Handlers{
		Auth: handlers.NewAuthHandler(authService),
		Public: handlers.NewPublicHandler(propertyService, geoService, cmsService,
			engagementService, recommendationService, accountService, visitService,
			reportService, analyticsService, seoService),
		Customer: handlers.NewCustomerHandler(accountService, engagementService,
			recommendationService, visitService, alertService),
		Client: handlers.NewClientHandler(propertyService, mediaService, engagementService,
			accountService, analyticsService, visitService),
		Admin: handlers.NewAdminHandler(adminService, analyticsService, propertyService,
			geoService, cmsService, engagementService, mediaService, reportService,
			expiryService),
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = validators.New()
	e.Server.ReadTimeout = time.Duration(cfg.Server.ReadTimeoutSec) * time.Second
	e.Server.WriteTimeout = time.Duration(cfg.Server.WriteTimeoutSec) * time.Second

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())
	e.Use(middleware.SecurityHeaders())
	e.Use(middleware.Session(cfg.IsProduction()))
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     cfg.CORS.Origins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Session-Id"},
		AllowCredentials: true,
		MaxAge:           600,
	}))
	e.Use(echomw.BodyLimit("25M"))
	e.Use(echomw.GzipWithConfig(echomw.GzipConfig{Level: 5}))

	// Uploaded files are served from disk under the local storage driver.
	if cfg.Storage.Driver == "local" || cfg.Storage.Driver == "" {
		e.Static("/uploads", cfg.Storage.LocalDir)
	}

	routes.Register(e, handlerSet, routes.Deps{
		Config:    cfg,
		Auth:      middleware.NewAuth(jwtService),
		RateLimit: middleware.NewRateLimiter(redis),
	})
	return e
}
