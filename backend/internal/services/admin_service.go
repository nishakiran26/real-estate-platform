package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/notification"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// Audit action names recorded for privileged operations.
const (
	AuditPropertyApprove  = "PROPERTY_APPROVE"
	AuditPropertyReject   = "PROPERTY_REJECT"
	AuditPropertyPublish  = "PROPERTY_PUBLISH"
	AuditUserStatusChange = "USER_STATUS_CHANGE"
	AuditUserRoleChange   = "USER_ROLE_CHANGE"
	AuditFeatureChange    = "FEATURED_PROPERTY_CHANGE"
	AuditRERAVerify       = "RERA_VERIFICATION"
)

// AdminService implements the operations only an administrator may perform.
type AdminService struct {
	props    repositories.PropertyRepository
	users    repositories.UserRepository
	accounts repositories.AccountRepository
	engage   repositories.EngagementRepository
	notifier *notification.Dispatcher
	propSvc  *PropertyService
	alertSvc *AlertService
}

func NewAdminService(
	props repositories.PropertyRepository,
	users repositories.UserRepository,
	accounts repositories.AccountRepository,
	engage repositories.EngagementRepository,
	notifier *notification.Dispatcher,
	propSvc *PropertyService,
	alertSvc *AlertService,
) *AdminService {
	return &AdminService{props: props, users: users, accounts: accounts,
		engage: engage, notifier: notifier, propSvc: propSvc, alertSvc: alertSvc}
}

// ListProperties returns the moderation queue across every builder.
func (s *AdminService) ListProperties(ctx context.Context, q dto.AdminPropertyListQuery) ([]dto.PropertyCard, *dto.Meta, error) {
	page, perPage := normalizePage(q.Page, q.PerPage)
	filter := repositories.PropertyFilter{
		ClientID: q.ClientID, CityID: q.CityID, Search: q.Search,
		SortBy: q.SortBy, Statuses: allStatuses(),
		Purpose: strings.ToUpper(q.Purpose),
		Offset:  (page - 1) * perPage, Limit: perPage,
	}
	if q.Availability != "" {
		filter.AvailabilityStatuses = []models.AvailabilityStatus{
			models.AvailabilityStatus(strings.ToUpper(q.Availability)),
		}
	}
	if q.Status != "" {
		filter.Statuses = []models.PropertyStatus{models.PropertyStatus(strings.ToUpper(q.Status))}
	}
	props, total, err := s.props.List(ctx, filter)
	if err != nil {
		return nil, nil, Internal("load properties", err)
	}
	return ToPropertyCards(props), newMeta(page, perPage, total), nil
}

// GetProperty loads any listing regardless of state or owner.
func (s *AdminService) GetProperty(ctx context.Context, id uint) (*dto.PropertyResponse, error) {
	p, err := s.props.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	resp := ToPropertyResponse(p)
	return &resp, nil
}

// StartReview moves a submitted listing to UNDER_REVIEW so other administrators
// can see it has been picked up.
func (s *AdminService) StartReview(ctx context.Context, id, adminID uint) error {
	p, err := s.props.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Property not found")
		}
		return Internal("load property", err)
	}
	if p.PropertyStatus != models.PropertyStatusSubmitted {
		return Conflict("Only submitted properties can be moved to review")
	}
	if err := s.props.ChangeStatus(ctx, id, models.PropertyStatusUnderReview, adminID, "Review started"); err != nil {
		return Internal("update property status", err)
	}
	return nil
}

// Approve accepts a listing and optionally publishes it immediately.
func (s *AdminService) Approve(ctx context.Context, id, adminID uint, req dto.ApprovePropertyRequest, meta RequestMeta) error {
	p, err := s.props.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Property not found")
		}
		return Internal("load property", err)
	}
	switch p.PropertyStatus {
	case models.PropertyStatusSubmitted, models.PropertyStatusUnderReview, models.PropertyStatusRejected:
		// approvable
	default:
		return Conflict("Only a submitted property can be approved")
	}

	target := models.PropertyStatusApproved
	action := AuditPropertyApprove
	if req.Publish {
		target = models.PropertyStatusPublished
		action = AuditPropertyPublish
	}
	remarks := req.Remarks
	if remarks == "" {
		remarks = "Approved"
	}
	if err := s.props.ChangeStatus(ctx, id, target, adminID, remarks); err != nil {
		return Internal("approve property", err)
	}
	s.audit(ctx, adminID, action, "property", id, map[string]any{"remarks": remarks}, meta)
	s.notifyBuilder(ctx, p, "Your property has been approved",
		fmt.Sprintf("%s is now live on the platform.", p.Name), "/client/properties")

	// Renters watching a saved search hear about the flat the moment it is
	// publicly visible — and only then, so an approval that stops short of
	// publishing tells nobody about a listing they cannot yet open.
	if target == models.PropertyStatusPublished && s.alertSvc != nil {
		s.alertSvc.NotifyForProperty(ctx, p)
	}
	s.propSvc.InvalidateSearchCache(ctx)
	return nil
}

