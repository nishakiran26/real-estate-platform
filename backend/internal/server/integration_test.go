package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	pngenc "image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/database"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/server"
	"github.com/rera/property-platform/backend/internal/storage"
	"github.com/rera/property-platform/backend/pkg/logger"
	"gorm.io/gorm"
)

// These tests exercise the real HTTP stack against a real MySQL schema, because
// the rules worth protecting (approval workflow, ownership, RBAC) only exist
// once the router, services and database are wired together.
//
// The suite creates and drops its own schema and is skipped when MySQL is not
// reachable, so `go test ./...` stays green on a machine without a database.

type testEnv struct {
	e   *echo.Echo
	db  *gorm.DB
	cfg *config.Config
}

func setup(t *testing.T) *testEnv {
	t.Helper()

	config.LoadDotEnv()
	logger.Init("test")

	cfg := config.Load()
	cfg.App.Env = "test"
	cfg.Database.Name = testSchema
	cfg.Database.AutoMigrate = true
	cfg.Database.Seed = false
	cfg.Seed.DemoData = false
	// Rate limits must not interfere with a test that logs in repeatedly.
	cfg.RateLimit.AuthPerMinute = 10000
	cfg.RateLimit.OTPPerHour = 10000
	cfg.RateLimit.PublicPerMinute = 10000
	cfg.Storage.LocalDir = t.TempDir()
	// A developer's .env may point at a real SMTP or SMS vendor. Tests must
	// never send to it: registration would depend on the network, take seconds
	// per account, and post real messages to real inboxes.
	cfg.Mail.Driver = "log"
	cfg.SMS.Driver = "log"
	cfg.WhatsApp.Driver = "log"

	if err := database.EnsureDatabase(cfg); err != nil {
		t.Skipf("MySQL is not reachable, skipping integration tests: %v", err)
	}

	db, err := database.NewMySQL(cfg.DSN())
	if err != nil {
		t.Skipf("MySQL is not reachable, skipping integration tests: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	truncateAll(t, db)

	// Roles must exist before anyone can register.
	if err := database.Seed(db, cfg); err != nil {
		t.Fatalf("seed reference data: %v", err)
	}

	store, err := storage.NewLocal(cfg.Storage.LocalDir, cfg.Storage.BaseURL)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}

	// A nil-address cache exercises the fail-soft path rather than requiring
	// Redis to be running for the suite.
	redis := cache.New("127.0.0.1:1", "", 0)

	// Each test opens its own pool of up to 25 connections. Left open, a suite
	// of this size exhausts MySQL's connection limit and later tests fail for
	// reasons that have nothing to do with what they assert.
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = redis.Close()
	})

	return &testEnv{e: server.New(cfg, db, redis, store), db: db, cfg: cfg}
}

