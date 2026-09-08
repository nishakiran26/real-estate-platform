package handlers

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/middleware"
	"github.com/rera/property-platform/backend/internal/services"
)

// PublicHandler serves everything a visitor can reach without logging in:
// search, property detail, homepage content, leads and event tracking.
type PublicHandler struct {
	props    *services.PropertyService
	geo      *services.GeoService
	cms      *services.CMSService
	engage   *services.EngagementService
	recs     *services.RecommendationService
	accounts *services.AccountService
	visits   *services.VisitService
	reports  *services.ReportService
	stats    *services.AnalyticsService
	seo      *services.SEOService
}

func NewPublicHandler(
	props *services.PropertyService,
	geo *services.GeoService,
	cms *services.CMSService,
	engage *services.EngagementService,
	recs *services.RecommendationService,
	accounts *services.AccountService,
	visits *services.VisitService,
	reports *services.ReportService,
	stats *services.AnalyticsService,
	seo *services.SEOService,
) *PublicHandler {
	return &PublicHandler{props: props, geo: geo, cms: cms, engage: engage,
		recs: recs, accounts: accounts, visits: visits, reports: reports,
		stats: stats, seo: seo}
}

// Homepage returns banners, featured listings and popular locations in one call.
func (h *PublicHandler) Homepage(c echo.Context) error {
	data, err := h.cms.Homepage(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Homepage content fetched successfully", data)
}

// Meta returns the reference data the search filters are built from.
func (h *PublicHandler) Meta(c echo.Context) error {
	ctx := c.Request().Context()
	cities, err := h.geo.ListCities(ctx, false)
	if err != nil {
		return fail(c, err)
	}
	amenities, err := h.geo.ListAmenities(ctx)
	if err != nil {
		return fail(c, err)
	}
	localities, err := h.geo.ListLocalities(ctx, uint(intQuery(c, "city_id", 0)), false)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Reference data fetched successfully", echo.Map{
		"cities":               cities,
		"localities":           localities,
		"amenities":            amenities,
		"property_types":       propertyTypeOptions(),
		"construction_status":  constructionStatusOptions(),
		"rera_status":          reraStatusOptions(),
		"budget_ranges":        budgetRanges(),
		"rent_ranges":          rentRanges(),
		"furnishing":           furnishingOptions(),
		"tenant_preference":    tenantPreferenceOptions(),
		"availability":         availabilityOptions(),
		"rental_photo_types":   rentalPhotoTypes(),
		"included_items":       includedItemOptions(),
		"bhk_options":          []int{1, 2, 3, 4, 5},
		"sort_options":         sortOptions(),
		"inquiry_status_flow":  inquiryStatuses(),
		"property_status_flow": propertyStatusFlow(),
	})
}

// ListProperties runs the public search. Every filter is a query parameter so a
// result page can be shared as a link.
func (h *PublicHandler) ListProperties(c echo.Context) error {
	var q dto.PropertyListQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid search parameters"))
	}
	cards, meta, err := h.props.ListPublic(c.Request().Context(), q, middleware.OptionalUserID(c))
	if err != nil {
		return fail(c, err)
	}
	// A search is an engagement signal; it powers the popular-locality report.
	if q.Search != "" || q.CitySlug != "" || q.LocalitySlug != "" {
		h.trackSearch(c, q)
	}
	return paginated(c, "Properties fetched successfully", cards, meta)
}

// trackSearch records the search terms. Values are read from the request before
// the write so nothing touches the Echo context afterwards.
func (h *PublicHandler) trackSearch(c echo.Context, q dto.PropertyListQuery) {
	metadata, err := json.Marshal(map[string]any{
		"q": q.Search, "city": q.CitySlug, "locality": q.LocalitySlug,
		"bhk": q.BHK, "min_price": q.MinPrice, "max_price": q.MaxPrice,
		"property_type": q.PropertyType, "status": q.ConstructionStatus,
	})
	if err != nil {
		return
	}
	_ = h.engage.TrackEvent(c.Request().Context(), dto.TrackEventRequest{
		EventType: "SEARCH", Metadata: string(metadata),
	}, visitor(c))
}

// GetProperty returns one published property and records the view. The path
// parameter is either the numeric ID or the SEO slug.
func (h *PublicHandler) GetProperty(c echo.Context) error {
	property, err := h.resolveProperty(c)
	if err != nil {
		return fail(c, err)
	}
	if err := h.engage.RecordView(c.Request().Context(), property.ID, visitor(c)); err != nil {
		// A failed analytics write must never break the page.
		_ = err
	}
	return ok(c, "Property fetched successfully", property)
}

