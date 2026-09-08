// Package repositories is the only layer that talks to the database. Every
// method takes a context and returns models or domain errors, never HTTP types.
package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// Domain errors shared across repositories.
var (
	ErrNotFound     = errors.New("record not found")
	ErrEmailExists  = errors.New("email already registered")
	ErrMobileExists = errors.New("mobile number already registered")
	ErrDuplicate    = errors.New("record already exists")
	ErrInUse        = errors.New("record is referenced by other data and cannot be deleted")
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User, roleCodes []string) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByMobile(ctx context.Context, mobile string) (*models.User, error)
	GetByEmailOrMobile(ctx context.Context, identifier string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateFields(ctx context.Context, id uint, fields map[string]any) error
	EmailExists(ctx context.Context, email string) (bool, error)
	MobileExists(ctx context.Context, mobile string) (bool, error)
	List(ctx context.Context, f UserFilter) ([]models.User, int64, error)
	Count(ctx context.Context, roleCode string) (int64, error)
	CountCreatedBetween(ctx context.Context, from, to time.Time) (int64, error)
	SetRoles(ctx context.Context, userID uint, roleIDs []uint) error
	RoleByCode(ctx context.Context, code string) (*models.Role, error)
	ListRoles(ctx context.Context) ([]models.Role, error)
	RecentRegistrations(ctx context.Context, limit int) ([]models.User, error)
}

// UserFilter drives admin user listings.
type UserFilter struct {
	Search   string
	RoleCode string
	Status   string
	Offset   int
	Limit    int
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository { return &userRepo{db: db} }

// Create inserts the user and its role assignments in one transaction so a
// half-registered account can never exist.
func (r *userRepo) Create(ctx context.Context, user *models.User, roleCodes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Where("email = ?", user.Email).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrEmailExists
		}
		if err := tx.Model(&models.User{}).Where("mobile = ?", user.Mobile).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrMobileExists
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if len(roleCodes) > 0 {
			var roles []models.Role
			if err := tx.Where("code IN ?", roleCodes).Find(&roles).Error; err != nil {
				return err
			}
			if len(roles) == 0 {
				return ErrNotFound
			}
			if err := tx.Model(user).Association("Roles").Replace(roles); err != nil {
				return err
			}
			user.Roles = roles
		}
		// Every user gets notification preferences so the settings page always
		// has a row to update.
		return tx.Create(&models.NotificationPreference{UserID: user.ID}).Error
	})
}

func (r *userRepo) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&u, id).Error
	return userOrErr(&u, err)
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Roles").Where("email = ?", email).First(&u).Error
	return userOrErr(&u, err)
}

func (r *userRepo) GetByMobile(ctx context.Context, mobile string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Roles").Where("mobile = ?", mobile).First(&u).Error
	return userOrErr(&u, err)
}

func (r *userRepo) GetByEmailOrMobile(ctx context.Context, identifier string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Roles").
		Where("email = ? OR mobile = ?", identifier, identifier).First(&u).Error
	return userOrErr(&u, err)
}

func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepo) UpdateFields(ctx context.Context, id uint, fields map[string]any) error {
	res := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&n).Error
	return n > 0, err
}

func (r *userRepo) MobileExists(ctx context.Context, mobile string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("mobile = ?", mobile).Count(&n).Error
	return n > 0, err
}

func (r *userRepo) List(ctx context.Context, f UserFilter) ([]models.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.User{})
	if f.RoleCode != "" {
		q = q.Joins("JOIN user_roles ur ON ur.user_id = users.id").
			Joins("JOIN roles ON roles.id = ur.role_id").
			Where("roles.code = ?", f.RoleCode)
	}
	if f.Status != "" {
		q = q.Where("users.status = ?", f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("users.full_name LIKE ? OR users.email LIKE ? OR users.mobile LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []models.User
	err := q.Preload("Roles").Order("users.created_at DESC").
		Offset(f.Offset).Limit(f.Limit).Find(&users).Error
	return users, total, err
}

func (r *userRepo) Count(ctx context.Context, roleCode string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.User{})
	if roleCode != "" {
		q = q.Joins("JOIN user_roles ur ON ur.user_id = users.id").
			Joins("JOIN roles ON roles.id = ur.role_id").
			Where("roles.code = ?", roleCode)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *userRepo) CountCreatedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at BETWEEN ? AND ?", from, to).Count(&n).Error
	return n, err
}

func (r *userRepo) SetRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return translate(err)
	}
	var roles []models.Role
	if err := r.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return err
	}
	if len(roles) != len(roleIDs) {
		return ErrNotFound
	}
	return r.db.WithContext(ctx).Model(&user).Association("Roles").Replace(roles)
}

func (r *userRepo) RoleByCode(ctx context.Context, code string) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error; err != nil {
		return nil, translate(err)
	}
	return &role, nil
}

func (r *userRepo) ListRoles(ctx context.Context) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.WithContext(ctx).Order("id").Find(&roles).Error
	return roles, err
}

func (r *userRepo) RecentRegistrations(ctx context.Context, limit int) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Preload("Roles").
		Order("created_at DESC").Limit(limit).Find(&users).Error
	return users, err
}

func userOrErr(u *models.User, err error) (*models.User, error) {
	if err != nil {
		return nil, translate(err)
	}
	return u, nil
}

// translate maps the ORM sentinel errors onto repository domain errors so the
// service layer never needs to import gorm.
func translate(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicate
	}
	return err
}
