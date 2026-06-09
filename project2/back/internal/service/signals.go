package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"

	_ "golang.org/x/image/webp"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
	"github.com/tochka-pamyati/tochka-pamyati/internal/moderation"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/storage"
)

type SignalsService struct {
	signals       *repo.Signals
	signalPhotos  *repo.SignalPhotos
	comments      *repo.SignalComments
	monuments     *repo.Monuments
	audit         *repo.AuditLog
	notifications *repo.Notifications
	adminLogs     *repo.AdminEventLogs
	users         *repo.Users
	sanctions     *SanctionsService
	trust         *TrustService
	geo           *GeographyService

	textChecker  moderation.TextChecker
	imageChecker moderation.ImageChecker
	uploader     storage.Uploader
}

type SignalsDeps struct {
	Signals       *repo.Signals
	SignalPhotos  *repo.SignalPhotos
	Comments      *repo.SignalComments
	Monuments     *repo.Monuments
	Audit         *repo.AuditLog
	Notifications *repo.Notifications
	AdminLogs     *repo.AdminEventLogs
	Users         *repo.Users
	Sanctions     *SanctionsService
	Trust         *TrustService
	Geography     *GeographyService

	TextChecker  moderation.TextChecker
	ImageChecker moderation.ImageChecker
	Uploader     storage.Uploader
}

func NewSignalsService(deps SignalsDeps) *SignalsService {
	return &SignalsService{
		signals:       deps.Signals,
		signalPhotos:  deps.SignalPhotos,
		comments:      deps.Comments,
		monuments:     deps.Monuments,
		audit:         deps.Audit,
		notifications: deps.Notifications,
		adminLogs:     deps.AdminLogs,
		users:         deps.Users,
		sanctions:     deps.Sanctions,
		trust:         deps.Trust,
		geo:           deps.Geography,
		textChecker:   deps.TextChecker,
		imageChecker:  deps.ImageChecker,
		uploader:      deps.Uploader,
	}
}

type CreateSignalInput struct {
	AuthorID uuid.UUID

	MonumentID *uuid.UUID

	NewMonumentName string
	NewLon          float64
	NewLat          float64

	SignalType  string
	Urgency     string
	Description string
	Photos      []*multipart.FileHeader
	ContentAck  bool
}

type CreateSignalOutput struct {
	SignalID   uuid.UUID  `json:"signal_id"`
	HighRisk   bool       `json:"high_risk"`
	MonumentID *uuid.UUID `json:"monument_id,omitempty"`
}

func (s *SignalsService) ValidateCreate(ctx context.Context, in CreateSignalInput) (ContentValidationResult, error) {
	fields := map[string]string{}

	in.SignalType = strings.TrimSpace(in.SignalType)
	in.Urgency = strings.TrimSpace(in.Urgency)
	in.Description = normalizeUserText(in.Description)

	if in.SignalType == "" {
		fields["signal_type"] = "required"
	}
	if in.Urgency == "" {
		fields["urgency"] = "required"
	}
	if in.Description == "" {
		fields["description"] = "required"
	}

	if in.MonumentID == nil {
		if strings.TrimSpace(in.NewMonumentName) == "" {
			fields["monument_name"] = "required"
		}
		if in.NewLon < -180 || in.NewLon > 180 {
			fields["lon"] = "invalid"
		}
		if in.NewLat < -90 || in.NewLat > 90 {
			fields["lat"] = "invalid"
		}
	} else {
		if _, err := s.monuments.GetByID(ctx, *in.MonumentID); err != nil {
			fields["monument_id"] = "not_found"
		}
	}

	if len(in.Photos) > 10 {
		fields["photos"] = "max_10"
	}

	if len(fields) > 0 {
		return ContentValidationResult{
			RequiresAck: false,
			Reasons:     []string{"invalid_input"},
			Fields:      fields,
			HighRisk:    false,
		}, nil
	}

	var reasons []string
	fields = map[string]string{}

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	if len(textFlags) > 0 {
		fields["description"] = "flagged"
		reasons = append(reasons, "text_flagged")
	}
	if textUnavailable {
		reasons = append(reasons, "text_filter_unavailable")
	}

	_, imageIssues, imgUnavailableAny, err := s.checkImages(ctx, in.Photos)
	if err != nil {
		return ContentValidationResult{}, err
	}
	for k, v := range imageIssues {
		fields[k] = v
		reasons = append(reasons, "image_flagged")
	}
	if imgUnavailableAny {
		reasons = append(reasons, "image_filter_unavailable")
	}

	if textScore != nil && *textScore >= 0.7 && !containsReason(reasons, "text_flagged") {
		reasons = append(reasons, "text_flagged")
	}

	highRisk := len(fields) > 0 || textUnavailable || imgUnavailableAny
	return ContentValidationResult{
		RequiresAck: len(fields) > 0,
		Reasons:     uniqueReasons(reasons),
		Fields:      fields,
		HighRisk:    highRisk,
	}, nil
}

