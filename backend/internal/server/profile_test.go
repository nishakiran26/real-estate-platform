package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

// registerWith performs a signup with an explicit profile type and intents,
// which is the newer shape of the registration request.
func (env *testEnv) registerWith(t *testing.T, body map[string]any) response {
	t.Helper()
	return env.do(t, http.MethodPost, "/api/v1/auth/register", "", body)
}

func baseSignup(email, mobile, name string) map[string]any {
	return map[string]any{
		"full_name": name, "email": email, "mobile": mobile,
		"password": "Password@123",
	}
}

// Each self-declared intent must resolve to the right system role and, for
// listing accounts, the right profile type. Builders, owners and agents share
// the CLIENT role on purpose.
func TestProfileTypeResolvesToRoleAndAccount(t *testing.T) {
	env := setup(t)

	cases := []struct {
		profileType string
		email       string
		mobile      string
		wantRole    string
		wantProfile models.ProfileType
	}{
		{"BUYER", "buyer.intent@example.com", "9800000201", "CUSTOMER", ""},
		{"RENTER", "renter.intent@example.com", "9800000202", "CUSTOMER", ""},
		{"OWNER", "owner.intent@example.com", "9800000203", "CLIENT", models.ProfileOwner},
		{"BUILDER", "builder.intent@example.com", "9800000204", "CLIENT", models.ProfileBuilder},
		{"AGENT", "agent.intent@example.com", "9800000205", "CLIENT", models.ProfileAgent},
	}

	for _, tc := range cases {
		t.Run(tc.profileType, func(t *testing.T) {
			body := baseSignup(tc.email, tc.mobile, tc.profileType+" User")
			body["profile_type"] = tc.profileType

			res := env.registerWith(t, body)
			if res.Status != http.StatusCreated {
				t.Fatalf("register: status %d (%s)", res.Status, res.Message)
			}

			var user models.User
			if err := env.db.Preload("Roles").Where("email = ?", tc.email).First(&user).Error; err != nil {
				t.Fatalf("load user: %v", err)
			}
			if len(user.Roles) != 1 || user.Roles[0].Code != tc.wantRole {
				t.Fatalf("roles = %v, want [%s]", user.Roles, tc.wantRole)
			}

			if tc.wantProfile == "" {
				// A buyer or renter must not be handed a listing account.
				var count int64
				env.db.Model(&models.Client{}).Where("user_id = ?", user.ID).Count(&count)
				if count != 0 {
					t.Error("a customer was given a listing account")
				}
				return
			}

			var client models.Client
			if err := env.db.Where("user_id = ?", user.ID).First(&client).Error; err != nil {
				t.Fatalf("load listing account: %v", err)
			}
			if client.ProfileType != tc.wantProfile {
				t.Errorf("ProfileType = %q, want %q", client.ProfileType, tc.wantProfile)
			}
		})
	}
}

// The headline requirement: one account covers buying and renting. A customer
// selecting both intents gets a single login with both recorded.
func TestCustomerCanDeclareBothIntents(t *testing.T) {
	env := setup(t)

	body := baseSignup("both.intents@example.com", "9800000210", "Rupali Both")
	body["profile_type"] = "BUYER"
	body["intents"] = []string{"BUY", "RENT"}

	if res := env.registerWith(t, body); res.Status != http.StatusCreated {
		t.Fatalf("register: status %d (%s)", res.Status, res.Message)
	}

	var user models.User
	if err := env.db.Where("email = ?", "both.intents@example.com").First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}

	var profile models.CustomerProfile
	if err := env.db.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		t.Fatalf("load customer profile: %v", err)
	}
	if !profile.LookingToBuy || !profile.LookingToRent {
		t.Errorf("LookingToBuy=%v LookingToRent=%v, want both true",
			profile.LookingToBuy, profile.LookingToRent)
	}
}

func TestRenterOnlyIntentIsRecorded(t *testing.T) {
	env := setup(t)

	body := baseSignup("renter.only@example.com", "9800000211", "Renter Only")
	body["profile_type"] = "RENTER"

	if res := env.registerWith(t, body); res.Status != http.StatusCreated {
		t.Fatalf("register: status %d (%s)", res.Status, res.Message)
	}

	var user models.User
	env.db.Where("email = ?", "renter.only@example.com").First(&user)

	var profile models.CustomerProfile
	if err := env.db.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		t.Fatalf("load customer profile: %v", err)
	}
	if profile.LookingToBuy {
		t.Error("a renter was marked as looking to buy")
	}
	if !profile.LookingToRent {
		t.Error("LookingToRent should be true for a renter")
	}
}

// The original two-option field must keep working, because existing clients
// still send it.
func TestLegacyRoleCodeStillWorks(t *testing.T) {
	env := setup(t)

	customer := baseSignup("legacy.customer@example.com", "9800000220", "Legacy Customer")
	customer["role_code"] = "customer"
	if res := env.registerWith(t, customer); res.Status != http.StatusCreated {
		t.Fatalf("legacy customer register: status %d (%s)", res.Status, res.Message)
	}

	client := baseSignup("legacy.client@example.com", "9800000221", "Legacy Builders")
	client["role_code"] = "client"
	client["company_name"] = "Legacy Builders"
	if res := env.registerWith(t, client); res.Status != http.StatusCreated {
		t.Fatalf("legacy client register: status %d (%s)", res.Status, res.Message)
	}

	// A legacy client is still a builder, not an owner.
	var account models.Client
	if err := env.db.Joins("JOIN users ON users.id = clients.user_id").
		Where("users.email = ?", "legacy.client@example.com").First(&account).Error; err != nil {
		t.Fatalf("load listing account: %v", err)
	}
	if account.ProfileType != models.ProfileBuilder {
		t.Errorf("ProfileType = %q, want BUILDER for a legacy client signup", account.ProfileType)
	}
}

