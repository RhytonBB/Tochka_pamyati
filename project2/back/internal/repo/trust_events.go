package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type TrustEvents struct {
	db *pgxpool.Pool
}

func NewTrustEvents(db *pgxpool.Pool) *TrustEvents {
	return &TrustEvents{db: db}
}

type TrustEvent struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Delta      int        `json:"delta"`
	ReasonCode string     `json:"reason_code"`
	SourceType string     `json:"source_type"`
	SourceID   *uuid.UUID `json:"source_id,omitempty"`
	Comment    string     `json:"comment,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateTrustEventParams struct {
	UserID     uuid.UUID
	Delta      int
	ReasonCode string
	SourceType string
	SourceID   *uuid.UUID
	Comment    string
}

func (r *TrustEvents) Create(ctx context.Context, in CreateTrustEventParams) (TrustEvent, error) {
	var item TrustEvent
	item.ID = ids.NewV7()
	err := r.db.QueryRow(ctx, `
		insert into trust_events (id, user_id, delta, reason_code, source_type, source_id, comment)
		values ($1,$2,$3,$4,$5,$6,$7)
		returning user_id, delta, reason_code, source_type, source_id, coalesce(comment, ''), created_at
	`, item.ID, in.UserID, in.Delta, in.ReasonCode, in.SourceType, in.SourceID, nullIfEmpty(in.Comment)).
		Scan(&item.UserID, &item.Delta, &item.ReasonCode, &item.SourceType, &item.SourceID, &item.Comment, &item.CreatedAt)
	if err != nil {
		return TrustEvent{}, err
	}
	return item, nil
}

func (r *TrustEvents) ListLatestByUser(ctx context.Context, userID uuid.UUID, limit int) ([]TrustEvent, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := r.db.Query(ctx, `
		select id, user_id, delta, reason_code, source_type, source_id, coalesce(comment, ''), created_at
		from trust_events
		where user_id=$1
		order by created_at desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TrustEvent
	for rows.Next() {
		var item TrustEvent
		if err := rows.Scan(&item.ID, &item.UserID, &item.Delta, &item.ReasonCode, &item.SourceType, &item.SourceID, &item.Comment, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
