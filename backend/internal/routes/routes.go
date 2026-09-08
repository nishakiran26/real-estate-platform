// Package routes maps URLs onto handlers and attaches the middleware each
// group needs. It is the single place to see the platform's public surface.
package routes

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/handlers"
	"github.com/rera/property-platform/backend/internal/middleware"
)

// Handlers bundles the controllers the router wires up.
type Handlers struct {
	Auth     *handlers.AuthHandler
	Public   *handlers.PublicHandler
	Customer *handlers.CustomerHandler
	Client   *handlers.ClientHandler
	Admin    *handlers.AdminHandler
}

// Deps carries the middleware and configuration the router needs.
type Deps struct {
	Config    *config.Config
	Auth      *middleware.AuthMiddleware
	RateLimit *middleware.RateLimiter
}

// Register mounts every route on e.
func Register(e *echo.Echo, h Handlers, d Deps) {
	minute := time.Minute
	hour := time.Hour

	publicLimit := d.RateLimit.Limit("public", d.Config.RateLimit.PublicPerMinute, minute)
	authLimit := d.RateLimit.Limit("auth", d.Config.RateLimit.AuthPerMinute, minute)
	otpLimit := d.RateLimit.Limit("otp", d.Config.RateLimit.OTPPerHour, hour)
	leadLimit := d.RateLimit.Limit("lead", 10, hour)

	e.GET("/health", h.Public.Health)
	e.GET("/sitemap.xml", h.Public.Sitemap)
	e.GET("/robots.txt", h.Public.Robots)

	v1 := e.Group("/api/v1")

	registerAuth(v1, h, d, authLimit, otpLimit)
	registerPublic(v1, h, d, publicLimit, leadLimit)
	registerCustomer(v1, h, d)
	registerClient(v1, h, d)
	registerAdmin(v1, h, d)
}

// registerAuth mounts the authentication endpoints. Credential and OTP routes
// are rate limited because they are the ones worth brute-forcing.
func registerAuth(v1 *echo.Group, h Handlers, d Deps, authLimit, otpLimit echo.MiddlewareFunc) {
	g := v1.Group("/auth")

	g.POST("/register", h.Auth.Register, authLimit)
	g.POST("/login", h.Auth.Login, authLimit)
	g.POST("/login-otp", h.Auth.LoginWithOTP, authLimit)
	g.POST("/send-otp", h.Auth.SendOTP, otpLimit)
	g.POST("/resend-otp", h.Auth.SendOTP, otpLimit)
	g.POST("/verify-otp", h.Auth.VerifyOTP, authLimit)
	g.POST("/forgot-password", h.Auth.ForgotPassword, otpLimit)
	g.POST("/reset-password", h.Auth.ResetPassword, authLimit)
	g.POST("/refresh-token", h.Auth.Refresh, authLimit)

	// Endpoints that require a valid access token.
	secured := v1.Group("/auth", d.Auth.Require)
	secured.POST("/logout", h.Auth.Logout)
	secured.POST("/change-password", h.Auth.ChangePassword)
	secured.GET("/me", h.Auth.Me)
	secured.GET("/login-history", h.Auth.LoginHistory)
}

// registerPublic mounts everything a visitor can reach without an account.
// Optional auth is applied so responses can be personalised (favourite flags)
// when a token happens to be present.
func registerPublic(v1 *echo.Group, h Handlers, d Deps, publicLimit, leadLimit echo.MiddlewareFunc) {
	g := v1.Group("", d.Auth.Optional, publicLimit)

	g.GET("/homepage", h.Public.Homepage)
	g.GET("/meta", h.Public.Meta)

	// A property is addressable by numeric ID or by SEO slug through the same
	// route; static segments below take priority over the :key parameter.
	g.GET("/properties", h.Public.ListProperties)
	g.GET("/properties/compare", h.Public.CompareProperties)
	g.GET("/properties/recently-viewed", h.Public.RecentlyViewed)
	g.GET("/properties/recommended", h.Public.Recommendations)
	g.GET("/properties/:key", h.Public.GetProperty)
	g.GET("/properties/:key/similar", h.Public.SimilarProperties)
	g.GET("/properties/:key/whatsapp", h.Public.WhatsAppLink)
	g.GET("/properties/:key/documents/:doc_id/download", h.Public.DownloadDocument)

	g.GET("/cities", h.Public.Cities)
	g.GET("/cities/:slug", h.Public.City)
	g.GET("/localities", h.Public.Localities)
	g.GET("/cities/:slug/localities/:locality_slug", h.Public.Locality)
	g.GET("/amenities", h.Public.Amenities)
	// What a flat here actually goes for, from live listings only.
	g.GET("/rent-guide", h.Public.RentGuide)
	g.GET("/builders/:id", h.Public.Builder)

	g.GET("/pages", h.Public.Pages)
	g.GET("/pages/:slug", h.Public.Page)

	g.POST("/events", h.Public.TrackEvent)
	// Lead submission is limited separately: it is open to anonymous visitors.
	g.POST("/inquiries", h.Public.CreateInquiry, leadLimit)
	// A viewing request is the same kind of action, so it shares the limit.
	g.POST("/properties/:key/visits", h.Public.RequestVisit, leadLimit)
	// Reporting a listing is open to anonymous visitors for the same reason a
	// lead is: the person who notices a flat is already let is the one searching
	// for it, and they will not create an account to say so.
	g.POST("/properties/:key/report", h.Public.ReportProperty, leadLimit)
}

