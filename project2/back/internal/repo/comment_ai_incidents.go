package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type CommentAIIncidents struct {
	db *pgxpool.Pool
}

func NewCommentAIIncidents(db *pgxpool.Pool) *CommentAIIncidents {
	return &CommentAIIncidents{db: db}
}

type CommentAIIncident struct {
	ID              uuid.UUID      `json:"id"`
	CommentID       uuid.UUID      `json:"comment_id"`
	UserID          uuid.UUID      `json:"user_id"`
	SignalID        uuid.UUID      `json:"signal_id"`
	ContentSnapshot string         `json:"content_snapshot"`
	ToxicScore      *float64       `json:"toxic_score,omitempty"`
	EventType       string         `json:"event_type"`
	Meta            map[string]any `json:"meta"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (r *CommentAIIncidents) Create(ctx context.Context, incident CommentAIIncident) (uuid.UUID, error) {
	metaJSON, err := json.Marshal(incident.Meta)
	if err != nil {
		return uuid.Nil, err
	}
	id := ids.NewV7()
	err = r.db.QueryRow(ctx, `
		insert into comment_ai_incidents (id, comment_id, user_id, signal_id, content_snapshot, toxic_score, event_type, meta)
		values ($1,$2,$3,$4,$5,$6,$7,$8)
		returning id
	`, id, incident.CommentID, incident.UserID, incident.SignalID, incident.ContentSnapshot, incident.ToxicScore, incident.EventType, metaJSON).Scan(&id)
	return id, err
}

func (r *CommentAIIncidents) CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		select count(*)
		from comment_ai_incidents
		where user_id=$1 and created_at >= $2
	`, userID, since).Scan(&count)
	return count, err
}

func (r *CommentAIIncidents) ListByUser(ctx context.Context, userID uuid.UUID, since *time.Time) ([]CommentAIIncident, error) {
	query := `
		select id, comment_id, user_id, signal_id, content_snapshot, toxic_score, event_type, meta, created_at
		from comment_ai_incidents
		where user_id=$1
	`
	args := []any{userID}
	if since != nil {
		query += ` and created_at >= $2`
		args = append(args, *since)
	}
	query += ` order by created_at desc`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CommentAIIncident
	for rows.Next() {
		var item CommentAIIncident
		var metaRaw []byte
		if err := rows.Scan(&item.ID, &item.CommentID, &item.UserID, &item.SignalID, &item.ContentSnapshot, &item.ToxicScore, &item.EventType, &metaRaw, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metaRaw, &item.Meta)
		out = append(out, item)
	}
	return out, rows.Err()
}
