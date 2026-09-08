package services

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
)

// AnalyticsService turns the raw event stream into dashboard figures. The same
// queries serve administrators and builders; the caller decides the scope, and
// a builder scope is always pinned to their own client ID.
type AnalyticsService struct {
	analytics repositories.AnalyticsRepository
	props     repositories.PropertyRepository
	engage    repositories.EngagementRepository
	users     repositories.UserRepository
	accounts  repositories.AccountRepository
	stats     repositories.StatsRepository
	reports   repositories.ReportRepository
}

func NewAnalyticsService(
	analytics repositories.AnalyticsRepository,
	props repositories.PropertyRepository,
	engage repositories.EngagementRepository,
	users repositories.UserRepository,
	accounts repositories.AccountRepository,
	stats repositories.StatsRepository,
	reports repositories.ReportRepository,
) *AnalyticsService {
	return &AnalyticsService{analytics: analytics, props: props, engage: engage,
		users: users, accounts: accounts, stats: stats, reports: reports}
}

// ResolveRange turns the dashboard period selector into concrete bounds.
// Unknown or malformed input falls back to the last 30 days.
func ResolveRange(q dto.AnalyticsQuery) repositories.AnalyticsRange {
	now := time.Now()
	end := now
	var start time.Time

	switch strings.ToLower(q.Period) {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "7d", "week":
		start = now.AddDate(0, 0, -7)
	case "90d", "quarter":
		start = now.AddDate(0, 0, -90)
	case "custom":
		if from, err := time.Parse("2006-01-02", q.From); err == nil {
			start = from
		} else {
			start = now.AddDate(0, 0, -30)
		}
		if to, err := time.Parse("2006-01-02", q.To); err == nil {
			end = to.Add(24*time.Hour - time.Second)
		}
	default: // "30d" and anything unrecognised
		start = now.AddDate(0, 0, -30)
	}
	if end.Before(start) {
		start, end = end, start
	}
	return repositories.AnalyticsRange{
		From: start, To: end,
		CityID: q.CityID, ClientID: q.ClientID, PropertyID: q.PropertyID,
	}
}

// AdminDashboard builds the platform-wide dashboard.
func (s *AnalyticsService) AdminDashboard(ctx context.Context, q dto.AnalyticsQuery) (*dto.AdminDashboardResponse, error) {
	rng := ResolveRange(q)

	stats, err := s.adminStats(ctx, rng)
	if err != nil {
		return nil, err
	}
	charts, err := s.charts(ctx, rng)
	if err != nil {
		return nil, err
	}
	mostViewed, err := s.analytics.TopProperties(ctx, rng, "views", 10)
	if err != nil {
		return nil, Internal("load most viewed properties", err)
	}
	mostContacted, err := s.analytics.TopProperties(ctx, rng, "inquiries", 10)
	if err != nil {
		return nil, Internal("load most contacted properties", err)
	}
	cities, err := s.analytics.TopCities(ctx, rng, 8)
	if err != nil {
		return nil, Internal("load top cities", err)
	}
	localities, err := s.analytics.TopSearchedLocalities(ctx, rng, 8)
	if err != nil {
		return nil, Internal("load top localities", err)
	}
	recentUsers, err := s.users.RecentRegistrations(ctx, 8)
	if err != nil {
		return nil, Internal("load recent registrations", err)
	}
	recentInquiries, err := s.engage.RecentInquiries(ctx, 0, 8)
	if err != nil {
		return nil, Internal("load recent inquiries", err)
	}

	resp := &dto.AdminDashboardResponse{
		Stats:              *stats,
		Charts:             *charts,
		MostViewed:         toStatDTOs(mostViewed),
		MostContacted:      toStatDTOs(mostContacted),
		PopularCities:      toLabelDTOs(cities),
		SearchedLocalities: toLabelDTOs(localities),
	}
	for i := range recentUsers {
		resp.RecentRegistrations = append(resp.RecentRegistrations, ToUserResponse(&recentUsers[i]))
	}
	for i := range recentInquiries {
		resp.RecentInquiries = append(resp.RecentInquiries, ToInquiryResponse(&recentInquiries[i]))
	}
	return resp, nil
}

