package repositories

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// AnalyticsRange scopes every analytics query. ClientID restricts results to one
// builder so the client dashboard can reuse the same queries as the admin one.
type AnalyticsRange struct {
	From       time.Time
	To         time.Time
	ClientID   uint
	CityID     uint
	PropertyID uint
}

// TimeSeriesPoint is one bucket of a daily chart.
type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// PropertyStat summarises engagement for one listing.
type PropertyStat struct {
	PropertyID        uint    `json:"property_id"`
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	Views             int64   `json:"views"`
	UniqueVisitors    int64   `json:"unique_visitors"`
	Inquiries         int64   `json:"inquiries"`
	WhatsAppClicks    int64   `json:"whatsapp_clicks"`
	PhoneClicks       int64   `json:"phone_clicks"`
	Favorites         int64   `json:"favorites"`
	BrochureDownloads int64   `json:"brochure_downloads"`
	ConversionRate    float64 `json:"conversion_rate"`
}

// LabelCount is a generic name/value pair for ranked lists.
type LabelCount struct {
	Label string `json:"label"`
	Slug  string `json:"slug"`
	Count int64  `json:"count"`
}

type AnalyticsRepository interface {
	CountEvents(ctx context.Context, eventType models.EventType, r AnalyticsRange) (int64, error)
	CountViews(ctx context.Context, r AnalyticsRange) (int64, error)
	CountUniqueVisitors(ctx context.Context, r AnalyticsRange) (int64, error)
	ViewsSeries(ctx context.Context, r AnalyticsRange) ([]TimeSeriesPoint, error)
	EventSeries(ctx context.Context, eventType models.EventType, r AnalyticsRange) ([]TimeSeriesPoint, error)
	InquiriesSeries(ctx context.Context, r AnalyticsRange) ([]TimeSeriesPoint, error)
	RegistrationsSeries(ctx context.Context, r AnalyticsRange) ([]TimeSeriesPoint, error)
	TopProperties(ctx context.Context, r AnalyticsRange, orderBy string, limit int) ([]PropertyStat, error)
	PropertyStats(ctx context.Context, propertyID uint) (*PropertyStat, error)
	TopCities(ctx context.Context, r AnalyticsRange, limit int) ([]LabelCount, error)
	TopSearchedLocalities(ctx context.Context, r AnalyticsRange, limit int) ([]LabelCount, error)
}

type analyticsRepo struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository { return &analyticsRepo{db: db} }

// scopeEvents restricts the event stream by date and, when a client or city is
// given, by joining through the property that the event refers to.
func (r *analyticsRepo) scopeEvents(ctx context.Context, rng AnalyticsRange) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&models.UserPropertyEvent{}).
		Where("user_property_events.created_at BETWEEN ? AND ?", rng.From, rng.To)
	if rng.PropertyID > 0 {
		q = q.Where("user_property_events.property_id = ?", rng.PropertyID)
	}
	if rng.ClientID > 0 || rng.CityID > 0 {
		q = q.Joins("JOIN properties ON properties.id = user_property_events.property_id")
		if rng.ClientID > 0 {
			q = q.Where("properties.client_id = ?", rng.ClientID)
		}
		if rng.CityID > 0 {
			q = q.Where("properties.city_id = ?", rng.CityID)
		}
	}
	return q
}

func (r *analyticsRepo) scopeViews(ctx context.Context, rng AnalyticsRange) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&models.PropertyView{}).
		Where("property_views.created_at BETWEEN ? AND ?", rng.From, rng.To)
	if rng.PropertyID > 0 {
		q = q.Where("property_views.property_id = ?", rng.PropertyID)
	}
	if rng.ClientID > 0 || rng.CityID > 0 {
		q = q.Joins("JOIN properties ON properties.id = property_views.property_id")
		if rng.ClientID > 0 {
			q = q.Where("properties.client_id = ?", rng.ClientID)
		}
		if rng.CityID > 0 {
			q = q.Where("properties.city_id = ?", rng.CityID)
		}
	}
	return q
}

func (r *analyticsRepo) CountEvents(ctx context.Context, eventType models.EventType, rng AnalyticsRange) (int64, error) {
	var n int64
	err := r.scopeEvents(ctx, rng).Where("user_property_events.event_type = ?", eventType).Count(&n).Error
	return n, err
}

