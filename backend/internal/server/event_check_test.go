package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

// Engagement events are written on a fire-and-forget path that only logs
// failures, so a storage error is invisible in the API response. This asserts
// they actually land — the analytics dashboards are worthless otherwise.
func TestEngagementEventsArePersisted(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "90")

	buyer, _ := env.register(t, "customer", "eventer@example.com", "9812349001", "Eventer")

	env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/customer/favorites/%d", propertyID), buyer, nil)
	env.do(t, http.MethodGet, fmt.Sprintf("/api/v1/properties/%d/whatsapp", propertyID), "", nil)
	env.do(t, http.MethodPost, "/api/v1/inquiries", "", map[string]any{
		"property_id": propertyID, "name": "Event Lead", "mobile": "9812349002",
		"consent_contact": true,
	})
	env.do(t, http.MethodPost, "/api/v1/events", "", map[string]any{
		"event_type": "CALL_CLICK", "property_id": propertyID,
	})

	for _, eventType := range []models.EventType{
		models.EventFavorite, models.EventWhatsappClick,
		models.EventInquiry, models.EventCallClick,
	} {
		var count int64
		env.db.Model(&models.UserPropertyEvent{}).
			Where("property_id = ? AND event_type = ?", propertyID, eventType).Count(&count)
		if count == 0 {
			t.Errorf("%s was not persisted", eventType)
		}
	}
}