func (s *AnalyticsService) adminStats(ctx context.Context, rng repositories.AnalyticsRange) (*dto.DashboardStats, error) {
	stats := &dto.DashboardStats{}
	var err error

	if stats.TotalUsers, err = s.users.Count(ctx, ""); err != nil {
		return nil, Internal("count users", err)
	}
	if stats.TotalClients, err = s.accounts.CountClients(ctx); err != nil {
		return nil, Internal("count clients", err)
	}
	if stats.TotalCustomers, err = s.users.Count(ctx, "CUSTOMER"); err != nil {
		return nil, Internal("count customers", err)
	}
	if stats.TotalProperties, err = s.props.CountAll(ctx, 0); err != nil {
		return nil, Internal("count properties", err)
	}
	counts := map[models.PropertyStatus]*int64{
		models.PropertyStatusPublished: &stats.ActiveProperties,
		models.PropertyStatusSubmitted: &stats.PendingProperties,
		models.PropertyStatusApproved:  &stats.ApprovedProperties,
		models.PropertyStatusRejected:  &stats.RejectedProperties,
		models.PropertyStatusDraft:     &stats.DraftProperties,
	}
	for status, target := range counts {
		n, err := s.props.CountByStatus(ctx, status, 0)
		if err != nil {
			return nil, Internal("count properties by status", err)
		}
		*target = n
	}
	// Anything under review is also awaiting an administrator.
	underReview, err := s.props.CountByStatus(ctx, models.PropertyStatusUnderReview, 0)
	if err != nil {
		return nil, Internal("count properties under review", err)
	}
	stats.PendingProperties += underReview
	// Approved listings are live alongside explicitly published ones.
	stats.ActiveProperties += stats.ApprovedProperties

	if stats.TotalInquiries, err = s.engage.CountInquiries(ctx, 0, ""); err != nil {
		return nil, Internal("count inquiries", err)
	}
	if stats.NewInquiries, err = s.engage.CountInquiries(ctx, 0, "NEW"); err != nil {
		return nil, Internal("count new inquiries", err)
	}
	if stats.WhatsAppLeads, err = s.analytics.CountEvents(ctx, models.EventWhatsappClick, rng); err != nil {
		return nil, Internal("count whatsapp leads", err)
	}
	if stats.TotalViews, err = s.analytics.CountViews(ctx, rng); err != nil {
		return nil, Internal("count views", err)
	}
	if stats.UniqueVisitors, err = s.analytics.CountUniqueVisitors(ctx, rng); err != nil {
		return nil, Internal("count unique visitors", err)
	}

	sale, rental, rented, buyLeads, rentLeads, err := s.purposeSplit(ctx, 0)
	if err != nil {
		return nil, err
	}
	stats.SaleProperties, stats.RentalProperties, stats.RentedProperties = sale, rental, rented
	stats.BuyLeads, stats.RentLeads = buyLeads, rentLeads

	// Moderation is work waiting on an administrator, so it belongs in the same
	// headline block as the approval queue.
	if stats.OpenReports, err = s.reports.Count(ctx, string(models.ReportOpen)); err != nil {
		return nil, Internal("count open reports", err)
	}
	return stats, nil
}