// Reject sends a listing back to the builder with a reason.
func (s *AdminService) Reject(ctx context.Context, id, adminID uint, req dto.RejectPropertyRequest, meta RequestMeta) error {
	p, err := s.props.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Property not found")
		}
		return Internal("load property", err)
	}
	if p.PropertyStatus == models.PropertyStatusDraft {
		return Conflict("A draft has not been submitted for review")
	}
	if err := s.props.ChangeStatus(ctx, id, models.PropertyStatusRejected, adminID, req.Reason); err != nil {
		return Internal("reject property", err)
	}
	s.audit(ctx, adminID, AuditPropertyReject, "property", id, map[string]any{"reason": req.Reason}, meta)
	s.notifyBuilder(ctx, p, "Your property needs changes",
		fmt.Sprintf("%s was not approved. Reason: %s", p.Name, req.Reason), "/client/properties")
	s.propSvc.InvalidateSearchCache(ctx)
	return nil
}

// VerifyRERA records the administrator decision on a registration number.
func (s *AdminService) VerifyRERA(ctx context.Context, propertyID, adminID uint, req dto.VerifyRERARequest, meta RequestMeta) error {
	status := models.RERAStatus(req.Status)
	if err := s.props.SetRERAVerification(ctx, propertyID, status, adminID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("This property has no RERA details to verify")
		}
		return Internal("update RERA verification", err)
	}
	s.audit(ctx, adminID, AuditRERAVerify, "property", propertyID, map[string]any{"status": req.Status}, meta)
	s.propSvc.InvalidateSearchCache(ctx)
	return nil
}

// SetFeatured pins or unpins a listing on the homepage.
func (s *AdminService) SetFeatured(ctx context.Context, propertyID, adminID uint, featured bool, meta RequestMeta) error {
	if err := s.props.SetFeatured(ctx, propertyID, featured); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Property not found")
		}
		return Internal("update featured flag", err)
	}
	s.audit(ctx, adminID, AuditFeatureChange, "property", propertyID, map[string]any{"featured": featured}, meta)
	s.propSvc.InvalidateSearchCache(ctx)
	return nil
}

// ListUsers powers the user, client and customer management screens.
func (s *AdminService) ListUsers(ctx context.Context, q dto.UserListQuery) ([]dto.UserResponse, *dto.Meta, error) {
	page, perPage := normalizePage(q.Page, q.PerPage)
	users, total, err := s.users.List(ctx, repositories.UserFilter{
		Search: q.Search, RoleCode: strings.ToUpper(q.Role), Status: strings.ToUpper(q.Status),
		Offset: (page - 1) * perPage, Limit: perPage,
	})
	if err != nil {
		return nil, nil, Internal("load users", err)
	}
	out := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		resp := ToUserResponse(&users[i])
		resp.CreatedAt = formatTime(users[i].CreatedAt)
		if users[i].LastLoginAt != nil {
			resp.LastLoginAt = formatTime(*users[i].LastLoginAt)
		}
		out = append(out, resp)
	}
	return out, newMeta(page, perPage, total), nil
}

// ListClients returns builder accounts with their company details.
func (s *AdminService) ListClients(ctx context.Context, search string, page, perPage int) ([]dto.ClientResponse, *dto.Meta, error) {
	page, perPage = normalizePage(page, perPage)
	clients, total, err := s.accounts.ListClients(ctx, search, (page-1)*perPage, perPage)
	if err != nil {
		return nil, nil, Internal("load clients", err)
	}
	out := make([]dto.ClientResponse, 0, len(clients))
	for i := range clients {
		out = append(out, ToClientResponse(&clients[i]))
	}
	return out, newMeta(page, perPage, total), nil
}

// UpdateUserStatus suspends or reactivates an account and, on suspension, ends
// its active sessions.
func (s *AdminService) UpdateUserStatus(ctx context.Context, userID, adminID uint, req dto.UpdateUserStatusRequest, meta RequestMeta) error {
	if userID == adminID {
		return Invalid("You cannot change the status of your own account")
	}
	target, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("User not found")
		}
		return Internal("load user", err)
	}
	for _, r := range target.Roles {
		if r.Code == "ADMIN" {
			return Forbidden("Administrator accounts cannot be suspended from this screen")
		}
	}
	if err := s.users.UpdateFields(ctx, userID, map[string]any{"status": req.Status}); err != nil {
		return Internal("update user status", err)
	}
	s.audit(ctx, adminID, AuditUserStatusChange, "user", userID,
		map[string]any{"status": req.Status, "reason": req.Reason}, meta)
	return nil
}

// AssignRoles replaces a user's roles.
func (s *AdminService) AssignRoles(ctx context.Context, userID, adminID uint, req dto.AssignRoleRequest, meta RequestMeta) error {
	if err := s.users.SetRoles(ctx, userID, req.RoleIDs); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("User or role not found")
		}
		return Internal("assign roles", err)
	}
	s.audit(ctx, adminID, AuditUserRoleChange, "user", userID,
		map[string]any{"role_ids": req.RoleIDs}, meta)
	return nil
}

