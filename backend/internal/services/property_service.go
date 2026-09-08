package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/pkg/logger"
	"github.com/rera/property-platform/backend/pkg/utils"
)

// searchCacheTTL is deliberately short: listings change often and a stale search
// page is more damaging than an extra query.
const searchCacheTTL = 2 * time.Minute

// PropertyService covers the public catalogue and the builder-facing lifecycle.
type PropertyService struct {
	props    repositories.PropertyRepository
	geo      repositories.GeoRepository
	accounts repositories.AccountRepository
	engage   repositories.EngagementRepository
	cache    *cache.Client
	cfg      *config.Config
}

func NewPropertyService(
	props repositories.PropertyRepository,
	geo repositories.GeoRepository,
	accounts repositories.AccountRepository,
	engage repositories.EngagementRepository,
	c *cache.Client,
	cfg *config.Config,
) *PropertyService {
	return &PropertyService{props: props, geo: geo, accounts: accounts, engage: engage, cache: c, cfg: cfg}
}

// ListPublic runs the public catalogue search. Results are cached per query.
func (s *PropertyService) ListPublic(ctx context.Context, q dto.PropertyListQuery, viewerID *uint) ([]dto.PropertyCard, *dto.Meta, error) {
	// The public catalogue defaults to sale listings: /properties is the Buy
	// page, and the Rent page passes purpose=rent explicitly. Callers wanting
	// both send purpose=all.
	if strings.TrimSpace(q.Purpose) == "" {
		q.Purpose = string(models.PurposeBuy)
	}

	filter, page, perPage := s.buildFilter(q, nil)

	// Rented and withdrawn listings never appear in search, whatever their
	// approval status. Existing sale listings are all AVAILABLE, so this changes
	// nothing for the Buy journey.
	filter.AvailabilityStatuses = []models.AvailabilityStatus{
		models.AvailabilityAvailable,
		models.AvailabilityReserved,
	}

	type cached struct {
		Cards []dto.PropertyCard `json:"cards"`
		Meta  dto.Meta           `json:"meta"`
	}
	key := searchCacheKey(q)
	var hit cached
	if err := s.cache.GetJSON(ctx, key, &hit); err == nil {
		s.markFavorites(ctx, hit.Cards, viewerID)
		return hit.Cards, &hit.Meta, nil
	}

	props, total, err := s.props.List(ctx, filter)
	if err != nil {
		return nil, nil, Internal("search properties", err)
	}
	cards := ToPropertyCards(props)
	meta := newMeta(page, perPage, total)

	s.cache.SetJSON(ctx, key, cached{Cards: cards, Meta: *meta}, searchCacheTTL)
	s.markFavorites(ctx, cards, viewerID)
	return cards, meta, nil
}

