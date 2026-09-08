package repositories

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// VisitFilter scopes a visit listing. As with leads, ClientID is the ownership
// key: a builder or owner only ever sees visits on their own properties.
type VisitFilter struct {
	ClientID   uint
	UserID     uint
	PropertyID uint
	Status     string
	SessionID  string
	From       *time.Time
	To         *time.Time
	Offset     int
	Limit      int
}

// VisitRepository owns property viewing requests.
type VisitRepository interface {
	Create(ctx context.Context, v *models.PropertyVisit) error
	Get(ctx context.Context, id uint) (*models.PropertyVisit, error)
	List(ctx context.Context, f VisitFilter) ([]models.PropertyVisit, int64, error)
	Update(ctx context.Context, v *models.PropertyVisit) error
	Count(ctx context.Context, clientID uint, status string) (int64, error)
	// RecentForProperty backs the duplicate check: the same visitor asking twice
	// for the same flat within a short window is one request, not two.
	RecentForProperty(ctx context.Context, propertyID uint, mobile string, within time.Duration) (int64, error)
}

type visitRepo struct {
	db *gorm.DB
}

func NewVisitRepository(db *gorm.DB) VisitRepository { return &visitRepo{db: db} }

func (r *visitRepo) Create(ctx context.Context, v *models.PropertyVisit) error {
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *visitRepo) Get(ctx context.Context, id uint) (*models.PropertyVisit, error) {
	var v models.PropertyVisit
	err := r.db.WithContext(ctx).Preload("Property").First(&v, id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &v, nil
}

func (r *visitRepo) List(ctx context.Context, f VisitFilter) ([]models.PropertyVisit, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyVisit{})

	if f.ClientID > 0 {
		q = q.Where("client_id = ?", f.ClientID)
	}
	if f.UserID > 0 {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.PropertyID > 0 {
		q = q.Where("property_id = ?", f.PropertyID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	// A signed-out visitor is identified by session, so they can still see the
	// visits they asked for during this session.
	if f.SessionID != "" && f.UserID == 0 {
		q = q.Where("session_id = ?", f.SessionID)
	}
	if f.From != nil {
		q = q.Where("requested_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("requested_at <= ?", *f.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	var rows []models.PropertyVisit
	err := q.Preload("Property").
		Order("requested_at DESC, id DESC").
		Offset(f.Offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *visitRepo) Update(ctx context.Context, v *models.PropertyVisit) error {
	return r.db.WithContext(ctx).Omit("Property").Save(v).Error
}

func (r *visitRepo) Count(ctx context.Context, clientID uint, status string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyVisit{})
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *visitRepo) RecentForProperty(ctx context.Context, propertyID uint, mobile string, within time.Duration) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PropertyVisit{}).
		Where("property_id = ? AND mobile = ? AND created_at >= ?",
			propertyID, mobile, time.Now().Add(-within)).
		Count(&count).Error
	return count, err
}