// ListRoles returns the role catalogue for the assignment UI.
func (s *AdminService) ListRoles(ctx context.Context) ([]dto.RoleResponse, error) {
	roles, err := s.users.ListRoles(ctx)
	if err != nil {
		return nil, Internal("load roles", err)
	}
	out := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		out = append(out, dto.RoleResponse{ID: r.ID, Code: r.Code, Name: r.Name})
	}
	return out, nil
}

// ListAuditLogs returns the administrative action trail.
func (s *AdminService) ListAuditLogs(ctx context.Context, page, perPage int) ([]dto.AuditLogResponse, *dto.Meta, error) {
	page, perPage = normalizePage(page, perPage)
	logs, total, err := s.accounts.ListAudit(ctx, (page-1)*perPage, perPage)
	if err != nil {
		return nil, nil, Internal("load audit logs", err)
	}
	out := make([]dto.AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		out = append(out, dto.AuditLogResponse{
			ID: l.ID, ActorID: l.ActorID, Action: l.Action, Entity: l.Entity,
			EntityID: l.EntityID, Details: l.Details, IPAddress: l.IPAddress,
			CreatedAt: formatTime(l.CreatedAt),
		})
	}
	return out, newMeta(page, perPage, total), nil
}

// audit records a privileged action. A logging failure never blocks the action
// itself, but it is surfaced in the application log.
func (s *AdminService) audit(ctx context.Context, actorID uint, action, entity string, entityID uint, details map[string]any, meta RequestMeta) {
	payload, err := json.Marshal(details)
	if err != nil {
		payload = []byte("{}")
	}
	if err := s.accounts.WriteAudit(ctx, &models.AuditLog{
		ActorID: actorID, Action: action, Entity: entity, EntityID: entityID,
		Details: string(payload), IPAddress: meta.IP,
	}); err != nil {
		logger.Log.Error("could not write audit log", "action", action, "entity_id", entityID, "error", err)
	}
}

// notifyBuilder informs the owning builder about a moderation decision.
func (s *AdminService) notifyBuilder(ctx context.Context, p *models.Property, title, body, link string) {
	client, err := s.accounts.GetClient(ctx, p.ClientID)
	if err != nil {
		logger.Log.Warn("could not load client for notification", "client_id", p.ClientID, "error", err)
		return
	}
	if err := s.accounts.CreateNotification(ctx, &models.Notification{
		UserID: client.UserID, Type: models.NotificationTypeInApp,
		Title: title, Message: body, Link: link,
	}); err != nil {
		logger.Log.Warn("could not create notification", "error", err)
	}
	to := client.ContactEmail
	if to == "" && client.User.ID != 0 {
		to = client.User.Email
	}
	if to == "" {
		return
	}
	if err := s.notifier.Send(ctx, notification.ChannelEmail, notification.Message{
		To: to, Subject: title, Body: body,
	}); err != nil {
		logger.Log.Warn("notification email failed", "error", err)
	}
}

// SetClientVerification records how far a listing account has been checked.
// The level drives the trust badge on every listing that account posts, so it
// is an administrator decision with an audit entry, never a self-service claim.
//
// It deliberately does not accept or store identity documents: collecting those
// would need consent, retention and access controls this platform does not yet
// have, so an administrator records the outcome of a check made elsewhere.
func (s *AdminService) SetClientVerification(ctx context.Context, clientID, adminID uint, req dto.SetVerificationRequest, meta RequestMeta) (*dto.ClientResponse, error) {
	client, err := s.accounts.GetClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Listing account not found")
		}
		return nil, Internal("load listing account", err)
	}

	level := models.VerificationLevel(strings.ToUpper(req.VerificationLevel))
	previous := client.VerificationLevel
	if previous == level {
		return nil, Invalid("This account is already at that verification level")
	}

	now := time.Now()
	client.VerificationLevel = level
	client.VerifiedAt = &now
	client.VerifiedBy = &adminID
	if err := s.accounts.UpdateClient(ctx, client); err != nil {
		return nil, Internal("update verification level", err)
	}

	s.audit(ctx, adminID, "CLIENT_VERIFICATION_CHANGED", "client", clientID, map[string]any{
		"from": string(previous), "to": string(level), "note": req.Note,
	}, meta)

	// The owner should know their badge changed, in both directions.
	title := "Your account is now " + humanizeEnum(string(level))
	body := req.Note
	if body == "" {
		body = "An administrator updated the verification status of your listing account."
	}
	if err := s.accounts.CreateNotification(ctx, &models.Notification{
		UserID: client.UserID, Type: models.NotificationTypeInApp,
		Title: title, Message: body, Link: "/client/profile",
	}); err != nil {
		logger.Log.Warn("could not notify account of verification change",
			"client_id", clientID, "error", err)
	}

	resp := ToClientResponse(client)
	return &resp, nil
}
