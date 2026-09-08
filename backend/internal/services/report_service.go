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
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// duplicateReportWindow collapses repeat complaints from the same visitor about
// the same listing. A refresh must not inflate the count that drives moderation
// priority.
const duplicateReportWindow = 24 * time.Hour

// ReportService owns abuse reports on listings. Once anyone can post a flat,
// some listings are wrong or dishonest, and the people best placed to notice a
// flat that is already let are the ones still searching for it.
type ReportService struct {
	reports  repositories.ReportRepository
	props    repositories.PropertyRepository
	accounts repositories.AccountRepository
	users    repositories.UserRepository
}

func NewReportService(
	reports repositories.ReportRepository,
	props repositories.PropertyRepository,
	accounts repositories.AccountRepository,
	users repositories.UserRepository,
) *ReportService {
	return &ReportService{reports: reports, props: props, accounts: accounts, users: users}
}

// Report files a complaint against a published listing.
func (s *ReportService) Report(ctx context.Context, req dto.CreateReportRequest, v Visitor) (*dto.ReportResponse, error) {
	p, err := s.props.GetByID(ctx, req.PropertyID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	// Only a listing the public can see can be reported. Reporting a draft
	// would confirm that a listing exists at an id the reporter cannot read.
	if !p.IsPublic() {
		return nil, NotFound("Property not found")
	}

	count, err := s.reports.RecentFromReporter(ctx, p.ID, v.UserID, v.SessionID, duplicateReportWindow)
	if err != nil {
		return nil, Internal("check existing reports", err)
	}
	if count > 0 {
		return nil, Conflict("You have already reported this listing. Our team is looking into it.")
	}

	report := &models.PropertyReport{
		PropertyID: p.ID,
		ReporterID: v.UserID,
		Reason:     models.ReportReason(strings.ToUpper(req.Reason)),
		Details:    strings.TrimSpace(req.Details),
		SessionID:  v.SessionID,
		IPAddress:  v.IP,
		Status:     models.ReportOpen,
	}
	if err := s.reports.Create(ctx, report); err != nil {
		return nil, Internal("record report", err)
	}

	s.notifyModerators(ctx, p, report)
	return s.toResponse(ctx, report, p), nil
}

// List returns the moderation queue, oldest complaint first.
func (s *ReportService) List(ctx context.Context, q dto.ReportListQuery) ([]dto.ReportResponse, *dto.Meta, error) {
	page, perPage := normalizePage(q.Page, q.PerPage)
	rows, total, err := s.reports.List(ctx, repositories.ReportFilter{
		PropertyID: q.PropertyID,
		Status:     strings.ToUpper(q.Status),
		Reason:     strings.ToUpper(q.Reason),
		Offset:     (page - 1) * perPage,
		Limit:      perPage,
	})
	if err != nil {
		return nil, nil, Internal("load reports", err)
	}

	out := make([]dto.ReportResponse, 0, len(rows))
	for i := range rows {
		out = append(out, *s.toResponse(ctx, &rows[i], &rows[i].Property))
	}
	return out, newMeta(page, perPage, total), nil
}

// Resolve records an administrator's decision on a complaint, and optionally
// takes the listing down in the same step — which is the usual outcome when a
// complaint is upheld.
func (s *ReportService) Resolve(ctx context.Context, id, adminID uint, req dto.ResolveReportRequest, meta RequestMeta) (*dto.ReportResponse, error) {
	report, err := s.reports.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Report not found")
		}
		return nil, Internal("load report", err)
	}

	status := models.ReportStatus(strings.ToUpper(req.Status))
	if report.Status == status && strings.TrimSpace(req.ActionNote) == "" && !req.Unpublish {
		return nil, Invalid("This report is already in that state")
	}

	now := time.Now()
	report.Status = status
	report.ActionNote = strings.TrimSpace(req.ActionNote)
	report.ReviewedBy = &adminID
	report.ReviewedAt = &now
	if err := s.reports.Update(ctx, report); err != nil {
		return nil, Internal("update report", err)
	}

	if req.Unpublish {
		// A listing taken down over a complaint goes back to the owner as
		// REJECTED with the reason attached, so they can see what was wrong and
		// fix it, rather than the listing silently vanishing.
		reason := report.ActionNote
		if reason == "" {
			reason = "Taken down after a report: " + humanizeEnum(string(report.Reason))
		}
		if err := s.props.ChangeStatus(ctx, report.PropertyID,
			models.PropertyStatusRejected, adminID, reason); err != nil {
			return nil, Internal("unpublish reported listing", err)
		}
	}

	s.record(ctx, adminID, "REPORT_RESOLVED", report.ID, map[string]any{
		"status": string(status), "property_id": report.PropertyID,
		"unpublished": req.Unpublish,
	}, meta)

	return s.toResponse(ctx, report, &report.Property), nil
}

// OpenCount backs the badge on the administrator's dashboard.
func (s *ReportService) OpenCount(ctx context.Context) (int64, error) {
	count, err := s.reports.Count(ctx, string(models.ReportOpen))
	if err != nil {
		return 0, Internal("count open reports", err)
	}
	return count, nil
}

func (s *ReportService) toResponse(ctx context.Context, r *models.PropertyReport, p *models.Property) *dto.ReportResponse {
	out := &dto.ReportResponse{
		ID: r.ID, PropertyID: r.PropertyID,
		Reason: string(r.Reason), Details: r.Details,
		Status: string(r.Status), ActionNote: r.ActionNote,
		CreatedAt: formatTime(r.CreatedAt),
	}
	if r.ReviewedAt != nil {
		out.ReviewedAt = formatTime(*r.ReviewedAt)
	}
	if p != nil && p.ID != 0 {
		out.Property = &dto.PropertySummary{
			ID: p.ID, Name: p.Name, Slug: p.Slug,
		}
	}
	if open, err := s.reports.CountForProperty(ctx, r.PropertyID, string(models.ReportOpen)); err == nil {
		out.OpenReports = open
	}
	return out
}

// notifyModerators is best effort: a report that is filed but not announced is
// still in the queue, so a notification failure must not fail the request.
func (s *ReportService) notifyModerators(ctx context.Context, p *models.Property, r *models.PropertyReport) {
	admins, _, err := s.users.List(ctx, repositories.UserFilter{RoleCode: "ADMIN", Limit: 50})
	if err != nil {
		logger.Log.Warn("could not load moderators for report notification", "error", err)
		return
	}
	title := "Listing reported: " + p.Name
	body := fmt.Sprintf("%s was reported as %s.", p.Name, humanizeEnum(string(r.Reason)))
	for _, admin := range admins {
		if err := s.accounts.CreateNotification(ctx, &models.Notification{
			UserID: admin.ID, Type: models.NotificationTypeInApp,
			Title: title, Message: body, Link: "/admin/reports",
		}); err != nil {
			logger.Log.Warn("could not create report notification", "user_id", admin.ID, "error", err)
		}
	}
}

func (s *ReportService) record(ctx context.Context, actorID uint, action string, entityID uint, details map[string]any, meta RequestMeta) {
	payload, err := json.Marshal(details)
	if err != nil {
		payload = []byte("{}")
	}
	if err := s.accounts.WriteAudit(ctx, &models.AuditLog{
		ActorID: actorID, Action: action, Entity: "property_report", EntityID: entityID,
		Details: string(payload), IPAddress: meta.IP,
	}); err != nil {
		logger.Log.Error("could not write audit log", "action", action, "entity_id", entityID, "error", err)
	}
}
