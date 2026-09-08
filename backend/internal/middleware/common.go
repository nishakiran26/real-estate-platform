package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// sessionCookie tracks anonymous visitors so property views, WhatsApp clicks and
// inquiries can be attributed without requiring a login.
const sessionCookie = "rera_sid"

// Session assigns a session ID cookie to every visitor and exposes it on the
// request context.
func Session(secure bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var sid string
			if ck, err := c.Cookie(sessionCookie); err == nil && ck.Value != "" {
				sid = ck.Value
			} else {
				sid = uuid.NewString()
				c.SetCookie(&http.Cookie{
					Name:     sessionCookie,
					Value:    sid,
					Path:     "/",
					Expires:  time.Now().Add(365 * 24 * time.Hour),
					HttpOnly: true,
					Secure:   secure,
					SameSite: http.SameSiteLaxMode,
				})
			}
			// An explicit header wins so the SPA can keep one session across tabs.
			if h := c.Request().Header.Get("X-Session-Id"); h != "" {
				sid = h
			}
			c.Set(ContextKeySessionID, sid)
			return next(c)
		}
	}
}

// RequestLogger emits one structured line per request.
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			if err != nil {
				c.Error(err)
			}
			req, res := c.Request(), c.Response()
			logger.Log.Info("request",
				"method", req.Method,
				"path", req.URL.Path,
				"status", res.Status,
				"duration_ms", time.Since(start).Milliseconds(),
				"ip", c.RealIP(),
				"user_id", UserID(c),
			)
			return nil
		}
	}
}

// Recover converts a panic into a 500 response and logs the stack.
func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Log.Error("panic recovered",
						"error", r, "path", c.Request().URL.Path)
					err = c.JSON(http.StatusInternalServerError,
						dto.NewErrorResponse("An unexpected error occurred", nil))
				}
			}()
			return next(c)
		}
	}
}

// SecurityHeaders sets conservative browser hardening headers on every response.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "SAMEORIGIN")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("X-XSS-Protection", "1; mode=block")
			return next(c)
		}
	}
}
