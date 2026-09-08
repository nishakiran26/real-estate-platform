package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/middleware"
	"github.com/rera/property-platform/backend/internal/services"
)

// AdminHandler serves the administrator console: the approval queue, user and
// builder management, reference data, CMS content and platform analytics.
type AdminHandler struct {
	admin     *services.AdminService
	analytics *services.AnalyticsService
	props     *services.PropertyService
	geo       *services.GeoService
	cms       *services.CMSService
	engage    *services.EngagementService
	media     *services.MediaService
	reports   *services.ReportService
	expiry    *services.ExpiryService
}

func NewAdminHandler(
	admin *services.AdminService,
	analytics *services.AnalyticsService,
	props *services.PropertyService,
	geo *services.GeoService,
	cms *services.CMSService,
	engage *services.EngagementService,
	media *services.MediaService,
	reports *services.ReportService,
	expiry *services.ExpiryService,
) *AdminHandler {
	return &AdminHandler{admin: admin, analytics: analytics, props: props,
		geo: geo, cms: cms, engage: engage, media: media, reports: reports,
		expiry: expiry}
}

// Dashboard returns platform-wide counts, charts and recent activity.
func (h *AdminHandler) Dashboard(c echo.Context) error {
	var q dto.AnalyticsQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid analytics parameters"))
	}
	data, err := h.analytics.AdminDashboard(c.Request().Context(), q)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Dashboard fetched successfully", data)
}

// Analytics serves the same payload as the dashboard for the reports screen,
// where the date range and scope filters are applied explicitly.
func (h *AdminHandler) Analytics(c echo.Context) error {
	return h.Dashboard(c)
}

