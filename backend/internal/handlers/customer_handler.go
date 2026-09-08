package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/middleware"
	"github.com/rera/property-platform/backend/internal/services"
)

// CustomerHandler serves the buyer account area. Every endpoint is scoped to
// the signed-in user; no customer can read another customer's data.
type CustomerHandler struct {
	accounts *services.AccountService
	engage   *services.EngagementService
	recs     *services.RecommendationService
	visits   *services.VisitService
	alerts   *services.AlertService
}

func NewCustomerHandler(
	accounts *services.AccountService,
	engage *services.EngagementService,
	recs *services.RecommendationService,
	visits *services.VisitService,
	alerts *services.AlertService,
) *CustomerHandler {
	return &CustomerHandler{accounts: accounts, engage: engage, recs: recs,
		visits: visits, alerts: alerts}
}

// Dashboard summarises saved properties, inquiries and recommendations.
func (h *CustomerHandler) Dashboard(c echo.Context) error {
	data, err := h.accounts.CustomerDashboard(c.Request().Context(),
		middleware.UserID(c), middleware.SessionID(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Dashboard fetched successfully", data)
}

// UpdateProfile edits name, avatar and contact details. Changing an email or
// mobile clears its verified flag inside the service.
func (h *CustomerHandler) UpdateProfile(c echo.Context) error {
	var req dto.UpdateProfileRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := h.accounts.UpdateProfile(c.Request().Context(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Profile updated successfully", user)
}

// GetPreferences returns the buyer's saved search preferences.
func (h *CustomerHandler) GetPreferences(c echo.Context) error {
	prefs, err := h.accounts.GetPreferences(c.Request().Context(), middleware.UserID(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Preferences fetched successfully", prefs)
}

// SavePreferences stores the criteria the recommendation engine reads.
func (h *CustomerHandler) SavePreferences(c echo.Context) error {
	var req dto.CustomerPreferenceRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	prefs, err := h.accounts.SavePreferences(c.Request().Context(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Preferences saved successfully", prefs)
}

// UpdateConsent records a change to marketing consent, with a timestamp.
func (h *CustomerHandler) UpdateConsent(c echo.Context) error {
	var req dto.UpdateConsentRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	consent, err := h.accounts.UpdateConsent(c.Request().Context(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Communication preferences updated", consent)
}

// Unsubscribe withdraws consent for every non-essential communication.
func (h *CustomerHandler) Unsubscribe(c echo.Context) error {
	if err := h.accounts.Unsubscribe(c.Request().Context(), middleware.UserID(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "You have been unsubscribed from promotional communication", nil)
}

// EnableListingAccount lets a signed-in customer start listing property —
// renting out their own flat, most often — without registering a second time.
func (h *CustomerHandler) EnableListingAccount(c echo.Context) error {
	var req dto.EnableListingAccountRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	account, err := h.accounts.EnableListingAccount(c.Request().Context(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	// The caller's token still holds the old role set; refreshing it is what
	// admits them to the builder endpoints.
	return created(c, "Listing account created. Refresh your session to start posting.", account)
}

// ListFavorites returns the saved properties of the signed-in customer.
func (h *CustomerHandler) ListFavorites(c echo.Context) error {
	cards, meta, err := h.engage.ListFavorites(c.Request().Context(), middleware.UserID(c),
		intQuery(c, "page", 1), intQuery(c, "per_page", 20))
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Saved properties fetched successfully", cards, meta)
}

// AddFavorite saves a property.
func (h *CustomerHandler) AddFavorite(c echo.Context) error {
	id, err := uintParam(c, "property_id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.engage.AddFavorite(c.Request().Context(), middleware.UserID(c), id, visitor(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Property saved", nil)
}

// RemoveFavorite unsaves a property.
func (h *CustomerHandler) RemoveFavorite(c echo.Context) error {
	id, err := uintParam(c, "property_id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.engage.RemoveFavorite(c.Request().Context(), middleware.UserID(c), id, visitor(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Property removed from saved list", nil)
}

// RecentlyViewed lists the properties this customer opened most recently.
func (h *CustomerHandler) RecentlyViewed(c echo.Context) error {
	cards, err := h.engage.RecentlyViewed(c.Request().Context(), visitor(c), intQuery(c, "limit", 12))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Recently viewed properties fetched successfully", cards)
}

// ListInquiries returns the inquiries this customer submitted.
func (h *CustomerHandler) ListInquiries(c echo.Context) error {
	var q dto.InquiryListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	items, meta, err := h.engage.ListInquiriesForCustomer(c.Request().Context(), middleware.UserID(c), q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Inquiries fetched successfully", items, meta)
}

// Recommendations returns personalised suggestions for this customer.
func (h *CustomerHandler) Recommendations(c echo.Context) error {
	cards, err := h.recs.ForUser(c.Request().Context(), middleware.UserID(c), intQuery(c, "limit", 12))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Recommendations fetched successfully", cards)
}

// ListNotifications returns in-app notifications, newest first.
func (h *CustomerHandler) ListNotifications(c echo.Context) error {
	items, meta, err := h.accounts.ListNotifications(c.Request().Context(), middleware.UserID(c),
		boolQuery(c, "unread", false), intQuery(c, "page", 1), intQuery(c, "per_page", 20))
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Notifications fetched successfully", items, meta)
}

// MarkNotificationRead marks one notification as read.
func (h *CustomerHandler) MarkNotificationRead(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.accounts.MarkNotificationRead(c.Request().Context(), middleware.UserID(c), id); err != nil {
		return fail(c, err)
	}
	return ok(c, "Notification marked as read", nil)
}

// MarkAllNotificationsRead clears the unread badge.
func (h *CustomerHandler) MarkAllNotificationsRead(c echo.Context) error {
	if err := h.accounts.MarkAllNotificationsRead(c.Request().Context(), middleware.UserID(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "All notifications marked as read", nil)
}

// GetNotificationPreferences returns the per-channel delivery switches.
func (h *CustomerHandler) GetNotificationPreferences(c echo.Context) error {
	prefs, err := h.accounts.GetNotificationPreferences(c.Request().Context(), middleware.UserID(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Notification preferences fetched successfully", prefs)
}

// SaveNotificationPreferences updates the per-channel delivery switches.
func (h *CustomerHandler) SaveNotificationPreferences(c echo.Context) error {
	var req dto.NotificationPreferenceRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	prefs, err := h.accounts.SaveNotificationPreferences(c.Request().Context(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Notification preferences updated", prefs)
}

// ListVisits returns the viewing requests this customer has made.
func (h *CustomerHandler) ListVisits(c echo.Context) error {
	var q dto.VisitListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	items, meta, err := h.visits.ListForCustomer(c.Request().Context(), middleware.UserID(c), q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Visits fetched successfully", items, meta)
}

// ListRentalAlerts returns the searches this customer has saved.
func (h *CustomerHandler) ListRentalAlerts(c echo.Context) error {
	items, err := h.alerts.List(c.Request().Context(), middleware.UserID(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Saved searches fetched successfully", items)
}

// CreateRentalAlert saves the current rent search.
func (h *CustomerHandler) CreateRentalAlert(c echo.Context) error {
	var req dto.CreateRentalAlertRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	alert, err := h.alerts.Create(c.Request().Context(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Search saved. We will tell you when a matching flat is listed.", alert)
}

// UpdateRentalAlert pauses or resumes a saved search.
func (h *CustomerHandler) UpdateRentalAlert(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req struct {
		IsActive *bool `json:"is_active" validate:"required"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.alerts.SetActive(c.Request().Context(), id, middleware.UserID(c), *req.IsActive); err != nil {
		return fail(c, err)
	}
	if *req.IsActive {
		return ok(c, "Saved search resumed", nil)
	}
	return ok(c, "Saved search paused", nil)
}

// DeleteRentalAlert removes a saved search.
func (h *CustomerHandler) DeleteRentalAlert(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.alerts.Delete(c.Request().Context(), id, middleware.UserID(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Saved search deleted", nil)
}
