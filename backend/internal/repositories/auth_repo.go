package repositories

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// AuthRepository persists everything that supports a login session: refresh
// tokens, OTP requests and login history.
type AuthRepository interface {
	// Refresh tokens
	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uint) error
	RevokeTokenFamily(ctx context.Context, familyID string) error
	RevokeAllUserTokens(ctx context.Context, userID uint) error
	PurgeExpiredTokens(ctx context.Context) (int64, error)

	// OTP
	CreateOTP(ctx context.Context, otp *models.OTPRequest) error
	LatestOTP(ctx context.Context, identifier string, purpose models.OTPPurpose) (*models.OTPRequest, error)
	GetOTPByID(ctx context.Context, id uint) (*models.OTPRequest, error)
	IncrementOTPAttempts(ctx context.Context, id uint) error
	MarkOTPVerified(ctx context.Context, id uint) error
	CountRecentOTPs(ctx context.Context, identifier string, since time.Time) (int64, error)

	// Login history
	RecordLogin(ctx context.Context, entry *models.LoginHistory) error
	RecordLogout(ctx context.Context, userID uint) error
	ListLoginHistory(ctx context.Context, userID uint, limit int) ([]models.LoginHistory, error)
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository { return &authRepo{db: db} }

func (r *authRepo) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *authRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var t models.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&t).Error; err != nil {
		return nil, translate(err)
	}
	return &t, nil
}

func (r *authRepo) RevokeRefreshToken(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

// RevokeTokenFamily invalidates every token descended from one login. It is the
// response to detecting a replayed refresh token.
func (r *authRepo) RevokeTokenFamily(ctx context.Context, familyID string) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now()).Error
}

func (r *authRepo) RevokeAllUserTokens(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

func (r *authRepo) PurgeExpiredTokens(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Unscoped().
		Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{})
	return res.RowsAffected, res.Error
}

func (r *authRepo) CreateOTP(ctx context.Context, otp *models.OTPRequest) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

// LatestOTP returns the most recent unverified, unexpired OTP for an email or
// mobile identifier and purpose.
func (r *authRepo) LatestOTP(ctx context.Context, identifier string, purpose models.OTPPurpose) (*models.OTPRequest, error) {
	var otp models.OTPRequest
	err := r.db.WithContext(ctx).
		Where("(email = ? OR mobile = ?) AND purpose = ? AND verified_at IS NULL AND expires_at > ?",
			identifier, identifier, purpose, time.Now()).
		Order("created_at DESC").First(&otp).Error
	if err != nil {
		return nil, translate(err)
	}
	return &otp, nil
}

func (r *authRepo) GetOTPByID(ctx context.Context, id uint) (*models.OTPRequest, error) {
	var otp models.OTPRequest
	if err := r.db.WithContext(ctx).First(&otp, id).Error; err != nil {
		return nil, translate(err)
	}
	return &otp, nil
}

func (r *authRepo) IncrementOTPAttempts(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.OTPRequest{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *authRepo) MarkOTPVerified(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.OTPRequest{}).
		Where("id = ?", id).Update("verified_at", time.Now()).Error
}

// CountRecentOTPs backs the per-identifier send throttle, which stays enforced
// even when Redis is unavailable.
func (r *authRepo) CountRecentOTPs(ctx context.Context, identifier string, since time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.OTPRequest{}).
		Where("(email = ? OR mobile = ?) AND created_at > ?", identifier, identifier, since).
		Count(&n).Error
	return n, err
}

func (r *authRepo) RecordLogin(ctx context.Context, entry *models.LoginHistory) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

// RecordLogout stamps the most recent open session for the user.
func (r *authRepo) RecordLogout(ctx context.Context, userID uint) error {
	var entry models.LoginHistory
	err := r.db.WithContext(ctx).Where("user_id = ? AND logout_at IS NULL", userID).
		Order("login_at DESC").First(&entry).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // nothing open to close
		}
		return err
	}
	now := time.Now()
	entry.LogoutAt = &now
	return r.db.WithContext(ctx).Save(&entry).Error
}

func (r *authRepo) ListLoginHistory(ctx context.Context, userID uint, limit int) ([]models.LoginHistory, error) {
	var list []models.LoginHistory
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("login_at DESC").Limit(limit).Find(&list).Error
	return list, err
}