// SimilarProperties powers the "you may also like" strip on a detail page.
func (h *PublicHandler) SimilarProperties(c echo.Context) error {
	property, err := h.resolveProperty(c)
	if err != nil {
		return fail(c, err)
	}
	cards, err := h.props.Similar(c.Request().Context(), property.Slug, intQuery(c, "limit", 8))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Similar properties fetched successfully", cards)
}

// resolveProperty loads a published property from the :key path parameter,
// which may be either its numeric ID or its slug.
func (h *PublicHandler) resolveProperty(c echo.Context) (*dto.PropertyResponse, error) {
	key := strings.TrimSpace(c.Param("key"))
	if key == "" {
		return nil, services.Invalid("Invalid property")
	}
	viewer := middleware.OptionalUserID(c)
	if id, err := strconv.ParseUint(key, 10, 64); err == nil && id > 0 {
		return h.props.GetPublicByID(c.Request().Context(), uint(id), viewer)
	}
	return h.props.GetPublicBySlug(c.Request().Context(), key, viewer)
}

// CompareProperties returns full detail for the selected properties.
func (h *PublicHandler) CompareProperties(c echo.Context) error {
	ids := parseIDList(c.QueryParam("ids"))
	if len(ids) == 0 {
		return fail(c, services.Invalid("Select at least one property to compare"))
	}
	items, err := h.props.Compare(c.Request().Context(), ids)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Comparison fetched successfully", items)
}

// RecentlyViewed works for anonymous visitors through the session cookie.
func (h *PublicHandler) RecentlyViewed(c echo.Context) error {
	cards, err := h.engage.RecentlyViewed(c.Request().Context(), visitor(c), intQuery(c, "limit", 8))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Recently viewed properties fetched successfully", cards)
}

// Recommendations returns rule-based suggestions for the signed-in customer and
// popular listings for anonymous visitors.
func (h *PublicHandler) Recommendations(c echo.Context) error {
	limit := intQuery(c, "limit", 8)
	cards, err := h.recs.ForUser(c.Request().Context(), middleware.UserID(c), limit)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Recommendations fetched successfully", cards)
}

// Cities lists the cities the platform covers.
func (h *PublicHandler) Cities(c echo.Context) error {
	cities, err := h.geo.ListCities(c.Request().Context(), boolQuery(c, "popular", false))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Cities fetched successfully", cities)
}

// Localities lists localities, optionally scoped to one city.
func (h *PublicHandler) Localities(c echo.Context) error {
	localities, err := h.geo.ListLocalities(c.Request().Context(),
		uint(intQuery(c, "city_id", 0)), boolQuery(c, "popular", false))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Localities fetched successfully", localities)
}

