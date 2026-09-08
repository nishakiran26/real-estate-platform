package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/pkg/auth"
)

// newTestAuth returns the middleware plus a helper that mints a token for a set
// of roles, so each case reads as "caller with role X hits route Y".
func newTestAuth() (*AuthMiddleware, func(roles ...string) string) {
	jwt := auth.NewJWTService("test-secret-that-is-long-enough-32", 15, 7, "rera-test")
	mint := func(roles ...string) string {
		pair, err := jwt.GenerateTokenPair(1, "user@example.com", "9000000000", "Test User", roles, nil)
		if err != nil {
			panic(err)
		}
		return pair.AccessToken
	}
	return NewAuth(jwt), mint
}

func okHandler(c echo.Context) error { return c.NoContent(http.StatusOK) }

func TestRequireRejectsAnonymousRequests(t *testing.T) {
	mw, _ := newTestAuth()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/dashboard", nil)
	rec := httptest.NewRecorder()

	if err := mw.Require(okHandler)(e.NewContext(req, rec)); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRequireRejectsAGarbageToken(t *testing.T) {
	mw, _ := newTestAuth()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/dashboard", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer not-a-real-token")
	rec := httptest.NewRecorder()

	_ = mw.Require(okHandler)(e.NewContext(req, rec))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRequireAcceptsAValidToken(t *testing.T) {
	mw, mint := newTestAuth()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/dashboard", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+mint("CUSTOMER"))
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	if err := mw.Require(okHandler)(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if UserID(c) != 1 {
		t.Errorf("UserID = %d, want 1", UserID(c))
	}
}

// TestRequireRole covers the permission matrix: a caller reaches a route only
// when they hold one of the roles it names.
func TestRequireRole(t *testing.T) {
	cases := []struct {
		name       string
		callerRole string
		allowed    []string
		wantStatus int
	}{
		{"admin reaches an admin route", RoleAdmin, []string{RoleAdmin}, http.StatusOK},
		{"customer is blocked from an admin route", RoleCustomer, []string{RoleAdmin}, http.StatusForbidden},
		{"builder is blocked from an admin route", RoleClient, []string{RoleAdmin}, http.StatusForbidden},
		{"builder reaches the builder portal", RoleClient, []string{RoleClient, RoleAdmin}, http.StatusOK},
		{"admin reaches the builder portal", RoleAdmin, []string{RoleClient, RoleAdmin}, http.StatusOK},
		{"customer is blocked from the builder portal", RoleCustomer, []string{RoleClient, RoleAdmin}, http.StatusForbidden},
		{"customer reaches their own account", RoleCustomer, []string{RoleCustomer, RoleAdmin}, http.StatusOK},
		{"builder is blocked from the customer account", RoleClient, []string{RoleCustomer, RoleAdmin}, http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mw, mint := newTestAuth()
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+mint(tc.callerRole))
			rec := httptest.NewRecorder()

			handler := mw.Require(mw.RequireRole(tc.allowed...)(okHandler))
			if err := handler(e.NewContext(req, rec)); err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestRequireRoleWithoutAuthenticationIsUnauthorized(t *testing.T) {
	mw, _ := newTestAuth()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	rec := httptest.NewRecorder()

	// RequireRole on its own (no Require ahead of it) must not fall open.
	_ = mw.RequireRole(RoleAdmin)(okHandler)(e.NewContext(req, rec))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestOptionalAllowsAnonymousButAttachesClaims(t *testing.T) {
	mw, mint := newTestAuth()
	e := echo.New()

	// Anonymous: allowed through with no identity.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/properties", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = mw.Optional(okHandler)(c)

	if rec.Code != http.StatusOK {
		t.Errorf("anonymous status = %d, want 200", rec.Code)
	}
	if UserID(c) != 0 {
		t.Error("an anonymous request was given a user id")
	}
	if OptionalUserID(c) != nil {
		t.Error("OptionalUserID should be nil for an anonymous request")
	}

	// Authenticated: same route, now personalised.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/properties", nil)
	req2.Header.Set(echo.HeaderAuthorization, "Bearer "+mint("CUSTOMER"))
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	_ = mw.Optional(okHandler)(c2)

	if UserID(c2) != 1 {
		t.Errorf("UserID = %d, want 1", UserID(c2))
	}
	if !HasRole(c2, RoleCustomer) {
		t.Error("HasRole did not see the CUSTOMER role")
	}
	if IsAdmin(c2) {
		t.Error("a customer was reported as an administrator")
	}
}
