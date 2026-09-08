package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/middleware"
	"github.com/rera/property-platform/backend/internal/services"
)

// AuthHandler exposes registration, login, OTP and token lifecycle endpoints.
type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register creates a customer or builder account and returns a session.
func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	res, err := h.auth.Register(c.Request().Context(), req, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return created(c, "Registration successful. Please verify your contact details.", res)
}

// Login authenticates with email or mobile plus password.
func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	res, err := h.auth.Login(c.Request().Context(), req, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Login successful", res)
}

// LoginWithOTP completes a passwordless mobile login.
func (h *AuthHandler) LoginWithOTP(c echo.Context) error {
	var req dto.MobileOTPLoginRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	res, err := h.auth.LoginWithOTP(c.Request().Context(), req.Mobile, req.Code, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Login successful", res)
}

// SendOTP dispatches a verification code over email or SMS.
func (h *AuthHandler) SendOTP(c echo.Context) error {
	var req dto.SendOTPRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	res, err := h.auth.SendOTP(c.Request().Context(), req, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Verification code sent", res)
}

// VerifyOTP confirms a code and, for password resets, returns a reset token.
func (h *AuthHandler) VerifyOTP(c echo.Context) error {
	var req dto.VerifyOTPRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	res, err := h.auth.VerifyOTP(c.Request().Context(), req, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, res.Message, res)
}

// ForgotPassword starts a reset by sending a code to the registered contact.
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req dto.ForgotPasswordRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.auth.ForgotPassword(c.Request().Context(), req, requestMeta(c)); err != nil {
		return fail(c, err)
	}
	// The response is deliberately identical whether or not the account exists.
	return ok(c, "If the account exists, a reset code has been sent", nil)
}

// ResetPassword sets a new password using the token from OTP verification.
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req dto.ResetPasswordRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.auth.ResetPassword(c.Request().Context(), req); err != nil {
		return fail(c, err)
	}
	return ok(c, "Password reset successfully. Please log in.", nil)
}

// ChangePassword updates the password of the signed-in user.
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	var req dto.ChangePasswordRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.auth.ChangePassword(c.Request().Context(), middleware.UserID(c), req); err != nil {
		return fail(c, err)
	}
	return ok(c, "Password changed successfully. Please log in again.", nil)
}

// Refresh rotates a refresh token and issues a new token pair.
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req dto.RefreshTokenRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	tokens, err := h.auth.Refresh(c.Request().Context(), req.RefreshToken, requestMeta(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Token refreshed", tokens)
}

// Logout revokes the current refresh token, or every token when all_devices is set.
func (h *AuthHandler) Logout(c echo.Context) error {
	var req dto.LogoutRequest
	if err := c.Bind(&req); err != nil {
		return fail(c, services.Invalid("The request body could not be read"))
	}
	if err := h.auth.Logout(c.Request().Context(), middleware.UserID(c), req.RefreshToken, req.AllDevices); err != nil {
		return fail(c, err)
	}
	return ok(c, "Logged out successfully", nil)
}

// Me returns the signed-in user profile.
func (h *AuthHandler) Me(c echo.Context) error {
	user, err := h.auth.Me(c.Request().Context(), middleware.UserID(c))
	if err != nil {
		return fail(c, err)
	}
	return ok(c, "Profile fetched successfully", user)
}

// LoginHistory lists recent sign-in activity for the current account.
func (h *AuthHandler) LoginHistory(c echo.Context) error {
	entries, err := h.auth.LoginHistory(c.Request().Context(), middleware.UserID(c), intQuery(c, "limit", 20))
	if err != nil {
		return fail(c, err)
	}
	out := make([]dto.LoginHistoryResponse, 0, len(entries))
	for i := range entries {
		out = append(out, services.ToLoginHistoryResponse(&entries[i]))
	}
	return ok(c, "Login history fetched successfully", out)
}