// Amenities lists the amenity catalogue used by filters and property forms.
func (h *PublicHandler) Amenities(c echo.Context) error {
	amenities, err := h.geo.ListAmenities(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Amenities fetched successfully", amenities)
}

// City returns one city by slug for its landing page.
func (h *PublicHandler) City(c echo.Context) error {
	city, err := h.geo.GetCityBySlug(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "City fetched successfully", city)
}

// Locality returns one locality by city and locality slug.
func (h *PublicHandler) Locality(c echo.Context) error {
	locality, err := h.geo.GetLocalityBySlug(c.Request().Context(), c.Param("slug"), c.Param("locality_slug"))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Locality fetched successfully", locality)
}

// Builder returns the public profile of a builder.
func (h *PublicHandler) Builder(c echo.Context) error {
	id, err := uintParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	builder, err := h.accounts.PublicBuilder(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Builder fetched successfully", builder)
}

// Page returns a published static page such as the RERA disclaimer.
func (h *PublicHandler) Page(c echo.Context) error {
	page, err := h.cms.GetPage(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return fail(c, err)
	}
	if !page.IsPublished {
		return fail(c, services.NotFound("Page not found"))
	}
	return ok(c, "Page fetched successfully", page)
}

// Pages lists the published static pages, for footer navigation.
func (h *PublicHandler) Pages(c echo.Context) error {
	pages, err := h.cms.ListPages(c.Request().Context(), true)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Pages fetched successfully", pages)
}

// CreateInquiry captures a lead. It is open to anonymous visitors, who must
// still pass validation and give explicit contact consent.
func (h *PublicHandler) CreateInquiry(c echo.Context) error {
	var req dto.CreateInquiryRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	inquiry, err := h.engage.CreateInquiry(c.Request().Context(), req, visitor(c))
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Thank you. The builder will contact you shortly.", inquiry)
}

// TrackEvent records an engagement event from the browser.
func (h *PublicHandler) TrackEvent(c echo.Context) error {
	var req dto.TrackEventRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.engage.TrackEvent(c.Request().Context(), req, visitor(c)); err != nil {
		return fail(c, err)
	}
	return ok(c, "Event recorded", nil)
}

// WhatsAppLink returns the prefilled deep link for a property and counts the
// click as a lead signal.
func (h *PublicHandler) WhatsAppLink(c echo.Context) error {
	property, err := h.resolveProperty(c)
	if err != nil {
		return fail(c, err)
	}
	link, err := h.engage.WhatsAppLink(c.Request().Context(), property.ID, c.QueryParam("configuration"), visitor(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "WhatsApp link generated", link)
}

// DownloadDocument records the download and redirects to the stored file.
func (h *PublicHandler) DownloadDocument(c echo.Context) error {
	property, err := h.resolveProperty(c)
	if err != nil {
		return fail(c, err)
	}
	docID, err := uintParam(c, "doc_id")
	if err != nil {
		return fail(c, err)
	}
	doc, err := h.engage.DownloadDocument(c.Request().Context(), property.ID, docID, visitor(c))
	if err != nil {
		return fail(c, err)
	}
	return c.Redirect(http.StatusFound, doc.FileURL)
}

// Sitemap renders the XML sitemap for search engines.
func (h *PublicHandler) Sitemap(c echo.Context) error {
	urls, err := h.seo.Sitemap(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	type urlEntry struct {
		XMLName    xml.Name `xml:"url"`
		Loc        string   `xml:"loc"`
		LastMod    string   `xml:"lastmod,omitempty"`
		ChangeFreq string   `xml:"changefreq,omitempty"`
		Priority   string   `xml:"priority,omitempty"`
	}
	type urlSet struct {
		XMLName xml.Name   `xml:"urlset"`
		Xmlns   string     `xml:"xmlns,attr"`
		URLs    []urlEntry `xml:"url"`
	}
	set := urlSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, u := range urls {
		set.URLs = append(set.URLs, urlEntry{
			Loc: u.Loc, LastMod: u.LastMod, ChangeFreq: u.ChangeFreq, Priority: u.Priority,
		})
	}
	return c.XMLPretty(http.StatusOK, set, "  ")
}

// Robots renders robots.txt.
func (h *PublicHandler) Robots(c echo.Context) error {
	return c.String(http.StatusOK, h.seo.Robots())
}

// Health is the container and load-balancer probe.
func (h *PublicHandler) Health(c echo.Context) error {
	return ok(c, "Service is healthy", echo.Map{"status": "ok"})
}

// parseIDList reads a comma-separated list of positive integers.
func parseIDList(raw string) []uint {
	out := make([]uint, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		if v, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64); err == nil && v > 0 {
			out = append(out, uint(v))
		}
	}
	return out
}

// RequestVisit records a request to view a property. It is open to anonymous
// visitors, like an inquiry, and rate limited for the same reason.
func (h *PublicHandler) RequestVisit(c echo.Context) error {
	property, err := h.resolveProperty(c)
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreateVisitRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	// The property comes from the path, so a visitor cannot book a viewing of
	// one listing while looking at another.
	req.PropertyID = property.ID

	visit, err := h.visits.Request(c.Request().Context(), req, visitor(c))
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Visit requested. The owner will confirm a time with you.", visit)
}

// ReportProperty flags a listing as wrong or abusive. Like an inquiry it is
// open to visitors who have not signed in: the people best placed to notice a
// flat that is already let are the ones still searching for it.
func (h *PublicHandler) ReportProperty(c echo.Context) error {
	property, err := h.resolveProperty(c)
	if err != nil {
		return fail(c, err)
	}
	var req dto.CreateReportRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	req.PropertyID = property.ID

	report, err := h.reports.Report(c.Request().Context(), req, visitor(c))
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Thank you. Our team will review this listing.", report)
}

// RentGuide answers the question every renter and every owner asks first: what
// does a flat here actually go for? The figures come from live listings, so a
// thin area honestly reports how few listings are behind them.
func (h *PublicHandler) RentGuide(c echo.Context) error {
	var q dto.RentGuideQuery
	if err := c.Bind(&q); err != nil {
		return fail(c, services.Invalid("Invalid rent guide parameters"))
	}
	guide, err := h.stats.RentGuide(c.Request().Context(), q)
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Rent guide fetched successfully", guide)
}
