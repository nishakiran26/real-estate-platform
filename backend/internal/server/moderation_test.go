package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

func reportPayload(reason string) map[string]any {
	return map[string]any{
		"reason":  reason,
		"details": "I called the number and the flat was let three weeks ago.",
	}
}

// The people best placed to notice a wrong listing are the ones searching, and
// they will not create an account to say so.
func TestAnonymousVisitorCanReportAListing(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "50")

	res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/report", propertyID), "",
		reportPayload("ALREADY_RENTED"))
	if res.Status != http.StatusCreated {
		t.Fatalf("report: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["status"] != "OPEN" {
		t.Errorf("status = %v, want OPEN", res.Data["status"])
	}
	if res.Data["reason"] != "ALREADY_RENTED" {
		t.Errorf("reason = %v, want ALREADY_RENTED", res.Data["reason"])
	}

	var stored int64
	env.db.Model(&models.PropertyReport{}).Where("property_id = ?", propertyID).Count(&stored)
	if stored != 1 {
		t.Errorf("reports stored = %d, want 1", stored)
	}
}

// A draft is not public, so reporting one must not confirm it exists.
func TestUnpublishedListingsCannotBeReported(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	token := env.registerOwner(t, "draftowner@example.com", "9700005100", "Draft Owner")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", token,
		rentalPayload(cityID, localityID, "Unlisted Flat", 21000))
	if created.Status != http.StatusCreated {
		t.Fatalf("create draft: status %d (%s)", created.Status, created.Message)
	}
	draftID := int(created.Data["id"].(float64))

	res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/report", draftID), "",
		reportPayload("FAKE_LISTING"))
	if res.Status != http.StatusNotFound {
		t.Errorf("status %d (%s), want 404", res.Status, res.Message)
	}
}

// A refresh must not inflate the count that drives moderation priority.
func TestRepeatReportFromTheSameVisitorIsRejected(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "51")
	reporter, _ := env.register(t, "customer", "reporter51@example.com", "9700005110", "Reporter")

	first := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/report", propertyID), reporter,
		reportPayload("INCORRECT_PRICE"))
	if first.Status != http.StatusCreated {
		t.Fatalf("first report: status %d (%s)", first.Status, first.Message)
	}

	second := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/report", propertyID), reporter,
		reportPayload("INCORRECT_PRICE"))
	if second.Status != http.StatusConflict {
		t.Errorf("second report: status %d (%s), want 409", second.Status, second.Message)
	}
}

// The moderation queue is administrators only: showing an owner the complaints
// against their own flat would tell them who reported it.
func TestModerationQueueIsAdminOnly(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "52")
	env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/properties/%d/report", propertyID), "",
		reportPayload("FRAUD"))

	if res := env.do(t, http.MethodGet, "/api/v1/admin/reports", "", nil); res.Status != http.StatusUnauthorized {
		t.Errorf("anonymous: status %d, want 401", res.Status)
	}
	if res := env.do(t, http.MethodGet, "/api/v1/admin/reports", ownerToken, nil); res.Status != http.StatusForbidden {
		t.Errorf("owner: status %d, want 403", res.Status)
	}

	adminToken := env.promoteToAdmin(t, "visitadmin52@example.com")
	queue := env.do(t, http.MethodGet, "/api/v1/admin/reports", adminToken, nil)
	if queue.Status != http.StatusOK {
		t.Fatalf("admin queue: status %d (%s)", queue.Status, queue.Message)
	}
	if len(queue.List) != 1 {
		t.Fatalf("queue length = %d, want 1", len(queue.List))
	}
}

// An upheld complaint takes the listing down and tells the owner why, rather
// than the listing silently vanishing.
func TestUpheldReportUnpublishesAndExplains(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "53")
	env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/properties/%d/report", propertyID), "",
		reportPayload("ALREADY_RENTED"))
	adminToken := env.promoteToAdmin(t, "visitadmin53@example.com")

	queue := env.do(t, http.MethodGet, "/api/v1/admin/reports", adminToken, nil)
	row := queue.List[0].(map[string]any)
	reportID := int(row["id"].(float64))

	res := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/reports/%d", reportID), adminToken,
		map[string]any{
			"status":      "ACTIONED",
			"action_note": "The owner confirmed the flat is let.",
			"unpublish":   true,
		})
	if res.Status != http.StatusOK {
		t.Fatalf("resolve: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["status"] != "ACTIONED" {
		t.Errorf("status = %v, want ACTIONED", res.Data["status"])
	}

	// The listing is gone from the public search...
	if got := env.countRentResults(t); got != 0 {
		t.Errorf("an unpublished listing is still searchable (%d results)", got)
	}
	// ...and the owner can see what happened, with the reason attached.
	owned := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), ownerToken, nil)
	if owned.Data["property_status"] != "REJECTED" {
		t.Errorf("property_status = %v, want REJECTED", owned.Data["property_status"])
	}
	if owned.Data["rejection_reason"] == nil || owned.Data["rejection_reason"] == "" {
		t.Error("the owner is not told why the listing was taken down")
	}
}