func (s *SignalsService) Create(ctx context.Context, in CreateSignalInput) (CreateSignalOutput, error) {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeContentCreate); err != nil {
			return CreateSignalOutput{}, err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentCreate(ctx, in.AuthorID); err != nil {
			return CreateSignalOutput{}, err
		}
	}
	validation, err := s.ValidateCreate(ctx, in)
	if err != nil {
		return CreateSignalOutput{}, err
	}
	if containsReason(validation.Reasons, "invalid_input") {
		return CreateSignalOutput{}, apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: validation.Fields}
	}
	if validation.RequiresAck && !in.ContentAck {
		return CreateSignalOutput{}, apierr.Error{
			Code:    "content_requires_ack",
			Message: "content requires confirmation",
			Fields:  validation.Fields,
			Data: map[string]any{
				"requires_ack": true,
				"reasons":      validation.Reasons,
			},
		}
	}

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	imageResults, _, imgUnavailableAny, err := s.checkImages(ctx, in.Photos)
	if err != nil {
		return CreateSignalOutput{}, err
	}

	aiFlags := map[string]any{
		"reasons":           validation.Reasons,
		"text_toxic_score":  textScore,
		"text_categories":   textFlags,
		"image_results":     imageResults,
		"content_ack":       in.ContentAck,
		"filters_available": map[string]any{"text": !textUnavailable, "image": !imgUnavailableAny},
	}
	if s.trust != nil {
		if err := s.trust.ApplySubmissionTrustFlags(ctx, in.AuthorID, &validation.HighRisk, &validation.Reasons, aiFlags); err != nil {
			return CreateSignalOutput{}, err
		}
	}

	var monumentName *string
	var lonPtr *float64
	var latPtr *float64
	var region string
	if in.MonumentID == nil {
		n := strings.TrimSpace(in.NewMonumentName)
		monumentName = &n
		lon := in.NewLon
		lat := in.NewLat
		lonPtr = &lon
		latPtr = &lat
		region = s.geo.GetRegionByCoords(lon, lat)
	} else {
		mon, err := s.monuments.GetByID(ctx, *in.MonumentID)
		if err == nil {
			region = mon.Region
			monName := mon.Name
			monumentName = &monName
		}
	}

	signalID, err := s.signals.Create(ctx, repo.Signal{
		MonumentID:   in.MonumentID,
		MonumentName: monumentName,
		Lon:          lonPtr,
		Lat:          latPtr,
		SignalType:   in.SignalType,
		Urgency:      in.Urgency,
		Description:  in.Description,
		Region:       region,
		AuthorID:     &in.AuthorID,
		Status:       "pending",
		HighRisk:     validation.HighRisk,
		AIFlags:      aiFlags,
	})
	if err != nil {
		return CreateSignalOutput{}, err
	}

	if err := s.saveSignalPhotos(ctx, signalID, in.Photos, imageResults); err != nil {
		return CreateSignalOutput{}, err
	}

	if s.adminLogs != nil {
		entityID := signalID
		_, _ = s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
			ActorUserID:  &in.AuthorID,
			TargetUserID: &in.AuthorID,
			EntityType:   "signal",
			EntityID:     &entityID,
			Action:       "создание_сигнала",
			Result:       "success",
			Message:      "Пользователь создал новый сигнал",
			Meta: map[string]any{
				"signal_type": in.SignalType,
				"region":      region,
				"high_risk":   validation.HighRisk,
				"monument_id": func() string {
					if in.MonumentID == nil {
						return ""
					}
					return in.MonumentID.String()
				}(),
			},
		})
	}

	return CreateSignalOutput{SignalID: signalID, HighRisk: validation.HighRisk, MonumentID: in.MonumentID}, nil
}

type AddCommentInput struct {
	AuthorID uuid.UUID
	SignalID uuid.UUID
	ParentID *uuid.UUID
	Content  string
}

