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
	TrustLevelTrusted    = "trusted"
	TrustLevelStandard   = "standard"
	TrustLevelRisky      = "risky"
	TrustLevelRestricted = "restricted"
)

type TrustEvent struct {
	ID         uuid.UUID  `json:"id"`
	Delta      int        `json:"delta"`
	ReasonCode string     `json:"reason_code"`
	SourceType string     `json:"source_type"`
	SourceID   *uuid.UUID `json:"source_id,omitempty"`
	Comment    string     `json:"comment,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type TrustSummary struct {
	Score          int          `json:"score"`
	Level          string       `json:"level"`
	Label          string       `json:"label"`
	Message        string       `json:"message"`
	Restrictions   []string     `json:"restrictions"`
	MinScore       int          `json:"min_score"`
	MaxScore       int          `json:"max_score"`
	NextLevelLabel string       `json:"next_level_label,omitempty"`
	NextLevelScore *int         `json:"next_level_score,omitempty"`
	RecentEvents   []TrustEvent `json:"recent_events"`
}

type TrustAdjustment struct {
	UserID     uuid.UUID
	Delta      int
	ReasonCode string
	SourceType string
	SourceID   *uuid.UUID
	Comment    string
}

type TrustService struct {
	users         *repo.Users
	events        *repo.TrustEvents
	notifications *repo.Notifications
	monuments     *repo.Monuments
	posts         *repo.Posts
	signals       *repo.Signals
	comments      *repo.SignalComments
}

type TrustDeps struct {
	Users         *repo.Users
	Events        *repo.TrustEvents
	Notifications *repo.Notifications
	Monuments     *repo.Monuments
	Posts         *repo.Posts
	Signals       *repo.Signals
	Comments      *repo.SignalComments
}

func NewTrustService(deps TrustDeps) *TrustService {
	return &TrustService{
		users:         deps.Users,
		events:        deps.Events,
		notifications: deps.Notifications,
		monuments:     deps.Monuments,
		posts:         deps.Posts,
		signals:       deps.Signals,
		comments:      deps.Comments,
	}
}

func trustLevelForScore(score int) string {
	switch {
	case score >= 10:
		return TrustLevelTrusted
	case score >= 0:
		return TrustLevelStandard
	case score >= -9:
		return TrustLevelRisky
	default:
		return TrustLevelRestricted
	}
}

func (s *TrustService) Summary(ctx context.Context, user repo.User) (TrustSummary, error) {
	level := trustLevelForScore(user.TrustScore)
	recentEvents := []TrustEvent{}
	if s.events != nil {
		items, err := s.events.ListLatestByUser(ctx, user.ID, 5)
		if err == nil {
			for _, item := range items {
				recentEvents = append(recentEvents, TrustEvent{
					ID:         item.ID,
					Delta:      item.Delta,
					ReasonCode: item.ReasonCode,
					SourceType: item.SourceType,
					SourceID:   item.SourceID,
					Comment:    s.eventComment(item),
					CreatedAt:  item.CreatedAt,
				})
			}
		}
	}
	return TrustSummary{
		Score:          user.TrustScore,
		Level:          level,
		Label:          trustLevelLabel(level),
		Message:        trustLevelMessage(level),
		Restrictions:   trustRestrictions(level),
		MinScore:       trustLevelMin(level),
		MaxScore:       trustLevelMax(level),
		NextLevelLabel: trustNextLevelLabel(level),
		NextLevelScore: trustNextLevelScore(user.TrustScore),
		RecentEvents:   recentEvents,
	}, nil
}

func (s *TrustService) AdjustScore(ctx context.Context, in TrustAdjustment) error {
	if s.users == nil || in.UserID == uuid.Nil || in.Delta == 0 {
		return nil
	}
	before, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	oldLevel := trustLevelForScore(before.TrustScore)
	if err := s.users.AdjustTrustScore(ctx, in.UserID, in.Delta); err != nil {
		return err
	}
	after, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if s.events != nil {
		_, _ = s.events.Create(ctx, repo.CreateTrustEventParams{
			UserID:     in.UserID,
			Delta:      in.Delta,
			ReasonCode: strings.TrimSpace(in.ReasonCode),
			SourceType: strings.TrimSpace(in.SourceType),
			SourceID:   in.SourceID,
			Comment:    strings.TrimSpace(in.Comment),
		})
	}
	newLevel := trustLevelForScore(after.TrustScore)
	if newLevel != oldLevel {
		s.notifyLevelChanged(ctx, after, oldLevel, newLevel, in.ReasonCode)
	}
	return nil
}

func (s *TrustService) CheckContentCreate(ctx context.Context, userID uuid.UUID) error {
	if s.users == nil || userID == uuid.Nil {
		return nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	level := trustLevelForScore(user.TrustScore)
	if level == TrustLevelRestricted {
		return apierr.Error{Code: "trust_content_create_blocked", Message: "Уровень доверия слишком низкий: создание нового контента временно недоступно"}
	}
	if level == TrustLevelRisky {
		count, err := s.dailyContentCount(ctx, userID)
		if err != nil {
			return err
		}
		if count >= 5 {
			return apierr.Error{Code: "trust_rate_limited", Message: "Для этого уровня доверия достигнут дневной лимит публикаций"}
		}
	}
	return nil
}

func (s *TrustService) CheckContentEdit(ctx context.Context, userID uuid.UUID) error {
	if s.users == nil || userID == uuid.Nil {
		return nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if trustLevelForScore(user.TrustScore) == TrustLevelRestricted {
		return apierr.Error{Code: "trust_edit_blocked", Message: "Уровень доверия слишком низкий: редактирование временно недоступно"}
	}
	return nil
}

func (s *TrustService) CheckCommentCreate(ctx context.Context, userID uuid.UUID) error {
	if s.users == nil || userID == uuid.Nil {
		return nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	level := trustLevelForScore(user.TrustScore)
	if level == TrustLevelRestricted {
		return apierr.Error{Code: "trust_comment_create_blocked", Message: "Уровень доверия слишком низкий: комментарии временно недоступны"}
	}
	if level == TrustLevelRisky && s.comments != nil {
		count, err := s.comments.CountCreatedByAuthorSince(ctx, userID, time.Now().Add(-1*time.Hour))
		if err != nil {
			return err
		}
		if count >= 5 {
			return apierr.Error{Code: "trust_rate_limited", Message: "Для этого уровня доверия достигнут лимит комментариев на ближайший час"}
		}
	}
	return nil
}

func (s *TrustService) IsRisky(ctx context.Context, userID uuid.UUID) (bool, error) {
	if s.users == nil || userID == uuid.Nil {
		return false, nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	level := trustLevelForScore(user.TrustScore)
	return level == TrustLevelRisky || level == TrustLevelRestricted, nil
}

func (s *TrustService) ApplySubmissionTrustFlags(ctx context.Context, userID uuid.UUID, highRisk *bool, reasons *[]string, aiFlags map[string]any) error {
	risky, err := s.IsRisky(ctx, userID)
	if err != nil {
		return err
	}
	if !risky {
		return nil
	}
	if highRisk != nil {
		*highRisk = true
	}
	if reasons != nil && !containsReason(*reasons, "low_trust_author") {
		*reasons = append(*reasons, "low_trust_author")
	}
	if aiFlags != nil {
		aiFlags["trust_level"] = TrustLevelRisky
		aiFlags["trust_forced_review"] = true
	}
	return nil
}

func (s *TrustService) RegisterHiddenComment(ctx context.Context, userID, commentID uuid.UUID, source string) error {
	comment := "Комментарий был скрыт автоматической проверкой"
	if strings.TrimSpace(source) == "edit" {
		comment = "Отредактированный комментарий был скрыт автоматической проверкой"
	}
	return s.AdjustScore(ctx, TrustAdjustment{
		UserID:     userID,
		Delta:      -3,
		ReasonCode: "ai_hidden_comment",
		SourceType: "comment",
		SourceID:   &commentID,
		Comment:    comment,
	})
}

func (s *TrustService) dailyContentCount(ctx context.Context, userID uuid.UUID) (int, error) {
	since := time.Now().Add(-24 * time.Hour)
	total := 0
	if s.monuments != nil {
		count, err := s.monuments.CountCreatedByAuthorSince(ctx, userID, since)
		if err != nil {
			return 0, err
		}
		total += count
	}
	if s.posts != nil {
		count, err := s.posts.CountCreatedByAuthorSince(ctx, userID, since)
		if err != nil {
			return 0, err
		}
		total += count
	}
	if s.signals != nil {
		count, err := s.signals.CountCreatedByAuthorSince(ctx, userID, since)
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func trustLevelLabel(level string) string {
	switch level {
	case TrustLevelTrusted:
		return "Высокий уровень доверия"
	case TrustLevelRisky:
		return "Пониженный уровень доверия"
	case TrustLevelRestricted:
		return "Ограниченный уровень доверия"
	default:
		return "Стандартный уровень доверия"
	}
}

func trustLevelMessage(level string) string {
	switch level {
	case TrustLevelTrusted:
		return "Аккаунт считается надежным, дополнительные ограничения не применяются."
	case TrustLevelRisky:
		return "Новые публикации отправляются на усиленную проверку, а активность частично ограничена."
	case TrustLevelRestricted:
		return "Создание и редактирование части контента временно недоступно из-за низкого уровня доверия."
	default:
		return "Базовый режим доверия без специальных ограничений."
	}
}

func trustRestrictions(level string) []string {
	switch level {
	case TrustLevelRisky:
		return []string{
			"Новые публикации помечаются как повышенный риск и требуют усиленной проверки.",
			"Есть лимит на количество комментариев в час.",
			"Есть лимит на количество новых публикаций в сутки.",
		}
	case TrustLevelRestricted:
		return []string{
			"Нельзя создавать новые точки, посты, сигналы и фотографии.",
			"Нельзя отправлять пользовательские правки контента.",
			"Комментарии временно недоступны.",
		}
	default:
		return []string{}
	}
}

func trustLevelMin(level string) int {
	switch level {
	case TrustLevelTrusted:
		return 10
	case TrustLevelRisky:
		return -9
	case TrustLevelRestricted:
		return -15
	default:
		return 0
	}
}

func trustLevelMax(level string) int {
	switch level {
	case TrustLevelTrusted:
		return 15
	case TrustLevelRisky:
		return -1
	case TrustLevelRestricted:
		return -10
	default:
		return 9
	}
}

func trustNextLevelLabel(level string) string {
	switch level {
	case TrustLevelRestricted:
		return "Пониженный"
	case TrustLevelRisky:
		return "Стандартный"
	case TrustLevelStandard:
		return "Высокий"
	default:
		return ""
	}
}

func trustNextLevelScore(score int) *int {
	var target int
	switch trustLevelForScore(score) {
	case TrustLevelRestricted:
		target = -9
	case TrustLevelRisky:
		target = 0
	case TrustLevelStandard:
		target = 10
	default:
		return nil
	}
	return &target
}

func (s *TrustService) eventComment(item repo.TrustEvent) string {
	if strings.TrimSpace(item.Comment) != "" {
		return strings.TrimSpace(item.Comment)
	}
	switch item.ReasonCode {
	case "abuse_report_confirmed":
		return "Подтверждена жалоба на нарушение"
	case "integrity_report_confirmed":
		return "Подтверждена полезная жалоба на данные"
	case "ai_hidden_comment":
		return "Комментарий скрыт автоматической проверкой"
	default:
		return "Изменение уровня доверия"
	}
}

func (s *TrustService) notifyLevelChanged(ctx context.Context, user repo.User, oldLevel, newLevel, reasonCode string) {
	if s.notifications == nil {
		return
	}
	title := "Изменился уровень доверия"
	content := fmt.Sprintf("Новый уровень: %s. %s", trustLevelLabel(newLevel), trustLevelMessage(newLevel))
	if strings.TrimSpace(reasonCode) != "" {
		content += " Причина: " + s.reasonLabel(reasonCode) + "."
	}
	if restrictions := trustRestrictions(newLevel); len(restrictions) > 0 {
		content += " Что изменилось: " + restrictions[0]
	}
	link := "/profile"
	_, _ = s.notifications.Create(ctx, user.ID, "trust_level_changed", title, content, &link)
}

func (s *TrustService) reasonLabel(reasonCode string) string {
	switch strings.TrimSpace(reasonCode) {
	case "abuse_report_confirmed":
		return "жалоба на нарушение подтверждена"
	case "integrity_report_confirmed":
		return "жалоба на данные подтверждена"
	case "ai_hidden_comment":
		return "автоматическая модерация скрыла комментарий"
	default:
		return "изменение доверия"
	}
}
