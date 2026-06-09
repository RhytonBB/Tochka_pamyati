package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type EmailVerifications struct {
	db *pgxpool.Pool
}

func NewEmailVerifications(db *pgxpool.Pool) *EmailVerifications {
	return &EmailVerifications{db: db}
}

type EmailVerification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Email     string
	Purpose   string
	Code      string
	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time
	Attempts  int
}

func (r *EmailVerifications) Create(ctx context.Context, userID uuid.UUID, email, code string, expiresAt time.Time) (EmailVerification, error) {
	return r.CreateWithPurpose(ctx, userID, email, "verify_email", code, expiresAt)
}

func (r *EmailVerifications) CreateWithPurpose(ctx context.Context, userID uuid.UUID, email, purpose, code string, expiresAt time.Time) (EmailVerification, error) {
	var out EmailVerification
	email = strings.ToLower(strings.TrimSpace(email))
	out.ID = ids.NewV7()
	err := r.db.QueryRow(ctx, `
		insert into email_verifications (id, user_id, email, purpose, code, expires_at)
		values ($1,$2,$3,$4,$5,$6)
		returning created_at
	`, out.ID, userID, email, purpose, code, expiresAt).Scan(&out.CreatedAt)
	if err != nil {
		return EmailVerification{}, err
	}
	out.UserID = userID
	out.Email = email
	out.Purpose = purpose
	out.Code = code
	out.ExpiresAt = expiresAt
	return out, nil
}

func (r *EmailVerifications) LatestActiveByEmail(ctx context.Context, email string) (EmailVerification, error) {
	return r.LatestActiveByEmailAndPurpose(ctx, email, "verify_email")
}

func (r *EmailVerifications) LatestActiveByEmailAndPurpose(ctx context.Context, email, purpose string) (EmailVerification, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var out EmailVerification
	var usedAt *time.Time

	err := r.db.QueryRow(ctx, `
		select id, user_id, email, purpose, code, expires_at, created_at, used_at, attempts
		from email_verifications
		where email=$1 and purpose=$2 and used_at is null
		order by created_at desc
		limit 1
	`, email, purpose).Scan(
		&out.ID,
		&out.UserID,
		&out.Email,
		&out.Purpose,
		&out.Code,
		&out.ExpiresAt,
		&out.CreatedAt,
		&usedAt,
		&out.Attempts,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailVerification{}, ErrNotFound
	}
	if err != nil {
		return EmailVerification{}, err
	}
	out.UsedAt = usedAt
	return out, nil
}

func (r *EmailVerifications) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	ct, err := r.db.Exec(ctx, `update email_verifications set used_at=$2 where id=$1 and used_at is null`, id, usedAt)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *EmailVerifications) IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error) {
	var attempts int
	err := r.db.QueryRow(ctx, `update email_verifications set attempts=attempts+1 where id=$1 returning attempts`, id).Scan(&attempts)
	return attempts, err
}

func (r *EmailVerifications) ResendLimits(ctx context.Context, email string, now time.Time) (sentLast2Min bool, sentTodayCount int, err error) {
	return r.ResendLimitsByPurpose(ctx, email, "verify_email", now)
}

func (r *EmailVerifications) ResendLimitsByPurpose(ctx context.Context, email, purpose string, now time.Time) (sentLast2Min bool, sentTodayCount int, err error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var lastCreatedAt *time.Time
	err = r.db.QueryRow(ctx, `
		select max(created_at) from email_verifications where email=$1 and purpose=$2
	`, email, purpose).Scan(&lastCreatedAt)
	if err != nil {
		return false, 0, err
	}

	if lastCreatedAt != nil {
		sentLast2Min = now.Sub(*lastCreatedAt) < 2*time.Minute
	}

	err = r.db.QueryRow(ctx, `
		select count(*) from email_verifications where email=$1 and purpose=$2 and created_at >= $3
	`, email, purpose, now.Add(-24*time.Hour)).Scan(&sentTodayCount)
	if err != nil {
		return false, 0, err
	}

	return sentLast2Min, sentTodayCount, nil
}
