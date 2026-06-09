package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
)

const (
	SanctionScopeCommentWrite  = "comment_write"
	SanctionScopeContentCreate = "content_create"
	SanctionScopeContentEdit   = "content_edit"
	SanctionScopeReportCreate  = "report_create"
	SanctionScopeLogin         = "login"
)

type RestrictionSummary struct {
	Status          string              `json:"status"`
	Scopes          []string            `json:"scopes"`
	Message         string              `json:"message"`
	ActiveSanctions []repo.UserSanction `json:"active_sanctions"`
	EndsAt          *time.Time          `json:"ends_at,omitempty"`
}

type SanctionsService struct {
	sanctions     *repo.UserSanctions
	incidents     *repo.CommentAIIncidents
	sessions      *repo.Sessions
	users         *repo.Users
	trust         *TrustService
	notifications *repo.Notifications
}

type SanctionsDeps struct {
	Sanctions     *repo.UserSanctions
	Incidents     *repo.CommentAIIncidents
	Sessions      *repo.Sessions
	Users         *repo.Users
	Trust         *TrustService
	Notifications *repo.Notifications
}

func NewSanctionsService(deps SanctionsDeps) *SanctionsService {
	return &SanctionsService{
		sanctions:     deps.Sanctions,
		incidents:     deps.Incidents,
		sessions:      deps.Sessions,
		users:         deps.Users,
		trust:         deps.Trust,
		notifications: deps.Notifications,
	}
}

func (s *SanctionsService) Check(ctx context.Context, userID uuid.UUID, scopes ...string) error {
	if userID == uuid.Nil || len(scopes) == 0 {
		return nil
	}
	now := time.Now()
	if s.sanctions != nil {
		_ = s.sanctions.ExpireFinished(ctx, now)
	}
	items, err := s.ActiveByUser(ctx, userID)
	if err != nil {
		return err
	}
	required := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		required[strings.TrimSpace(scope)] = struct{}{}
	}
	for _, item := range items {
		for _, scope := range item.Scopes {
			if _, ok := required[scope]; ok {
				return apierr.Error{
					Code:    "account_restricted",
					Message: s.blockMessage(scope, item.EndsAt),
					Data: map[string]any{
						"scope":   scope,
						"ends_at": item.EndsAt,
					},
				}
			}
		}
	}
	return nil
}

func (s *SanctionsService) ActiveByUser(ctx context.Context, userID uuid.UUID) ([]repo.UserSanction, error) {
	if s.sanctions == nil || userID == uuid.Nil {
		return nil, nil
	}
	items, err := s.sanctions.ListActiveByUser(ctx, userID, time.Now())
	if err == repo.ErrNotFound {
		return nil, nil
	}
	return items, err
}

func (s *SanctionsService) Summary(ctx context.Context, user repo.User) (RestrictionSummary, error) {
	items, err := s.ActiveByUser(ctx, user.ID)
	if err != nil {
		return RestrictionSummary{}, err
	}
	if user.IsBlocked {
		return RestrictionSummary{
			Status:          "login_banned",
			Scopes:          []string{SanctionScopeLogin},
			Message:         "Доступ к аккаунту заблокирован",
			ActiveSanctions: items,
		}, nil
	}
	if len(items) == 0 {
		return RestrictionSummary{Status: "active", Scopes: []string{}}, nil
	}
	scopes := map[string]struct{}{}
	var endsAt *time.Time
	for _, item := range items {
		for _, scope := range item.Scopes {
			scopes[scope] = struct{}{}
		}
		if item.EndsAt != nil && (endsAt == nil || item.EndsAt.After(*endsAt)) {
			t := *item.EndsAt
			endsAt = &t
		}
	}
	flat := make([]string, 0, len(scopes))
	for scope := range scopes {
		flat = append(flat, scope)
	}
	status := "restricted"
	if _, ok := scopes[SanctionScopeLogin]; ok {
		status = "login_banned"
	}
	return RestrictionSummary{
		Status:          status,
		Scopes:          flat,
		Message:         s.summaryMessage(flat, endsAt),
		ActiveSanctions: items,
		EndsAt:          endsAt,
	}, nil
}

func (s *SanctionsService) CreateManual(ctx context.Context, in repo.CreateUserSanctionParams) (repo.UserSanction, error) {
	if in.StartsAt.IsZero() {
		in.StartsAt = time.Now()
	}
	if strings.TrimSpace(in.Kind) == "" {
		in.Kind = "manual_ban"
	}
	if strings.TrimSpace(in.Source) == "" {
		in.Source = "manual"
	}
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "active"
	}
	item, err := s.sanctions.Create(ctx, in)
	if err != nil {
		return repo.UserSanction{}, err
	}
	if hasScope(item.Scopes, SanctionScopeLogin) && s.sessions != nil {
		_ = s.sessions.RevokeByUserID(ctx, item.UserID, time.Now())
	}
	s.notifySanction(ctx, item)
	return item, nil
}

func (s *SanctionsService) Revoke(ctx context.Context, sanctionID, actorID uuid.UUID, reason string) error {
	return s.sanctions.Revoke(ctx, sanctionID, actorID, reason, time.Now())
}

func (s *SanctionsService) Update(ctx context.Context, sanctionID uuid.UUID, scopes []string, endsAt *time.Time, reasonText string, meta map[string]any) error {
	return s.sanctions.Update(ctx, sanctionID, scopes, endsAt, reasonText, meta)
}

func (s *SanctionsService) ListByUser(ctx context.Context, userID uuid.UUID) ([]repo.UserSanction, error) {
	if s.sanctions == nil {
		return nil, nil
	}
	items, err := s.sanctions.ListByUser(ctx, userID)
	if err == repo.ErrNotFound {
		return nil, nil
	}
	return items, err
}

