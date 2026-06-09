package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type Sessions struct {
	db *pgxpool.Pool
}

func NewSessions(db *pgxpool.Pool) *Sessions {
	return &Sessions{db: db}
}

type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	RefreshJTI uuid.UUID
	ExpiresAt  time.Time
	CreatedAt  time.Time
	RevokedAt  *time.Time
	IP         string
	UserAgent  string
}

func (s *Sessions) Create(ctx context.Context, userID, refreshJTI uuid.UUID, expiresAt time.Time, ip, userAgent string) error {
	sessionID := ids.NewV7()
	_, err := s.db.Exec(ctx, `
		insert into user_sessions (id, user_id, refresh_jti, expires_at, ip, user_agent)
		values ($1,$2,$3,$4,$5,$6)
	`, sessionID, userID, refreshJTI, expiresAt, ip, userAgent)
	return err
}

func (s *Sessions) GetActiveByJTI(ctx context.Context, refreshJTI uuid.UUID) (Session, error) {
	var out Session
	var revokedAt *time.Time
	var ip *string
	var ua *string

	err := s.db.QueryRow(ctx, `
		select id, user_id, refresh_jti, expires_at, created_at, revoked_at, ip, user_agent
		from user_sessions
		where refresh_jti=$1
	`, refreshJTI).Scan(&out.ID, &out.UserID, &out.RefreshJTI, &out.ExpiresAt, &out.CreatedAt, &revokedAt, &ip, &ua)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	out.RevokedAt = revokedAt
	if ip != nil {
		out.IP = *ip
	}
	if ua != nil {
		out.UserAgent = *ua
	}
	if out.RevokedAt != nil {
		return Session{}, ErrNotFound
	}
	if time.Now().After(out.ExpiresAt) {
		return Session{}, ErrNotFound
	}
	return out, nil
}

func (s *Sessions) RevokeByJTI(ctx context.Context, refreshJTI uuid.UUID, revokedAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		update user_sessions
		set revoked_at=$2
		where refresh_jti=$1 and revoked_at is null
	`, refreshJTI, revokedAt)
	return err
}

func (s *Sessions) RevokeByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		update user_sessions
		set revoked_at=$2
		where user_id=$1 and revoked_at is null
	`, userID, revokedAt)
	return err
}