// The trust badge reflects an administrator decision, never a self-claim.
func TestOnlyAnAdminCanRaiseAnAccountVerificationLevel(t *testing.T) {
	env := setup(t)
	ownerToken := env.registerOwner(t, "verifyme@example.com", "9700005400", "Verify Me")
	_, _ = env.register(t, "customer", "verifyadmin@example.com", "9700005401", "Admin")
	adminToken := env.promoteToAdmin(t, "verifyadmin@example.com")

	var client models.Client
	if err := env.db.Joins("JOIN users ON users.id = clients.user_id").
		Where("users.email = ?", "verifyme@example.com").First(&client).Error; err != nil {
		t.Fatalf("load listing account: %v", err)
	}
	if client.VerificationLevel != models.VerificationUnverified {
		t.Fatalf("a new account starts at %s, want UNVERIFIED", client.VerificationLevel)
	}

	// The owner cannot award themselves a badge.
	self := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/clients/%d/verification", client.ID), ownerToken,
		map[string]any{"verification_level": "ADMIN_VERIFIED"})
	if self.Status != http.StatusForbidden {
		t.Errorf("self-verification: status %d, want 403", self.Status)
	}

	res := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/clients/%d/verification", client.ID), adminToken,
		map[string]any{"verification_level": "DOCUMENT_VERIFIED", "note": "Lease deed seen in person."})
	if res.Status != http.StatusOK {
		t.Fatalf("verify: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["verification_level"] != "DOCUMENT_VERIFIED" {
		t.Errorf("verification_level = %v, want DOCUMENT_VERIFIED", res.Data["verification_level"])
	}

	// The decision is auditable and the owner is told.
	var audits int64
	env.db.Model(&models.AuditLog{}).
		Where("action = ? AND entity_id = ?", "CLIENT_VERIFICATION_CHANGED", client.ID).Count(&audits)
	if audits != 1 {
		t.Errorf("audit entries = %d, want 1", audits)
	}
	var notes int64
	env.db.Model(&models.Notification{}).Where("user_id = ?", client.UserID).Count(&notes)
	if notes == 0 {
		t.Error("the owner was not told their verification level changed")
	}
}

// An administrator taking details over the phone posts into the owner's own
// account, but the provenance must not claim the owner wrote it.
func TestAdminPostsOnBehalfOfAnOwner(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	ownerToken := env.registerOwner(t, "phoneowner@example.com", "9700005500", "Phone Owner")
	_, _ = env.register(t, "customer", "phoneadmin@example.com", "9700005501", "Admin")
	adminToken := env.promoteToAdmin(t, "phoneadmin@example.com")

	var client models.Client
	if err := env.db.Joins("JOIN users ON users.id = clients.user_id").
		Where("users.email = ?", "phoneowner@example.com").First(&client).Error; err != nil {
		t.Fatalf("load listing account: %v", err)
	}

	res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/clients/%d/properties", client.ID), adminToken,
		rentalPayload(cityID, localityID, "Phoned In Flat", 24000))
	if res.Status != http.StatusCreated {
		t.Fatalf("post on behalf: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["posted_by_type"] != "ADMIN" {
		t.Errorf("posted_by_type = %v, want ADMIN", res.Data["posted_by_type"])
	}
	propertyID := int(res.Data["id"].(float64))

	// The listing is the owner's: they can open and edit it as their own.
	owned := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), ownerToken, nil)
	if owned.Status != http.StatusOK {
		t.Fatalf("owner cannot see the listing posted for them: status %d (%s)",
			owned.Status, owned.Message)
	}

	var property models.Property
	if err := env.db.First(&property, propertyID).Error; err != nil {
		t.Fatalf("load property: %v", err)
	}
	if property.OwnerUserID == nil || *property.OwnerUserID != client.UserID {
		t.Errorf("owner_user_id = %v, want %d", property.OwnerUserID, client.UserID)
	}
}

// With two catalogues in one queue, an administrator reviewing rentals should
// not have to read past sale listings.
func TestAdminQueueSplitsByPurpose(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)
	adminToken := env.promoteToAdmin(t, "rentadmin@example.com")

	all := env.do(t, http.MethodGet, "/api/v1/admin/properties", adminToken, nil)
	if len(all.List) != 4 {
		t.Fatalf("unfiltered queue = %d, want 4", len(all.List))
	}
	rent := env.do(t, http.MethodGet, "/api/v1/admin/properties?purpose=RENT", adminToken, nil)
	if len(rent.List) != 3 {
		t.Errorf("rent queue = %d, want 3", len(rent.List))
	}
	buy := env.do(t, http.MethodGet, "/api/v1/admin/properties?purpose=BUY", adminToken, nil)
	if len(buy.List) != 1 {
		t.Errorf("buy queue = %d, want 1", len(buy.List))
	}
}