type AddCommentOutput struct {
	CommentID uuid.UUID `json:"comment_id"`
	IsHidden  bool      `json:"is_hidden"`
}

type UpdateOwnSignalInput struct {
	AuthorID    uuid.UUID
	SignalID    uuid.UUID
	SignalType  string
	Urgency     string
	Description string
}

const (
	SignalResolutionSuccessful   = "successful"
	SignalResolutionPartial      = "partial"
	SignalResolutionUnsuccessful = "unsuccessful"
)

func (s *SignalsService) AddComment(ctx context.Context, in AddCommentInput) (AddCommentOutput, error) {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeCommentWrite); err != nil {
			return AddCommentOutput{}, err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckCommentCreate(ctx, in.AuthorID); err != nil {
			return AddCommentOutput{}, err
		}
	}
	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" {
		return AddCommentOutput{}, apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"content": "required"}}
	}

	if _, err := s.signals.GetByID(ctx, in.SignalID, nil); err != nil {
		return AddCommentOutput{}, apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
	}

	if in.ParentID != nil {
		if _, err := s.comments.GetByID(ctx, *in.ParentID); err != nil {
			return AddCommentOutput{}, apierr.Error{Code: "validation_failed", Message: "parent comment not found", Fields: map[string]string{"parent_id": "not_found"}}
		}
	}

	textScore, categories, unavailable := s.checkText(ctx, in.Content)

	isHidden := false
	if !unavailable && (len(categories) > 0 || (textScore != nil && *textScore >= 0.85)) {
		isHidden = true
	}
	if s.trust != nil {
		risky, err := s.trust.IsRisky(ctx, in.AuthorID)
		if err != nil {
			return AddCommentOutput{}, err
		}
		if risky {
			isHidden = true
		}
	}

	id, err := s.comments.Create(ctx, in.SignalID, in.AuthorID, in.ParentID, in.Content, isHidden, textScore)
	if err != nil {
		return AddCommentOutput{}, err
	}
	if isHidden && s.sanctions != nil {
		_ = s.sanctions.RegisterHiddenComment(ctx, id, in.AuthorID, in.SignalID, in.Content, textScore, map[string]any{
			"source":    "create",
			"parent_id": stringifyUUIDPtr(in.ParentID),
		})
	}
	if s.adminLogs != nil {
		commentID := id
		_, _ = s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
			ActorUserID:  &in.AuthorID,
			TargetUserID: &in.AuthorID,
			EntityType:   "comment",
			EntityID:     &commentID,
			Action:       "создание_комментария",
			Result:       "success",
			Message:      "Пользователь добавил комментарий к сигналу",
			Meta: map[string]any{
				"signal_id": in.SignalID.String(),
				"is_hidden": isHidden,
			},
		})
	}
	return AddCommentOutput{CommentID: id, IsHidden: isHidden}, nil
}

type SignalDetail struct {
	Signal      repo.Signal          `json:"signal"`
	Photos      []repo.SignalPhoto   `json:"photos"`
	Comments    []repo.SignalComment `json:"comments"`
	MonumentRef *repo.Monument       `json:"monument,omitempty"`
}

func (s *SignalsService) GetDetail(ctx context.Context, signalID uuid.UUID, userID *uuid.UUID) (SignalDetail, error) {
	sig, err := s.signals.GetByID(ctx, signalID, userID)
	if err != nil {
		return SignalDetail{}, apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
	}

	photos, err := s.signalPhotos.ListBySignal(ctx, signalID)
	if err != nil {
		return SignalDetail{}, err
	}
	comments, err := s.comments.ListBySignal(ctx, signalID)
	if err != nil {
		return SignalDetail{}, err
	}

	var monumentRef *repo.Monument
	if sig.MonumentID != nil {
		m, err := s.monuments.GetByID(ctx, *sig.MonumentID)
		if err == nil {
			monumentRef = &m
		}
	}

	if photos == nil {
		photos = []repo.SignalPhoto{}
	}
	if comments == nil {
		comments = []repo.SignalComment{}
	}

	return SignalDetail{Signal: sig, Photos: photos, Comments: comments, MonumentRef: monumentRef}, nil
}

func (s *SignalsService) List(ctx context.Context, f repo.ListSignalsFilter) ([]repo.Signal, error) {
	return s.signals.List(ctx, f)
}

func (s *SignalsService) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]repo.Signal, error) {
	return s.signals.ListByAuthor(ctx, authorID)
}