// ClientDashboard builds the builder dashboard, restricted to clientID.
func (s *AnalyticsService) ClientDashboard(ctx context.Context, clientID uint, q dto.AnalyticsQuery) (*dto.ClientDashboardResponse, error) {
	q.ClientID = clientID // ignore any client filter supplied by the caller
	rng := ResolveRange(q)
	rng.ClientID = clientID

	stats := dto.ClientStats{}
	var err error
	if stats.TotalProperties, err = s.props.CountAll(ctx, clientID); err != nil {
		return nil, Internal("count properties", err)
	}
	for status, target := range map[models.PropertyStatus]*int64{
		models.PropertyStatusPublished: &stats.ActiveProperties,
		models.PropertyStatusSubmitted: &stats.PendingProperties,
		models.PropertyStatusRejected:  &stats.RejectedProperties,
		models.PropertyStatusDraft:     &stats.DraftProperties,
	} {
		n, err := s.props.CountByStatus(ctx, status, clientID)
		if err != nil {
			return nil, Internal("count properties by status", err)
		}
		*target = n
	}
	approved, err := s.props.CountByStatus(ctx, models.PropertyStatusApproved, clientID)
	if err != nil {
		return nil, Internal("count approved properties", err)
	}
	stats.ActiveProperties += approved
	underReview, err := s.props.CountByStatus(ctx, models.PropertyStatusUnderReview, clientID)
	if err != nil {
		return nil, Internal("count properties under review", err)
	}
	stats.PendingProperties += underReview

	if stats.TotalViews, err = s.analytics.CountViews(ctx, rng); err != nil {
		return nil, Internal("count views", err)
	}
	if stats.UniqueVisitors, err = s.analytics.CountUniqueVisitors(ctx, rng); err != nil {
		return nil, Internal("count unique visitors", err)
	}
	if stats.TotalInquiries, err = s.engage.CountInquiries(ctx, clientID, ""); err != nil {
		return nil, Internal("count inquiries", err)
	}
	for eventType, target := range map[models.EventType]*int64{
		models.EventWhatsappClick:    &stats.WhatsAppLeads,
		models.EventPhoneClick:       &stats.PhoneClicks,
		models.EventBrochureDownload: &stats.BrochureDownloads,
		models.EventFavorite:         &stats.Favorites,
	} {
		n, err := s.analytics.CountEvents(ctx, eventType, rng)
		if err != nil {
			return nil, Internal("count events", err)
		}
		*target = n
	}
	stats.ConversionRate = rate(stats.TotalInquiries, stats.TotalViews)

	sale, rental, rented, buyLeads, rentLeads, err := s.purposeSplit(ctx, clientID)
	if err != nil {
		return nil, err
	}
	stats.SaleProperties, stats.RentalProperties, stats.RentedProperties = sale, rental, rented
	stats.BuyLeads, stats.RentLeads = buyLeads, rentLeads

	if stats.PendingVisits, err = s.stats.CountVisitsByStatus(ctx, models.VisitRequested, clientID); err != nil {
		return nil, Internal("count pending visits", err)
	}
	// Rentals lapse, and an owner who does not know cannot renew.
	expiring, err := s.props.ExpiringRentals(ctx, time.Time{}, time.Now().AddDate(0, 0, 14))
	if err != nil {
		return nil, Internal("count expiring listings", err)
	}
	for _, p := range expiring {
		if clientID == 0 || p.ClientID == clientID {
			stats.ExpiringSoon++
		}
	}

	top, err := s.analytics.TopProperties(ctx, rng, "views", 20)
	if err != nil {
		return nil, Internal("load property performance", err)
	}
	recent, err := s.engage.RecentInquiries(ctx, clientID, 8)
	if err != nil {
		return nil, Internal("load recent inquiries", err)
	}
	charts, err := s.charts(ctx, rng)
	if err != nil {
		return nil, err
	}

	resp := &dto.ClientDashboardResponse{
		Stats: stats, TopProperties: toStatDTOs(top), Charts: *charts,
	}
	for i := range recent {
		resp.RecentInquiries = append(resp.RecentInquiries, ToInquiryResponse(&recent[i]))
	}
	return resp, nil
}

// PropertyAnalytics returns per-listing performance. clientID, when non-zero,
// asserts ownership before any figures are returned.
func (s *AnalyticsService) PropertyAnalytics(ctx context.Context, propertyID, clientID uint) (*dto.PropertyStatDTO, error) {
	if clientID != 0 {
		if _, err := s.props.GetForOwner(ctx, propertyID, clientID); err != nil {
			return nil, NotFound("Property not found")
		}
	}
	stat, err := s.analytics.PropertyStats(ctx, propertyID)
	if err != nil {
		return nil, Internal("load property analytics", err)
	}
	out := toStatDTO(*stat)
	return &out, nil
}

// charts assembles the daily series shown on both dashboards.
func (s *AnalyticsService) charts(ctx context.Context, rng repositories.AnalyticsRange) (*dto.AnalyticsChartsDTO, error) {
	views, err := s.analytics.ViewsSeries(ctx, rng)
	if err != nil {
		return nil, Internal("load views series", err)
	}
	inquiries, err := s.analytics.InquiriesSeries(ctx, rng)
	if err != nil {
		return nil, Internal("load inquiries series", err)
	}
	registrations, err := s.analytics.RegistrationsSeries(ctx, rng)
	if err != nil {
		return nil, Internal("load registrations series", err)
	}
	whatsapp, err := s.analytics.EventSeries(ctx, models.EventWhatsappClick, rng)
	if err != nil {
		return nil, Internal("load whatsapp series", err)
	}
	return &dto.AnalyticsChartsDTO{
		Views:          toSeriesDTOs(views),
		Inquiries:      toSeriesDTOs(inquiries),
		Registrations:  toSeriesDTOs(registrations),
		WhatsAppClicks: toSeriesDTOs(whatsapp),
	}, nil
}

