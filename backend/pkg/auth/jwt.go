package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID   uint     `json:"user_id"`
	Email    string   `json:"email"`
	Mobile   string   `json:"mobile"`
	FullName string   `json:"full_name"`
	Roles    []string `json:"roles"`
	ClientID *uint    `json:"client_id,omitempty"`
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type JWTService struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Issuer     string
}

func NewJWTService(secret string, accessTTLMin, refreshTTLDays int, issuer string) *JWTService {
	return &JWTService{
		Secret:     []byte(secret),
		AccessTTL:  time.Duration(accessTTLMin) * time.Minute,
		RefreshTTL: time.Duration(refreshTTLDays) * 24 * time.Hour,
		Issuer:     issuer,
	}
}

func (s *JWTService) GenerateTokenPair(userID uint, email, mobile, fullName string, roles []string, clientID *uint) (*TokenPair, error) {
	accessClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.AccessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.Issuer,
			Subject:   fmt.Sprintf("%d", userID),
		},
		UserID:   userID,
		Email:    email,
		Mobile:   mobile,
		FullName: fullName,
		Roles:    roles,
		ClientID: clientID,
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.Secret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.RefreshTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    s.Issuer,
		Subject:   fmt.Sprintf("%d", userID),
	})
	refreshStr, err := refreshToken.SignedString(s.Secret)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}
	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int(s.AccessTTL.Seconds()),
	}, nil
}

func (s *JWTService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (s *JWTService) ValidateRefreshToken(tokenStr string) (uint, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return s.Secret, nil
	})
	if err != nil {
		return 0, err
	}
	if !token.Valid {
		return 0, fmt.Errorf("invalid refresh token")
	}
	var uid uint
	if _, err := fmt.Sscanf(claims.Subject, "%d", &uid); err != nil {
		return 0, err
	}
	return uid, nil
}