// truncateAll gives each run a clean schema without dropping it, so the test
// database can be reused between runs.
func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"user_property_events", "property_views", "favorites", "inquiries",
		"property_status_histories", "rera_details", "property_documents",
		"property_media", "property_amenities", "property_configurations",
		// Rental marketplace tables: omitting one leaves rows behind that
		// collide with the fresh IDs the next test creates.
		"rental_details", "property_visits", "property_reports", "rental_alerts",
		"featured_properties", "properties", "notifications",
		"notification_preferences", "audit_logs", "customer_profiles", "clients",
		"otp_requests", "refresh_tokens", "login_histories", "user_roles", "users",
		"home_banners", "static_pages", "localities", "cities", "amenities", "roles",
	}
	// FOREIGN_KEY_CHECKS is a session variable, and every db.Exec may take a
	// different connection from the pool. Issued separately, the disable lands on
	// one connection and the TRUNCATEs run on others with checks still on, so
	// some tables are silently left populated and later tests fail on foreign
	// keys in whatever order the pool happened to hand out connections.
	//
	// Pinning the whole sequence to one connection is what makes truncation
	// deterministic.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		t.Fatalf("disable foreign key checks: %v", err)
	}
	for _, table := range tables {
		if _, err := conn.ExecContext(ctx, "TRUNCATE TABLE "+table); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		t.Fatalf("restore foreign key checks: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// --- request helpers --------------------------------------------------------

type response struct {
	Status  int
	Success bool                       `json:"success"`
	Message string                     `json:"message"`
	Data    map[string]any             `json:"-"`
	List    []any                      `json:"-"`
	Raw     map[string]json.RawMessage `json:"-"`
}

func (env *testEnv) do(t *testing.T, method, path, token string, body any) response {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	env.e.ServeHTTP(rec, req)

	var envelope struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &envelope)

	out := response{Status: rec.Code, Success: envelope.Success, Message: envelope.Message}
	if len(envelope.Data) > 0 {
		_ = json.Unmarshal(envelope.Data, &out.Data)
		_ = json.Unmarshal(envelope.Data, &out.List)
	}
	return out
}

// register creates an account and returns its access token.
func (env *testEnv) register(t *testing.T, role, email, mobile, name string) (string, map[string]any) {
	t.Helper()
	res := env.do(t, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"full_name":    name,
		"email":        email,
		"mobile":       mobile,
		"password":     "Password@123",
		"role_code":    role,
		"company_name": name + " Constructions",
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("register %s: status %d (%s)", email, res.Status, res.Message)
	}
	tokens, _ := res.Data["tokens"].(map[string]any)
	user, _ := res.Data["user"].(map[string]any)
	token, _ := tokens["access_token"].(string)
	if token == "" {
		t.Fatalf("register %s returned no access token", email)
	}
	return token, user
}

// promoteToAdmin grants the ADMIN role directly, the way an operator would
// provision the first administrator.
func (env *testEnv) promoteToAdmin(t *testing.T, email string) string {
	t.Helper()

	var user models.User
	if err := env.db.Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("find user: %v", err)
	}
	var role models.Role
	if err := env.db.Where("code = ?", "ADMIN").First(&role).Error; err != nil {
		t.Fatalf("find admin role: %v", err)
	}
	env.db.Exec("DELETE FROM user_roles WHERE user_id = ?", user.ID)
	if err := env.db.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	// Sign in again so the new role is inside the token.
	res := env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": email, "password": "Password@123",
	})
	if res.Status != http.StatusOK {
		t.Fatalf("admin login: status %d (%s)", res.Status, res.Message)
	}
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)
	return token
}

func (env *testEnv) seedLocation(t *testing.T) (uint, uint) {
	t.Helper()
	city := models.City{Name: "Pune", Slug: "pune", State: "Maharashtra", IsPopular: true}
	if err := env.db.Where("slug = ?", "pune").FirstOrCreate(&city).Error; err != nil {
		t.Fatalf("seed city: %v", err)
	}
	locality := models.Locality{CityID: city.ID, Name: "Baner", Slug: "baner"}
	if err := env.db.Where("city_id = ? AND slug = ?", city.ID, "baner").
		FirstOrCreate(&locality).Error; err != nil {
		t.Fatalf("seed locality: %v", err)
	}
	return city.ID, locality.ID
}

// uploadImage posts a real multipart image to the gallery. A listing cannot be
// submitted for review without one, and this also exercises the upload
// validation path (size, sniffed content type, storage key).
func (env *testEnv) uploadImage(t *testing.T, propertyID int, token string) {
	t.Helper()
	env.uploadMediaAs(t, propertyID, token, "PROPERTY_IMAGE")
}

// uploadMediaAs posts a real multipart image into a named gallery bucket.
func (env *testEnv) uploadMediaAs(t *testing.T, propertyID int, token, mediaType string) {
	t.Helper()

	var png bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	if err := pngenc.Encode(&png, img); err != nil {
		t.Fatalf("encode test image: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "cover.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(png.Bytes()); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	_ = writer.WriteField("media_type", mediaType)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/media", propertyID), &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)

	rec := httptest.NewRecorder()
	env.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload image: status %d (%s)", rec.Code, rec.Body.String())
	}
}

// publish takes a listing all the way from draft to published.
func (env *testEnv) publish(t *testing.T, propertyID int, builderToken, adminToken string) {
	t.Helper()
	env.uploadImage(t, propertyID, builderToken)

	submit := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), builderToken, nil)
	if submit.Status != http.StatusOK {
		t.Fatalf("submit: status %d (%s)", submit.Status, submit.Message)
	}

	approve := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/properties/%d/approve", propertyID), adminToken,
		map[string]any{"publish": true})
	if approve.Status != http.StatusOK {
		t.Fatalf("approve: status %d (%s)", approve.Status, approve.Message)
	}
}

