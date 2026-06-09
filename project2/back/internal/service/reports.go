package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
)

type ReportsService struct {
	reports       *repo.Reports
	notifications *repo.Notifications
	audit         *repo.AuditLog
	adminLogs     *repo.AdminEventLogs
	users         *repo.Users
	trust         *TrustService
	monuments     *repo.Monuments
	posts         *repo.Posts
	photos        *repo.Photos
	signalPhotos  *repo.SignalPhotos
	signals       *repo.Signals
	comments      *repo.SignalComments
	monumentSvc   *MonumentsService
}

type ReportsDeps struct {
	Reports       *repo.Reports
	Notifications *repo.Notifications
	Audit         *repo.AuditLog
	AdminLogs     *repo.AdminEventLogs
	Users         *repo.Users
	Trust         *TrustService
	Monuments     *repo.Monuments
	Posts         *repo.Posts
	Photos        *repo.Photos
	SignalPhotos  *repo.SignalPhotos
	Signals       *repo.Signals
	Comments      *repo.SignalComments
	MonumentSvc   *MonumentsService
}

func NewReportsService(deps ReportsDeps) *ReportsService {
	return &ReportsService{
		reports:       deps.Reports,
		notifications: deps.Notifications,
		audit:         deps.Audit,
		adminLogs:     deps.AdminLogs,
		users:         deps.Users,
		trust:         deps.Trust,
		monuments:     deps.Monuments,
		posts:         deps.Posts,
		photos:        deps.Photos,
		signalPhotos:  deps.SignalPhotos,
		signals:       deps.Signals,
		comments:      deps.Comments,
		monumentSvc:   deps.MonumentSvc,
	}
}

type CreateReportInput struct {
	ReporterID        uuid.UUID
	EntityType        string
	EntityID          uuid.UUID
	ReasonCode        string
	Comment           string
	SuggestedTitle    string
	SuggestedLon      *float64
	SuggestedLat      *float64
	DuplicateTargetID *uuid.UUID
}