// ListProperties returns listings in every workflow state, for the review queue.
func (h *AdminHandler) ListProperties(c echo.Context) error {
	var q dto.AdminPropertyListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	cards, meta, err := h.admin.ListProperties(c.Request().Context(), q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Properties fetched successfully", cards, meta)
}

// GetProperty returns one listing in any state.
func (h *AdminHandler) GetProperty(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	property, err := h.admin.GetProperty(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Property fetched successfully", property)
}

// StartReview moves a submitted listing into UNDER_REVIEW.
func (h *AdminHandler) StartReview(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.admin.StartReview(c.Request().Context(), id, middleware.UserID(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Property moved to review", nil)
}

// ApproveProperty approves and optionally publishes a listing.
func (h *AdminHandler) ApproveProperty(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.ApprovePropertyRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.admin.Approve(c.Request().Context(), id, middleware.UserID(c), req, requestMeta(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Property approved successfully", nil)
}

// RejectProperty rejects a listing with a reason the builder can act on.
func (h *AdminHandler) RejectProperty(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.RejectPropertyRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.admin.Reject(c.Request().Context(), id, middleware.UserID(c), req, requestMeta(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Property rejected", nil)
}

// VerifyRERA records the administrator decision on a registration number.
func (h *AdminHandler) VerifyRERA(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.VerifyRERARequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.admin.VerifyRERA(c.Request().Context(), id, middleware.UserID(c), req, requestMeta(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "RERA verification status updated", nil)
}

// SetFeatured toggles the featured flag on a listing.
func (h *AdminHandler) SetFeatured(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	featured := boolQuery(c, "featured", true)
	if err := h.admin.SetFeatured(c.Request().Context(), id, middleware.UserID(c), featured, requestMeta(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Featured status updated", nil)
}

// PropertyStatusHistory returns the approval trail of a listing.
func (h *AdminHandler) PropertyStatusHistory(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	history, err := h.props.StatusHistory(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Status history fetched successfully", history)
}

// ListUsers returns platform accounts, filterable by role and status.
func (h *AdminHandler) ListUsers(c echo.Context) error {
	var q dto.UserListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	users, meta, err := h.admin.ListUsers(c.Request().Context(), q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Users fetched successfully", users, meta)
}

// ListCustomers is ListUsers pinned to the customer role.
func (h *AdminHandler) ListCustomers(c echo.Context) error {
	var q dto.UserListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	q.Role = middleware.RoleCustomer
	users, meta, err := h.admin.ListUsers(c.Request().Context(), q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Customers fetched successfully", users, meta)
}

// ListClients returns builder accounts with their portfolio counts.
func (h *AdminHandler) ListClients(c echo.Context) error {
	clients, meta, err := h.admin.ListClients(c.Request().Context(),
		c.QueryParam("q"), intQuery(c, "page", 1), intQuery(c, "per_page", 20))
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Clients fetched successfully", clients, meta)
}

// UpdateUserStatus suspends or reactivates an account.
func (h *AdminHandler) UpdateUserStatus(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdateUserStatusRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.admin.UpdateUserStatus(c.Request().Context(), id, middleware.UserID(c), req, requestMeta(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Account status updated", nil)
}

// AssignRoles replaces the role set of an account.
func (h *AdminHandler) AssignRoles(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.AssignRoleRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.admin.AssignRoles(c.Request().Context(), id, middleware.UserID(c), req, requestMeta(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Roles updated", nil)
}

// ListRoles returns the assignable roles.
func (h *AdminHandler) ListRoles(c echo.Context) error {
	roles, err := h.admin.ListRoles(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Roles fetched successfully", roles)
}

// ListAuditLogs returns the administrative action trail.
func (h *AdminHandler) ListAuditLogs(c echo.Context) error {
	logs, meta, err := h.admin.ListAuditLogs(c.Request().Context(),
		intQuery(c, "page", 1), intQuery(c, "per_page", 50))
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Audit logs fetched successfully", logs, meta)
}

// ListInquiries returns every lead on the platform.
func (h *AdminHandler) ListInquiries(c echo.Context) error {
	var q dto.InquiryListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	items, meta, err := h.engage.ListInquiriesForAdmin(c.Request().Context(), q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Inquiries fetched successfully", items, meta)
}

// --- Reference data -------------------------------------------------------

func (h *AdminHandler) CreateCity(c echo.Context) error {
	var req dto.CreateCityRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	city, err := h.geo.CreateCity(c.Request().Context(), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "City created successfully", city)
}

func (h *AdminHandler) UpdateCity(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreateCityRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	city, err := h.geo.UpdateCity(c.Request().Context(), id, req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "City updated successfully", city)
}

func (h *AdminHandler) DeleteCity(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.geo.DeleteCity(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return ok(c, "City deleted successfully", nil)
}

func (h *AdminHandler) CreateLocality(c echo.Context) error {
	var req dto.CreateLocalityRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	locality, err := h.geo.CreateLocality(c.Request().Context(), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Locality created successfully", locality)
}

func (h *AdminHandler) UpdateLocality(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreateLocalityRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	locality, err := h.geo.UpdateLocality(c.Request().Context(), id, req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Locality updated successfully", locality)
}

func (h *AdminHandler) DeleteLocality(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.geo.DeleteLocality(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return ok(c, "Locality deleted successfully", nil)
}

func (h *AdminHandler) CreateAmenity(c echo.Context) error {
	var req dto.CreateAmenityRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	amenity, err := h.geo.CreateAmenity(c.Request().Context(), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Amenity created successfully", amenity)
}

func (h *AdminHandler) UpdateAmenity(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreateAmenityRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	amenity, err := h.geo.UpdateAmenity(c.Request().Context(), id, req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Amenity updated successfully", amenity)
}

func (h *AdminHandler) DeleteAmenity(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.geo.DeleteAmenity(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return ok(c, "Amenity deleted successfully", nil)
}

// --- Content management ---------------------------------------------------

func (h *AdminHandler) ListBanners(c echo.Context) error {
	banners, err := h.cms.ListBanners(c.Request().Context(), boolQuery(c, "active", false))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Banners fetched successfully", banners)
}

func (h *AdminHandler) CreateBanner(c echo.Context) error {
	var req dto.CreateBannerRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	banner, err := h.cms.CreateBanner(c.Request().Context(), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Banner created successfully", banner)
}

func (h *AdminHandler) UpdateBanner(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreateBannerRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	banner, err := h.cms.UpdateBanner(c.Request().Context(), id, req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Banner updated successfully", banner)
}

func (h *AdminHandler) DeleteBanner(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.cms.DeleteBanner(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return ok(c, "Banner deleted successfully", nil)
}

func (h *AdminHandler) ListFeatured(c echo.Context) error {
	items, err := h.cms.ListFeatured(c.Request().Context(), boolQuery(c, "active", false), intQuery(c, "limit", 20))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Featured properties fetched successfully", items)
}

func (h *AdminHandler) AddFeatured(c echo.Context) error {
	var req dto.CreateFeaturedPropertyRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	item, err := h.cms.AddFeatured(c.Request().Context(), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Property featured successfully", item)
}

func (h *AdminHandler) RemoveFeatured(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.cms.RemoveFeatured(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return ok(c, "Featured property removed", nil)
}

func (h *AdminHandler) ListPages(c echo.Context) error {
	pages, err := h.cms.ListPages(c.Request().Context(), false)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Pages fetched successfully", pages)
}

func (h *AdminHandler) SavePage(c echo.Context) error {
	var req dto.StaticPageRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	page, err := h.cms.SavePage(c.Request().Context(), req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Page saved successfully", page)
}

func (h *AdminHandler) DeletePage(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.cms.DeletePage(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return ok(c, "Page deleted successfully", nil)
}

// UploadAsset stores banner artwork and other console images.
func (h *AdminHandler) UploadAsset(c echo.Context) error {
	up, file, err := openUpload(c, "file")
	if err != nil {
		return fail(c, err)
	}
	defer file.Close()

	folder := c.FormValue("folder")
	if folder == "" {
		folder = "assets/cms"
	}
	asset, err := h.media.UploadImageAsset(c.Request().Context(), folder, up)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "File uploaded successfully", asset)
}

// ListReports returns the moderation queue, oldest complaint first.
func (h *AdminHandler) ListReports(c echo.Context) error {
	var q dto.ReportListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	items, meta, err := h.reports.List(c.Request().Context(), q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Reports fetched successfully", items, meta)
}

// ResolveReport records a decision on a complaint, optionally taking the
// listing down in the same step.
func (h *AdminHandler) ResolveReport(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.ResolveReportRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	report, err := h.reports.Resolve(c.Request().Context(), id,
		middleware.UserID(c), req, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Report updated", report)
}

// SetClientVerification records how far a listing account has been checked,
// which is what the trust badge on its listings reflects.
func (h *AdminHandler) SetClientVerification(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.SetVerificationRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	client, err := h.admin.SetClientVerification(c.Request().Context(), id,
		middleware.UserID(c), req, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Verification level updated", client)
}

// CreatePropertyForClient posts a listing on an owner's behalf — details taken
// over the phone, most often. The listing belongs to the owner's account; the
// provenance records that an administrator entered it.
func (h *AdminHandler) CreatePropertyForClient(c echo.Context) error {
	clientID, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreatePropertyRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	property, err := h.props.CreateOnBehalf(c.Request().Context(), clientID,
		middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Property saved as draft for this account", property)
}

// SweepExpiredListings retires stale rentals and warns the owners of listings
// about to lapse. The sweep also runs on a timer; this endpoint exists so an
// administrator can run it now and see what it did, rather than waiting for the
// next tick and guessing.
func (h *AdminHandler) SweepExpiredListings(c echo.Context) error {
	result, err := h.expiry.Sweep(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Expiry sweep completed", result)
}
