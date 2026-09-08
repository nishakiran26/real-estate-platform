// Package middleware holds the Echo middleware chain: authentication, RBAC,
// rate limiting, anonymous session tracking, logging and panic recovery.
package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/pkg/auth"
)

// Context keys shared between middleware and handlers.
const (
	ContextKeyClaims    = "auth_claims"
	ContextKeyUserID    = "auth_user_id"
	ContextKeySessionID = "session_id"
)

// Role codes recognised by the platform.
const (
	RoleAdmin    = "ADMIN"
	RoleClient   = "CLIENT"
	RoleCustomer = "CUSTOMER"
)

// Claims returns the authenticated user's claims, or nil for anonymous requests.
func Claims(c echo.Context) *auth.Claims {
	if v, ok := c.Get(ContextKeyClaims).(*auth.Claims); ok {
		return v
	}
	return nil
}

// UserID returns the authenticated user's ID, or 0 when anonymous.
func UserID(c echo.Context) uint {
	if cl := Claims(c); cl != nil {
		return cl.UserID
	}
	return 0
}

// OptionalUserID returns a pointer to the user ID, or nil when anonymous. Event
// and inquiry records use it to attribute activity without requiring login.
func OptionalUserID(c echo.Context) *uint {
	if id := UserID(c); id != 0 {
		return &id
	}
	return nil
}

// ClientID returns the builder account ID attached to the token, or nil.
func ClientID(c echo.Context) *uint {
	if cl := Claims(c); cl != nil {
		return cl.ClientID
	}
	return nil
}

// HasRole reports whether the caller holds the given role.
func HasRole(c echo.Context, role string) bool {
	cl := Claims(c)
	if cl == nil {
		return false
	}
	for _, r := range cl.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin is a shorthand used by services that widen access for administrators.
func IsAdmin(c echo.Context) bool { return HasRole(c, RoleAdmin) }

// SessionID returns the anonymous-visitor session identifier.
func SessionID(c echo.Context) string {
	if v, ok := c.Get(ContextKeySessionID).(string); ok {
		return v
	}
	return ""
}
