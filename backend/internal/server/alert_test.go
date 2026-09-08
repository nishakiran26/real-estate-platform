package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

// alertEnv sets up the cast every alert test needs: a renter with an account, an
// owner who can post, and an administrator who can publish.
type alertEnv struct {
	cityID, localityID uint
	renterToken        string
	renterEmail        string
	ownerToken         string
	adminToken         string
}

func (env *testEnv) alertSetup(t *testing.T, suffix string) alertEnv {
	t.Helper()
	cityID, localityID := env.seedLocation(t)

	renterToken, _ := env.register(t, "customer",
		"renter"+suffix+"@example.com", "97000011"+suffix, "Renter")
	ownerToken := env.registerOwner(t,
		"alertowner"+suffix+"@example.com", "97000012"+suffix, "Alert Owner")
	_, _ = env.register(t, "customer",
		"alertadmin"+suffix+"@example.com", "97000013"+suffix, "Admin")
	adminToken := env.promoteToAdmin(t, "alertadmin"+suffix+"@example.com")

	return alertEnv{
		cityID: cityID, localityID: localityID,
		renterToken: renterToken, renterEmail: "renter" + suffix + "@example.com",
		ownerToken: ownerToken, adminToken: adminToken,
	}
}

// postRental creates a rental as the owner and returns its id, leaving it in
// draft so the test controls when it goes live.
func (env *testEnv) postRental(t *testing.T, a alertEnv, name string, rent float64) int {
	t.Helper()
	created := env.do(t, http.MethodPost, "/api/v1/client/properties", a.ownerToken,
		rentalPayload(a.cityID, a.localityID, name, rent))
	if created.Status != http.StatusCreated {
		t.Fatalf("post rental: status %d (%s)", created.Status, created.Message)
	}
	return int(created.Data["id"].(float64))
}

// notificationCount reports how many notices the renter has been sent.
func (env *testEnv) notificationCount(t *testing.T, email string) int64 {
	t.Helper()
	var user models.User
	if err := env.db.Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("find user: %v", err)
	}
	var n int64
	env.db.Model(&models.Notification{}).Where("user_id = ?", user.ID).Count(&n)
	return n
}

func (env *testEnv) saveSearch(t *testing.T, token string, body map[string]any) map[string]any {
	t.Helper()
	res := env.do(t, http.MethodPost, "/api/v1/customer/rental-alerts", token, body)
	if res.Status != http.StatusCreated {
		t.Fatalf("save search: status %d (%s)", res.Status, res.Message)
	}
	return res.Data
}

// A saved search is only worth having if it survives being saved and can be
// reopened as the search that created it.
func TestSavingASearchKeepsWhatWasSearchedFor(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "50")

	saved := env.saveSearch(t, a.renterToken, map[string]any{
		"city_id": a.cityID, "locality_id": a.localityID,
		"bhk": 2, "min_rent": 15000, "max_rent": 30000,
		"furnishing": "SEMI_FURNISHED",
	})

	// An unnamed search is labelled from what was actually searched for, so the
	// list does not read as "Untitled" three times over.
	if saved["name"] != "2 BHK in Baner under ₹30,000" {
		t.Errorf("name = %v", saved["name"])
	}
	if saved["is_active"] != true {
		t.Errorf("is_active = %v, want true", saved["is_active"])
	}

	// The query string must reproduce the search on the site, including the
	// criteria that are not matched on.
	query := fmt.Sprint(saved["search_query"])
	for _, want := range []string{"purpose=rent", "bhk=2", "max_rent=30000", "furnishing=SEMI_FURNISHED"} {
		if !contains(query, want) {
			t.Errorf("search_query %q is missing %q", query, want)
		}
	}

	listed := env.do(t, http.MethodGet, "/api/v1/customer/rental-alerts", a.renterToken, nil)
	if len(listed.List) != 1 {
		t.Fatalf("saved searches = %d, want 1", len(listed.List))
	}
}

// The point of the feature: a matching flat reaches the renter without them
// checking the site.
func TestPublishingAMatchingRentalNotifiesTheRenter(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "51")

	env.saveSearch(t, a.renterToken, map[string]any{
		"city_id": a.cityID, "bhk": 2, "max_rent": 30000,
	})
	before := env.notificationCount(t, a.renterEmail)

	propertyID := env.postRental(t, a, "Matching Flat", 26000)
	env.publish(t, propertyID, a.ownerToken, a.adminToken)

	if got := env.notificationCount(t, a.renterEmail); got != before+1 {
		t.Fatalf("notifications = %d, want %d", got, before+1)
	}

	// The alert records that it matched, so a search that has never produced
	// anything is distinguishable from one that has.
	var alert models.RentalAlert
	if err := env.db.First(&alert).Error; err != nil {
		t.Fatalf("load alert: %v", err)
	}
	if alert.MatchCount != 1 {
		t.Errorf("match_count = %d, want 1", alert.MatchCount)
	}
	if alert.LastNotifiedAt == nil {
		t.Error("last_notified_at was not recorded")
	}
}

// An alert that fires on everything is worse than no alert, so the criteria
// have to actually exclude.
func TestARentalOutsideTheSavedCriteriaIsNotNotified(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "52")

	env.saveSearch(t, a.renterToken, map[string]any{
		"city_id": a.cityID, "bhk": 2, "max_rent": 20000,
	})
	before := env.notificationCount(t, a.renterEmail)

	// Above the renter's ceiling.
	propertyID := env.postRental(t, a, "Too Expensive", 45000)
	env.publish(t, propertyID, a.ownerToken, a.adminToken)

	if got := env.notificationCount(t, a.renterEmail); got != before {
		t.Errorf("notifications = %d, want %d — a flat over budget was reported", got, before)
	}
}