func (r *analyticsRepo) CountViews(ctx context.Context, rng AnalyticsRange) (int64, error) {
	var n int64
	err := r.scopeViews(ctx, rng).Count(&n).Error
	return n, err
}

// CountUniqueVisitors counts distinct sessions, which covers anonymous traffic
// as well as logged-in users.
func (r *analyticsRepo) CountUniqueVisitors(ctx context.Context, rng AnalyticsRange) (int64, error) {
	var n int64
	err := r.scopeViews(ctx, rng).
		Distinct("COALESCE(property_views.session_id, CAST(property_views.user_id AS CHAR))").
		Count(&n).Error
	return n, err
}

func (r *analyticsRepo) ViewsSeries(ctx context.Context, rng AnalyticsRange) ([]TimeSeriesPoint, error) {
	var points []TimeSeriesPoint
	err := r.scopeViews(ctx, rng).
		Select("DATE(property_views.created_at) AS date, COUNT(*) AS count").
		Group("DATE(property_views.created_at)").
		Order("date ASC").Scan(&points).Error
	return points, err
}

func (r *analyticsRepo) EventSeries(ctx context.Context, eventType models.EventType, rng AnalyticsRange) ([]TimeSeriesPoint, error) {
	var points []TimeSeriesPoint
	err := r.scopeEvents(ctx, rng).
		Where("user_property_events.event_type = ?", eventType).
		Select("DATE(user_property_events.created_at) AS date, COUNT(*) AS count").
		Group("DATE(user_property_events.created_at)").
		Order("date ASC").Scan(&points).Error
	return points, err
}

func (r *analyticsRepo) InquiriesSeries(ctx context.Context, rng AnalyticsRange) ([]TimeSeriesPoint, error) {
	q := r.db.WithContext(ctx).Model(&models.Inquiry{}).
		Where("inquiries.created_at BETWEEN ? AND ?", rng.From, rng.To)
	if rng.ClientID > 0 {
		q = q.Where("inquiries.client_id = ?", rng.ClientID)
	}
	if rng.PropertyID > 0 {
		q = q.Where("inquiries.property_id = ?", rng.PropertyID)
	}
	var points []TimeSeriesPoint
	err := q.Select("DATE(inquiries.created_at) AS date, COUNT(*) AS count").
		Group("DATE(inquiries.created_at)").Order("date ASC").Scan(&points).Error
	return points, err
}

func (r *analyticsRepo) RegistrationsSeries(ctx context.Context, rng AnalyticsRange) ([]TimeSeriesPoint, error) {
	var points []TimeSeriesPoint
	err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at BETWEEN ? AND ?", rng.From, rng.To).
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Group("DATE(created_at)").Order("date ASC").Scan(&points).Error
	return points, err
}

// topOrderColumns whitelists the ranking column so the caller cannot inject SQL.
var topOrderColumns = map[string]string{
	"views":      "views",
	"inquiries":  "inquiries",
	"whatsapp":   "whats_app_clicks",
	"favorites":  "favorites",
	"conversion": "inquiries",
}

// TopProperties ranks listings by engagement inside the range. Counts come from
// correlated sub-queries so one property never multiplies another's totals.
func (r *analyticsRepo) TopProperties(ctx context.Context, rng AnalyticsRange, orderBy string, limit int) ([]PropertyStat, error) {
	col, ok := topOrderColumns[orderBy]
	if !ok {
		col = "views"
	}
	q := r.db.WithContext(ctx).Model(&models.Property{}).
		Select(`properties.id AS property_id, properties.name, properties.slug,
			(SELECT COUNT(*) FROM property_views pv
				WHERE pv.property_id = properties.id AND pv.created_at BETWEEN ? AND ?) AS views,
			(SELECT COUNT(DISTINCT COALESCE(pv.session_id, CAST(pv.user_id AS CHAR))) FROM property_views pv
				WHERE pv.property_id = properties.id AND pv.created_at BETWEEN ? AND ?) AS unique_visitors,
			(SELECT COUNT(*) FROM inquiries i
				WHERE i.property_id = properties.id AND i.deleted_at IS NULL
				AND i.created_at BETWEEN ? AND ?) AS inquiries,
			(SELECT COUNT(*) FROM user_property_events e
				WHERE e.property_id = properties.id AND e.event_type = ?
				AND e.created_at BETWEEN ? AND ?) AS whats_app_clicks,
			(SELECT COUNT(*) FROM user_property_events e
				WHERE e.property_id = properties.id AND e.event_type = ?
				AND e.created_at BETWEEN ? AND ?) AS phone_clicks,
			(SELECT COUNT(*) FROM user_property_events e
				WHERE e.property_id = properties.id AND e.event_type = ?
				AND e.created_at BETWEEN ? AND ?) AS brochure_downloads,
			properties.favorite_count AS favorites`,
			rng.From, rng.To,
			rng.From, rng.To,
			rng.From, rng.To,
			models.EventWhatsappClick, rng.From, rng.To,
			models.EventPhoneClick, rng.From, rng.To,
			models.EventBrochureDownload, rng.From, rng.To,
		)
	if rng.ClientID > 0 {
		q = q.Where("properties.client_id = ?", rng.ClientID)
	}
	if rng.CityID > 0 {
		q = q.Where("properties.city_id = ?", rng.CityID)
	}
	var stats []PropertyStat
	if err := q.Order(col + " DESC").Limit(limit).Scan(&stats).Error; err != nil {
		return nil, err
	}
	for i := range stats {
		stats[i].ConversionRate = conversionRate(stats[i].Inquiries, stats[i].Views)
	}
	return stats, nil
}

