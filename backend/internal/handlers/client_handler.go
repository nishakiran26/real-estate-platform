package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/middleware"
	"github.com/rera/property-platform/backend/internal/services"
)

// ClientHandler serves the builder portal. Every property, lead and analytics
// endpoint is scoped by the caller's client ID, so one builder can never reach
// another builder's data.
type ClientHandler struct {
	props     *services.PropertyService
	media     *services.MediaService
	engage    *services.EngagementService
	accounts  *services.AccountService
	analytics *services.AnalyticsService
	visits    *services.VisitService
}

func NewClientHandler(
	props *services.PropertyService,
	media *services.MediaService,
	engage *services.EngagementService,
	accounts *services.AccountService,
	analytics *services.AnalyticsService,
	visits *services.VisitService,
) *ClientHandler {
	return &ClientHandler{props: props, media: media, engage: engage,
		accounts: accounts, analytics: analytics, visits: visits}
}

// Dashboard returns portfolio-wide statistics and charts for this builder.
func (h *ClientHandler) Dashboard(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	var q dto.AnalyticsQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid analytics parameters"))
	}
	data, err := h.analytics.ClientDashboard(c.Request().Context(), clientID, q)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Dashboard fetched successfully", data)
}

// GetProfile returns the builder company record.
func (h *ClientHandler) GetProfile(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	profile, err := h.accounts.GetClientProfile(c.Request().Context(), clientID)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Company profile fetched successfully", profile)
}

