package server_test

import (
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

// The rental extension is additive: it must leave every listing that existed
// before it looking exactly like a sale listing. These tests assert the
// invariants the migration promises, so a future change to the defaults or the
// backfill cannot silently reclassify existing inventory.

func TestRentalColumnsExistAfterMigration(t *testing.T) {
	env := setup(t)
	m := env.db.Migrator()

	cases := []struct {
		table   string
		model   any
		columns []string
	}{
		{"properties", &models.Property{}, []string{
			"purpose", "posted_by_type", "owner_user_id",
			"availability_status", "listing_expiry_date",
		}},
		{"clients", &models.Client{}, []string{"profile_type", "verification_level"}},
		{"inquiries", &models.Inquiry{}, []string{"lead_type", "assigned_to_user_id"}},
	}

	for _, tc := range cases {
		for _, column := range tc.columns {
			if !m.HasColumn(tc.model, column) {
				t.Errorf("%s.%s is missing after migration", tc.table, column)
			}
		}
	}

	for _, model := range []any{
		&models.RentalDetails{}, &models.PropertyVisit{},
		&models.PropertyReport{}, &models.RentalAlert{},
	} {
		if !m.HasTable(model) {
			t.Errorf("table for %T was not created", model)
		}
	}
}

// A property created through the existing sale path must come out as BUY,
// available, and posted by a client — without the caller saying so. This is
// what keeps the Buy journey working unchanged while rental support lands.
func TestExistingPropertyPathDefaultsToBuy(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	builderToken, _ := env.register(t, "client", "defaults@example.com", "9800000101", "Defaults")

	created := env.do(t, "POST", "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Defaults Residency"))
	if created.Status != 201 {
		t.Fatalf("create property: status %d (%s)", created.Status, created.Message)
	}

	var property models.Property
	if err := env.db.First(&property, uint(created.Data["id"].(float64))).Error; err != nil {
		t.Fatalf("reload property: %v", err)
	}

	if property.Purpose != models.PurposeBuy {
		t.Errorf("Purpose = %q, want BUY", property.Purpose)
	}
	if property.AvailabilityStatus != models.AvailabilityAvailable {
		t.Errorf("AvailabilityStatus = %q, want AVAILABLE", property.AvailabilityStatus)
	}
	if property.PostedByType != models.PostedByClient {
		t.Errorf("PostedByType = %q, want CLIENT", property.PostedByType)
	}
	// A sale listing never expires and has no separate owner record.
	if property.ListingExpiryDate != nil {
		t.Error("a sale listing was given a listing expiry date")
	}
	if property.OwnerUserID != nil {
		t.Error("OwnerUserID should be nil when the poster is the owner")
	}
}

// Registering as a builder must produce a BUILDER listing account. Owners will
// reuse this same table with a different profile type, so the default matters.
func TestBuilderRegistrationCreatesBuilderProfile(t *testing.T) {
	env := setup(t)
	env.register(t, "client", "profile@example.com", "9800000102", "Profile Co")

	var client models.Client
	if err := env.db.Joins("JOIN users ON users.id = clients.user_id").
		Where("users.email = ?", "profile@example.com").First(&client).Error; err != nil {
		t.Fatalf("load client: %v", err)
	}

	if client.ProfileType != models.ProfileBuilder {
		t.Errorf("ProfileType = %q, want BUILDER", client.ProfileType)
	}
	// A badge must never appear without the verification actually happening.
	if client.VerificationLevel != models.VerificationUnverified {
		t.Errorf("VerificationLevel = %q, want UNVERIFIED on a fresh account", client.VerificationLevel)
	}
}

// Leads captured through the existing inquiry path are buy leads, so rental
// reporting can be split by type without reclassifying history.
func TestInquiryDefaultsToBuyLead(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	builderToken, _ := env.register(t, "client", "leadtype@example.com", "9800000103", "LeadType")
	_, _ = env.register(t, "customer", "leadadmin@example.com", "9800000104", "Admin")
	adminToken := env.promoteToAdmin(t, "leadadmin@example.com")

	created := env.do(t, "POST", "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Lead Type Towers"))
	propertyID := int(created.Data["id"].(float64))
	env.publish(t, propertyID, builderToken, adminToken)

	inquiry := env.do(t, "POST", "/api/v1/inquiries", "", map[string]any{
		"property_id": propertyID, "name": "Lead Tester", "mobile": "9812345600",
		"consent_contact": true,
	})
	if inquiry.Status != 201 {
		t.Fatalf("create inquiry: status %d (%s)", inquiry.Status, inquiry.Message)
	}

	var lead models.Inquiry
	if err := env.db.Where("property_id = ?", propertyID).First(&lead).Error; err != nil {
		t.Fatalf("load inquiry: %v", err)
	}
	if lead.LeadType != models.LeadBuy {
		t.Errorf("LeadType = %q, want BUY_LEAD", lead.LeadType)
	}
	if lead.AssignedToUserID != nil {
		t.Error("a new lead should be unassigned, visible to the whole listing account")
	}
}

// The approval workflow and availability must stay independent: approving a
// listing says nothing about whether it is still on the market.
func TestApprovalDoesNotTouchAvailability(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	builderToken, _ := env.register(t, "client", "avail@example.com", "9800000105", "Avail")
	_, _ = env.register(t, "customer", "availadmin@example.com", "9800000106", "Admin")
	adminToken := env.promoteToAdmin(t, "availadmin@example.com")

	created := env.do(t, "POST", "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Availability Heights"))
	propertyID := int(created.Data["id"].(float64))
	env.publish(t, propertyID, builderToken, adminToken)

	var property models.Property
	if err := env.db.First(&property, propertyID).Error; err != nil {
		t.Fatalf("reload property: %v", err)
	}

	if property.PropertyStatus != models.PropertyStatusPublished {
		t.Errorf("PropertyStatus = %q, want PUBLISHED", property.PropertyStatus)
	}
	if property.AvailabilityStatus != models.AvailabilityAvailable {
		t.Errorf("AvailabilityStatus = %q, want AVAILABLE — approval must not change it",
			property.AvailabilityStatus)
	}
}
