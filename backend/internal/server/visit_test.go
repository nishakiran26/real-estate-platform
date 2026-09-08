package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
)

func tomorrow() string  { return time.Now().AddDate(0, 0, 1).Format("2006-01-02") }
func nextWeek() string  { return time.Now().AddDate(0, 0, 7).Format("2006-01-02") }
func yesterday() string { return time.Now().AddDate(0, 0, -1).Format("2006-01-02") }

// publishedRental returns a live rental plus its owner's token, which most of
// the visit tests need as their starting point.
func (env *testEnv) publishedRental(t *testing.T, suffix string) (propertyID int, ownerToken string) {
	t.Helper()
	cityID, localityID := env.seedLocation(t)

	ownerToken = env.registerOwner(t,
		"visitowner"+suffix+"@example.com", "97000005"+suffix, "Visit Owner "+suffix)
	_, _ = env.register(t,
		"customer", "visitadmin"+suffix+"@example.com", "97000006"+suffix, "Admin")
	adminToken := env.promoteToAdmin(t, "visitadmin"+suffix+"@example.com")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", ownerToken,
		rentalPayload(cityID, localityID, "Visit Flat "+suffix, 26000))
	if created.Status != http.StatusCreated {
		t.Fatalf("create rental: status %d (%s)", created.Status, created.Message)
	}
	propertyID = int(created.Data["id"].(float64))
	env.publish(t, propertyID, ownerToken, adminToken)
	return propertyID, ownerToken
}

func visitPayload(date string) map[string]any {
	return map[string]any{
		"name": "Anjali Rao", "mobile": "9812340001",
		"email": "anjali@example.com", "message": "Could I see it this weekend?",
		"visit_date": date, "time_slot": "EVENING",
		"consent_contact": true,
	}
}

func TestAnonymousVisitorCanRequestAVisit(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "10")

	res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), "", visitPayload(tomorrow()))
	if res.Status != http.StatusCreated {
		t.Fatalf("request visit: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["status"] != "REQUESTED" {
		t.Errorf("status = %v, want REQUESTED", res.Data["status"])
	}
	if res.Data["time_slot"] != "EVENING" {
		t.Errorf("time_slot = %v, want EVENING", res.Data["time_slot"])
	}

	// The request is recorded in the engagement stream, so it reaches owner
	// analytics beside views and WhatsApp clicks.
	var events int64
	env.db.Model(&models.UserPropertyEvent{}).
		Where("property_id = ? AND event_type = ?", propertyID, models.EventVisitRequest).
		Count(&events)
	if events != 1 {
		t.Errorf("visit events recorded = %d, want 1", events)
	}
}

func TestVisitRequiresConsentAndASensibleDate(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "11")
	path := fmt.Sprintf("/api/v1/properties/%d/visits", propertyID)

	noConsent := visitPayload(tomorrow())
	noConsent["consent_contact"] = false
	if res := env.do(t, http.MethodPost, path, "", noConsent); res.Status == http.StatusCreated {
		t.Error("a visit was booked without contact consent")
	}

	past := visitPayload(yesterday())
	past["mobile"] = "9812340002"
	if res := env.do(t, http.MethodPost, path, "", past); res.Status == http.StatusCreated {
		t.Error("a visit was booked for a date that has already passed")
	}

	tooFar := visitPayload(time.Now().AddDate(1, 0, 0).Format("2006-01-02"))
	tooFar["mobile"] = "9812340003"
	if res := env.do(t, http.MethodPost, path, "", tooFar); res.Status == http.StatusCreated {
		t.Error("a visit was booked a year ahead")
	}
}