// PropertyStats returns lifetime engagement for a single listing.
func (r *analyticsRepo) PropertyStats(ctx context.Context, propertyID uint) (*PropertyStat, error) {
	stat := PropertyStat{PropertyID: propertyID}
	if err := r.db.WithContext(ctx).Model(&models.Property{}).
		Select("id AS property_id, name, slug, view_count AS views, inquiry_count AS inquiries, favorite_count AS favorites").
		Where("id = ?", propertyID).Scan(&stat).Error; err != nil {
		return nil, err
	}
	countEvent := func(t models.EventType) int64 {
		var n int64
		r.db.WithContext(ctx).Model(&models.UserPropertyEvent{}).
			Where("property_id = ? AND event_type = ?", propertyID, t).Count(&n)
		return n
	}
	stat.WhatsAppClicks = countEvent(models.EventWhatsappClick)
	stat.PhoneClicks = countEvent(models.EventPhoneClick)
	stat.BrochureDownloads = countEvent(models.EventBrochureDownload)

	var unique int64
	if err := r.db.WithContext(ctx).Model(&models.PropertyView{}).
		Where("property_id = ?", propertyID).
		Distinct("COALESCE(session_id, CAST(user_id AS CHAR))").Count(&unique).Error; err != nil {
		return nil, err
	}
	stat.UniqueVisitors = unique
	stat.ConversionRate = conversionRate(stat.Inquiries, stat.Views)
	return &stat, nil
}

func conversionRate(inquiries, views int64) float64 {
	if views == 0 {
		return 0
	}
	return float64(inquiries) / float64(views) * 100
}

// TopCities ranks cities by property views in the window.
func (r *analyticsRepo) TopCities(ctx context.Context, rng AnalyticsRange, limit int) ([]LabelCount, error) {
	var out []LabelCount
	err := r.db.WithContext(ctx).Model(&models.PropertyView{}).
		Select("cities.name AS label, cities.slug AS slug, COUNT(*) AS count").
		Joins("JOIN properties ON properties.id = property_views.property_id").
		Joins("JOIN cities ON cities.id = properties.city_id").
		Where("property_views.created_at BETWEEN ? AND ?", rng.From, rng.To).
		Group("cities.id, cities.name, cities.slug").
		Order("count DESC").Limit(limit).Scan(&out).Error
	return out, err
}

// TopSearchedLocalities ranks localities by views, standing in for search-term
// popularity until a dedicated search-log table is introduced.
func (r *analyticsRepo) TopSearchedLocalities(ctx context.Context, rng AnalyticsRange, limit int) ([]LabelCount, error) {
	var out []LabelCount
	err := r.db.WithContext(ctx).Model(&models.PropertyView{}).
		Select("localities.name AS label, localities.slug AS slug, COUNT(*) AS count").
		Joins("JOIN properties ON properties.id = property_views.property_id").
		Joins("JOIN localities ON localities.id = properties.locality_id").
		Where("property_views.created_at BETWEEN ? AND ?", rng.From, rng.To).
		Group("localities.id, localities.name, localities.slug").
		Order("count DESC").Limit(limit).Scan(&out).Error
	return out, err
}