func (s *SignalsService) UpdateOwnSignal(ctx context.Context, in UpdateOwnSignalInput) error {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeContentEdit); err != nil {
			return err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, in.AuthorID); err != nil {
			return err
		}
	}
	in.SignalType = strings.TrimSpace(in.SignalType)
	in.Urgency = strings.TrimSpace(in.Urgency)
	in.Description = normalizeUserText(in.Description)
	if in.SignalType == "" || in.Urgency == "" || in.Description == "" {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{
			"signal_type": "required",
			"urgency":     "required",
			"description": "required",
		}}
	}
	signal, err := s.signals.GetByID(ctx, in.SignalID, &in.AuthorID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
	}
	if signal.AuthorID == nil || *signal.AuthorID != in.AuthorID {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	highRisk := len(textFlags) > 0 || textUnavailable
	reasons := []string{}
	if len(textFlags) > 0 {
		reasons = append(reasons, "text_flagged")
	}
	if textUnavailable {
		reasons = append(reasons, "text_filter_unavailable")
	}
	aiFlags := map[string]any{
		"reasons":           uniqueReasons(reasons),
		"text_toxic_score":  textScore,
		"text_categories":   textFlags,
		"filters_available": map[string]any{"text": !textUnavailable},
	}
	if s.trust != nil {
		if err := s.trust.ApplySubmissionTrustFlags(ctx, in.AuthorID, &highRisk, &reasons, aiFlags); err != nil {
			return err
		}
		aiFlags["reasons"] = uniqueReasons(reasons)
	}
	if err := s.signals.UpdateOwn(ctx, repo.UpdateOwnSignalParams{
		ID:          in.SignalID,
		AuthorID:    in.AuthorID,
		SignalType:  in.SignalType,
		Description: in.Description,
		Urgency:     in.Urgency,
		HighRisk:    highRisk,
		AIFlags:     aiFlags,
	}); err != nil {
		return err
	}
	if s.notifications != nil {
		link := signalLink(in.SignalID)
		_, _ = s.notifications.Create(ctx, in.AuthorID, "signal_status", "Сигнал отправлен на повторную проверку", "Измененный сигнал отправлен на повторную модерацию.", &link)
	}
	return nil
}

func (s *SignalsService) DeleteOwnSignal(ctx context.Context, authorID, signalID uuid.UUID) error {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, authorID, SanctionScopeContentEdit); err != nil {
			return err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, authorID); err != nil {
			return err
		}
	}
	signal, err := s.signals.GetByID(ctx, signalID, &authorID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
	}
	if signal.AuthorID == nil || *signal.AuthorID != authorID {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}
	if err := s.signals.DeleteByAuthor(ctx, signalID, authorID); err != nil {
		return err
	}
	if s.adminLogs != nil {
		entityID := signalID
		_, _ = s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
			ActorUserID:  &authorID,
			TargetUserID: &authorID,
			EntityType:   "signal",
			EntityID:     &entityID,
			Action:       "удаление_сигнала_автором",
			Result:       "success",
			Message:      "Пользователь удалил свой сигнал",
			Meta: map[string]any{
				"region": signal.Region,
			},
		})
	}
	return nil
}

func (s *SignalsService) DeleteSignalAsAdmin(ctx context.Context, actorID, signalID uuid.UUID) error {
	signal, err := s.signals.GetByID(ctx, signalID, nil)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
	}
	photos, _ := s.signalPhotos.ListBySignalIncludeHidden(ctx, signalID)
	for _, photo := range photos {
		_ = s.uploader.Delete(ctx, photo.FilePath)
		_ = s.uploader.Delete(ctx, photo.PreviewPath)
		_ = s.uploader.Delete(ctx, photo.ThumbnailPath)
	}
	if err := s.signals.Delete(ctx, signalID); err != nil {
		return err
	}
	if s.notifications != nil && signal.AuthorID != nil {
		link := signalLink(signalID)
		_, _ = s.notifications.Create(ctx, *signal.AuthorID, "content_hidden_by_report", "Сигнал удален администратором", "Один из сигналов был удален администратором системы.", &link)
	}
	if s.adminLogs != nil {
		var target *uuid.UUID
		if signal.AuthorID != nil {
			target = signal.AuthorID
		}
		_, _ = s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
			ActorUserID:  &actorID,
			TargetUserID: target,
			EntityType:   "signal",
			EntityID:     &signalID,
			Action:       "удаление_сигнала_администратором",
			Result:       "success",
			Message:      "Администратор удалил сигнал",
			Meta: map[string]any{
				"signal_type": signal.SignalType,
				"region":      signal.Region,
			},
		})
	}
	return nil
}