// Tapping the button twice is one request, not two.
func TestDuplicateVisitRequestIsRejected(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "12")
	path := fmt.Sprintf("/api/v1/properties/%d/visits", propertyID)

	if res := env.do(t, http.MethodPost, path, "", visitPayload(tomorrow())); res.Status != http.StatusCreated {
		t.Fatalf("first request: status %d (%s)", res.Status, res.Message)
	}
	second := env.do(t, http.MethodPost, path, "", visitPayload(nextWeek()))
	if second.Status == http.StatusCreated {
		t.Error("the same number booked the same property twice")
	}

	var count int64
	env.db.Model(&models.PropertyVisit{}).Where("property_id = ?", propertyID).Count(&count)
	if count != 1 {
		t.Errorf("visit rows = %d, want 1", count)
	}
}

func TestVisitCannotBeRequestedForARentedFlat(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "13")

	if err := env.db.Model(&models.Property{}).Where("id = ?", propertyID).
		Update("availability_status", models.AvailabilityRented).Error; err != nil {
		t.Fatalf("mark rented: %v", err)
	}

	res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), "", visitPayload(tomorrow()))
	if res.Status == http.StatusCreated {
		t.Error("a visit was booked for a flat that has already been let")
	}
}

// The owner sees the request, confirms a time, and the visitor's own list shows
// the outcome — the whole loop on one property.
func TestOwnerConfirmsAVisit(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "14")

	buyerToken, _ := env.register(t, "customer", "visitbuyer@example.com", "9812340010", "Buyer")

	booked := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), buyerToken, visitPayload(tomorrow()))
	if booked.Status != http.StatusCreated {
		t.Fatalf("book visit: status %d (%s)", booked.Status, booked.Message)
	}
	visitID := int(booked.Data["id"].(float64))

	// The owner's inbox shows it, with the visitor's contact details.
	inbox := env.do(t, http.MethodGet, "/api/v1/client/visits", ownerToken, nil)
	if len(inbox.List) != 1 {
		t.Fatalf("owner sees %d visit(s), want 1", len(inbox.List))
	}
	row, _ := inbox.List[0].(map[string]any)
	if row["mobile"] == "" || row["mobile"] == nil {
		t.Error("the owner cannot see the visitor's number, so cannot call them")
	}

	confirmed := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/visits/%d", visitID), ownerToken, map[string]any{
			"status": "CONFIRMED", "scheduled_date": tomorrow(),
			"time_slot": "MORNING", "remarks": "Please call when you reach the gate.",
		})
	if confirmed.Status != http.StatusOK {
		t.Fatalf("confirm visit: status %d (%s)", confirmed.Status, confirmed.Message)
	}
	if confirmed.Data["status"] != "CONFIRMED" {
		t.Errorf("status = %v, want CONFIRMED", confirmed.Data["status"])
	}
	if confirmed.Data["scheduled_at"] != tomorrow() {
		t.Errorf("scheduled_at = %v, want %s", confirmed.Data["scheduled_at"], tomorrow())
	}

	// The visitor sees the confirmation in their own list.
	mine := env.do(t, http.MethodGet, "/api/v1/customer/visits", buyerToken, nil)
	if len(mine.List) != 1 {
		t.Fatalf("customer sees %d visit(s), want 1", len(mine.List))
	}
	own, _ := mine.List[0].(map[string]any)
	if own["status"] != "CONFIRMED" {
		t.Errorf("customer sees status %v, want CONFIRMED", own["status"])
	}

	// And they were notified.
	var notifications int64
	env.db.Model(&models.Notification{}).
		Where("title LIKE ?", "%visit is confirmed%").Count(&notifications)
	if notifications == 0 {
		t.Error("the visitor was not notified that the visit was confirmed")
	}
}

func TestConfirmingRequiresADate(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "15")

	booked := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), "", visitPayload(tomorrow()))
	visitID := int(booked.Data["id"].(float64))

	res := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/visits/%d", visitID), ownerToken,
		map[string]any{"status": "CONFIRMED"})
	if res.Status == http.StatusOK {
		t.Error("a visit was confirmed without committing to a date")
	}
}