// registerCustomer mounts the buyer account area.
func registerCustomer(v1 *echo.Group, h Handlers, d Deps) {
	g := v1.Group("/customer", d.Auth.Require,
		d.Auth.RequireRole(middleware.RoleCustomer, middleware.RoleAdmin))

	g.GET("/dashboard", h.Customer.Dashboard)
	g.PUT("/profile", h.Customer.UpdateProfile)

	g.GET("/preferences", h.Customer.GetPreferences)
	g.PUT("/preferences", h.Customer.SavePreferences)

	g.PUT("/consent", h.Customer.UpdateConsent)
	g.POST("/unsubscribe", h.Customer.Unsubscribe)

	// One account, many intents: a buyer who wants to rent out a flat upgrades
	// here rather than registering again.
	g.POST("/listing-account", h.Customer.EnableListingAccount)

	g.GET("/favorites", h.Customer.ListFavorites)
	g.POST("/favorites/:property_id", h.Customer.AddFavorite)
	g.DELETE("/favorites/:property_id", h.Customer.RemoveFavorite)

	g.GET("/recently-viewed", h.Customer.RecentlyViewed)
	g.GET("/visits", h.Customer.ListVisits)

	// Saved searches: a renter who has not found the flat yet should not have to
	// check the site every day.
	g.GET("/rental-alerts", h.Customer.ListRentalAlerts)
	g.POST("/rental-alerts", h.Customer.CreateRentalAlert)
	g.PUT("/rental-alerts/:id", h.Customer.UpdateRentalAlert)
	g.DELETE("/rental-alerts/:id", h.Customer.DeleteRentalAlert)
	g.GET("/recommendations", h.Customer.Recommendations)
	g.GET("/inquiries", h.Customer.ListInquiries)

	g.GET("/notifications", h.Customer.ListNotifications)
	g.PUT("/notifications/:id/read", h.Customer.MarkNotificationRead)
	g.PUT("/notifications/read-all", h.Customer.MarkAllNotificationsRead)

	g.GET("/notification-preferences", h.Customer.GetNotificationPreferences)
	g.PUT("/notification-preferences", h.Customer.SaveNotificationPreferences)
}

// registerClient mounts the builder portal. Ownership is enforced inside the
// services, not by the route, so an administrator can use the same endpoints.
func registerClient(v1 *echo.Group, h Handlers, d Deps) {
	g := v1.Group("/client", d.Auth.Require,
		d.Auth.RequireRole(middleware.RoleClient, middleware.RoleAdmin))

	g.GET("/dashboard", h.Client.Dashboard)
	g.GET("/profile", h.Client.GetProfile)
	g.PUT("/profile", h.Client.UpdateProfile)
	g.POST("/logo", h.Client.UploadLogo)

	g.GET("/properties", h.Client.ListProperties)
	g.POST("/properties", h.Client.CreateProperty)
	g.GET("/properties/:id", h.Client.GetProperty)
	g.PUT("/properties/:id", h.Client.UpdateProperty)
	g.DELETE("/properties/:id", h.Client.DeleteProperty)
	g.POST("/properties/:id/submit", h.Client.SubmitProperty)
	g.GET("/properties/:id/status-history", h.Client.StatusHistory)
	g.GET("/properties/:id/analytics", h.Client.PropertyAnalytics)

	g.GET("/properties/:id/media", h.Client.ListMedia)
	g.POST("/properties/:id/media", h.Client.UploadMedia)
	g.DELETE("/properties/:id/media/:media_id", h.Client.DeleteMedia)
	g.PUT("/properties/:id/media/cover", h.Client.SetCoverImage)
	g.PUT("/properties/:id/media/reorder", h.Client.ReorderMedia)

	g.GET("/properties/:id/documents", h.Client.ListDocuments)
	g.POST("/properties/:id/documents", h.Client.UploadDocument)
	g.DELETE("/properties/:id/documents/:doc_id", h.Client.DeleteDocument)

	// Availability and renewal are separate from the approval workflow: letting
	// a flat, or confirming it is still free, must not send the listing back for
	// review.
	g.PUT("/properties/:id/availability", h.Client.SetAvailability)
	g.POST("/properties/:id/renew", h.Client.RenewListing)
	g.GET("/expiring-listings", h.Client.ExpiringListings)

	g.GET("/visits", h.Client.ListVisits)
	g.PUT("/visits/:id", h.Client.RespondToVisit)

	g.GET("/inquiries", h.Client.ListInquiries)
	g.GET("/inquiries/:id", h.Client.GetInquiry)
	g.PUT("/inquiries/:id/status", h.Client.UpdateInquiryStatus)
}