func TestRegistrationRejectsMissingIntent(t *testing.T) {
	env := setup(t)

	body := baseSignup("no.intent@example.com", "9800000222", "No Intent")
	res := env.registerWith(t, body) // neither profile_type nor role_code

	if res.Status == http.StatusCreated {
		t.Error("registration succeeded without any stated intent")
	}
}

// An individual owner rarely has a company. Their own name should stand in
// rather than the signup being rejected.
func TestOwnerWithoutCompanyNameUsesTheirOwnName(t *testing.T) {
	env := setup(t)

	body := baseSignup("solo.owner@example.com", "9800000223", "Rahul Sharma")
	body["profile_type"] = "OWNER"

	if res := env.registerWith(t, body); res.Status != http.StatusCreated {
		t.Fatalf("register: status %d (%s)", res.Status, res.Message)
	}

	var account models.Client
	if err := env.db.Joins("JOIN users ON users.id = clients.user_id").
		Where("users.email = ?", "solo.owner@example.com").First(&account).Error; err != nil {
		t.Fatalf("load listing account: %v", err)
	}
	if account.CompanyName != "Rahul Sharma" {
		t.Errorf("CompanyName = %q, want the owner's own name", account.CompanyName)
	}
}

// The architectural requirement in full: a customer who decides to rent out a
// flat gains posting ability on the same login, keeping their buyer access.
func TestCustomerCanBecomeAnOwnerOnTheSameAccount(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	token, _ := env.register(t, "customer", "upgrader@example.com", "9800000230", "Upgrader")

	// Before upgrading, the builder portal is closed to them.
	before := env.do(t, http.MethodGet, "/api/v1/client/properties", token, nil)
	if before.Status != http.StatusForbidden {
		t.Fatalf("customer reached the builder portal before upgrading: status %d", before.Status)
	}

	upgrade := env.do(t, http.MethodPost, "/api/v1/customer/listing-account", token,
		map[string]any{"profile_type": "OWNER"})
	if upgrade.Status != http.StatusCreated {
		t.Fatalf("enable listing account: status %d (%s)", upgrade.Status, upgrade.Message)
	}
	if upgrade.Data["profile_type"] != "OWNER" {
		t.Errorf("profile_type = %v, want OWNER", upgrade.Data["profile_type"])
	}

	// Both roles are held: becoming an owner must not cost them buyer access.
	var user models.User
	if err := env.db.Preload("Roles").Where("email = ?", "upgrader@example.com").
		First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	codes := map[string]bool{}
	for _, role := range user.Roles {
		codes[role.Code] = true
	}
	if !codes["CUSTOMER"] || !codes["CLIENT"] {
		t.Fatalf("roles = %v, want both CUSTOMER and CLIENT", codes)
	}

	// A fresh sign-in carries the new role, and the builder portal now admits
	// them — on the same account, with no second registration.
	login := env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": "upgrader@example.com", "password": "Password@123",
	})
	tokens, _ := login.Data["tokens"].(map[string]any)
	upgraded, _ := tokens["access_token"].(string)

	after := env.do(t, http.MethodGet, "/api/v1/client/properties", upgraded, nil)
	if after.Status != http.StatusOK {
		t.Fatalf("owner could not reach the builder portal after upgrading: status %d (%s)",
			after.Status, after.Message)
	}

	// And they can actually post.
	created := env.do(t, http.MethodPost, "/api/v1/client/properties", upgraded,
		propertyPayload(cityID, localityID, "Upgraders Flat"))
	if created.Status != http.StatusCreated {
		t.Fatalf("owner could not post a property: status %d (%s)", created.Status, created.Message)
	}

	// Their customer access still works too.
	saved := env.do(t, http.MethodGet, "/api/v1/customer/favorites", upgraded, nil)
	if saved.Status != http.StatusOK {
		t.Errorf("upgraded user lost customer access: status %d", saved.Status)
	}
}

// Upgrading twice must not create a second listing account, which the unique
// index would reject anyway.
func TestEnableListingAccountIsIdempotent(t *testing.T) {
	env := setup(t)
	token, _ := env.register(t, "customer", "twice@example.com", "9800000231", "Twice")

	first := env.do(t, http.MethodPost, "/api/v1/customer/listing-account", token,
		map[string]any{"profile_type": "OWNER"})
	second := env.do(t, http.MethodPost, "/api/v1/customer/listing-account", token,
		map[string]any{"profile_type": "OWNER"})

	if first.Status != http.StatusCreated || second.Status != http.StatusCreated {
		t.Fatalf("statuses were %d and %d, want both 201", first.Status, second.Status)
	}
	if fmt.Sprint(first.Data["id"]) != fmt.Sprint(second.Data["id"]) {
		t.Errorf("a second listing account was created: %v then %v",
			first.Data["id"], second.Data["id"])
	}
}