func (s *SanctionsService) RegisterHiddenComment(ctx context.Context, commentID, userID, signalID uuid.UUID, content string, toxicScore *float64, meta map[string]any) error {
	if s.incidents == nil || s.sanctions == nil {
		return nil
	}
	_, err := s.incidents.Create(ctx, repo.CommentAIIncident{
		CommentID:       commentID,
		UserID:          userID,
		SignalID:        signalID,
		ContentSnapshot: content,
		ToxicScore:      toxicScore,
		EventType:       "ai_hidden",
		Meta:            meta,
	})
	if err != nil {
		return err
	}
	if s.trust != nil {
		source, _ := meta["source"].(string)
		if err := s.trust.RegisterHiddenComment(ctx, userID, commentID, source); err != nil {
			return err
		}
	}
	return s.applyAutoEscalation(ctx, userID, commentID, signalID)
}

func (s *SanctionsService) applyAutoEscalation(ctx context.Context, userID, commentID, signalID uuid.UUID) error {
	count24, err := s.incidents.CountSince(ctx, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if count24 < 3 {
		return nil
	}

	reasonCode := "ai_comment_abuse"
	escalations30, err := s.sanctions.CountActiveByReasonSince(ctx, userID, reasonCode, time.Now().Add(-30*24*time.Hour))
	if err != nil {
		return err
	}
	escalations90, err := s.sanctions.CountActiveByReasonSince(ctx, userID, reasonCode, time.Now().Add(-90*24*time.Hour))
	if err != nil {
		return err
	}

	now := time.Now()
	var endsAt *time.Time
	scopes := []string{SanctionScopeCommentWrite}
	reasonText := "Несколько комментариев подряд были скрыты автоматической модерацией"
	meta := map[string]any{
		"trigger_comment_id": commentID.String(),
		"signal_id":          signalID.String(),
		"window_hours":       24,
	}

	if escalations90 >= 2 {
		scopes = []string{SanctionScopeLogin}
		reasonText = "Аккаунт заблокирован после повторяющихся нарушений в комментариях"
	} else if escalations30 >= 1 {
		t := now.Add(7 * 24 * time.Hour)
		endsAt = &t
		scopes = []string{SanctionScopeCommentWrite, SanctionScopeContentCreate, SanctionScopeContentEdit}
		reasonText = "На аккаунт наложено временное ограничение на публикацию и редактирование контента"
	} else {
		t := now.Add(6 * time.Hour)
		endsAt = &t
	}

	item, err := s.sanctions.Create(ctx, repo.CreateUserSanctionParams{
		UserID:            userID,
		Kind:              "auto_ban",
		Source:            "auto",
		ReasonCode:        reasonCode,
		ReasonText:        reasonText,
		Scopes:            scopes,
		StartsAt:          now,
		EndsAt:            endsAt,
		Status:            "active",
		RelatedEntityType: "comment",
		RelatedEntityID:   &commentID,
		Meta:              meta,
	})
	if err != nil {
		return err
	}
	if hasScope(scopes, SanctionScopeLogin) && s.sessions != nil {
		_ = s.sessions.RevokeByUserID(ctx, userID, now)
	}
	s.notifySanction(ctx, item)
	return nil
}

func (s *SanctionsService) blockMessage(scope string, endsAt *time.Time) string {
	switch scope {
	case SanctionScopeCommentWrite:
		return formatRestrictionMessage("Оставление комментариев временно недоступно", endsAt)
	case SanctionScopeContentCreate:
		return formatRestrictionMessage("Создание нового контента временно недоступно", endsAt)
	case SanctionScopeContentEdit:
		return formatRestrictionMessage("Редактирование контента временно недоступно", endsAt)
	case SanctionScopeLogin:
		return "Вход в аккаунт заблокирован"
	default:
		return formatRestrictionMessage("Действие временно недоступно", endsAt)
	}
}

func (s *SanctionsService) summaryMessage(scopes []string, endsAt *time.Time) string {
	if hasScope(scopes, SanctionScopeLogin) {
		return "Вход в аккаунт заблокирован"
	}
	if hasScope(scopes, SanctionScopeContentCreate) || hasScope(scopes, SanctionScopeContentEdit) {
		return formatRestrictionMessage("Публикация и редактирование контента временно ограничены", endsAt)
	}
	if hasScope(scopes, SanctionScopeCommentWrite) {
		return formatRestrictionMessage("Комментарии временно ограничены", endsAt)
	}
	return formatRestrictionMessage("На аккаунте есть активные ограничения", endsAt)
}

func (s *SanctionsService) notifySanction(ctx context.Context, sanction repo.UserSanction) {
	if s.notifications == nil {
		return
	}
	title := "Ограничение аккаунта"
	typ := "account_restricted"
	if hasScope(sanction.Scopes, SanctionScopeLogin) {
		title = "Аккаунт заблокирован"
		typ = "account_banned"
	}
	content := sanction.ReasonText
	if content == "" {
		content = s.summaryMessage(sanction.Scopes, sanction.EndsAt)
	}
	link := "/profile"
	_, _ = s.notifications.Create(ctx, sanction.UserID, typ, title, content, &link)
}

func hasScope(scopes []string, expected string) bool {
	for _, scope := range scopes {
		if scope == expected {
			return true
		}
	}
	return false
}

func formatRestrictionMessage(base string, endsAt *time.Time) string {
	if endsAt == nil {
		return base
	}
	return fmt.Sprintf("%s до %s", base, endsAt.Local().Format("02.01.2006 15:04"))
}