// markFavorites flags which cards the viewer has already saved. It is applied
// after caching so the cached payload stays viewer-independent.
func (s *PropertyService) markFavorites(ctx context.Context, cards []dto.PropertyCard, viewerID *uint) {
	if viewerID == nil || len(cards) == 0 {
		return
	}
	ids, err := s.engage.FavoriteIDs(ctx, *viewerID)
	if err != nil {
		logger.Log.Warn("could not load favourites", "user_id", *viewerID, "error", err)
		return
	}
	set := make(map[uint]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	for i := range cards {
		cards[i].IsFavorite = set[cards[i].ID]
	}
}

// GetPublicBySlug returns a published property for the detail page.
func (s *PropertyService) GetPublicBySlug(ctx context.Context, slug string, viewerID *uint) (*dto.PropertyResponse, error) {
	p, err := s.props.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	if !p.IsPublic() {
		// Unpublished listings must not be discoverable by guessing slugs.
		return nil, NotFound("Property not found")
	}
	resp := ToPropertyResponse(p)
	if viewerID != nil {
		if fav, err := s.engage.IsFavorite(ctx, *viewerID, p.ID); err == nil {
			resp.IsFavorite = fav
		}
	}
	return &resp, nil
}

// GetPublicByID backs endpoints that address a property numerically.
func (s *PropertyService) GetPublicByID(ctx context.Context, id uint, viewerID *uint) (*dto.PropertyResponse, error) {
	p, err := s.props.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	if !p.IsPublic() {
		return nil, NotFound("Property not found")
	}
	resp := ToPropertyResponse(p)
	if viewerID != nil {
		if fav, err := s.engage.IsFavorite(ctx, *viewerID, p.ID); err == nil {
			resp.IsFavorite = fav
		}
	}
	return &resp, nil
}

// Similar returns comparable listings for the detail page.
func (s *PropertyService) Similar(ctx context.Context, slug string, limit int) ([]dto.PropertyCard, error) {
	p, err := s.props.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	props, err := s.props.Similar(ctx, p, limit)
	if err != nil {
		return nil, Internal("load similar properties", err)
	}
	return ToPropertyCards(props), nil
}

// Compare returns the full detail of several properties side by side.
func (s *PropertyService) Compare(ctx context.Context, ids []uint) ([]dto.PropertyResponse, error) {
	if len(ids) < 2 {
		return nil, Invalid("Select at least two properties to compare")
	}
	if len(ids) > 4 {
		return nil, Invalid("You can compare up to four properties at a time")
	}
	props, err := s.props.ListByIDs(ctx, ids)
	if err != nil {
		return nil, Internal("load properties", err)
	}

	// A comparison table puts the figures side by side, and a monthly rent
	// beside a sale price is not a comparison — it reads as one flat being
	// hundreds of times cheaper than the other. Refuse rather than render it.
	seen := map[models.PropertyPurpose]bool{}
	for i := range props {
		if props[i].IsPublic() {
			seen[props[i].Purpose] = true
		}
	}
	if len(seen) > 1 {
		return nil, Invalid("Compare homes to rent with homes to rent, and property for sale with property for sale — the prices are not comparable")
	}

	out := make([]dto.PropertyResponse, 0, len(props))
	for i := range props {
		if !props[i].IsPublic() {
			continue
		}
		out = append(out, ToPropertyResponse(&props[i]))
	}
	if len(out) == 0 {
		return nil, NotFound("None of the selected properties are available")
	}
	return out, nil
}

// Featured returns the homepage carousel.
// The homepage strips below are sale strips. A monthly rent and a sale price
// are not comparable figures, and putting them in one row invites exactly the
// wrong reading: a flat listed at ₹17,000 a month sitting beside flats for sale
// looks like a flat that costs ₹17,000. Rentals get their own section instead.
func (s *PropertyService) Featured(ctx context.Context, limit int) ([]dto.PropertyCard, error) {
	yes := true
	props, _, err := s.props.List(ctx, repositories.PropertyFilter{
		IsFeatured: &yes, Purpose: string(models.PurposeBuy), Limit: limit, SortBy: "newest",
	})
	if err != nil {
		return nil, Internal("load featured properties", err)
	}
	return ToPropertyCards(props), nil
}

// NewLaunches, ReadyToMove and Upcoming back the homepage sections. All three
// are sale framings — a tenancy is not "under construction" or "newly
// launched" — so all three are restricted to the sale catalogue.
func (s *PropertyService) ByConstructionStatus(ctx context.Context, status string, limit int) ([]dto.PropertyCard, error) {
	props, _, err := s.props.List(ctx, repositories.PropertyFilter{
		ConstructionStatus: status, Purpose: string(models.PurposeBuy),
		Limit: limit, SortBy: "newest",
	})
	if err != nil {
		return nil, Internal("load properties", err)
	}
	return ToPropertyCards(props), nil
}

func (s *PropertyService) NewLaunches(ctx context.Context, limit int) ([]dto.PropertyCard, error) {
	props, _, err := s.props.List(ctx, repositories.PropertyFilter{
		Purpose: string(models.PurposeBuy), Limit: limit, SortBy: "newest",
	})
	if err != nil {
		return nil, Internal("load new launches", err)
	}
	return ToPropertyCards(props), nil
}

// LatestRentals backs the homepage's rental strip, kept apart from the sale
// strips so the two kinds of price are never read against each other. Rented
// and reserved stock is excluded: a homepage that advertises a let flat wastes
// the visitor's time.
func (s *PropertyService) LatestRentals(ctx context.Context, limit int) ([]dto.PropertyCard, error) {
	props, _, err := s.props.List(ctx, repositories.PropertyFilter{
		Purpose: string(models.PurposeRent),
		AvailabilityStatuses: []models.AvailabilityStatus{
			models.AvailabilityAvailable,
		},
		Limit: limit, SortBy: "newest",
	})
	if err != nil {
		return nil, Internal("load rentals", err)
	}
	return ToPropertyCards(props), nil
}

// ---------------------------------------------------------------------------
// Builder-facing lifecycle
// ---------------------------------------------------------------------------

// ListForClient returns a builder's own listings in any workflow state.
func (s *PropertyService) ListForClient(ctx context.Context, clientID uint, q dto.AdminPropertyListQuery) ([]dto.PropertyCard, *dto.Meta, error) {
	page, perPage := normalizePage(q.Page, q.PerPage)
	filter := repositories.PropertyFilter{
		ClientID: clientID,
		Search:   q.Search,
		SortBy:   q.SortBy,
		Statuses: allStatuses(),
		// Empty means both, so an account with only one kind sees no change.
		// An owner with flats to sell and flats to let needs them apart.
		Purpose: strings.ToUpper(q.Purpose),
		Offset:  (page - 1) * perPage,
		Limit:   perPage,
	}
	if q.Status != "" {
		filter.Statuses = []models.PropertyStatus{models.PropertyStatus(strings.ToUpper(q.Status))}
	}
	props, total, err := s.props.List(ctx, filter)
	if err != nil {
		return nil, nil, Internal("load properties", err)
	}
	return ToPropertyCards(props), newMeta(page, perPage, total), nil
}

// GetForClient loads one of the builder's own listings, in any state.
func (s *PropertyService) GetForClient(ctx context.Context, id, clientID uint) (*dto.PropertyResponse, error) {
	p, err := s.props.GetForOwner(ctx, id, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			// Do not distinguish "not yours" from "does not exist".
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	resp := ToPropertyResponse(p)
	return &resp, nil
}

// Create adds a listing in DRAFT state for the given builder.
func (s *PropertyService) Create(ctx context.Context, clientID, userID uint, req dto.CreatePropertyRequest) (*dto.PropertyResponse, error) {
	city, locality, err := s.resolveLocation(ctx, req.CityID, req.LocalityID)
	if err != nil {
		return nil, err
	}
	if len(req.Configurations) == 0 {
		return nil, Invalid("Add at least one unit configuration")
	}

	purpose := models.PurposeBuy
	if strings.EqualFold(req.Purpose, string(models.PurposeRent)) {
		purpose = models.PurposeRent
	}
	if err := validatePurposePayload(purpose, req.Configurations, req.Rental); err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(ctx, req.Name, locality.Name, city.Name, 0)
	if err != nil {
		return nil, err
	}

	builderName := strings.TrimSpace(req.BuilderName)
	if builderName == "" {
		if client, err := s.accounts.GetClient(ctx, clientID); err == nil {
			builderName = client.CompanyName
		}
	}

	p := &models.Property{
		ClientID: clientID, UserID: userID,
		Name: strings.TrimSpace(req.Name), Slug: slug,
		ProjectName: req.ProjectName, BuilderName: builderName,
		PropertyType:       models.PropertyType(req.PropertyType),
		ConstructionStatus: models.ConstructionStatus(req.ConstructionStatus),
		PropertyStatus:     models.PropertyStatusDraft,
		CityID:             city.ID, LocalityID: locality.ID,
		Address: req.Address, Pincode: req.Pincode,
		Latitude: req.Latitude, Longitude: req.Longitude,
		Description:        req.Description,
		Purpose:            purpose,
		AvailabilityStatus: models.AvailabilityAvailable,
		PostedByType:       s.postedByType(ctx, clientID),
	}
	// Rentals go stale, so they carry an expiry from the day they are created.
	// Sale listings leave it nil and never expire.
	if purpose == models.PurposeRent {
		expiry := time.Now().AddDate(0, 0, s.listingDays())
		p.ListingExpiryDate = &expiry
	}
	if d, err := parseDate(req.PossessionDate); err != nil {
		return nil, err
	} else if d != nil {
		p.PossessionDate = d
	}
	p.MetaTitle, p.MetaDescription = s.seoDefaults(req.MetaTitle, req.MetaDescription, p, city.Name, locality.Name, req.Configurations)

	configs := toConfigModels(req.Configurations)
	rera, err := toRERAModel(req.RERA)
	if err != nil {
		return nil, err
	}
	rental, err := toRentalModel(req.Rental)
	if err != nil {
		return nil, err
	}

	if err := s.props.Create(ctx, repositories.PropertyWrite{
		Property: p, Configurations: configs, AmenityIDs: req.AmenityIDs,
		RERA: rera, Rental: rental,
	}); err != nil {
		return nil, Internal("create property", err)
	}
	if err := s.accounts.RefreshClientPropertyCount(ctx, clientID); err != nil {
		logger.Log.Warn("could not refresh client property count", "client_id", clientID, "error", err)
	}
	return s.GetForClient(ctx, p.ID, clientID)
}

// Update edits a listing the builder owns. Editing a published listing sends it
// back for review so changes cannot bypass approval.
func (s *PropertyService) Update(ctx context.Context, id, clientID, userID uint, req dto.UpdatePropertyRequest) (*dto.PropertyResponse, error) {
	p, err := s.props.GetForOwner(ctx, id, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != p.Name {
		p.Name = strings.TrimSpace(*req.Name)
	}
	if req.ProjectName != nil {
		p.ProjectName = *req.ProjectName
	}
	if req.BuilderName != nil {
		p.BuilderName = *req.BuilderName
	}
	if req.PropertyType != nil {
		p.PropertyType = models.PropertyType(*req.PropertyType)
	}
	if req.ConstructionStatus != nil {
		p.ConstructionStatus = models.ConstructionStatus(*req.ConstructionStatus)
	}
	if req.CityID != nil || req.LocalityID != nil {
		cityID, localityID := p.CityID, p.LocalityID
		if req.CityID != nil {
			cityID = *req.CityID
		}
		if req.LocalityID != nil {
			localityID = *req.LocalityID
		}
		city, locality, err := s.resolveLocation(ctx, cityID, localityID)
		if err != nil {
			return nil, err
		}
		p.CityID, p.LocalityID = city.ID, locality.ID
		p.City, p.Locality = *city, *locality
	}
	if req.Address != nil {
		p.Address = *req.Address
	}
	if req.Pincode != nil {
		p.Pincode = *req.Pincode
	}
	if req.Latitude != nil {
		p.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		p.Longitude = *req.Longitude
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.PossessionDate != nil {
		d, err := parseDate(*req.PossessionDate)
		if err != nil {
			return nil, err
		}
		p.PossessionDate = d
	}
	if req.MetaTitle != nil {
		p.MetaTitle = *req.MetaTitle
	}
	if req.MetaDescription != nil {
		p.MetaDescription = *req.MetaDescription
	}

	// A published listing that changes returns to the review queue.
	if p.PropertyStatus == models.PropertyStatusApproved || p.PropertyStatus == models.PropertyStatusPublished {
		p.PropertyStatus = models.PropertyStatusSubmitted
	}

	var configs []models.PropertyConfiguration
	if req.Configurations != nil {
		if len(req.Configurations) == 0 {
			return nil, Invalid("A property must keep at least one unit configuration")
		}
		if err := validatePurposePayload(p.Purpose, req.Configurations, req.Rental); err != nil {
			return nil, err
		}
		configs = toConfigModels(req.Configurations)
	}
	rera, err := toRERAModel(req.RERA)
	if err != nil {
		return nil, err
	}
	rental, err := toRentalModel(req.Rental)
	if err != nil {
		return nil, err
	}
	// Rental terms only mean something on a rental listing; silently writing
	// them onto a sale listing would put it in an inconsistent state.
	if rental != nil && p.Purpose != models.PurposeRent {
		return nil, Invalid("Rental terms can only be set on a listing offered for rent")
	}

	if err := s.props.Update(ctx, repositories.PropertyWrite{
		Property: p, Configurations: configs, AmenityIDs: req.AmenityIDs,
		RERA: rera, Rental: rental,
	}); err != nil {
		return nil, Internal("update property", err)
	}
	s.invalidateSearchCache(ctx)
	return s.GetForClient(ctx, p.ID, clientID)
}

// Delete removes a listing the builder owns.
func (s *PropertyService) Delete(ctx context.Context, id, clientID uint) error {
	if _, err := s.props.GetForOwner(ctx, id, clientID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Property not found")
		}
		return Internal("load property", err)
	}
	if err := s.props.Delete(ctx, id); err != nil {
		return Internal("delete property", err)
	}
	if err := s.accounts.RefreshClientPropertyCount(ctx, clientID); err != nil {
		logger.Log.Warn("could not refresh client property count", "client_id", clientID, "error", err)
	}
	s.invalidateSearchCache(ctx)
	return nil
}

// Submit moves a draft or rejected listing into the review queue after checking
// that it carries enough information to be reviewed.
func (s *PropertyService) Submit(ctx context.Context, id, clientID, userID uint) (*dto.PropertyResponse, error) {
	p, err := s.props.GetForOwner(ctx, id, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	switch p.PropertyStatus {
	case models.PropertyStatusDraft, models.PropertyStatusRejected:
		// resubmission is allowed
	case models.PropertyStatusSubmitted, models.PropertyStatusUnderReview:
		return nil, Conflict("This property is already awaiting review")
	default:
		return nil, Conflict("This property has already been approved")
	}
	if err := validateForSubmission(p); err != nil {
		return nil, err
	}
	if err := s.props.ChangeStatus(ctx, id, models.PropertyStatusSubmitted, userID, "Submitted for review"); err != nil {
		return nil, Internal("submit property", err)
	}
	return s.GetForClient(ctx, id, clientID)
}

// validateForSubmission enforces the completeness rules a reviewer relies on.
func validateForSubmission(p *models.Property) error {
	var missing []string
	if len(p.Configurations) == 0 {
		missing = append(missing, "at least one unit configuration")
	}
	if strings.TrimSpace(p.Description) == "" {
		missing = append(missing, "a project description")
	}
	if strings.TrimSpace(p.Address) == "" {
		missing = append(missing, "the site address")
	}
	if !hasListingImage(p.Media) {
		missing = append(missing, "at least one property image")
	}

	// RERA registration applies to projects offered for sale. An individual
	// letting a flat has no project registration, so demanding one would make
	// rental listings impossible to submit.
	if p.Purpose == models.PurposeRent {
		if p.RentalDetails == nil || p.RentalDetails.MonthlyRent <= 0 {
			missing = append(missing, "the monthly rent")
		}
	} else if p.RERADetails.ID == 0 || strings.TrimSpace(p.RERADetails.RegistrationNumber) == "" {
		missing = append(missing, "the RERA registration number")
	}
	if len(missing) > 0 {
		return Invalid("Before submitting, please add " + joinWithAnd(missing) + ".")
	}
	return nil
}

// StatusHistory returns the approval trail for a listing the caller may see.
func (s *PropertyService) StatusHistory(ctx context.Context, propertyID uint) ([]dto.StatusHistoryResponse, error) {
	rows, err := s.props.StatusHistory(ctx, propertyID)
	if err != nil {
		return nil, Internal("load status history", err)
	}
	out := make([]dto.StatusHistoryResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.StatusHistoryResponse{
			ID: r.ID, FromStatus: string(r.FromStatus), ToStatus: string(r.ToStatus),
			ActorID: r.ActorID, Remarks: r.Remarks, CreatedAt: formatTime(r.CreatedAt),
		})
	}
	return out, nil
}

// OwnsProperty reports whether a builder owns a listing; media and lead
// endpoints use it before acting.
func (s *PropertyService) OwnsProperty(ctx context.Context, propertyID, clientID uint) error {
	if _, err := s.props.GetForOwner(ctx, propertyID, clientID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Property not found")
		}
		return Internal("load property", err)
	}
	return nil
}

// InvalidateSearchCache drops cached search pages after a catalogue change.
func (s *PropertyService) invalidateSearchCache(ctx context.Context) {
	if err := s.cache.DelByPrefix(ctx, "search:"); err != nil {
		logger.Log.Warn("could not clear search cache", "error", err)
	}
	// The homepage strips are built from the same listings, so every event that
	// invalidates a search invalidates them too: publishing, approving, letting
	// a flat. Without this a let flat keeps being advertised on the front page
	// until the entry expires on its own.
	if err := s.cache.Del(ctx, "homepage"); err != nil {
		logger.Log.Warn("could not clear homepage cache", "error", err)
	}
}

// InvalidateSearchCache is the exported form used by the admin service.
func (s *PropertyService) InvalidateSearchCache(ctx context.Context) { s.invalidateSearchCache(ctx) }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildFilter translates a URL query into a repository filter.
func (s *PropertyService) buildFilter(q dto.PropertyListQuery, statuses []models.PropertyStatus) (repositories.PropertyFilter, int, int) {
	page, perPage := normalizePage(q.Page, q.PerPage)
	f := repositories.PropertyFilter{
		CitySlug: q.CitySlug, LocalitySlug: q.LocalitySlug,
		CityID: q.CityID, LocalityID: q.LocalityID,
		PropertyType:       q.PropertyType,
		ConstructionStatus: q.ConstructionStatus,
		MinPrice:           q.MinPrice, MaxPrice: q.MaxPrice,
		MinArea: q.MinArea, MaxArea: q.MaxArea,
		BHK:         q.BHK,
		BuilderName: q.Builder,
		Search:      strings.TrimSpace(q.Search),
		SortBy:      q.SortBy, SortOrder: q.SortOrder,
		Statuses: statuses,
		Offset:   (page - 1) * perPage,
		Limit:    perPage,
	}
	if ids := parseUintList(q.Amenities); len(ids) > 0 {
		f.AmenityIDs = ids
	}
	if b := parseBool(q.Featured); b != nil {
		f.IsFeatured = b
	}
	if b := parseBool(q.RERAVerified); b != nil {
		f.IsRERAVerified = b
	}
	if q.PossessionBefore != "" {
		if d, err := time.Parse("2006-01-02", q.PossessionBefore); err == nil {
			f.PossessionBefore = &d
		}
	}

	// Purpose. "all" searches both catalogues; anything else is taken literally.
	// An empty value is left alone here — ListPublic decides the default, so the
	// client portal and admin queue keep seeing every listing they own.
	if purpose := strings.ToUpper(strings.TrimSpace(q.Purpose)); purpose != "" && purpose != "ALL" {
		f.Purpose = purpose
	}

	// Rental criteria are only meaningful for a rental search; applying them to
	// a sale search would silently exclude every sale listing, since none has a
	// rental_details row to join.
	if f.Purpose == string(models.PurposeRent) {
		f.MinRent, f.MaxRent = q.MinRent, q.MaxRent
		f.Furnishing = strings.ToUpper(strings.TrimSpace(q.Furnishing))
		f.TenantPreference = strings.ToUpper(strings.TrimSpace(q.TenantPreference))
		f.PetAllowed = parseBool(q.PetAllowed)
		f.AvailableBefore = parseAvailability(q.AvailableFrom)
	}

	return f, page, perPage
}

// parseAvailability accepts either a date or the shorthand the rent page offers,
// and returns the latest move-in date a listing may have to still qualify.
func parseAvailability(value string) *time.Time {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return nil
	}
	now := time.Now()
	switch value {
	case "now", "immediate":
		return &now
	case "7d":
		d := now.AddDate(0, 0, 7)
		return &d
	case "30d":
		d := now.AddDate(0, 0, 30)
		return &d
	}
	if d, err := time.Parse("2006-01-02", value); err == nil {
		return &d
	}
	return nil
}

// resolveLocation validates that the locality belongs to the given city.
func (s *PropertyService) resolveLocation(ctx context.Context, cityID, localityID uint) (*models.City, *models.Locality, error) {
	city, err := s.geo.GetCity(ctx, cityID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, nil, Invalid("The selected city does not exist")
		}
		return nil, nil, Internal("load city", err)
	}
	locality, err := s.geo.GetLocality(ctx, localityID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, nil, Invalid("The selected locality does not exist")
		}
		return nil, nil, Internal("load locality", err)
	}
	if locality.CityID != city.ID {
		return nil, nil, Invalid("The selected locality is not in the selected city")
	}
	return city, locality, nil
}