func (s *SignalsService) SetOwnResolved(ctx context.Context, authorID, signalID uuid.UUID, resolved bool, resolutionKind, resolutionComment string) error {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, authorID, SanctionScopeContentEdit); err != nil {
			return err
		}
	}
	signal, err := s.signals.GetByID(ctx, signalID, &authorID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
	}
	if signal.AuthorID == nil || *signal.AuthorID != authorID {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}

	resolutionKind = strings.TrimSpace(strings.ToLower(resolutionKind))
	resolutionComment = strings.TrimSpace(resolutionComment)
	status := "confirmed"
	var resolvedAt *time.Time
	response := signal.OfficialResponse
	var resolutionKindPtr *string
	var resolutionCommentPtr *string
	if resolved {
		switch resolutionKind {
		case SignalResolutionSuccessful, SignalResolutionPartial, SignalResolutionUnsuccessful:
		default:
			return apierr.Error{Code: "validation_failed", Message: "Некорректные данные", Fields: map[string]string{"resolution_kind": "required"}}
		}
		if resolutionKind == SignalResolutionPartial && resolutionComment == "" {
			return apierr.Error{Code: "validation_failed", Message: "Некорректные данные", Fields: map[string]string{"resolution_comment": "required"}}
		}
		status = "resolved"
		now := time.Now()
		resolvedAt = &now
		resolutionKindPtr = &resolutionKind
		if resolutionComment != "" {
			resolutionCommentPtr = &resolutionComment
		}
		text := buildResolutionResponse(resolutionKind, resolutionComment)
		response = &text
	} else {
		text := "Сигнал снова открыт пользователем"
		response = &text
	}
	if err := s.signals.UpdateStatus(ctx, signalID, status, response, resolvedAt, nil, resolutionKindPtr, resolutionCommentPtr); err != nil {
		return err
	}
	if s.adminLogs != nil {
		entityID := signalID
		action := "повторное_открытие_сигнала"
		message := "Пользователь снова открыл свой сигнал"
		if resolved {
			action = "завершение_сигнала_автором"
			message = "Пользователь отметил свой сигнал как завершенный"
		}
		_, _ = s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
			ActorUserID:  &authorID,
			TargetUserID: &authorID,
			EntityType:   "signal",
			EntityID:     &entityID,
			Action:       action,
			Result:       "success",
			Message:      message,
			Meta: map[string]any{
				"resolved":           resolved,
				"resolution_kind":    resolutionKind,
				"resolution_comment": resolutionComment,
			},
		})
	}
	return nil
}

func (s *SignalsService) ListCommentsByAuthor(ctx context.Context, authorID uuid.UUID) ([]repo.SignalComment, error) {
	return s.comments.ListByAuthor(ctx, authorID)
}

func (s *SignalsService) ConfirmedMapPoints(ctx context.Context) ([]repo.SignalMapPoint, error) {
	return s.signals.ConfirmedMapPoints(ctx)
}

func (s *SignalsService) Support(ctx context.Context, signalID, userID uuid.UUID, enable bool) error {
	if enable {
		_, err := s.signals.AddSupport(ctx, signalID, userID)
		return err
	}
	_, err := s.signals.RemoveSupport(ctx, signalID, userID)
	return err
}

type ModerateSignalInput struct {
	SignalID uuid.UUID
	ActorID  uuid.UUID
	Action   string
	Comment  string
	Urgency  string
}

func (s *SignalsService) Moderate(ctx context.Context, in ModerateSignalInput) error {
	in.Action = strings.TrimSpace(strings.ToLower(in.Action))
	in.Comment = strings.TrimSpace(in.Comment)
	in.Urgency = strings.TrimSpace(strings.ToLower(in.Urgency))
	if in.Action == "" {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"action": "required"}}
	}

	switch in.Action {
	case "confirm", "approve":
		if in.Urgency != "low" && in.Urgency != "medium" && in.Urgency != "high" {
			return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"urgency": "required"}}
		}
		return s.updateSignalStatus(ctx, in.SignalID, in.ActorID, "confirmed", nil, nil, &in.Urgency, nil, nil)
	case "resolve":
		now := time.Now()
		return s.updateSignalStatus(ctx, in.SignalID, in.ActorID, "resolved", &in.Comment, &now, nil, nil, nil)
	case "reject":
		if in.Comment == "" {
			return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"comment": "required"}}
		}
		return s.updateSignalStatus(ctx, in.SignalID, in.ActorID, "rejected", &in.Comment, nil, nil, nil, nil)
	default:
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"action": "invalid"}}
	}
}

