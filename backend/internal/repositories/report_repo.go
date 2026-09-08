package repositories

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// ReportFilter scopes the moderation queue. Unlike leads and visits there is no
// ownership key: a report is about a listing, and only administrators read it.
// Showing an owner the complaints against their own flat would tell them who
// reported it.
type ReportFilter struct {
	PropertyID uint
	Status     string
	Reason     string
	Offset     int
	Limit      int
}

// ReportRepository owns abuse reports on listings.
type ReportRepository interface {
	Create(ctx context.Context, r *models.PropertyReport) error
	Get(ctx context.Context, id uint) (*models.PropertyReport, error)
	List(ctx context.Context, f ReportFilter) ([]models.PropertyReport, int64, error)
	Update(ctx context.Context, r *models.PropertyReport) error
	Count(ctx context.Context, status string) (int64, error)
	// CountForProperty backs the badge on the moderation queue: a listing with
	// several open complaints deserves to be looked at before one with a single
	// complaint.
	CountForProperty(ctx context.Context, propertyID uint, status string) (int64, error)
	// RecentFromReporter backs the duplicate check. The same visitor filing the
	// same complaint twice is one report, not two, and a refresh must not
	// inflate the count that drives moderation priority.
	RecentFromReporter(ctx context.Context, propertyID uint, userID *uint, sessionID string, within time.Duration) (int64, error)
}

type reportRepo struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository { return &reportRepo{db: db} }

func (r *reportRepo) Create(ctx context.Context, report *models.PropertyReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

func (r *reportRepo) Get(ctx context.Context, id uint) (*models.PropertyReport, error) {
	var report models.PropertyReport
	if err := r.db.WithContext(ctx).Preload("Property").First(&report, id).Error; err != nil {
		return nil, translate(err)
	}
	return &report, nil
}

func (r *reportRepo) List(ctx context.Context, f ReportFilter) ([]models.PropertyReport, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyReport{})
	if f.PropertyID > 0 {
		q = q.Where("property_id = ?", f.PropertyID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Reason != "" {
		q = q.Where("reason = ?", f.Reason)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	var rows []models.PropertyReport
	// Oldest first: a complaint that has been waiting longest is the one the
	// queue should surface, which is the opposite of every other listing here.
	err := q.Preload("Property").
		Order("created_at ASC, id ASC").
		Offset(f.Offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *reportRepo) Update(ctx context.Context, report *models.PropertyReport) error {
	return r.db.WithContext(ctx).Omit("Property").Save(report).Error
}

func (r *reportRepo) Count(ctx context.Context, status string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyReport{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *reportRepo) CountForProperty(ctx context.Context, propertyID uint, status string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyReport{}).Where("property_id = ?", propertyID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *reportRepo) RecentFromReporter(ctx context.Context, propertyID uint, userID *uint, sessionID string, within time.Duration) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyReport{}).
		Where("property_id = ? AND created_at >= ?", propertyID, time.Now().Add(-within))
	switch {
	case userID != nil && *userID > 0:
		q = q.Where("reporter_id = ?", *userID)
	case sessionID != "":
		q = q.Where("session_id = ?", sessionID)
	default:
		// Nothing identifies the reporter, so nothing can be called a duplicate.
		return 0, nil
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}
