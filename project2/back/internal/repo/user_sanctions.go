package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type UserSanctions struct {
	db *pgxpool.Pool
}

func NewUserSanctions(db *pgxpool.Pool) *UserSanctions {
	return &UserSanctions{db: db}
}

type UserSanction struct {
	ID                uuid.UUID      `json:"id"`
	UserID            uuid.UUID      `json:"user_id"`
	Kind              string         `json:"kind"`
	Source            string         `json:"source"`
	ReasonCode        string         `json:"reason_code"`
	ReasonText        string         `json:"reason_text,omitempty"`
	Scopes            []string       `json:"scopes"`
	StartsAt          time.Time      `json:"starts_at"`
	EndsAt            *time.Time     `json:"ends_at,omitempty"`
	Status            string         `json:"status"`
	CreatedBy         *uuid.UUID     `json:"created_by,omitempty"`
	RelatedEntityType string         `json:"related_entity_type,omitempty"`
	RelatedEntityID   *uuid.UUID     `json:"related_entity_id,omitempty"`
	Meta              map[string]any `json:"meta"`
	CreatedAt         time.Time      `json:"created_at"`
	RevokedAt         *time.Time     `json:"revoked_at,omitempty"`
	RevokedBy         *uuid.UUID     `json:"revoked_by,omitempty"`
	RevokedReason     *string        `json:"revoked_reason,omitempty"`
}

type CreateUserSanctionParams struct {
	UserID            uuid.UUID
	Kind              string
	Source            string
	ReasonCode        string
	ReasonText        string
	Scopes            []string
	StartsAt          time.Time
	EndsAt            *time.Time
	Status            string
	CreatedBy         *uuid.UUID
	RelatedEntityType string
	RelatedEntityID   *uuid.UUID
	Meta              map[string]any
}

func (r *UserSanctions) Create(ctx context.Context, in CreateUserSanctionParams) (UserSanction, error) {
	metaJSON, err := json.Marshal(in.Meta)
	if err != nil {
		return UserSanction{}, err
	}
	var out UserSanction
	out.ID = ids.NewV7()
	var metaRaw []byte
	err = r.db.QueryRow(ctx, `
		insert into user_sanctions (
			id, user_id, kind, source, reason_code, reason_text, scopes, starts_at, ends_at,
			status, created_by, related_entity_type, related_entity_id, meta
		)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		returning user_id, kind, source, reason_code, reason_text, scopes, starts_at, ends_at,
		          status, created_by, related_entity_type, related_entity_id, meta, created_at,
		          revoked_at, revoked_by, revoked_reason
	`, out.ID, in.UserID, in.Kind, in.Source, in.ReasonCode, nullIfEmpty(in.ReasonText), in.Scopes, in.StartsAt, in.EndsAt,
		in.Status, in.CreatedBy, nullIfEmpty(in.RelatedEntityType), in.RelatedEntityID, metaJSON).
		Scan(&out.UserID, &out.Kind, &out.Source, &out.ReasonCode, &out.ReasonText, &out.Scopes, &out.StartsAt, &out.EndsAt,
			&out.Status, &out.CreatedBy, &out.RelatedEntityType, &out.RelatedEntityID, &metaRaw, &out.CreatedAt,
			&out.RevokedAt, &out.RevokedBy, &out.RevokedReason)
	if err != nil {
		return UserSanction{}, err
	}
	_ = json.Unmarshal(metaRaw, &out.Meta)
	return out, nil
}