// Approving without publishing leaves the listing invisible, so telling anyone
// about it would send them to a page they cannot open.
func TestApprovalWithoutPublishingNotifiesNobody(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "53")

	env.saveSearch(t, a.renterToken, map[string]any{"city_id": a.cityID, "bhk": 2})
	before := env.notificationCount(t, a.renterEmail)

	propertyID := env.postRental(t, a, "Approved Only", 24000)
	env.uploadImage(t, propertyID, a.ownerToken)
	submit := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), a.ownerToken, nil)
	if submit.Status != http.StatusOK {
		t.Fatalf("submit: status %d (%s)", submit.Status, submit.Message)
	}
	approve := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/properties/%d/approve", propertyID), a.adminToken,
		map[string]any{"publish": false})
	if approve.Status != http.StatusOK {
		t.Fatalf("approve: status %d (%s)", approve.Status, approve.Message)
	}

	if got := env.notificationCount(t, a.renterEmail); got != before {
		t.Errorf("notifications = %d, want %d — an unpublished listing was reported", got, before)
	}
}

// Pausing is offered so a renter can stop the notices without losing a search
// they spent time refining. It has to actually stop them.
func TestAPausedSearchStopsNotifying(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "54")

	saved := env.saveSearch(t, a.renterToken, map[string]any{"city_id": a.cityID, "bhk": 2})
	alertID := int(saved["id"].(float64))

	paused := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/customer/rental-alerts/%d", alertID), a.renterToken,
		map[string]any{"is_active": false})
	if paused.Status != http.StatusOK {
		t.Fatalf("pause: status %d (%s)", paused.Status, paused.Message)
	}
	before := env.notificationCount(t, a.renterEmail)

	propertyID := env.postRental(t, a, "Paused Match", 24000)
	env.publish(t, propertyID, a.ownerToken, a.adminToken)

	if got := env.notificationCount(t, a.renterEmail); got != before {
		t.Errorf("notifications = %d, want %d — a paused search still fired", got, before)
	}

	// Pausing keeps the search: it is still listed, just inactive.
	listed := env.do(t, http.MethodGet, "/api/v1/customer/rental-alerts", a.renterToken, nil)
	if len(listed.List) != 1 {
		t.Fatalf("saved searches after pausing = %d, want 1", len(listed.List))
	}
	row, _ := listed.List[0].(map[string]any)
	if row["is_active"] != false {
		t.Errorf("is_active = %v, want false", row["is_active"])
	}
}

// A saved search is personal. Nobody else may read or delete it.
func TestSavedSearchesAreOwnerScoped(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "55")
	saved := env.saveSearch(t, a.renterToken, map[string]any{"city_id": a.cityID})
	alertID := int(saved["id"].(float64))

	otherToken, _ := env.register(t, "customer", "other55@example.com", "9700001455", "Other")

	listed := env.do(t, http.MethodGet, "/api/v1/customer/rental-alerts", otherToken, nil)
	if len(listed.List) != 0 {
		t.Errorf("another customer can see %d saved searches", len(listed.List))
	}

	del := env.do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/customer/rental-alerts/%d", alertID), otherToken, nil)
	if del.Status != http.StatusNotFound {
		t.Errorf("delete by another customer: status %d, want 404", del.Status)
	}

	// Anonymous visitors have nowhere to save a search to.
	anon := env.do(t, http.MethodGet, "/api/v1/customer/rental-alerts", "", nil)
	if anon.Status != http.StatusUnauthorized {
		t.Errorf("anonymous: status %d, want 401", anon.Status)
	}
}

// The owner deletes their own search.
func TestDeletingASavedSearch(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "56")
	saved := env.saveSearch(t, a.renterToken, map[string]any{"city_id": a.cityID})
	alertID := int(saved["id"].(float64))

	del := env.do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/customer/rental-alerts/%d", alertID), a.renterToken, nil)
	if del.Status != http.StatusOK {
		t.Fatalf("delete: status %d (%s)", del.Status, del.Message)
	}
	listed := env.do(t, http.MethodGet, "/api/v1/customer/rental-alerts", a.renterToken, nil)
	if len(listed.List) != 0 {
		t.Errorf("saved searches after deleting = %d, want 0", len(listed.List))
	}
}

// Saved searches are capped, so an alert cannot become a subscription to the
// whole catalogue.
func TestSavedSearchesAreCapped(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "57")

	for i := 0; i < 20; i++ {
		res := env.do(t, http.MethodPost, "/api/v1/customer/rental-alerts", a.renterToken,
			map[string]any{"city_id": a.cityID, "max_rent": 20000 + i})
		if res.Status != http.StatusCreated {
			t.Fatalf("saving search %d: status %d (%s)", i+1, res.Status, res.Message)
		}
	}

	over := env.do(t, http.MethodPost, "/api/v1/customer/rental-alerts", a.renterToken,
		map[string]any{"city_id": a.cityID})
	if over.Status != http.StatusBadRequest {
		t.Errorf("21st search: status %d (%s), want 400", over.Status, over.Message)
	}
}

// A minimum above the maximum is a search that can never match.
func TestASavedSearchWithAnImpossibleRentBandIsRejected(t *testing.T) {
	env := setup(t)
	a := env.alertSetup(t, "58")

	res := env.do(t, http.MethodPost, "/api/v1/customer/rental-alerts", a.renterToken,
		map[string]any{"city_id": a.cityID, "min_rent": 50000, "max_rent": 20000})
	if res.Status != http.StatusBadRequest {
		t.Errorf("status %d (%s), want 400", res.Status, res.Message)
	}
}