func toSeriesDTOs(points []repositories.TimeSeriesPoint) []dto.TimeSeriesDTO {
	out := make([]dto.TimeSeriesDTO, 0, len(points))
	for _, p := range points {
		// MySQL returns DATE() as a full timestamp through the driver; keep only
		// the date part so the chart labels stay short.
		date := p.Date
		if len(date) > 10 {
			date = date[:10]
		}
		out = append(out, dto.TimeSeriesDTO{Date: date, Count: p.Count})
	}
	return out
}

func toStatDTOs(stats []repositories.PropertyStat) []dto.PropertyStatDTO {
	out := make([]dto.PropertyStatDTO, 0, len(stats))
	for _, s := range stats {
		out = append(out, toStatDTO(s))
	}
	return out
}

func toStatDTO(s repositories.PropertyStat) dto.PropertyStatDTO {
	return dto.PropertyStatDTO{
		PropertyID: s.PropertyID, Name: s.Name, Slug: s.Slug,
		Views: s.Views, UniqueVisitors: s.UniqueVisitors,
		Inquiries: s.Inquiries, WhatsAppClicks: s.WhatsAppClicks,
		PhoneClicks: s.PhoneClicks, Favorites: s.Favorites,
		BrochureDownloads: s.BrochureDownloads, ConversionRate: s.ConversionRate,
	}
}

func toLabelDTOs(items []repositories.LabelCount) []dto.LabelCountDTO {
	out := make([]dto.LabelCountDTO, 0, len(items))
	for _, i := range items {
		out = append(out, dto.LabelCountDTO{Label: i.Label, Slug: i.Slug, Count: i.Count})
	}
	return out
}

func rate(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

// applyPurposeSplit fills in the buy/rent breakdown shared by both dashboards.
// A single "properties" figure hides which half of the business is growing, and
// once an account can hold both a flat for sale and one to let, a single leads
// figure hides which kind of enquiry is actually arriving.
//
// clientID of 0 means the whole platform, which is what the administrator sees.
func (s *AnalyticsService) purposeSplit(ctx context.Context, clientID uint) (sale, rental, rented, buyLeads, rentLeads int64, err error) {
	if sale, err = s.stats.CountByPurpose(ctx, models.PurposeBuy, clientID); err != nil {
		return 0, 0, 0, 0, 0, Internal("count sale listings", err)
	}
	if rental, err = s.stats.CountByPurpose(ctx, models.PurposeRent, clientID); err != nil {
		return 0, 0, 0, 0, 0, Internal("count rental listings", err)
	}
	if rented, err = s.stats.CountByAvailability(ctx, models.AvailabilityRented, clientID); err != nil {
		return 0, 0, 0, 0, 0, Internal("count let listings", err)
	}
	if buyLeads, err = s.stats.CountLeadsByType(ctx, models.LeadBuy, clientID); err != nil {
		return 0, 0, 0, 0, 0, Internal("count buy leads", err)
	}
	if rentLeads, err = s.stats.CountLeadsByType(ctx, models.LeadRent, clientID); err != nil {
		return 0, 0, 0, 0, 0, Internal("count rent leads", err)
	}
	return sale, rental, rented, buyLeads, rentLeads, nil
}

// RentGuide summarises what live rentals are asking in an area. It reports the
// number of listings behind the figures so a thin sample is visible as one,
// rather than a single flat being presented as the market rate.
func (s *AnalyticsService) RentGuide(ctx context.Context, q dto.RentGuideQuery) (*dto.RentGuideResponse, error) {
	summary, err := s.stats.RentSummary(ctx, repositories.RentStatsFilter{
		CityID: q.CityID, LocalityID: q.LocalityID, BHK: q.BHK,
	})
	if err != nil {
		return nil, Internal("summarise rents", err)
	}
	return &dto.RentGuideResponse{
		Listings:    summary.Listings,
		AverageRent: math.Round(summary.AverageRent),
		MinRent:     summary.MinRent,
		MaxRent:     summary.MaxRent,
		CityID:      q.CityID, LocalityID: q.LocalityID, BHK: q.BHK,
	}, nil
}