func propertyPayload(cityID, localityID uint, name string) map[string]any {
	return map[string]any{
		"name":                name,
		"property_type":       "APARTMENT",
		"construction_status": "UNDER_CONSTRUCTION",
		"city_id":             cityID,
		"locality_id":         localityID,
		"address":             "Baner Road",
		"pincode":             "411045",
		"description":         "A RERA-registered project used by the integration tests.",
		"possession_date":     time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		"configurations": []map[string]any{
			{"name": "2 BHK", "carpet_area": 850, "bedrooms": 2, "bathrooms": 2, "price": 7500000},
			{"name": "3 BHK", "carpet_area": 1200, "bedrooms": 3, "bathrooms": 3, "price": 10500000},
		},
		"rera": map[string]any{
			"registration_number": "P52100000001",
			"authority":           "MahaRERA",
		},
	}
}

// --- tests ------------------------------------------------------------------

func TestRegistrationAndLogin(t *testing.T) {
	env := setup(t)

	token, user := env.register(t, "customer", "buyer1@example.com", "9800000001", "Test Buyer")
	if user["email"] != "buyer1@example.com" {
		t.Errorf("registered email = %v", user["email"])
	}

	// The session works immediately.
	me := env.do(t, http.MethodGet, "/api/v1/auth/me", token, nil)
	if me.Status != http.StatusOK {
		t.Fatalf("GET /auth/me: status %d (%s)", me.Status, me.Message)
	}

	// A duplicate email is rejected rather than creating a second account.
	dup := env.do(t, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"full_name": "Someone Else", "email": "buyer1@example.com", "mobile": "9800000002",
		"password": "Password@123", "role_code": "customer",
	})
	if dup.Status != http.StatusConflict && dup.Status != http.StatusUnprocessableEntity {
		t.Errorf("duplicate registration status = %d, want a client error", dup.Status)
	}

	// Wrong password is rejected.
	bad := env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": "buyer1@example.com", "password": "not-the-password",
	})
	if bad.Status != http.StatusUnauthorized {
		t.Errorf("bad password status = %d, want 401", bad.Status)
	}

	// Correct password succeeds.
	good := env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": "buyer1@example.com", "password": "Password@123",
	})
	if good.Status != http.StatusOK {
		t.Errorf("login status = %d (%s)", good.Status, good.Message)
	}
}

func TestOTPVerificationFlow(t *testing.T) {
	env := setup(t)
	token, _ := env.register(t, "customer", "otp@example.com", "9800000010", "OTP Buyer")

	send := env.do(t, http.MethodPost, "/api/v1/auth/send-otp", token, map[string]any{
		"mobile": "9800000010", "purpose": "MOBILE_VERIFY",
	})
	if send.Status != http.StatusOK {
		t.Fatalf("send-otp: status %d (%s)", send.Status, send.Message)
	}

	// A wrong code must not verify the account.
	wrong := env.do(t, http.MethodPost, "/api/v1/auth/verify-otp", token, map[string]any{
		"mobile": "9800000010", "code": "000000", "purpose": "MOBILE_VERIFY",
	})
	if wrong.Status == http.StatusOK {
		t.Error("an incorrect OTP was accepted")
	}

	var user models.User
	if err := env.db.Where("email = ?", "otp@example.com").First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.MobileVerified {
		t.Fatal("the mobile was marked verified before a correct code was supplied")
	}
}