func (r *UserSanctions) ListActiveByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]UserSanction, error) {
	rows, err := r.db.Query(ctx, `
		select id, user_id, kind, source, reason_code, coalesce(reason_text, ''), scopes, starts_at, ends_at,
		       status, created_by, coalesce(related_entity_type, ''), related_entity_id, meta, created_at,
		       revoked_at, revoked_by, revoked_reason
		from user_sanctions
		where user_id=$1
		  and status='active'
		  and starts_at <= $2
		  and (ends_at is null or ends_at > $2)
		order by created_at desc
	`, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserSanctions(rows)
}

func (r *UserSanctions) ListByUser(ctx context.Context, userID uuid.UUID) ([]UserSanction, error) {
	rows, err := r.db.Query(ctx, `
		select id, user_id, kind, source, reason_code, coalesce(reason_text, ''), scopes, starts_at, ends_at,
		       status, created_by, coalesce(related_entity_type, ''), related_entity_id, meta, created_at,
		       revoked_at, revoked_by, revoked_reason
		from user_sanctions
		where user_id=$1
		order by created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserSanctions(rows)
}

func (r *UserSanctions) GetByID(ctx context.Context, id uuid.UUID) (UserSanction, error) {
	var item UserSanction
	var metaRaw []byte
	err := r.db.QueryRow(ctx, `
		select id, user_id, kind, source, reason_code, coalesce(reason_text, ''), scopes, starts_at, ends_at,
		       status, created_by, coalesce(related_entity_type, ''), related_entity_id, meta, created_at,
		       revoked_at, revoked_by, revoked_reason
		from user_sanctions
		where id=$1
	`, id).Scan(&item.ID, &item.UserID, &item.Kind, &item.Source, &item.ReasonCode, &item.ReasonText, &item.Scopes, &item.StartsAt, &item.EndsAt,
		&item.Status, &item.CreatedBy, &item.RelatedEntityType, &item.RelatedEntityID, &metaRaw, &item.CreatedAt,
		&item.RevokedAt, &item.RevokedBy, &item.RevokedReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserSanction{}, err
	}
	if err != nil {
		return UserSanction{}, err
	}
	_ = json.Unmarshal(metaRaw, &item.Meta)
	return item, nil
}

func (r *UserSanctions) Revoke(ctx context.Context, id uuid.UUID, actorID uuid.UUID, reason string, revokedAt time.Time) error {
	ct, err := r.db.Exec(ctx, `
		update user_sanctions
		set status='revoked', revoked_at=$3, revoked_by=$4, revoked_reason=$5
		where id=$1 and status='active'
	`, id, "revoked", revokedAt, actorID, nullIfEmpty(reason))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserSanctions) Update(ctx context.Context, id uuid.UUID, scopes []string, endsAt *time.Time, reasonText string, meta map[string]any) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	ct, err := r.db.Exec(ctx, `
		update user_sanctions
		set scopes=$2, ends_at=$3, reason_text=$4, meta=$5
		where id=$1
	`, id, scopes, endsAt, nullIfEmpty(reasonText), metaJSON)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserSanctions) CountActiveByReasonSince(ctx context.Context, userID uuid.UUID, reasonCode string, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		select count(*)
		from user_sanctions
		where user_id=$1 and reason_code=$2 and source='auto' and created_at >= $3 and status in ('active', 'expired', 'revoked')
	`, userID, reasonCode, since).Scan(&count)
	return count, err
}

func (r *UserSanctions) ExpireFinished(ctx context.Context, now time.Time) error {
	_, err := r.db.Exec(ctx, `
		update user_sanctions
		set status='expired'
		where status='active' and ends_at is not null and ends_at <= $1
	`, now)
	return err
}

type sanctionScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanUserSanctions(rows sanctionScanner) ([]UserSanction, error) {
	var out []UserSanction
	for rows.Next() {
		var item UserSanction
		var metaRaw []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.Kind, &item.Source, &item.ReasonCode, &item.ReasonText, &item.Scopes, &item.StartsAt, &item.EndsAt,
			&item.Status, &item.CreatedBy, &item.RelatedEntityType, &item.RelatedEntityID, &metaRaw, &item.CreatedAt,
			&item.RevokedAt, &item.RevokedBy, &item.RevokedReason); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		_ = json.Unmarshal(metaRaw, &item.Meta)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}