func (s *SignalsService) updateSignalStatus(ctx context.Context, signalID uuid.UUID, actorID uuid.UUID, status string, officialResponse *string, resolvedAt *time.Time, urgency *string, resolutionKind *string, resolutionComment *string) error {
	prev, err := s.signals.GetByID(ctx, signalID, nil)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
	}

	if err := s.signals.UpdateStatus(ctx, signalID, status, officialResponse, resolvedAt, urgency, resolutionKind, resolutionComment); err != nil {
		return err
	}

	// Отправляем уведомление автору
	if prev.AuthorID != nil {
		msg := "Ваш сигнал "
		if prev.MonumentID != nil {
			mon, _ := s.monuments.GetByID(ctx, *prev.MonumentID)
			msg += "по памятнику «" + mon.Name + "» "
		} else if prev.MonumentName != nil {
			msg += "«" + *prev.MonumentName + "» "
		}

		switch status {
		case "confirmed":
			msg += "был подтвержден модератором."
		case "resolved":
			msg += "был решен."
			if officialResponse != nil {
				msg += " Официальный ответ: " + *officialResponse
			}
		case "rejected":
			msg += "был отклонен модератором."
			if officialResponse != nil {
				msg += " Причина: " + *officialResponse
			}
		}

		link := signalLink(signalID)
		if prev.AuthorID != nil && *prev.AuthorID != uuid.Nil {
			_, err := s.notifications.Create(ctx, *prev.AuthorID, "signal_status", "Статус сигнала", msg, &link)
			if err != nil {
				log.Printf("[NOTIF] Failed to notify signal author %s: %v", prev.AuthorID, err)
			} else {
				log.Printf("[NOTIF] Sent status update to signal author %s", prev.AuthorID)
			}
		} else {
			log.Printf("[NOTIF] Skipping notification for signal %s: author is nil/empty", signalID)
		}
	}

	// Если сигнал подтвержден, уведомляем всех пользователей региона
	if status == "confirmed" {
		region := prev.Region
		if region != "" && region != "Неизвестный регион" {
			users, _, err := s.users.List(ctx, repo.ListUsersFilter{Region: region, Limit: 1000}) // Filter by region
			if err == nil {
				signalMsg := "Новое сообщение об угрозе в вашем регионе"
				if prev.MonumentName != nil {
					signalMsg += ": " + *prev.MonumentName
				}
				signalLinkValue := signalLink(signalID)

				for _, u := range users {
					if prev.AuthorID != nil && *prev.AuthorID == u.ID {
						continue // Don't notify double
					}
					if !notificationSettingEnabled(u.NotificationSettings, "new_signal_city") {
						continue
					}
					_, _ = s.notifications.Create(ctx, u.ID, "regional_threat", "Угроза в регионе!", signalMsg, &signalLinkValue)
				}
			}
		}
	}

	if s.audit != nil {
		oldV := prev.Status
		newV := status
		actor := actorID
		auditStatus := "approved"
		if status == "rejected" {
			auditStatus = "rejected"
		}
		_ = s.audit.Add(ctx, "signal", signalID, "status", &oldV, &newV, &actor, auditStatus)
	}

	return nil
}

func (s *SignalsService) DeleteComment(ctx context.Context, commentID uuid.UUID, reason string) error {
	comment, err := s.comments.GetByID(ctx, commentID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "comment not found"}
	}

	if err := s.comments.Delete(ctx, commentID, nil, reason); err != nil {
		return err
	}

	// Notify author
	if comment.AuthorID != uuid.Nil {
		_ = s.audit.Add(ctx, "comment", commentID, "deleted", nil, nil, nil, "approved")
		if s.notifications != nil {
			_, _ = s.notifications.Create(ctx, comment.AuthorID, "comment_deleted", "Комментарий удален", "Причина: "+reason, nil)
		}
	}

	return nil
}