type CreateReportOutput struct {
	CaseID  uuid.UUID `json:"case_id"`
	VoteID  uuid.UUID `json:"vote_id"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

type ModerateReportCaseInput struct {
	CaseID            uuid.UUID
	ActorID           uuid.UUID
	Action            string
	ModeratorComment  string
	EditedContent     string
	ReturnStatus      string
	FixPayload        map[string]any
	TargetPartType    string
	TargetPartID      *uuid.UUID
	DuplicateTargetID *uuid.UUID
}

type reportReasonMeta struct {
	Label                 string
	Category              string
	RequiresComment       bool
	RequiresStructuredFix bool
	AvailableActions      []string
}

var allowedReasons = map[string]map[string]reportReasonMeta{
	"monument": {
		"wrong_name":       {Label: "Неверное название", Category: "integrity", RequiresStructuredFix: true, AvailableActions: []string{"apply_structural_fix", "edit_and_return", "reject_case"}},
		"wrong_coords":     {Label: "Неверные координаты", Category: "integrity", RequiresStructuredFix: true, AvailableActions: []string{"apply_structural_fix", "reject_case"}},
		"duplicate":        {Label: "Дубликат", Category: "integrity", RequiresStructuredFix: true, AvailableActions: []string{"merge_duplicate", "reject_case"}},
		"wrong_photo":      {Label: "Нерелевантные фото", Category: "integrity", AvailableActions: []string{"hide_part", "reject_case"}},
		"fake_object":      {Label: "Фейковый объект", Category: "abuse", AvailableActions: []string{"hide_entity", "return_for_revision", "reject_case"}},
		"offensive_object": {Label: "Оскорбительный объект", Category: "abuse", AvailableActions: []string{"hide_entity", "return_for_revision", "reject_case"}},
		"other":            {Label: "Другое", Category: "abuse", RequiresComment: true, AvailableActions: []string{"hide_entity", "reject_case"}},
	},
	"post": {
		"false_info":             {Label: "Ложная информация", Category: "abuse", AvailableActions: []string{"hide_entity", "edit_and_return", "reject_case"}},
		"offensive":              {Label: "Оскорбительный контент", Category: "abuse", AvailableActions: []string{"hide_entity", "edit_and_return", "reject_case"}},
		"spam":                   {Label: "Спам", Category: "abuse", AvailableActions: []string{"hide_entity", "reject_case"}},
		"flood":                  {Label: "Флуд", Category: "abuse", AvailableActions: []string{"hide_entity", "reject_case"}},
		"duplicate":              {Label: "Дубликат", Category: "integrity", AvailableActions: []string{"hide_entity", "reject_case"}},
		"irrelevant_to_monument": {Label: "Не относится к памятнику", Category: "abuse", AvailableActions: []string{"hide_entity", "edit_and_return", "reject_case"}},
		"other":                  {Label: "Другое", Category: "abuse", RequiresComment: true, AvailableActions: []string{"hide_entity", "reject_case"}},
	},
	"photo": {
		"not_relevant": {Label: "Фото не относится к объекту", Category: "abuse", AvailableActions: []string{"hide_part", "reject_case"}},
		"duplicate":    {Label: "Дубликат", Category: "integrity", AvailableActions: []string{"hide_part", "reject_case"}},
		"offensive":    {Label: "Оскорбительное фото", Category: "abuse", AvailableActions: []string{"hide_part", "hide_entity", "reject_case"}},
		"low_quality":  {Label: "Плохое качество", Category: "abuse", AvailableActions: []string{"hide_part", "reject_case"}},
		"private_data": {Label: "Личные данные", Category: "abuse", AvailableActions: []string{"hide_part", "hide_entity", "reject_case"}},
		"other":        {Label: "Другое", Category: "abuse", RequiresComment: true, AvailableActions: []string{"hide_part", "reject_case"}},
	},
	"signal": {
		"false_threat": {Label: "Ложная угроза", Category: "abuse", AvailableActions: []string{"hide_entity", "edit_and_return", "reject_case"}},
		"spam":         {Label: "Спам", Category: "abuse", AvailableActions: []string{"hide_entity", "reject_case"}},
		"offensive":    {Label: "Оскорбительный контент", Category: "abuse", AvailableActions: []string{"hide_entity", "edit_and_return", "reject_case"}},
		"duplicate":    {Label: "Дубликат", Category: "integrity", AvailableActions: []string{"hide_entity", "reject_case"}},
		"manipulation": {Label: "Манипуляция", Category: "abuse", AvailableActions: []string{"hide_entity", "edit_and_return", "reject_case"}},
		"other":        {Label: "Другое", Category: "abuse", RequiresComment: true, AvailableActions: []string{"hide_entity", "reject_case"}},
	},
	"comment": {
		"offensive":   {Label: "Оскорбление", Category: "abuse", AvailableActions: []string{"hide_part", "edit_and_return", "reject_case"}},
		"spam":        {Label: "Спам", Category: "abuse", AvailableActions: []string{"hide_part", "reject_case"}},
		"offtopic":    {Label: "Не по теме", Category: "abuse", AvailableActions: []string{"hide_part", "reject_case"}},
		"harassment":  {Label: "Травля", Category: "abuse", AvailableActions: []string{"hide_part", "edit_and_return", "reject_case"}},
		"provocation": {Label: "Провокация", Category: "abuse", AvailableActions: []string{"hide_part", "edit_and_return", "reject_case"}},
		"other":       {Label: "Другое", Category: "abuse", RequiresComment: true, AvailableActions: []string{"hide_part", "reject_case"}},
	},
}

func isAllowedEntityType(v string) bool {
	switch v {
	case "monument", "post", "photo", "signal", "comment":
		return true
	default:
		return false
	}
}

func (s *ReportsService) reasonMeta(entityType, reasonCode string) reportReasonMeta {
	if items, ok := allowedReasons[entityType]; ok {
		if meta, ok := items[reasonCode]; ok {
			return meta
		}
	}
	return reportReasonMeta{}
}

func (s *ReportsService) reasonCategory(entityType, reasonCode string) string {
	meta := s.reasonMeta(entityType, reasonCode)
	if meta.Category == "" {
		return "abuse"
	}
	return meta.Category
}

func (s *ReportsService) reasonSeverity(reasonCode string) string {
	switch reasonCode {
	case "offensive", "offensive_object", "fake_object", "false_threat", "private_data", "harassment":
		return "high"
	case "wrong_coords", "duplicate", "wrong_name":
		return "medium"
	default:
		return "low"
	}
}

func priorityFor(category, severity string) int {
	base := 10
	if category == "abuse" {
		base = 30
	}
	switch severity {
	case "high":
		return base + 30
	case "medium":
		return base + 20
	default:
		return base + 10
	}
}

func (s *ReportsService) validateCreateInput(in CreateReportInput) error {
	fields := map[string]string{}
	in.EntityType = strings.TrimSpace(in.EntityType)
	in.ReasonCode = strings.TrimSpace(in.ReasonCode)
	in.Comment = strings.TrimSpace(in.Comment)
	in.SuggestedTitle = strings.TrimSpace(in.SuggestedTitle)

	if in.EntityID == uuid.Nil {
		fields["entity_id"] = "required"
	}
	if in.EntityType == "" {
		fields["entity_type"] = "required"
	} else if !isAllowedEntityType(in.EntityType) {
		fields["entity_type"] = "invalid"
	}
	if in.ReasonCode == "" {
		fields["reason_code"] = "required"
	} else if reasons, ok := allowedReasons[in.EntityType]; !ok {
		fields["reason_code"] = "invalid"
	} else if _, exists := reasons[in.ReasonCode]; !exists {
		fields["reason_code"] = "invalid"
	}

	meta := s.reasonMeta(in.EntityType, in.ReasonCode)
	if meta.RequiresComment && in.Comment == "" {
		fields["comment"] = "required"
	}
	if in.ReasonCode == "wrong_name" && in.SuggestedTitle == "" {
		fields["suggested_title"] = "required"
	}
	if in.ReasonCode == "wrong_coords" && (in.SuggestedLon == nil || in.SuggestedLat == nil) {
		fields["suggested_coords"] = "required"
	}
	if in.ReasonCode == "duplicate" && in.DuplicateTargetID == nil && in.EntityType == "monument" {
		fields["duplicate_target_id"] = "required"
	}
	if len(fields) > 0 {
		return apierr.Error{Code: "validation_failed", Message: "Некорректные данные", Fields: fields}
	}
	return nil
}

func (s *ReportsService) buildSuggestedFix(in CreateReportInput) map[string]any {
	out := map[string]any{}
	if in.SuggestedTitle != "" {
		out["suggested_title"] = in.SuggestedTitle
	}
	if in.SuggestedLon != nil && in.SuggestedLat != nil {
		out["suggested_lon"] = *in.SuggestedLon
		out["suggested_lat"] = *in.SuggestedLat
	}
	if in.DuplicateTargetID != nil {
		out["duplicate_target_id"] = in.DuplicateTargetID.String()
	}
	return out
}

func (s *ReportsService) buildEntitySnapshot(ctx context.Context, entityType string, entityID uuid.UUID) (map[string]any, uuid.UUID, error) {
	switch entityType {
	case "monument":
		m, err := s.monuments.GetByID(ctx, entityID)
		if err != nil {
			return nil, uuid.Nil, err
		}
		posts, _ := s.posts.ListByMonument(ctx, entityID)
		authorID := uuid.Nil
		if m.AuthorID != nil {
			authorID = *m.AuthorID
		}
		return map[string]any{
			"id":                         m.ID.String(),
			"type":                       "monument",
			"name":                       m.Name,
			"lat":                        m.Lat,
			"lon":                        m.Lon,
			"region":                     m.Region,
			"status":                     m.Status,
			"author_id":                  authorID.String(),
			"author_name":                m.AuthorName,
			"is_orphaned":                m.IsOrphaned,
			"orphaned_at":                m.OrphanedAt,
			"thumbnail":                  m.Thumbnail,
			"created_at":                 m.CreatedAt,
			"ai_flags":                   m.AIFlags,
			"linked_posts_count":         len(posts),
			"public_posts_count":         s.countPublicPosts(posts),
			"linked_other_authors_count": s.countOtherAuthors(posts, authorID),
			"link":                       entityLink("monument", m.ID),
		}, authorID, nil
	case "post":
		p, err := s.posts.GetByID(ctx, entityID)
		if err != nil {
			return nil, uuid.Nil, err
		}
		return map[string]any{
			"id":            p.ID.String(),
			"type":          "post",
			"description":   p.Description,
			"status":        p.Status,
			"is_hidden":     p.IsHidden,
			"author_id":     p.AuthorID.String(),
			"author_name":   p.AuthorName,
			"monument_id":   p.MonumentID.String(),
			"monument_name": p.MonumentName,
			"thumbnail":     p.Thumbnail,
			"created_at":    p.CreatedAt,
			"ai_flags":      p.AIFlags,
			"link":          entityLink("monument", p.MonumentID),
		}, p.AuthorID, nil
	case "photo":
		if s.photos != nil {
			ph, err := s.photos.GetByID(ctx, entityID)
			if err == nil {
				post, postErr := s.posts.GetByID(ctx, ph.PostID)
				if postErr != nil {
					return nil, uuid.Nil, postErr
				}
				return map[string]any{
					"id":            ph.ID.String(),
					"type":          "photo",
					"source":        "post",
					"post_id":       ph.PostID.String(),
					"author_id":     post.AuthorID.String(),
					"author_name":   post.AuthorName,
					"monument_id":   post.MonumentID.String(),
					"monument_name": post.MonumentName,
					"thumbnail":     ph.ThumbnailPath,
					"preview":       ph.PreviewPath,
					"is_hidden":     ph.IsHidden,
					"created_at":    ph.UploadedAt,
					"ai_flags":      ph.AIFlags,
					"link":          entityLink("monument", post.MonumentID),
				}, post.AuthorID, nil
			} else if !errors.Is(err, repo.ErrNotFound) {
				return nil, uuid.Nil, err
			}
		}
		sph, err := s.signalPhotos.GetByID(ctx, entityID)
		if err != nil {
			return nil, uuid.Nil, err
		}
		sig, err := s.signals.GetByID(ctx, sph.SignalID, nil)
		if err != nil {
			return nil, uuid.Nil, err
		}
		authorID := uuid.Nil
		if sig.AuthorID != nil {
			authorID = *sig.AuthorID
		}
		return map[string]any{
			"id":            sph.ID.String(),
			"type":          "photo",
			"source":        "signal",
			"signal_id":     sph.SignalID.String(),
			"author_id":     authorID.String(),
			"author_name":   sig.AuthorName,
			"monument_id":   stringifyUUIDPtr(sig.MonumentID),
			"monument_name": stringifyStringPtr(sig.MonumentName),
			"region":        sig.Region,
			"thumbnail":     sph.ThumbnailPath,
			"preview":       sph.PreviewPath,
			"is_hidden":     sph.IsHidden,
			"created_at":    sph.UploadedAt,
			"ai_flags":      sph.AIFlags,
			"link":          entityLink("signal", sph.SignalID),
		}, authorID, nil
	case "signal":
		sig, err := s.signals.GetByID(ctx, entityID, nil)
		if err != nil {
			return nil, uuid.Nil, err
		}
		authorID := uuid.Nil
		if sig.AuthorID != nil {
			authorID = *sig.AuthorID
		}
		return map[string]any{
			"id":            sig.ID.String(),
			"type":          "signal",
			"signal_type":   sig.SignalType,
			"description":   sig.Description,
			"status":        sig.Status,
			"urgency":       sig.Urgency,
			"region":        sig.Region,
			"author_id":     authorID.String(),
			"author_name":   sig.AuthorName,
			"monument_id":   stringifyUUIDPtr(sig.MonumentID),
			"monument_name": stringifyStringPtr(sig.MonumentName),
			"lat":           derefFloat(sig.Lat),
			"lon":           derefFloat(sig.Lon),
			"thumbnail":     sig.Thumbnail,
			"created_at":    sig.CreatedAt,
			"resolution_kind": stringifyStringPtr(sig.ResolutionKind),
			"resolution_comment": stringifyStringPtr(sig.ResolutionComment),
			"ai_flags":      sig.AIFlags,
			"link":          entityLink("signal", sig.ID),
		}, authorID, nil
	case "comment":
		c, err := s.comments.GetByID(ctx, entityID)
		if err != nil {
			return nil, uuid.Nil, err
		}
		return map[string]any{
			"id":          c.ID.String(),
			"type":        "comment",
			"signal_id":   c.SignalID.String(),
			"author_id":   c.AuthorID.String(),
			"author_name": c.AuthorName,
			"content":     c.Content,
			"is_hidden":   c.IsHidden,
			"created_at":  c.CreatedAt,
			"toxic_score": c.ToxicScore,
			"edited_at":   c.EditedAt,
			"deleted_at":  c.DeletedAt,
			"link":        entityLink("signal", c.SignalID),
		}, c.AuthorID, nil
	default:
		return nil, uuid.Nil, apierr.Error{Code: "validation_failed", Message: "unsupported entity type"}
	}
}

func stringifyUUIDPtr(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func stringifyStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func (s *ReportsService) Create(ctx context.Context, in CreateReportInput) (CreateReportOutput, error) {
	in.EntityType = strings.TrimSpace(in.EntityType)
	in.ReasonCode = strings.TrimSpace(in.ReasonCode)
	in.Comment = strings.TrimSpace(in.Comment)
	if err := s.validateCreateInput(in); err != nil {
		return CreateReportOutput{}, err
	}
	exists, err := s.reports.HasActiveVote(ctx, in.ReporterID, in.EntityType, in.EntityID, in.ReasonCode)
	if err != nil {
		return CreateReportOutput{}, err
	}
	if exists {
		return CreateReportOutput{}, apierr.Error{
			Code:    "validation_failed",
			Message: "Такая жалоба уже отправлена",
			Fields:  map[string]string{"reason_code": "already_reported"},
		}
	}

	snapshot, authorID, err := s.buildEntitySnapshot(ctx, in.EntityType, in.EntityID)
	if err != nil {
		return CreateReportOutput{}, err
	}
	if authorID != uuid.Nil && authorID == in.ReporterID {
		return CreateReportOutput{}, apierr.Error{
			Code:    "validation_failed",
			Message: "Нельзя жаловаться на свой контент",
			Fields:  map[string]string{"entity_id": "own_content"},
		}
	}

	category := s.reasonCategory(in.EntityType, in.ReasonCode)
	severity := s.reasonSeverity(in.ReasonCode)
	priority := priorityFor(category, severity)
	commentPtr := (*string)(nil)
	if in.Comment != "" {
		commentPtr = &in.Comment
	}
	reportCase, vote, err := s.reports.CreateVote(ctx, repo.CreateReportVoteParams{
		EntityType:     in.EntityType,
		EntityID:       in.EntityID,
		ReporterID:     in.ReporterID,
		ReasonCode:     in.ReasonCode,
		Category:       category,
		Severity:       severity,
		Comment:        commentPtr,
		EntitySnapshot: snapshot,
		SuggestedFix:   s.buildSuggestedFix(in),
		PriorityScore:  priority,
	})
	if err != nil {
		return CreateReportOutput{}, err
	}

	if err := s.handleAutoCaseState(ctx, reportCase); err != nil {
		return CreateReportOutput{}, err
	}

	if s.adminLogs != nil {
		entityID := in.EntityID
		_, _ = s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
			ActorUserID:  &in.ReporterID,
			TargetUserID: &in.ReporterID,
			EntityType:   in.EntityType,
			EntityID:     &entityID,
			Action:       "создание_жалобы",
			Result:       "success",
			Message:      "Пользователь отправил жалобу на объект системы",
			Meta: map[string]any{
				"case_id":      reportCase.ID.String(),
				"reason_code":  in.ReasonCode,
				"report_vote":  vote.ID.String(),
				"entity_type":  in.EntityType,
			},
		})
	}

	return CreateReportOutput{
		CaseID:  reportCase.ID,
		VoteID:  vote.ID,
		Status:  reportCase.Status,
		Message: "Жалоба отправлена",
	}, nil
}

func (s *ReportsService) handleAutoCaseState(ctx context.Context, reportCase repo.ReportCase) error {
	if reportCase.EntityType == "monument" {
		return nil
	}
	if reportCase.DistinctReportersCount < 3 || reportCase.Status == "auto_hidden" {
		return nil
	}
	if err := s.hideEntity(ctx, reportCase.EntityType, reportCase.EntityID, "auto_hidden:"+reportCase.ReasonCode); err != nil {
		return err
	}
	if err := s.reports.SetCaseStatus(ctx, reportCase.ID, "auto_hidden"); err != nil {
		return err
	}
	if s.notifications != nil {
		link := "/moderation/item/reports/" + reportCase.ID.String()
		title := "Автоматическое скрытие по жалобам"
		content := fmt.Sprintf("%s скрыт после %d жалоб", reportCase.EntityType, reportCase.DistinctReportersCount)
		_ = s.notifications.CreateForRoleNames(ctx, []string{"moderator", "admin"}, "auto_hidden", title, content, &link)
	}
	return nil
}

func (s *ReportsService) hideEntity(ctx context.Context, entityType string, entityID uuid.UUID, reason string) error {
	switch entityType {
	case "post":
		return s.posts.SetHidden(ctx, entityID, true)
	case "comment":
		return s.comments.SetHidden(ctx, entityID, true)
	case "photo":
		if s.photos != nil {
			if err := s.photos.SetHidden(ctx, entityID, true); err == nil {
				return nil
			} else if !errors.Is(err, repo.ErrNotFound) {
				return err
			}
		}
		return s.signalPhotos.SetHidden(ctx, entityID, true)
	case "signal":
		off := reason
		return s.signals.UpdateStatus(ctx, entityID, "rejected", &off, nil, nil, nil, nil)
	case "monument":
		if snapshot, _, err := s.buildEntitySnapshot(ctx, "monument", entityID); err == nil {
			if linkedPosts, ok := snapshot["linked_posts_count"].(int); ok && linkedPosts > 0 && s.monumentSvc != nil {
				return s.monumentSvc.PreserveMonumentButHideInitialContent(ctx, entityID, reason)
			}
		}
		comment := reason
		return s.monuments.SetStatus(ctx, entityID, "rejected", &comment)
	default:
		return nil
	}
}

func (s *ReportsService) ListCases(ctx context.Context, status, entityType, reasonCode, category string, page, limit int) ([]repo.ReportCase, int, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit
	items, err := s.reports.ListCases(ctx, repo.ListReportCasesFilter{
		Status:     strings.TrimSpace(status),
		EntityType: strings.TrimSpace(entityType),
		ReasonCode: strings.TrimSpace(reasonCode),
		Category:   strings.TrimSpace(category),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, page, limit, err
	}
	for i := range items {
		s.decorateCase(&items[i])
	}
	return items, page, limit, nil
}

func (s *ReportsService) GetCase(ctx context.Context, caseID uuid.UUID) (repo.ReportCase, error) {
	item, err := s.reports.GetCaseByID(ctx, caseID)
	if err != nil {
		return repo.ReportCase{}, err
	}
	s.decorateCase(&item)
	return item, nil
}

func (s *ReportsService) ListMyReports(ctx context.Context, reporterID uuid.UUID, status string, page, limit int) ([]repo.ReportVote, int, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit
	items, err := s.reports.ListVotesByReporter(ctx, repo.ListReportVotesFilter{
		ReporterID: reporterID,
		Status:     strings.TrimSpace(status),
		Limit:      limit,
		Offset:     offset,
	})
	return items, page, limit, err
}

func (s *ReportsService) ModerateCase(ctx context.Context, in ModerateReportCaseInput) error {
	in.Action = strings.TrimSpace(in.Action)
	in.ModeratorComment = strings.TrimSpace(in.ModeratorComment)
	in.EditedContent = strings.TrimSpace(in.EditedContent)
	in.TargetPartType = strings.TrimSpace(in.TargetPartType)
	if in.CaseID == uuid.Nil {
		return apierr.Error{Code: "validation_failed", Message: "Некорректные данные", Fields: map[string]string{"case_id": "required"}}
	}
	if in.Action == "" {
		in.Action = "approve"
	}

	reportCase, err := s.reports.GetCaseByID(ctx, in.CaseID)
	if err != nil {
		return err
	}

	status := "resolved"
	resolutionAction := "approve"
	switch in.Action {
	case "approve":
		resolutionAction = "approve"
	case "reject", "reject_case":
		status = "rejected"
		resolutionAction = "reject"
	case "apply_fix", "apply_structural_fix":
		resolutionAction = "apply_structural_fix"
	case "hide", "hide_entity":
		resolutionAction = "hide_entity"
	case "hide_part":
		resolutionAction = "hide_part"
	case "edit_and_return", "return_for_revision":
		resolutionAction = "edit_and_return"
	case "merge_duplicate":
		resolutionAction = "merge_duplicate"
	default:
		return apierr.Error{Code: "validation_failed", Message: "Недопустимое действие", Fields: map[string]string{"action": "invalid"}}
	}

	if status == "resolved" {
		switch resolutionAction {
		case "approve":
			if err := s.applyApprovedCase(ctx, reportCase, false); err != nil {
				return err
			}
		case "hide_entity":
			if err := s.applyApprovedCase(ctx, reportCase, true); err != nil {
				return err
			}
		case "hide_part":
			if err := s.hideReportedPart(ctx, reportCase, in.TargetPartType, in.TargetPartID); err != nil {
				return err
			}
		case "apply_structural_fix":
			if err := s.applyFix(ctx, reportCase, in.FixPayload, in.DuplicateTargetID); err != nil {
				return err
			}
		case "edit_and_return":
			if err := s.editAndReturn(ctx, reportCase, in.EditedContent, in.ModeratorComment); err != nil {
				return err
			}
		case "merge_duplicate":
			if err := s.mergeDuplicate(ctx, reportCase, in.DuplicateTargetID); err != nil {
				return err
			}
		}
	}

	commentPtr := (*string)(nil)
	if in.ModeratorComment != "" {
		commentPtr = &in.ModeratorComment
	}
	if err := s.reports.ResolveCase(ctx, reportCase.ID, in.ActorID, status, resolutionAction, commentPtr); err != nil {
		return err
	}

	if s.audit != nil {
		oldStatus := reportCase.Status
		newStatus := status
		actorID := in.ActorID
		_ = s.audit.Add(ctx, "report_case", reportCase.ID, "status", &oldStatus, &newStatus, &actorID, resolutionAction)
	}
	if status == "rejected" {
		s.notifyReportersRejected(ctx, reportCase)
	}
	return nil
}

func (s *ReportsService) applyApprovedCase(ctx context.Context, reportCase repo.ReportCase, forceHide bool) error {
	if forceHide || reportCase.Category == "abuse" {
		if err := s.hideEntity(ctx, reportCase.EntityType, reportCase.EntityID, "report:"+reportCase.ReasonCode); err != nil {
			return err
		}
	}
	for _, vote := range reportCase.Votes {
		if s.trust != nil {
			delta := 1
			reasonCode := "integrity_report_confirmed"
			comment := "Подтверждена полезная жалоба на данные"
			if reportCase.Category == "abuse" {
				delta = 2
				reasonCode = "abuse_report_helpful"
				comment = "Подтверждена жалоба на нарушение"
			}
			_ = s.trust.AdjustScore(ctx, TrustAdjustment{
				UserID:     vote.ReporterID,
				Delta:      delta,
				ReasonCode: reasonCode,
				SourceType: "report_case",
				SourceID:   &reportCase.ID,
				Comment:    comment,
			})
		}
		if s.notifications != nil {
			link := "/moderation/item/reports/" + reportCase.ID.String()
			title := "Жалоба рассмотрена"
			content := "Результат: подтверждена"
			_, _ = s.notifications.Create(ctx, vote.ReporterID, "report_status", title, content, &link)
		}
	}

	if reportCase.Category == "abuse" && s.trust != nil {
		authorID := s.authorIDFromSnapshot(reportCase.EntitySnapshot)
		if authorID != uuid.Nil {
			_ = s.trust.AdjustScore(ctx, TrustAdjustment{
				UserID:     authorID,
				Delta:      -5,
				ReasonCode: "abuse_report_confirmed",
				SourceType: "report_case",
				SourceID:   &reportCase.ID,
				Comment:    "Подтверждена жалоба на нарушение правил",
			})
		}
	}
	if s.notifications != nil {
		s.notifyAuthorResolved(ctx, reportCase, reportCase.Category == "abuse", "content_hidden_report", "")
	}
	return nil
}

func (s *ReportsService) applyFix(ctx context.Context, reportCase repo.ReportCase, fixPayload map[string]any, duplicateTargetID *uuid.UUID) error {
	if reportCase.EntityType != "monument" {
		return s.applyApprovedCase(ctx, reportCase, false)
	}
	fix := reportCase.SuggestedFix
	if len(fixPayload) > 0 {
		fix = fixPayload
	}
	if reportCase.ReasonCode == "duplicate" {
		return s.mergeDuplicate(ctx, reportCase, duplicateTargetID)
	}

	var (
		newName *string
		lon     *float64
		lat     *float64
	)
	if raw, ok := fix["suggested_title"].(string); ok && strings.TrimSpace(raw) != "" {
		value := strings.TrimSpace(raw)
		newName = &value
	}
	if rawLon, ok := fix["suggested_lon"].(float64); ok {
		lon = &rawLon
	}
	if rawLat, ok := fix["suggested_lat"].(float64); ok {
		lat = &rawLat
	}
	if newName == nil && (lon == nil || lat == nil) {
		return s.applyApprovedCase(ctx, reportCase, false)
	}
	if err := s.monuments.UpdateCoreFields(ctx, reportCase.EntityID, newName, lon, lat); err != nil {
		return err
	}
	for _, vote := range reportCase.Votes {
		if s.trust != nil {
			_ = s.trust.AdjustScore(ctx, TrustAdjustment{
				UserID:     vote.ReporterID,
				Delta:      1,
				ReasonCode: "integrity_report_confirmed",
				SourceType: "report_case",
				SourceID:   &reportCase.ID,
				Comment:    "Подтверждена полезная жалоба на данные",
			})
		}
		if s.notifications != nil {
			link := "/moderation/item/reports/" + reportCase.ID.String()
			title := "Жалоба рассмотрена"
			content := "Результат: данные исправлены"
			_, _ = s.notifications.Create(ctx, vote.ReporterID, "report_status", title, content, &link)
		}
	}
	s.notifyAuthorResolved(ctx, reportCase, false, "monument_data_fixed", "Данные памятника были исправлены модератором")
	return nil
}

func (s *ReportsService) hideReportedPart(ctx context.Context, reportCase repo.ReportCase, targetPartType string, targetPartID *uuid.UUID) error {
	switch reportCase.EntityType {
	case "comment":
		return s.comments.SetHidden(ctx, reportCase.EntityID, true)
	case "photo":
		return s.hideEntity(ctx, "photo", reportCase.EntityID, "report:"+reportCase.ReasonCode)
	case "monument":
		if reportCase.ReasonCode != "wrong_photo" || targetPartType != "photo" || targetPartID == nil {
			return apierr.Error{Code: "validation_failed", Message: "Нужно выбрать конкретное фото", Fields: map[string]string{"target_part_id": "required"}}
		}
		return s.hideEntity(ctx, "photo", *targetPartID, "report:"+reportCase.ReasonCode)
	default:
		return s.applyApprovedCase(ctx, reportCase, true)
	}
}

func (s *ReportsService) mergeDuplicate(ctx context.Context, reportCase repo.ReportCase, duplicateTargetID *uuid.UUID) error {
	if reportCase.EntityType != "monument" {
		return s.applyApprovedCase(ctx, reportCase, false)
	}
	targetID := duplicateTargetID
	if targetID == nil {
		if raw, ok := reportCase.SuggestedFix["duplicate_target_id"].(string); ok {
			if parsed, err := uuid.Parse(strings.TrimSpace(raw)); err == nil {
				targetID = &parsed
			}
		}
	}
	if targetID == nil {
		return apierr.Error{Code: "validation_failed", Message: "Не указан памятник для объединения", Fields: map[string]string{"duplicate_target_id": "required"}}
	}
	comment := "Объединено с другой точкой: " + targetID.String()
	if err := s.monuments.SetStatus(ctx, reportCase.EntityID, "rejected", &comment); err != nil {
		return err
	}
	s.notifyAuthorResolved(ctx, reportCase, true, "monument_data_fixed", "Точка была объединена с существующим памятником")
	return nil
}

func (s *ReportsService) editAndReturn(ctx context.Context, reportCase repo.ReportCase, editedContent, moderatorComment string) error {
	switch reportCase.EntityType {
	case "post":
		return s.editPostAndReturn(ctx, reportCase, editedContent, moderatorComment)
	case "comment":
		return s.editCommentAndReturn(ctx, reportCase, editedContent, moderatorComment)
	case "signal":
		return s.editSignalAndReturn(ctx, reportCase, editedContent, moderatorComment)
	default:
		return s.applyApprovedCase(ctx, reportCase, true)
	}
}

func (s *ReportsService) editPostAndReturn(ctx context.Context, reportCase repo.ReportCase, editedContent, moderatorComment string) error {
	if strings.TrimSpace(editedContent) == "" {
		return apierr.Error{Code: "validation_failed", Message: "Нужен исправленный текст", Fields: map[string]string{"edited_content": "required"}}
	}
	if err := s.posts.SetHidden(ctx, reportCase.EntityID, true); err != nil {
		return err
	}
	s.notifyAuthorResolved(ctx, reportCase, true, "content_returned_for_revision", buildRevisionMessage(moderatorComment, editedContent))
	return nil
}

func (s *ReportsService) editCommentAndReturn(ctx context.Context, reportCase repo.ReportCase, editedContent, moderatorComment string) error {
	if strings.TrimSpace(editedContent) == "" {
		return apierr.Error{Code: "validation_failed", Message: "Нужен исправленный текст", Fields: map[string]string{"edited_content": "required"}}
	}
	if err := s.comments.UpdateContent(ctx, reportCase.EntityID, editedContent, true, nil); err != nil {
		return err
	}
	s.notifyAuthorResolved(ctx, reportCase, true, "content_returned_for_revision", buildRevisionMessage(moderatorComment, editedContent))
	return nil
}

func (s *ReportsService) editSignalAndReturn(ctx context.Context, reportCase repo.ReportCase, editedContent, moderatorComment string) error {
	reason := "Контент отправлен на доработку"
	if strings.TrimSpace(moderatorComment) != "" {
		reason = moderatorComment
	}
	if err := s.hideEntity(ctx, "signal", reportCase.EntityID, reason); err != nil {
		return err
	}
	s.notifyAuthorResolved(ctx, reportCase, true, "content_returned_for_revision", buildRevisionMessage(moderatorComment, editedContent))
	return nil
}

func (s *ReportsService) notifyReportersRejected(ctx context.Context, reportCase repo.ReportCase) {
	if s.notifications == nil {
		return
	}
	for _, vote := range reportCase.Votes {
		link := "/moderation/item/reports/" + reportCase.ID.String()
		title := "Жалоба рассмотрена"
		content := "Результат: отклонена"
		_, _ = s.notifications.Create(ctx, vote.ReporterID, "report_status", title, content, &link)
	}
}

func (s *ReportsService) notifyAuthorResolved(ctx context.Context, reportCase repo.ReportCase, hidden bool, typ string, customContent string) {
	if s.notifications == nil {
		return
	}
	authorID := s.authorIDFromSnapshot(reportCase.EntitySnapshot)
	if authorID == uuid.Nil {
		return
	}
	link := entityLink(reportCase.EntityType, reportCase.EntityID)
	title := "Контент затронут жалобой"
	content := "Жалоба подтверждена"
	if typ == "" {
		typ = "content_hidden_report"
	}
	if hidden {
		content = "Контент скрыт по жалобе"
	} else if reportCase.Category == "integrity" {
		content = "Жалоба подтверждена, данные проверены или исправлены"
	}
	if strings.TrimSpace(customContent) != "" {
		content = customContent
	}
	_, _ = s.notifications.Create(ctx, authorID, typ, title, content, &link)
}

func (s *ReportsService) authorIDFromSnapshot(snapshot map[string]any) uuid.UUID {
	raw, ok := snapshot["author_id"].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func entityLink(entityType string, entityID uuid.UUID) string {
	switch entityType {
	case "monument":
		return "/monument/" + entityID.String()
	case "signal":
		return "/signal/" + entityID.String()
	default:
		return "/"
	}
}

func (s *ReportsService) decorateCase(reportCase *repo.ReportCase) {
	meta := s.reasonMeta(reportCase.EntityType, reportCase.ReasonCode)
	reportCase.AvailableActions = append([]string{}, meta.AvailableActions...)
	reportCase.CanApplyFix = reportCase.EntityType == "monument" && (reportCase.ReasonCode == "wrong_name" || reportCase.ReasonCode == "wrong_coords" || reportCase.ReasonCode == "duplicate")
	reportCase.CanHidePart = reportCase.EntityType == "photo" || reportCase.EntityType == "comment" || reportCase.ReasonCode == "wrong_photo"
	reportCase.CanHideEntity = reportCase.EntityType != "monument" || reportCase.ReasonCode == "fake_object" || reportCase.ReasonCode == "offensive_object"
	reportCase.CanEdit = reportCase.EntityType == "post" || reportCase.EntityType == "comment" || reportCase.EntityType == "signal"
	reportCase.ActionProfile = map[string]any{
		"reason_label":        meta.Label,
		"category":            meta.Category,
		"available_actions":   reportCase.AvailableActions,
		"monument_safe_hide":  reportCase.EntityType != "monument",
		"linked_posts_count":  reportCase.EntitySnapshot["linked_posts_count"],
		"other_authors_count": reportCase.EntitySnapshot["linked_other_authors_count"],
	}
}

func buildRevisionMessage(moderatorComment, editedContent string) string {
	content := "Контент скрыт и отправлен на доработку"
	if strings.TrimSpace(moderatorComment) != "" {
		content += ". Причина: " + strings.TrimSpace(moderatorComment)
	}
	if strings.TrimSpace(editedContent) != "" {
		content += ". Предлагаемая редакция: " + strings.TrimSpace(editedContent)
	}
	return content
}

func (s *ReportsService) countOtherAuthors(posts []repo.Post, authorID uuid.UUID) int {
	seen := map[uuid.UUID]struct{}{}
	for _, post := range posts {
		if post.AuthorID == uuid.Nil || post.AuthorID == authorID {
			continue
		}
		seen[post.AuthorID] = struct{}{}
	}
	return len(seen)
}

func (s *ReportsService) countPublicPosts(posts []repo.Post) int {
	count := 0
	for _, post := range posts {
		if post.Status == "approved" && !post.IsHidden && !post.IsArchived {
			count++
		}
	}
	return count
}