// registerAdmin mounts the administrator console.
func registerAdmin(v1 *echo.Group, h Handlers, d Deps) {
	g := v1.Group("/admin", d.Auth.Require, d.Auth.RequireRole(middleware.RoleAdmin))

	g.GET("/dashboard", h.Admin.Dashboard)
	g.GET("/analytics", h.Admin.Analytics)
	g.GET("/audit-logs", h.Admin.ListAuditLogs)

	g.GET("/properties", h.Admin.ListProperties)
	g.GET("/properties/:id", h.Admin.GetProperty)
	g.GET("/properties/:id/status-history", h.Admin.PropertyStatusHistory)
	g.PUT("/properties/:id/review", h.Admin.StartReview)
	g.PUT("/properties/:id/approve", h.Admin.ApproveProperty)
	g.PUT("/properties/:id/reject", h.Admin.RejectProperty)
	g.PUT("/properties/:id/rera", h.Admin.VerifyRERA)
	g.PUT("/properties/:id/featured", h.Admin.SetFeatured)

	g.GET("/users", h.Admin.ListUsers)
	g.GET("/customers", h.Admin.ListCustomers)
	g.GET("/clients", h.Admin.ListClients)
	g.PUT("/users/:id/status", h.Admin.UpdateUserStatus)
	g.PUT("/users/:id/roles", h.Admin.AssignRoles)
	g.GET("/roles", h.Admin.ListRoles)

	g.GET("/inquiries", h.Admin.ListInquiries)

	// Moderation: once anyone can post a flat, some listings are wrong or
	// dishonest, and someone has to answer for them.
	g.GET("/reports", h.Admin.ListReports)
	g.PUT("/reports/:id", h.Admin.ResolveReport)
	// The sweep also runs on a timer; this is how an administrator runs it now
	// and sees what it did.
	g.POST("/listings/sweep-expiry", h.Admin.SweepExpiredListings)

	// Trust and coverage: an administrator records the outcome of a check made
	// elsewhere, and can post for an owner who is not at a keyboard.
	g.PUT("/clients/:id/verification", h.Admin.SetClientVerification)
	g.POST("/clients/:id/properties", h.Admin.CreatePropertyForClient)

	g.POST("/cities", h.Admin.CreateCity)
	g.PUT("/cities/:id", h.Admin.UpdateCity)
	g.DELETE("/cities/:id", h.Admin.DeleteCity)

	g.POST("/localities", h.Admin.CreateLocality)
	g.PUT("/localities/:id", h.Admin.UpdateLocality)
	g.DELETE("/localities/:id", h.Admin.DeleteLocality)

	g.POST("/amenities", h.Admin.CreateAmenity)
	g.PUT("/amenities/:id", h.Admin.UpdateAmenity)
	g.DELETE("/amenities/:id", h.Admin.DeleteAmenity)

	g.GET("/banners", h.Admin.ListBanners)
	g.POST("/banners", h.Admin.CreateBanner)
	g.PUT("/banners/:id", h.Admin.UpdateBanner)
	g.DELETE("/banners/:id", h.Admin.DeleteBanner)

	g.GET("/featured", h.Admin.ListFeatured)
	g.POST("/featured", h.Admin.AddFeatured)
	g.DELETE("/featured/:id", h.Admin.RemoveFeatured)

	g.GET("/pages", h.Admin.ListPages)
	g.POST("/pages", h.Admin.SavePage)
	g.DELETE("/pages/:id", h.Admin.DeletePage)

	g.POST("/uploads", h.Admin.UploadAsset)
}