func (s *SignalsService) EditOwnComment(ctx context.Context, authorID, commentID uuid.UUID, content string) (AddCommentOutput, error) {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, authorID, SanctionScopeCommentWrite, SanctionScopeContentEdit); err != nil {
			return AddCommentOutput{}, err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckCommentCreate(ctx, authorID); err != nil {
			return AddCommentOutput{}, err
		}
		if err := s.trust.CheckContentEdit(ctx, authorID); err != nil {
			return AddCommentOutput{}, err
		}
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return AddCommentOutput{}, apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"content": "required"}}
	}
	comment, err := s.comments.GetByID(ctx, commentID)
	if err != nil {
		return AddCommentOutput{}, apierr.Error{Code: "validation_failed", Message: "comment not found"}
	}
	if comment.AuthorID != authorID {
		return AddCommentOutput{}, apierr.Error{Code: "forbidden", Message: "forbidden"}
	}
	if comment.DeletedAt != nil {
		return AddCommentOutput{}, apierr.Error{Code: "validation_failed", Message: "comment deleted"}
	}

	textScore, categories, unavailable := s.checkText(ctx, content)
	isHidden := false
	if !unavailable && (len(categories) > 0 || (textScore != nil && *textScore >= 0.85)) {
		isHidden = true
	}
	if s.trust != nil {
		risky, err := s.trust.IsRisky(ctx, authorID)
		if err != nil {
			return AddCommentOutput{}, err
		}
		if risky {
			isHidden = true
		}
	}
	if err := s.comments.UpdateContent(ctx, commentID, content, isHidden, textScore); err != nil {
		return AddCommentOutput{}, err
	}
	if isHidden && s.sanctions != nil {
		_ = s.sanctions.RegisterHiddenComment(ctx, commentID, authorID, comment.SignalID, content, textScore, map[string]any{
			"source": "edit",
		})
	}
	if s.adminLogs != nil {
		entityID := commentID
		_, _ = s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
			ActorUserID:  &authorID,
			TargetUserID: &authorID,
			EntityType:   "comment",
			EntityID:     &entityID,
			Action:       "редактирование_комментария",
			Result:       "success",
			Message:      "Пользователь изменил свой комментарий к сигналу",
			Meta: map[string]any{
				"signal_id": comment.SignalID.String(),
				"is_hidden": isHidden,
			},
		})
	}
	return AddCommentOutput{CommentID: commentID, IsHidden: isHidden}, nil
}

func (s *SignalsService) DeleteOwnComment(ctx context.Context, authorID, commentID uuid.UUID) error {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, authorID, SanctionScopeContentEdit); err != nil {
			return err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, authorID); err != nil {
			return err
		}
	}
	comment, err := s.comments.GetByID(ctx, commentID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "comment not found"}
	}
	if comment.AuthorID != authorID {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}
	return s.comments.Delete(ctx, commentID, &authorID, "Удалено автором")
}

func (s *SignalsService) checkText(ctx context.Context, text string) (*float64, []string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil, false
	}

	if s.textChecker == nil {
		return nil, nil, true
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := s.textChecker.Check(ctx, text)
	if err != nil {
		log.Printf("AI Text Filter unavailable: %v", err)
		return nil, nil, true
	}

	var toxicScore *float64
	if res.Scores != nil {
		if v, ok := res.Scores["toxicity"]; ok {
			toxicScore = &v
		}
	}

	var flags []string
	for _, c := range res.Categories {
		if c == "profanity" || c == "toxicity" {
			flags = append(flags, c)
		}
	}
	return toxicScore, flags, false
}

func (s *SignalsService) checkImages(ctx context.Context, photos []*multipart.FileHeader) ([]map[string]any, map[string]string, bool, error) {
	var results []map[string]any
	issues := map[string]string{}
	anyUnavailable := false

	for i, fh := range photos {
		file, err := fh.Open()
		if err != nil {
			return nil, nil, false, err
		}
		b, err := io.ReadAll(io.LimitReader(file, 20<<20))
		_ = file.Close()
		if err != nil {
			return nil, nil, false, err
		}

		if s.imageChecker == nil {
			anyUnavailable = true
			results = append(results, map[string]any{"i": i, "unavailable": true})
			continue
		}

		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		res, err := s.imageChecker.Check(ctx2, fh.Filename, b)
		cancel()
		if err != nil {
			log.Printf("AI Image Filter unavailable (photo %d): %v", i, err)
			anyUnavailable = true
			results = append(results, map[string]any{"i": i, "unavailable": true, "err": err.Error()})
			continue
		}

		results = append(results, map[string]any{"i": i, "status": res.Status, "confidence": res.Confidence})
		if strings.EqualFold(res.Status, "garbage") {
			issues[fmt.Sprintf("photos.%d", i)] = "garbage"
		}
	}
	return results, issues, anyUnavailable, nil
}

