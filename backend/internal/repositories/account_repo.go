package repositories

import (
	"context"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AccountRepository owns the role-specific profiles that hang off a user:
// builder companies and customer preferences.
type AccountRepository interface {
	// Clients (builders)
	CreateClient(ctx context.Context, c *models.Client) error
	UpdateClient(ctx context.Context, c *models.Client) error
	GetClient(ctx context.Context, id uint) (*models.Client, error)
	GetClientByUserID(ctx context.Context, userID uint) (*models.Client, error)
	ListClients(ctx context.Context, search string, offset, limit int) ([]models.Client, int64, error)
	CountClients(ctx context.Context) (int64, error)
	RefreshClientPropertyCount(ctx context.Context, clientID uint) error

	// Customer profiles
	UpsertCustomerProfile(ctx context.Context, p *models.CustomerProfile) error
	GetCustomerProfile(ctx context.Context, userID uint) (*models.CustomerProfile, error)

	// Notification preferences
	GetNotificationPreference(ctx context.Context, userID uint) (*models.NotificationPreference, error)
	UpsertNotificationPreference(ctx context.Context, p *models.NotificationPreference) error

	// In-app notifications
	CreateNotification(ctx context.Context, n *models.Notification) error
	ListNotifications(ctx context.Context, userID uint, unreadOnly bool, offset, limit int) ([]models.Notification, int64, error)
	MarkNotificationRead(ctx context.Context, userID, id uint) error
	MarkAllNotificationsRead(ctx context.Context, userID uint) error
	UnreadNotificationCount(ctx context.Context, userID uint) (int64, error)

	// Audit log
	WriteAudit(ctx context.Context, entry *models.AuditLog) error
	ListAudit(ctx context.Context, offset, limit int) ([]models.AuditLog, int64, error)
}

type accountRepo struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository { return &accountRepo{db: db} }

func (r *accountRepo) CreateClient(ctx context.Context, c *models.Client) error {
	return translate(r.db.WithContext(ctx).Create(c).Error)
}

func (r *accountRepo) UpdateClient(ctx context.Context, c *models.Client) error {
	return r.db.WithContext(ctx).Omit("User", "Properties").Save(c).Error
}

func (r *accountRepo) GetClient(ctx context.Context, id uint) (*models.Client, error) {
	var c models.Client
	if err := r.db.WithContext(ctx).Preload("User").First(&c, id).Error; err != nil {
		return nil, translate(err)
	}
	return &c, nil
}

func (r *accountRepo) GetClientByUserID(ctx context.Context, userID uint) (*models.Client, error) {
	var c models.Client
	if err := r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, translate(err)
	}
	return &c, nil
}

func (r *accountRepo) ListClients(ctx context.Context, search string, offset, limit int) ([]models.Client, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Client{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("company_name LIKE ? OR contact_email LIKE ? OR contact_phone LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Client
	err := q.Preload("User").Order("created_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *accountRepo) CountClients(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Client{}).Count(&n).Error
	return n, err
}

// RefreshClientPropertyCount recomputes the denormalised listing count from the
// property table rather than trusting incremental updates.
func (r *accountRepo) RefreshClientPropertyCount(ctx context.Context, clientID uint) error {
	var n int64
	if err := r.db.WithContext(ctx).Model(&models.Property{}).
		Where("client_id = ?", clientID).Count(&n).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&models.Client{}).Where("id = ?", clientID).
		Update("total_properties", n).Error
}

// UpsertCustomerProfile creates or updates the buyer preference record keyed by
// user so repeated saves never duplicate a row.
func (r *accountRepo) UpsertCustomerProfile(ctx context.Context, p *models.CustomerProfile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"preferred_city", "preferred_locality", "min_budget", "max_budget",
			"property_type", "bhk", "purpose", "updated_at",
			// Rental intent and preferences are listed explicitly: an omitted
			// column here is silently never updated.
			"looking_to_buy", "looking_to_rent",
			"monthly_rent_min", "monthly_rent_max", "preferred_furnishing",
			"preferred_lease_months", "move_in_date", "family_type",
		}),
	}).Omit("User").Create(p).Error
}

func (r *accountRepo) GetCustomerProfile(ctx context.Context, userID uint) (*models.CustomerProfile, error) {
	var p models.CustomerProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error; err != nil {
		return nil, translate(err)
	}
	return &p, nil
}

func (r *accountRepo) GetNotificationPreference(ctx context.Context, userID uint) (*models.NotificationPreference, error) {
	var p models.NotificationPreference
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if err != nil {
		return nil, translate(err)
	}
	return &p, nil
}

func (r *accountRepo) UpsertNotificationPreference(ctx context.Context, p *models.NotificationPreference) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"email_enabled", "sms_enabled", "whatsapp_enabled", "in_app_enabled", "updated_at",
		}),
	}).Create(p).Error
}

func (r *accountRepo) CreateNotification(ctx context.Context, n *models.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *accountRepo) ListNotifications(ctx context.Context, userID uint, unreadOnly bool, offset, limit int) ([]models.Notification, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		q = q.Where("read_at IS NULL")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Notification
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *accountRepo) MarkNotificationRead(ctx context.Context, userID, id uint) error {
	res := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", id, userID).
		Update("read_at", gorm.Expr("NOW()"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *accountRepo) MarkAllNotificationsRead(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", gorm.Expr("NOW()")).Error
}

func (r *accountRepo) UnreadNotificationCount(ctx context.Context, userID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).Count(&n).Error
	return n, err
}

func (r *accountRepo) WriteAudit(ctx context.Context, entry *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *accountRepo) ListAudit(ctx context.Context, offset, limit int) ([]models.AuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.AuditLog{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.AuditLog
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}
