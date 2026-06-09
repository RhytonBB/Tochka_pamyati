package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
)

type UsersService struct {
	users     *repo.Users
	roles     *repo.Roles
	sanctions *SanctionsService
	trust     *TrustService
	incidents *repo.CommentAIIncidents
	sessions  *repo.Sessions
	monuments *repo.Monuments
	posts     *repo.Posts
	signals   *repo.Signals
	adminLogs *repo.AdminEventLogs
}

type UsersDeps struct {
	Users     *repo.Users
	Roles     *repo.Roles
	Sanctions *SanctionsService
	Trust     *TrustService
	Incidents *repo.CommentAIIncidents
	Sessions  *repo.Sessions
	Monuments *repo.Monuments
	Posts     *repo.Posts
	Signals   *repo.Signals
	AdminLogs *repo.AdminEventLogs
}

func NewUsersService(deps UsersDeps) *UsersService {
	return &UsersService{
		users:     deps.Users,
		roles:     deps.Roles,
		sanctions: deps.Sanctions,
		trust:     deps.Trust,
		incidents: deps.Incidents,
		sessions:  deps.Sessions,
		monuments: deps.Monuments,
		posts:     deps.Posts,
		signals:   deps.Signals,
		adminLogs: deps.AdminLogs,
	}
}

type ListUsersOutput struct {
	Users []map[string]any `json:"users"`
	Total int64            `json:"total"`
}

func (s *UsersService) ListUsers(ctx context.Context, f repo.ListUsersFilter) (ListUsersOutput, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	users, total, err := s.users.List(ctx, f)
	if err != nil {
		return ListUsersOutput{}, err
	}
	items := make([]map[string]any, 0, len(users))
	for _, user := range users {
		items = append(items, s.buildUserAdminPayload(ctx, user))
	}
	return ListUsersOutput{Users: items, Total: total}, nil
}

func (s *UsersService) UpdateRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.users.UpdateRole(ctx, userID, roleID)
}

func (s *UsersService) ResolveRoleID(ctx context.Context, roleName string) (uuid.UUID, error) {
	return s.roles.GetByName(ctx, roleName)
}

func (s *UsersService) SetBlocked(ctx context.Context, userID uuid.UUID, blocked bool) error {
	return s.users.SetBlocked(ctx, userID, blocked)
}

func (s *UsersService) Delete(ctx context.Context, userID uuid.UUID) error {
	return s.users.Delete(ctx, userID)
}

func (s *UsersService) GetUserByID(ctx context.Context, id uuid.UUID) (repo.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *UsersService) GetUserAdminDetail(ctx context.Context, id uuid.UUID) (map[string]any, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	payload := s.buildUserAdminPayload(ctx, user)
	if s.monuments != nil {
		if monuments, err := s.monuments.ListByAuthor(ctx, id); err == nil {
			payload["monuments"] = monuments
		}
	}
	if s.posts != nil {
		if posts, err := s.posts.ListByAuthor(ctx, id); err == nil {
			payload["posts"] = posts
		}
	}
	if s.signals != nil {
		if signals, err := s.signals.ListByAuthor(ctx, id); err == nil {
			payload["signals"] = signals
		}
	}
	if s.adminLogs != nil {
		if logs, err := s.adminLogs.List(ctx, repo.ListAdminEventLogsFilter{TargetUserID: &id, Limit: 20}); err == nil {
			payload["admin_logs"] = logs
		}
	}
	return payload, nil
}

func (s *UsersService) CreateSanction(ctx context.Context, in repo.CreateUserSanctionParams) (repo.UserSanction, error) {
	if in.Source == "" {
		in.Source = "manual"
	}
	if in.Kind == "" {
		in.Kind = "manual_ban"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	return s.sanctions.CreateManual(ctx, in)
}

func (s *UsersService) RevokeSanction(ctx context.Context, sanctionID, actorID uuid.UUID, reason string) error {
	return s.sanctions.Revoke(ctx, sanctionID, actorID, reason)
}

func (s *UsersService) UpdateSanction(ctx context.Context, sanctionID uuid.UUID, scopes []string, endsAt *time.Time, reasonText string, meta map[string]any) error {
	return s.sanctions.Update(ctx, sanctionID, scopes, endsAt, reasonText, meta)
}

func (s *UsersService) ListSanctions(ctx context.Context, userID uuid.UUID) ([]repo.UserSanction, error) {
	return s.sanctions.ListByUser(ctx, userID)
}

func (s *UsersService) ListAdminLogs(ctx context.Context, filter repo.ListAdminEventLogsFilter) ([]repo.AdminEventLog, error) {
	if s.adminLogs == nil {
		return []repo.AdminEventLog{}, nil
	}
	return s.adminLogs.List(ctx, filter)
}

func (s *UsersService) buildUserAdminPayload(ctx context.Context, user repo.User) map[string]any {
	payload := map[string]any{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"role_id":     user.RoleID,
		"trust_score": user.TrustScore,
		"city":        user.City,
		"region":      user.Region,
		"is_active":   user.IsActive,
		"is_blocked":  user.IsBlocked,
		"created_at":  user.CreatedAt,
		"last_login":  user.LastLogin,
	}
	if s.roles != nil {
		if roleName, err := s.roles.GetNameByID(ctx, user.RoleID); err == nil {
			payload["role_name"] = roleName
		}
	}
	if s.sanctions != nil {
		if active, err := s.sanctions.ActiveByUser(ctx, user.ID); err == nil {
			payload["active_sanctions"] = active
		}
		if history, err := s.sanctions.ListByUser(ctx, user.ID); err == nil {
			payload["sanctions_history"] = history
		}
		if summary, err := s.sanctions.Summary(ctx, user); err == nil {
			payload["restriction_summary"] = summary
			payload["status"] = summary.Status
		}
	}
	if s.trust != nil {
		if summary, err := s.trust.Summary(ctx, user); err == nil {
			payload["trust_summary"] = summary
			payload["trust_level"] = summary.Level
		}
	}
	if _, ok := payload["status"]; !ok {
		payload["status"] = "active"
	}
	if s.incidents != nil {
		if dayCount, err := s.incidents.CountSince(ctx, user.ID, time.Now().Add(-24*time.Hour)); err == nil {
			payload["ai_hidden_comments_24h"] = dayCount
		}
	}
	return payload
}