// UpdateProfile edits the builder company record.
func (h *ClientHandler) UpdateProfile(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdateClientProfileRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	profile, err := h.accounts.UpdateClientProfile(c.Request().Context(), clientID, req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Company profile updated successfully", profile)
}

// ListProperties returns this builder's listings in every workflow state.
func (h *ClientHandler) ListProperties(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	var q dto.AdminPropertyListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	cards, meta, err := h.props.ListForClient(c.Request().Context(), clientID, q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Properties fetched successfully", cards, meta)
}

// GetProperty returns one of this builder's listings, in any state.
func (h *ClientHandler) GetProperty(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	property, err := h.props.GetForClient(c.Request().Context(), id, clientID)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Property fetched successfully", property)
}

// CreateProperty saves a new listing as a draft.
func (h *ClientHandler) CreateProperty(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreatePropertyRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	property, err := h.props.Create(c.Request().Context(), clientID, middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Property saved as draft", property)
}

// UpdateProperty edits a listing. An approved listing returns to review after
// an edit, which the service enforces.
func (h *ClientHandler) UpdateProperty(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdatePropertyRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	property, err := h.props.Update(c.Request().Context(), id, clientID, middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Property updated successfully", property)
}

// DeleteProperty soft-deletes a listing.
func (h *ClientHandler) DeleteProperty(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.props.Delete(c.Request().Context(), id, clientID); err != nil {
		return fail(c, err)
	}
	return ok(c, "Property deleted successfully", nil)
}

// SubmitProperty sends a draft or corrected listing for administrator review.
func (h *ClientHandler) SubmitProperty(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	property, err := h.props.Submit(c.Request().Context(), id, clientID, middleware.UserID(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Property submitted for review", property)
}

// StatusHistory returns the approval audit trail of a listing.
func (h *ClientHandler) StatusHistory(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if clientID != 0 {
		if err := h.props.OwnsProperty(c.Request().Context(), id, clientID); err != nil {
			return fail(c, err)
		}
	}
	history, err := h.props.StatusHistory(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Status history fetched successfully", history)
}

// PropertyAnalytics returns engagement metrics for one of this builder's listings.
func (h *ClientHandler) PropertyAnalytics(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	stats, err := h.analytics.PropertyAnalytics(c.Request().Context(), id, clientID)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Property analytics fetched successfully", stats)
}

// ListMedia returns the gallery of one listing.
func (h *ClientHandler) ListMedia(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	media, err := h.media.ListMedia(c.Request().Context(), id, clientID)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Media fetched successfully", media)
}

// UploadMedia adds an image to a listing gallery.
func (h *ClientHandler) UploadMedia(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	up, file, err := openUpload(c, "file")
	if err != nil {
		return fail(c, err)
	}
	defer file.Close()

	mediaType := c.FormValue("media_type")
	if mediaType == "" {
		mediaType = "PROPERTY_IMAGE"
	}
	media, err := h.media.UploadMedia(c.Request().Context(), id, clientID, mediaType, up)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Image uploaded successfully", media)
}

// DeleteMedia removes an image from a gallery.
func (h *ClientHandler) DeleteMedia(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	mediaID, err := uintParam(c, "media_id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.media.DeleteMedia(c.Request().Context(), id, mediaID, clientID); err != nil {
		return fail(c, err)
	}
	return ok(c, "Image deleted successfully", nil)
}

// SetCoverImage promotes one image to the listing cover.
func (h *ClientHandler) SetCoverImage(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.SetCoverRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.media.SetCover(c.Request().Context(), id, req.MediaID, clientID); err != nil {
		return fail(c, err)
	}
	return ok(c, "Cover image updated", nil)
}

// ReorderMedia applies a new gallery order.
func (h *ClientHandler) ReorderMedia(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.ReorderMediaRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.media.Reorder(c.Request().Context(), id, clientID, req.MediaIDs); err != nil {
		return fail(c, err)
	}
	return ok(c, "Gallery reordered", nil)
}

// ListDocuments returns brochures and certificates for a listing.
func (h *ClientHandler) ListDocuments(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	docs, err := h.media.ListDocuments(c.Request().Context(), id, clientID)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Documents fetched successfully", docs)
}

// UploadDocument attaches a PDF brochure or RERA certificate to a listing.
func (h *ClientHandler) UploadDocument(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	up, file, err := openUpload(c, "file")
	if err != nil {
		return fail(c, err)
	}
	defer file.Close()

	docType := c.FormValue("doc_type")
	if docType == "" {
		docType = "BROCHURE"
	}
	doc, err := h.media.UploadDocument(c.Request().Context(), id, clientID, docType, c.FormValue("name"), up)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Document uploaded successfully", doc)
}

// DeleteDocument removes a document from a listing.
func (h *ClientHandler) DeleteDocument(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	docID, err := uintParam(c, "doc_id")
	if err != nil {
		return fail(c, err)
	}
	if err := h.media.DeleteDocument(c.Request().Context(), id, docID, clientID); err != nil {
		return fail(c, err)
	}
	return ok(c, "Document deleted successfully", nil)
}

// UploadLogo stores a company logo and returns its public URL.
func (h *ClientHandler) UploadLogo(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	up, file, err := openUpload(c, "file")
	if err != nil {
		return fail(c, err)
	}
	defer file.Close()

	asset, err := h.media.UploadImageAsset(c.Request().Context(), folderForClient(clientID), up)
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Logo uploaded successfully", asset)
}

// ListInquiries returns the leads raised on this builder's listings only.
func (h *ClientHandler) ListInquiries(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	var q dto.InquiryListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	items, meta, err := h.engage.ListInquiriesForClient(c.Request().Context(), clientID, q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Inquiries fetched successfully", items, meta)
}

// GetInquiry returns one lead belonging to this builder.
func (h *ClientHandler) GetInquiry(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	inquiry, err := h.engage.GetInquiryForClient(c.Request().Context(), id, clientID)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Inquiry fetched successfully", inquiry)
}

// UpdateInquiryStatus moves a lead along the follow-up pipeline.
func (h *ClientHandler) UpdateInquiryStatus(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdateInquiryStatusRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.engage.UpdateInquiryStatus(c.Request().Context(), id, clientID, middleware.UserID(c), req); err != nil {
		return fail(c, err)
	}
	return ok(c, "Inquiry updated successfully", nil)
}

// folderForClient keeps each builder's assets in its own storage prefix.
func folderForClient(clientID uint) string {
	if clientID == 0 {
		return "assets/admin"
	}
	return "clients/" + uintToString(clientID)
}

// ListVisits returns viewing requests on this account's own properties.
func (h *ClientHandler) ListVisits(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	var q dto.VisitListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid filter parameters"))
	}
	items, meta, err := h.visits.ListForClient(c.Request().Context(), clientID, q)
	if err != nil {
		return fail(c, err)
	}
	return paginated(c, "Visits fetched successfully", items, meta)
}

// RespondToVisit confirms, reschedules, completes or cancels a request.
func (h *ClientHandler) RespondToVisit(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdateVisitRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	visit, err := h.visits.Respond(c.Request().Context(), id, clientID, middleware.UserID(c), req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Visit updated", visit)
}

// SetAvailability marks a rental as available, reserved, rented or inactive.
// It is separate from the approval workflow so that letting a flat does not
// send the listing back for review.
func (h *ClientHandler) SetAvailability(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdateAvailabilityRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	property, err := h.props.SetAvailability(c.Request().Context(), id, clientID, req)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Availability updated", property)
}

// RenewListing extends a rental's expiry by another listing period.
func (h *ClientHandler) RenewListing(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	property, err := h.props.RenewListing(c.Request().Context(), id, clientID)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Listing renewed", property)
}

// ExpiringListings powers the "these lapse soon" panel on the owner dashboard.
func (h *ClientHandler) ExpiringListings(c echo.Context) error {
	clientID, err := callerClientID(c)
	if err != nil {
		return fail(c, err)
	}
	items, err := h.props.ExpiringForClient(c.Request().Context(), clientID, intQuery(c, "within_days", 14))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Expiring listings fetched successfully", items)
}