func TestPropertyLifecycleAndAuthorization(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	builderToken, _ := env.register(t, "client", "builder1@example.com", "9800000021", "Alpha")
	rivalToken, _ := env.register(t, "client", "builder2@example.com", "9800000022", "Beta")
	customerToken, _ := env.register(t, "customer", "buyer2@example.com", "9800000023", "Buyer")
	_, _ = env.register(t, "customer", "admin@example.com", "9800000024", "Admin")
	adminToken := env.promoteToAdmin(t, "admin@example.com")

	// A builder creates a draft.
	created := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Alpha Residency"))
	if created.Status != http.StatusCreated {
		t.Fatalf("create property: status %d (%s)", created.Status, created.Message)
	}
	propertyID := int(created.Data["id"].(float64))
	if created.Data["property_status"] != "DRAFT" {
		t.Errorf("new property status = %v, want DRAFT", created.Data["property_status"])
	}

	// A draft is not publicly visible.
	publicList := env.do(t, http.MethodGet, "/api/v1/properties", "", nil)
	if len(publicList.List) != 0 {
		t.Errorf("a draft property appeared in the public listing: %d result(s)", len(publicList.List))
	}

	// A rival builder can neither read nor edit it.
	foreignRead := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), rivalToken, nil)
	if foreignRead.Status != http.StatusNotFound && foreignRead.Status != http.StatusForbidden {
		t.Errorf("a rival builder read another builder's property: status %d", foreignRead.Status)
	}
	foreignEdit := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), rivalToken,
		map[string]any{"name": "Hijacked Project"})
	if foreignEdit.Status != http.StatusNotFound && foreignEdit.Status != http.StatusForbidden {
		t.Errorf("a rival builder edited another builder's property: status %d", foreignEdit.Status)
	}

	// A customer cannot use the builder portal at all.
	customerAttempt := env.do(t, http.MethodPost, "/api/v1/client/properties", customerToken,
		propertyPayload(cityID, localityID, "Customer Project"))
	if customerAttempt.Status != http.StatusForbidden {
		t.Errorf("a customer reached the builder portal: status %d", customerAttempt.Status)
	}

	// Only an administrator may approve.
	selfApprove := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/properties/%d/approve", propertyID), builderToken,
		map[string]any{"publish": true})
	if selfApprove.Status != http.StatusForbidden {
		t.Errorf("a builder approved their own listing: status %d", selfApprove.Status)
	}

	// Submit, then approve and publish. A listing needs an image first.
	env.uploadImage(t, propertyID, builderToken)
	submit := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), builderToken, nil)
	if submit.Status != http.StatusOK {
		t.Fatalf("submit: status %d (%s)", submit.Status, submit.Message)
	}
	if submit.Data["property_status"] != "SUBMITTED" {
		t.Errorf("status after submit = %v, want SUBMITTED", submit.Data["property_status"])
	}

	approve := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/properties/%d/approve", propertyID), adminToken,
		map[string]any{"publish": true, "remarks": "Looks good"})
	if approve.Status != http.StatusOK {
		t.Fatalf("approve: status %d (%s)", approve.Status, approve.Message)
	}

	// Now it is publicly visible.
	publicAfter := env.do(t, http.MethodGet, "/api/v1/properties", "", nil)
	if len(publicAfter.List) != 1 {
		t.Fatalf("published property count = %d, want 1", len(publicAfter.List))
	}

	// The status history records the transitions.
	var transitions int64
	env.db.Model(&models.PropertyStatusHistory{}).
		Where("property_id = ?", propertyID).Count(&transitions)
	if transitions < 2 {
		t.Errorf("status history rows = %d, want at least 2", transitions)
	}
}

func TestRejectionAndResubmission(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	builderToken, _ := env.register(t, "client", "builder3@example.com", "9800000031", "Gamma")
	_, _ = env.register(t, "customer", "admin2@example.com", "9800000032", "Admin")
	adminToken := env.promoteToAdmin(t, "admin2@example.com")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Gamma Heights"))
	propertyID := int(created.Data["id"].(float64))

	env.uploadImage(t, propertyID, builderToken)
	env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID),
		builderToken, nil)

	reject := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/properties/%d/reject", propertyID), adminToken,
		map[string]any{"reason": "The RERA number does not match the project name."})
	if reject.Status != http.StatusOK {
		t.Fatalf("reject: status %d (%s)", reject.Status, reject.Message)
	}

	// The builder sees the reason and can resubmit after correcting it.
	detail := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), builderToken, nil)
	if detail.Data["property_status"] != "REJECTED" {
		t.Errorf("status = %v, want REJECTED", detail.Data["property_status"])
	}
	if detail.Data["rejection_reason"] == "" || detail.Data["rejection_reason"] == nil {
		t.Error("the rejection reason was not returned to the builder")
	}

	resubmit := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), builderToken, nil)
	if resubmit.Status != http.StatusOK {
		t.Fatalf("resubmit: status %d (%s)", resubmit.Status, resubmit.Message)
	}
	if resubmit.Data["property_status"] != "SUBMITTED" {
		t.Errorf("status after resubmit = %v, want SUBMITTED", resubmit.Data["property_status"])
	}

	// A rejected listing is never public.
	publicList := env.do(t, http.MethodGet, "/api/v1/properties", "", nil)
	if len(publicList.List) != 0 {
		t.Errorf("a non-approved property was publicly visible: %d result(s)", len(publicList.List))
	}
}