// uniqueSlug builds an SEO-friendly slug of the form
// "green-valley-residency-baner-pune", adding a numeric suffix on collision.
func (s *PropertyService) uniqueSlug(ctx context.Context, name, locality, city string, excludeID uint) (string, error) {
	base := utils.Slugify(strings.Join([]string{name, locality, city}, " "))
	if base == "" {
		base = "property"
	}
	slug := base
	for i := 2; i < 100; i++ {
		exists, err := s.props.SlugExists(ctx, slug, excludeID)
		if err != nil {
			return "", Internal("check slug", err)
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return "", Internal("generate slug", errors.New("exhausted slug attempts"))
}

// seoDefaults derives the meta title and description when the builder has not
// supplied them, e.g. "2, 3 BHK Flats in Baner, Pune | Green Valley Residency".
func (s *PropertyService) seoDefaults(title, desc string, p *models.Property, city, locality string, configs []dto.ConfigurationRequest) (string, string) {
	if title == "" {
		bhk := bhkLabel(configs)
		unit := propertyTypeLabel(p.PropertyType)
		if bhk != "" {
			title = fmt.Sprintf("%s %s in %s, %s | %s", bhk, unit, locality, city, p.Name)
		} else {
			title = fmt.Sprintf("%s in %s, %s | %s", unit, locality, city, p.Name)
		}
	}
	if desc == "" {
		desc = fmt.Sprintf("%s in %s, %s by %s. RERA registered project with %d configuration(s). View price, floor plans, amenities and availability.",
			p.Name, locality, city, p.BuilderName, len(configs))
	}
	return truncate(title, 300), truncate(desc, 500)
}

func bhkLabel(configs []dto.ConfigurationRequest) string {
	seen := map[int]bool{}
	var bhks []int
	for _, c := range configs {
		if c.Bedrooms > 0 && !seen[c.Bedrooms] {
			seen[c.Bedrooms] = true
			bhks = append(bhks, c.Bedrooms)
		}
	}
	if len(bhks) == 0 {
		return ""
	}
	sortInts(bhks)
	parts := make([]string, 0, len(bhks))
	for _, b := range bhks {
		parts = append(parts, strconv.Itoa(b))
	}
	return strings.Join(parts, ", ") + " BHK"
}

func propertyTypeLabel(t models.PropertyType) string {
	switch t {
	case models.PropertyTypeApartment:
		return "Apartments"
	case models.PropertyTypeFlat:
		return "Flats"
	case models.PropertyTypeVilla:
		return "Villas"
	case models.PropertyTypeRowHouse:
		return "Row Houses"
	case models.PropertyTypePlot:
		return "Plots"
	case models.PropertyTypePenthouse:
		return "Penthouses"
	case models.PropertyTypeStudioApartment:
		return "Studio Apartments"
	case models.PropertyTypeCommercialOffice:
		return "Commercial Offices"
	case models.PropertyTypeCommercialShop:
		return "Commercial Shops"
	default:
		return "Properties"
	}
}

func toConfigModels(reqs []dto.ConfigurationRequest) []models.PropertyConfiguration {
	out := make([]models.PropertyConfiguration, 0, len(reqs))
	for _, c := range reqs {
		availability := c.Availability
		if availability == "" {
			availability = "AVAILABLE"
		}
		out = append(out, models.PropertyConfiguration{
			Name: c.Name, CarpetArea: c.CarpetArea, BuiltUpArea: c.BuiltUpArea,
			SuperBuiltUpArea: c.SuperBuiltUpArea,
			Bedrooms:         c.Bedrooms, Bathrooms: c.Bathrooms, Balconies: c.Balconies,
			Price: c.Price, MinPrice: c.MinPrice, MaxPrice: c.MaxPrice,
			Availability: availability, NumberOfUnits: c.NumberOfUnits,
		})
	}
	return out
}

func toRERAModel(req *dto.RERARequest) (*models.RERADetails, error) {
	if req == nil {
		return nil, nil
	}
	r := &models.RERADetails{
		RegistrationNumber: strings.TrimSpace(req.RegistrationNumber),
		Authority:          strings.TrimSpace(req.Authority),
		VerificationStatus: models.RERAStatusPendingVerification,
	}
	if r.RegistrationNumber == "" {
		r.VerificationStatus = models.RERAStatusNotAvailable
	}
	d, err := parseDate(req.RegistrationDate)
	if err != nil {
		return nil, err
	}
	r.RegistrationDate = d
	d, err = parseDate(req.CompletionDate)
	if err != nil {
		return nil, err
	}
	r.CompletionDate = d
	return r, nil
}

func parseDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, Invalid("Dates must be in YYYY-MM-DD format")
	}
	return &d, nil
}