func (s *SignalsService) saveSignalPhotos(ctx context.Context, signalID uuid.UUID, files []*multipart.FileHeader, imageResults []map[string]any) error {
	for i, fh := range files {
		file, err := fh.Open()
		if err != nil {
			return err
		}
		originalBytes, err := io.ReadAll(io.LimitReader(file, 20<<20))
		_ = file.Close()
		if err != nil {
			return err
		}

		var imageResult map[string]any
		if i < len(imageResults) {
			imageResult = imageResults[i]
		}
		assets, err := buildPreparedImageAssets(originalBytes, fh.Filename, imageResult, fitSignal, 97, 95, 95, encodeSignalJPG)
		if err != nil {
			return apierr.Error{Code: "validation_failed", Message: "invalid image", Fields: map[string]string{fmt.Sprintf("photos.%d", i): "invalid_image"}}
		}

		fileID := ids.NewV7()
		fileName := fileID.String() + assets.Ext

		origPath := filepath.ToSlash(filepath.Join("signal_originals", fileName))
		prevPath := filepath.ToSlash(filepath.Join("signal_previews", fileName))
		thumbPath := filepath.ToSlash(filepath.Join("signal_thumbnails", fileName))

		if err := s.uploader.Save(ctx, origPath, bytes.NewReader(assets.Original)); err != nil {
			return err
		}
		if err := s.uploader.Save(ctx, prevPath, bytes.NewReader(assets.Preview)); err != nil {
			_ = s.uploader.Delete(ctx, origPath)
			return err
		}
		if err := s.uploader.Save(ctx, thumbPath, bytes.NewReader(assets.Thumb)); err != nil {
			_ = s.uploader.Delete(ctx, origPath)
			_ = s.uploader.Delete(ctx, prevPath)
			return err
		}

		var relevanceScore *float64
		aiFlags := map[string]any{}
		if i < len(imageResults) {
			aiFlags["image_filter"] = imageResults[i]
			if v, ok := imageResults[i]["confidence"].(float64); ok {
				relevanceScore = &v
			}
		}

		exifData := map[string]any{"original_name": fh.Filename}
		for k, v := range extractExif(originalBytes) {
			exifData[k] = v
		}

		if _, err := s.signalPhotos.Create(ctx, signalID, origPath, thumbPath, prevPath, exifData, relevanceScore, aiFlags); err != nil {
			_ = s.uploader.Delete(ctx, origPath)
			_ = s.uploader.Delete(ctx, prevPath)
			_ = s.uploader.Delete(ctx, thumbPath)
			return err
		}
	}
	return nil
}

func fitSignal(img image.Image, max int) image.Image {
	if img.Bounds().Dx() > max || img.Bounds().Dy() > max {
		return imaging.Fit(img, max, max, imaging.Lanczos)
	}
	return img
}

func encodeSignalJPG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractExif(originalBytes []byte) map[string]any {
	x, err := exif.Decode(bytes.NewReader(originalBytes))
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if tm, err := x.DateTime(); err == nil {
		out["datetime"] = tm.UTC().Format(time.RFC3339)
	}
	if tag, err := x.Get(exif.Model); err == nil {
		if s, err := tag.StringVal(); err == nil {
			out["model"] = s
		}
	}
	lat, lon, err := x.LatLong()
	if err == nil {
		out["gps_lat"] = lat
		out["gps_lon"] = lon
	}
	return out
}

func buildResolutionResponse(resolutionKind, resolutionComment string) string {
	switch resolutionKind {
	case SignalResolutionSuccessful:
		return "Пользователь отметил, что проблема устранена результативно"
	case SignalResolutionPartial:
		if resolutionComment == "" {
			return "Пользователь отметил, что проблема решена частично"
		}
		return "Пользователь отметил частичное решение проблемы: " + resolutionComment
	case SignalResolutionUnsuccessful:
		return "Пользователь закрыл сигнал без результата"
	default:
		return "Пользователь обновил статус сигнала"
	}
}

func containsReason(reasons []string, reason string) bool {
	for _, item := range reasons {
		if item == reason {
			return true
		}
	}
	return false
}

func uniqueReasons(reasons []string) []string {
	if len(reasons) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(reasons))
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if strings.TrimSpace(reason) == "" {
			continue
		}
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		out = append(out, reason)
	}
	return out
}
