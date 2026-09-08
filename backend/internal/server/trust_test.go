package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

// A verification level an administrator sets is worth nothing if it never
// reaches the page a renter reads.
func TestVerificationLevelReachesThePublicListing(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "70")
	adminToken := env.promoteToAdmin(t, "visitadmin70@example.com")

	before := env.do(t, http.MethodGet, fmt.Sprintf("/api/v1/properties/%d", propertyID), "", nil)
	if before.Status != http.StatusOK {
		t.Fatalf("public listing: status %d (%s)", before.Status, before.Message)
	}
	builder, _ := before.Data["builder"].(map[string]any)
	if builder == nil {
		t.Fatal("the public listing carries no builder block")
	}
	// The account posting is an individual owner, and the page has to be able
	// to say so rather than calling them a builder.
	if builder["profile_type"] != "OWNER" {
		t.Errorf("profile_type = %v, want OWNER", builder["profile_type"])
	}
	if builder["verification_level"] != "UNVERIFIED" {
		t.Errorf("verification_level = %v, want UNVERIFIED", builder["verification_level"])
	}

	var client models.Client
	if err := env.db.First(&client, builder["id"].(float64)).Error; err != nil {
		t.Fatalf("load listing account: %v", err)
	}
	res := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/clients/%d/verification", client.ID), adminToken,
		map[string]any{"verification_level": "ADMIN_VERIFIED"})
	if res.Status != http.StatusOK {
		t.Fatalf("verify: status %d (%s)", res.Status, res.Message)
	}

	after := env.do(t, http.MethodGet, fmt.Sprintf("/api/v1/properties/%d", propertyID), "", nil)
	updated, _ := after.Data["builder"].(map[string]any)
	if updated == nil || updated["verification_level"] != "ADMIN_VERIFIED" {
		t.Errorf("verification_level on the public listing = %v, want ADMIN_VERIFIED",
			updated["verification_level"])
	}
}

// A builder's listing must keep reading as a builder's listing.
func TestBuilderListingsStillReportBuilderProvenance(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedSale(t, "71")

	res := env.do(t, http.MethodGet, fmt.Sprintf("/api/v1/properties/%d", propertyID), "", nil)
	builder, _ := res.Data["builder"].(map[string]any)
	if builder == nil {
		t.Fatal("the public listing carries no builder block")
	}
	if builder["profile_type"] != "BUILDER" {
		t.Errorf("profile_type = %v, want BUILDER", builder["profile_type"])
	}
	if res.Data["posted_by_type"] != "CLIENT" {
		t.Errorf("posted_by_type = %v, want CLIENT", res.Data["posted_by_type"])
	}
	if res.Data["purpose"] != "BUY" {
		t.Errorf("purpose = %v, want BUY", res.Data["purpose"])
	}
}