func parseUintList(s string) []uint {
	var out []uint
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if n, err := strconv.ParseUint(part, 10, 64); err == nil && n > 0 {
			out = append(out, uint(n))
		}
	}
	return out
}

func parseBool(s string) *bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		t := true
		return &t
	case "0", "false", "no":
		f := false
		return &f
	default:
		return nil
	}
}

func normalizePage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 12
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func newMeta(page, perPage int, total int64) *dto.Meta {
	pages := int((total + int64(perPage) - 1) / int64(perPage))
	return &dto.Meta{Page: page, PerPage: perPage, Total: total, TotalPages: pages}
}

func allStatuses() []models.PropertyStatus {
	return []models.PropertyStatus{
		models.PropertyStatusDraft, models.PropertyStatusSubmitted,
		models.PropertyStatusUnderReview, models.PropertyStatusApproved,
		models.PropertyStatusRejected, models.PropertyStatusPublished,
		models.PropertyStatusSoldOut,
	}
}

// searchCacheKey derives a stable cache key from the query parameters.
// searchCacheKey must include every parameter that changes the result set.
// Purpose and the rental filters are part of it: without them a rent search and
// a buy search with otherwise identical parameters would share a cache entry and
// serve each other's results.
func searchCacheKey(q dto.PropertyListQuery) string {
	return fmt.Sprintf(
		"search:%s|%s|%s|%d|%d|%s|%s|%.0f|%.0f|%.0f|%.0f|%d|%s|%s|%s|%s|%s|%s|%d|%d"+
			"|%.0f|%.0f|%s|%s|%s|%s",
		q.Purpose,
		q.CitySlug, q.LocalitySlug, q.CityID, q.LocalityID,
		q.PropertyType, q.ConstructionStatus,
		q.MinPrice, q.MaxPrice, q.MinArea, q.MaxArea, q.BHK,
		q.Amenities, q.Builder, q.PossessionBefore, q.Featured, q.RERAVerified,
		q.Search+"|"+q.SortBy+"|"+q.SortOrder, q.Page, q.PerPage,
		q.MinRent, q.MaxRent, q.Furnishing, q.TenantPreference, q.PetAllowed, q.AvailableFrom)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func joinWithAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

// validatePurposePayload enforces the rules that depend on the purpose, which a
// struct tag cannot express: a sale listing needs a price on every unit, and a
// rental needs its rent.
func validatePurposePayload(purpose models.PropertyPurpose, configs []dto.ConfigurationRequest, rental *dto.RentalRequest) error {
	if purpose == models.PurposeRent {
		if rental == nil {
			return Invalid("Add the rental terms, starting with the monthly rent")
		}
		if rental.MonthlyRent <= 0 {
			return Invalid("Enter the monthly rent")
		}
		if rental.MaxLeaseMonths > 0 && rental.MaxLeaseMonths < rental.MinLeaseMonths {
			return Invalid("The maximum lease cannot be shorter than the minimum")
		}
		return nil
	}

	// Sale listings price each unit configuration.
	for _, c := range configs {
		if c.Price <= 0 {
			return Invalid("Enter a price for every configuration, including " + c.Name)
		}
	}
	if rental != nil {
		return Invalid("Rental terms can only be set on a listing offered for rent")
	}
	return nil
}

// toRentalModel maps the rental block, applying the defaults an owner rarely
// thinks about: an 11-month lease and a month's notice, which is the Indian norm.
func toRentalModel(req *dto.RentalRequest) (*models.RentalDetails, error) {
	if req == nil {
		return nil, nil
	}
	r := &models.RentalDetails{
		MonthlyRent:         req.MonthlyRent,
		SecurityDeposit:     req.SecurityDeposit,
		MaintenanceAmount:   req.MaintenanceAmount,
		MaintenanceIncluded: req.MaintenanceIncluded,
		MinLeaseMonths:      req.MinLeaseMonths,
		MaxLeaseMonths:      req.MaxLeaseMonths,
		NoticePeriodDays:    req.NoticePeriodDays,
		FurnishingType:      models.FurnishingType(req.FurnishingType),
		TenantPreference:    models.TenantPreference(req.TenantPreference),
		PetAllowed:          req.PetAllowed,
	}
	if r.MinLeaseMonths == 0 {
		r.MinLeaseMonths = 11
	}
	if r.NoticePeriodDays == 0 {
		r.NoticePeriodDays = 30
	}
	if r.FurnishingType == "" {
		r.FurnishingType = models.FurnishingUnfurnished
	}
	if r.TenantPreference == "" {
		r.TenantPreference = models.TenantAnyone
	}

	d, err := parseDate(req.AvailableFrom)
	if err != nil {
		return nil, err
	}
	r.AvailableFrom = d

	if len(req.IncludedItems) > 0 {
		encoded, err := json.Marshal(req.IncludedItems)
		if err != nil {
			return nil, Invalid("Could not read the list of included items")
		}
		r.IncludedItems = string(encoded)
	}
	return r, nil
}

// postedByType records who created the listing, derived from the kind of
// account posting it. It feeds the trust badge and the source analytics.
func (s *PropertyService) postedByType(ctx context.Context, clientID uint) models.PostedByType {
	client, err := s.accounts.GetClient(ctx, clientID)
	if err != nil {
		return models.PostedByClient
	}
	switch client.ProfileType {
	case models.ProfileOwner:
		return models.PostedByOwner
	case models.ProfileAgent:
		return models.PostedByAgent
	default:
		return models.PostedByClient
	}
}

// listingDays is how long a new rental stays live before it needs renewing.
func (s *PropertyService) listingDays() int {
	if s.cfg != nil && s.cfg.Rental.ListingDays > 0 {
		return s.cfg.Rental.ListingDays
	}
	return 60
}

// hasListingImage reports whether a gallery contains something usable as the
// listing photo. Rentals are shot room by room, so those buckets count too — a
// flat photographed only as "living room" is still illustrated.
func hasListingImage(media []models.PropertyMedia) bool {
	for _, m := range media {
		switch m.MediaType {
		case models.MediaCoverImage, models.MediaPropertyImage, models.MediaSocietyImage,
			models.MediaLivingRoom, models.MediaBedroom, models.MediaKitchen,
			models.MediaBathroom, models.MediaBalcony, models.MediaBuildingExterior:
			return true
		}
	}
	return false
}

// SetAvailability takes a rental off the market or puts it back. Availability
// is independent of the approval workflow: the listing stays PUBLISHED, so the
// owner keeps its history, its leads and its analytics, and can relist in one
// click when the tenant moves out.
func (s *PropertyService) SetAvailability(ctx context.Context, id, clientID uint, req dto.UpdateAvailabilityRequest) (*dto.PropertyResponse, error) {
	p, err := s.loadOwned(ctx, id, clientID)
	if err != nil {
		return nil, err
	}
	if p.Purpose != models.PurposeRent {
		return nil, Invalid("Availability applies to rental listings only")
	}

	status := models.AvailabilityStatus(req.AvailabilityStatus)
	if err := s.props.SetAvailability(ctx, id, status); err != nil {
		return nil, Internal("update availability", err)
	}

	// Coming back on the market with a stale date would advertise a month that
	// has already passed, so the owner may re-date the listing in the same call.
	if req.AvailableFrom != "" {
		from, err := parseDate(req.AvailableFrom)
		if err != nil {
			return nil, err
		}
		if from != nil && p.RentalDetails != nil {
			rental := *p.RentalDetails
			rental.AvailableFrom = from
			// Update re-saves the property alongside the rental terms, and `p`
			// still carries the availability it had when it was loaded. Without
			// this line the write puts the old status straight back, so a
			// relisted flat stays out of the search results.
			p.AvailabilityStatus = status
			if err := s.props.Update(ctx, repositories.PropertyWrite{
				Property: p, Rental: &rental,
			}); err != nil {
				return nil, Internal("update availability date", err)
			}
		}
	}

	// A rented flat must leave the search results immediately.
	s.invalidateSearchCache(ctx)
	return s.reload(ctx, id, clientID)
}

// RenewListing pushes a rental's expiry out by another listing period. Renewing
// is deliberately not an edit: an owner confirming the flat is still free
// should not send the listing back through approval.
func (s *PropertyService) RenewListing(ctx context.Context, id, clientID uint) (*dto.PropertyResponse, error) {
	p, err := s.loadOwned(ctx, id, clientID)
	if err != nil {
		return nil, err
	}
	if p.Purpose != models.PurposeRent {
		return nil, Invalid("Only rental listings expire, so only they can be renewed")
	}

	// Renew from today rather than from the old expiry, so a listing renewed
	// late still gets a full period rather than one that is already half spent.
	until := time.Now().AddDate(0, 0, s.listingDays())
	if err := s.props.ExtendListing(ctx, id, until); err != nil {
		return nil, Internal("renew listing", err)
	}
	// An expired listing was taken off the market by the sweep; renewing puts
	// it back, which is the whole point of the button.
	if p.AvailabilityStatus == models.AvailabilityInactive {
		if err := s.props.SetAvailability(ctx, id, models.AvailabilityAvailable); err != nil {
			return nil, Internal("reactivate listing", err)
		}
	}
	s.invalidateSearchCache(ctx)
	return s.reload(ctx, id, clientID)
}

// ExpiringForClient lists the rentals that lapse within the given number of
// days, which is what the owner dashboard warns about.
func (s *PropertyService) ExpiringForClient(ctx context.Context, clientID uint, withinDays int) ([]dto.ExpiringListing, error) {
	if withinDays <= 0 {
		withinDays = 14
	}
	props, err := s.props.ExpiringRentals(ctx, time.Time{}, time.Now().AddDate(0, 0, withinDays))
	if err != nil {
		return nil, Internal("load expiring listings", err)
	}

	out := make([]dto.ExpiringListing, 0, len(props))
	for _, p := range props {
		// An administrator (clientID 0) sees every account's listings.
		if clientID != 0 && p.ClientID != clientID {
			continue
		}
		if p.ListingExpiryDate == nil {
			continue
		}
		row := dto.ExpiringListing{
			ID: p.ID, Name: p.Name, Slug: p.Slug,
			ExpiresOn:  p.ListingExpiryDate.Format("2006-01-02"),
			DaysLeft:   daysUntil(*p.ListingExpiryDate),
			HasExpired: p.ListingExpiryDate.Before(time.Now()),
		}
		if p.RentalDetails != nil {
			row.MonthlyRent = p.RentalDetails.MonthlyRent
		}
		out = append(out, row)
	}
	return out, nil
}

// loadOwned fetches a listing the caller is allowed to act on. A builder or
// owner is restricted to their own; an administrator (clientID 0) is not.
func (s *PropertyService) loadOwned(ctx context.Context, id, clientID uint) (*models.Property, error) {
	var (
		p   *models.Property
		err error
	)
	if clientID == 0 {
		p, err = s.props.GetByID(ctx, id)
	} else {
		p, err = s.props.GetForOwner(ctx, id, clientID)
	}
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	return p, nil
}

// reload returns the listing as the caller should now see it.
func (s *PropertyService) reload(ctx context.Context, id, clientID uint) (*dto.PropertyResponse, error) {
	p, err := s.loadOwned(ctx, id, clientID)
	if err != nil {
		return nil, err
	}
	resp := ToPropertyResponse(p)
	return &resp, nil
}

// daysUntil counts whole days from today, so "expires tomorrow" reads as 1 and
// anything already past reads as 0 rather than a negative number.
func daysUntil(t time.Time) int {
	d := int(time.Until(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// CreateOnBehalf posts a listing for an owner who is not doing it themselves —
// an administrator taking details over the phone, most often. The listing
// belongs to the owner's account, so their leads, analytics and edits all work
// normally, but the provenance records that an administrator entered it: a
// trust badge must not claim the owner wrote something they never saw.
func (s *PropertyService) CreateOnBehalf(ctx context.Context, clientID, adminID uint, req dto.CreatePropertyRequest) (*dto.PropertyResponse, error) {
	client, err := s.accounts.GetClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Listing account not found")
		}
		return nil, Internal("load listing account", err)
	}

	created, err := s.Create(ctx, clientID, adminID, req)
	if err != nil {
		return nil, err
	}
	ownerUserID := client.UserID
	if err := s.props.SetProvenance(ctx, created.ID, models.PostedByAdmin, &ownerUserID); err != nil {
		return nil, Internal("record listing provenance", err)
	}
	return s.reload(ctx, created.ID, 0)
}