func TestInquiryCreationAndLeadIsolation(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	builderToken, _ := env.register(t, "client", "builder4@example.com", "9800000041", "Delta")
	rivalToken, _ := env.register(t, "client", "builder5@example.com", "9800000042", "Epsilon")
	_, _ = env.register(t, "customer", "admin3@example.com", "9800000043", "Admin")
	adminToken := env.promoteToAdmin(t, "admin3@example.com")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Delta Enclave"))
	propertyID := int(created.Data["id"].(float64))
	env.publish(t, propertyID, builderToken, adminToken)

	// An anonymous visitor raises a lead, with explicit consent.
	inquiry := env.do(t, http.MethodPost, "/api/v1/inquiries", "", map[string]any{
		"property_id":     propertyID,
		"name":            "Anonymous Buyer",
		"mobile":          "9812345678",
		"email":           "anon@example.com",
		"message":         "Please share the price list.",
		"consent_contact": true,
	})
	if inquiry.Status != http.StatusCreated {
		t.Fatalf("create inquiry: status %d (%s)", inquiry.Status, inquiry.Message)
	}

	// Consent is mandatory: without it the lead is refused.
	noConsent := env.do(t, http.MethodPost, "/api/v1/inquiries", "", map[string]any{
		"property_id": propertyID, "name": "No Consent", "mobile": "9812345679",
		"consent_contact": false,
	})
	if noConsent.Status == http.StatusCreated {
		t.Error("a lead was stored without contact consent")
	}

	// The owning builder sees the lead.
	ownLeads := env.do(t, http.MethodGet, "/api/v1/client/inquiries", builderToken, nil)
	if len(ownLeads.List) != 1 {
		t.Errorf("owning builder sees %d lead(s), want 1", len(ownLeads.List))
	}

	// A rival builder sees none of it.
	rivalLeads := env.do(t, http.MethodGet, "/api/v1/client/inquiries", rivalToken, nil)
	if len(rivalLeads.List) != 0 {
		t.Errorf("a rival builder saw %d lead(s) that were not theirs", len(rivalLeads.List))
	}

	// The administrator sees every lead.
	adminLeads := env.do(t, http.MethodGet, "/api/v1/admin/inquiries", adminToken, nil)
	if len(adminLeads.List) != 1 {
		t.Errorf("administrator sees %d lead(s), want 1", len(adminLeads.List))
	}
}

func TestPublicEndpointsDoNotRequireAuthentication(t *testing.T) {
	env := setup(t)
	env.seedLocation(t)

	for _, path := range []string{
		"/api/v1/properties",
		"/api/v1/cities",
		"/api/v1/amenities",
		"/api/v1/homepage",
		"/api/v1/meta",
		"/health",
	} {
		res := env.do(t, http.MethodGet, path, "", nil)
		if res.Status != http.StatusOK {
			t.Errorf("GET %s without a token: status %d (%s)", path, res.Status, res.Message)
		}
	}
}

func TestCustomerAreaRequiresAuthentication(t *testing.T) {
	env := setup(t)

	for _, path := range []string{
		"/api/v1/customer/dashboard",
		"/api/v1/customer/favorites",
		"/api/v1/client/properties",
		"/api/v1/admin/dashboard",
	} {
		res := env.do(t, http.MethodGet, path, "", nil)
		if res.Status != http.StatusUnauthorized {
			t.Errorf("GET %s anonymously: status %d, want 401", path, res.Status)
		}
	}
}
