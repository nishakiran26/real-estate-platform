// Package handlers holds the HTTP controllers. A handler binds and validates
// the request, delegates to a service and renders the response envelope; it
// never contains business logic and never touches the database.
package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/middleware"
	"github.com/rera/property-platform/backend/internal/services"
	"github.com/rera/property-platform/backend/internal/validators"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// bind reads the request body (or query string) into req and validates it,
// returning a ready-to-send error response when either step fails.
func bind(c echo.Context, req any) error {
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest,
			dto.NewErrorResponse("The request body could not be read", nil))
	}
	if err := c.Validate(req); err != nil {
		if fields := validators.Errors(err); len(fields) > 0 {
			return c.JSON(http.StatusUnprocessableEntity,
				dto.NewErrorResponse("Validation failed", fields))
		}
		return c.JSON(http.StatusUnprocessableEntity,
			dto.NewErrorResponse("Validation failed", nil))
	}
	return nil
}

// ok renders a 200 success envelope.
func ok(c echo.Context, message string, data any) error {
	return c.JSON(http.StatusOK, dto.NewSuccessResponse(message, data))
}

// created renders a 201 success envelope.
func created(c echo.Context, message string, data any) error {
	return c.JSON(http.StatusCreated, dto.NewSuccessResponse(message, data))
}

// paginated renders a list plus its pagination metadata.
func paginated(c echo.Context, message string, data any, meta *dto.Meta) error {
	return c.JSON(http.StatusOK, dto.NewPaginatedResponse(message, data, meta))
}

// fail turns a service error into the matching HTTP status and message.
// Internal errors are logged in full but reported generically to the client.
func fail(c echo.Context, err error) error {
	status := services.HTTPStatus(err)
	if status >= http.StatusInternalServerError {
		logger.Log.Error("request failed",
			"path", c.Request().URL.Path, "method", c.Request().Method, "error", err.Error())
	}
	return c.JSON(status, dto.NewErrorResponse(services.Message(err), nil))
}

// uintParam reads a positive integer path parameter.
func uintParam(c echo.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || v == 0 {
		return 0, services.Invalid("Invalid " + strings.ReplaceAll(name, "_", " "))
	}
	return uint(v), nil
}

// uintToString renders an ID for use in storage paths and messages.
func uintToString(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

// intQuery reads an integer query parameter, falling back to def.
func intQuery(c echo.Context, name string, def int) int {
	if raw := c.QueryParam(name); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			return n
		}
	}
	return def
}

// boolQuery reads a boolean query parameter, falling back to def.
func boolQuery(c echo.Context, name string, def bool) bool {
	if raw := c.QueryParam(name); raw != "" {
		if b, err := strconv.ParseBool(raw); err == nil {
			return b
		}
	}
	return def
}

// requestMeta captures the transport details services record on audit trails.
func requestMeta(c echo.Context) services.RequestMeta {
	return services.RequestMeta{
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}
}

// visitor identifies the caller for engagement tracking; it works for anonymous
// visitors, who are keyed by session cookie instead of user ID.
func visitor(c echo.Context) services.Visitor {
	return services.Visitor{
		UserID:    middleware.OptionalUserID(c),
		SessionID: middleware.SessionID(c),
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		Referrer:  c.Request().Referer(),
	}
}

// callerClientID resolves the builder account of the caller. Administrators
// return 0, which services read as "not scoped to one builder".
func callerClientID(c echo.Context) (uint, error) {
	if middleware.IsAdmin(c) {
		return 0, nil
	}
	id := middleware.ClientID(c)
	if id == nil || *id == 0 {
		return 0, services.Forbidden("Your account is not linked to a builder profile")
	}
	return *id, nil
}
