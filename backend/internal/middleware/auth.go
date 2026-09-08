package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/pkg/auth"
)

// AuthMiddleware validates bearer tokens and enforces role membership.
type AuthMiddleware struct {
	jwt *auth.JWTService
}

func NewAuth(jwt *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwt: jwt}
}

// Require rejects any request without a valid access token.
func (m *AuthMiddleware) Require(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims, err := m.parse(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Authentication required", nil))
		}
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyUserID, claims.UserID)
		return next(c)
	}
}

// Optional attaches claims when a valid token is present but still allows
// anonymous access. Public property pages use it to personalise responses.
func (m *AuthMiddleware) Optional(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if claims, err := m.parse(c); err == nil {
			c.Set(ContextKeyClaims, claims)
			c.Set(ContextKeyUserID, claims.UserID)
		}
		return next(c)
	}
}

// RequireRole allows the request only if the caller holds one of the roles.
// It must be chained after Require.
func (m *AuthMiddleware) RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := Claims(c)
			if claims == nil {
				return c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Authentication required", nil))
			}
			for _, want := range roles {
				for _, got := range claims.Roles {
					if got == want {
						return next(c)
					}
				}
			}
			return c.JSON(http.StatusForbidden, dto.NewErrorResponse("You do not have permission to perform this action", nil))
		}
	}
}

func (m *AuthMiddleware) parse(c echo.Context) (*auth.Claims, error) {
	header := c.Request().Header.Get(echo.HeaderAuthorization)
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
	if token == "" || token == header {
		// Fall back to a cookie so brochure links and sitemap crawls work from
		// the browser address bar.
		cookie, err := c.Cookie("access_token")
		if err != nil || cookie.Value == "" {
			return nil, echo.ErrUnauthorized
		}
		token = cookie.Value
	}
	return m.jwt.ValidateAccessToken(token)
}