func TestClosedVisitCannotBeReopened(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "16")

	booked := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), "", visitPayload(tomorrow()))
	visitID := int(booked.Data["id"].(float64))
	path := fmt.Sprintf("/api/v1/client/visits/%d", visitID)

	if res := env.do(t, http.MethodPut, path, ownerToken,
		map[string]any{"status": "CANCELLED", "remarks": "Flat is let"}); res.Status != http.StatusOK {
		t.Fatalf("cancel: status %d (%s)", res.Status, res.Message)
	}
	if res := env.do(t, http.MethodPut, path, ownerToken,
		map[string]any{"status": "CONFIRMED", "scheduled_date": tomorrow()}); res.Status == http.StatusOK {
		t.Error("a cancelled visit was reopened")
	}
}

// The gate for this phase: one owner must never see or touch another's visits.
func TestVisitIsolationBetweenOwners(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "17")

	rivalToken := env.registerOwner(t, "visitrival@example.com", "9700000777", "Rival Owner")

	booked := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), "", visitPayload(tomorrow()))
	visitID := int(booked.Data["id"].(float64))

	rivalInbox := env.do(t, http.MethodGet, "/api/v1/client/visits", rivalToken, nil)
	if len(rivalInbox.List) != 0 {
		t.Errorf("a rival owner saw %d visit(s) that were not theirs", len(rivalInbox.List))
	}

	hijack := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/visits/%d", visitID), rivalToken,
		map[string]any{"status": "CANCELLED"})
	if hijack.Status != http.StatusNotFound && hijack.Status != http.StatusForbidden {
		t.Errorf("a rival owner cancelled another owner's visit: status %d", hijack.Status)
	}

	var visit models.PropertyVisit
	env.db.First(&visit, visitID)
	if visit.Status != models.VisitRequested {
		t.Errorf("the visit was altered by a rival: status %q", visit.Status)
	}
}

func TestCustomerSeesOnlyTheirOwnVisits(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "18")

	first, _ := env.register(t, "customer", "visitor.a@example.com", "9812340021", "Visitor A")
	second, _ := env.register(t, "customer", "visitor.b@example.com", "9812340022", "Visitor B")

	payload := visitPayload(tomorrow())
	payload["mobile"] = "9812340021"
	if res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), first, payload); res.Status != http.StatusCreated {
		t.Fatalf("book visit: status %d (%s)", res.Status, res.Message)
	}

	mine := env.do(t, http.MethodGet, "/api/v1/customer/visits", first, nil)
	if len(mine.List) != 1 {
		t.Errorf("visitor A sees %d visit(s), want 1", len(mine.List))
	}
	theirs := env.do(t, http.MethodGet, "/api/v1/customer/visits", second, nil)
	if len(theirs.List) != 0 {
		t.Errorf("visitor B saw %d of visitor A's visits", len(theirs.List))
	}
}

// A rental enquiry must be reported as a rental lead, derived from the property
// rather than trusted from the caller.
func TestRentalInquiryIsARentLead(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "19")

	res := env.do(t, http.MethodPost, "/api/v1/inquiries", "", map[string]any{
		"property_id": propertyID, "name": "Rent Enquirer", "mobile": "9812340030",
		"consent_contact": true,
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("create inquiry: status %d (%s)", res.Status, res.Message)
	}

	var lead models.Inquiry
	if err := env.db.Where("property_id = ?", propertyID).First(&lead).Error; err != nil {
		t.Fatalf("load lead: %v", err)
	}
	if lead.LeadType != models.LeadRent {
		t.Errorf("LeadType = %q, want RENT_LEAD", lead.LeadType)
	}
}

// The prefilled WhatsApp message must speak to a renter, and carry the rent.
func TestWhatsAppMessageIsRentalAware(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "20")

	res := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/properties/%d/whatsapp", propertyID), "", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("whatsapp link: status %d (%s)", res.Status, res.Message)
	}

	message, _ := res.Data["message"].(string)
	for _, want := range []string{"interested in renting", "Monthly rent", "26,000", "visit"} {
		if !contains(message, want) {
			t.Errorf("the rental message does not mention %q:\n%s", want, message)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOfSubstring(haystack, needle) >= 0
}

func indexOfSubstring(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
