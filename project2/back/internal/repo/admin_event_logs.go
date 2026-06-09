package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type AdminEventLogs struct {
	db *pgxpool.Pool
}

func NewAdminEventLogs(db *pgxpool.Pool) *AdminEventLogs {
	return &AdminEventLogs{db: db}
}

type AdminEventLog struct {
	ID           uuid.UUID      `json:"id"`
	ActorUserID  *uuid.UUID     `json:"actor_user_id,omitempty"`
	TargetUserID *uuid.UUID     `json:"target_user_id,omitempty"`
	EntityType   string         `json:"entity_type"`
	EntityID     *uuid.UUID     `json:"entity_id,omitempty"`
	Action       string         `json:"action"`
	Result       string         `json:"result"`
	Message      string         `json:"message"`
	Meta         map[string]any `json:"meta"`
	CreatedAt    time.Time      `json:"created_at"`
}

type CreateAdminEventLogParams struct {
	ActorUserID  *uuid.UUID
	TargetUserID *uuid.UUID
	EntityType   string
	EntityID     *uuid.UUID
	Action       string
	Result       string
	Message      string
	Meta         map[string]any
}

type ListAdminEventLogsFilter struct {
	ActorUserID  *uuid.UUID
	TargetUserID *uuid.UUID
	EntityType   string
	EntityID     *uuid.UUID
	Action       string
	Result       string
	Limit        int
	Offset       int
}

func (r *AdminEventLogs) Create(ctx context.Context, in CreateAdminEventLogParams) (uuid.UUID, error) {
	if strings.TrimSpace(in.Result) == "" {
		in.Result = "success"
	}
	meta := in.Meta
	if meta == nil {
		meta = map[string]any{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return uuid.Nil, err
	}
	id := ids.NewV7()
	err = r.db.QueryRow(ctx, `
		insert into admin_event_logs (id, actor_user_id, target_user_id, entity_type, entity_id, action, result, message, meta)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		returning id
	`, id, in.ActorUserID, in.TargetUserID, strings.TrimSpace(in.EntityType), in.EntityID, strings.TrimSpace(in.Action), strings.TrimSpace(in.Result), strings.TrimSpace(in.Message), metaJSON).Scan(&id)
	return id, err
}

func (r *AdminEventLogs) List(ctx context.Context, f ListAdminEventLogsFilter) ([]AdminEventLog, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 30
	}
	query := `
		select id, actor_user_id, target_user_id, entity_type, entity_id, action, result, message, meta, created_at
		from admin_event_logs
		where 1=1
	`
	args := []any{}
	if f.ActorUserID != nil {
		query += fmt.Sprintf(" and actor_user_id = $%d", len(args)+1)
		args = append(args, *f.ActorUserID)
	}
	if f.TargetUserID != nil {
		query += fmt.Sprintf(" and target_user_id = $%d", len(args)+1)
		args = append(args, *f.TargetUserID)
	}
	if strings.TrimSpace(f.EntityType) != "" {
		query += fmt.Sprintf(" and entity_type = $%d", len(args)+1)
		args = append(args, strings.TrimSpace(f.EntityType))
	}
	if f.EntityID != nil {
		query += fmt.Sprintf(" and entity_id = $%d", len(args)+1)
		args = append(args, *f.EntityID)
	}
	if strings.TrimSpace(f.Action) != "" {
		query += fmt.Sprintf(" and action = $%d", len(args)+1)
		args = append(args, strings.TrimSpace(f.Action))
	}
	if strings.TrimSpace(f.Result) != "" {
		query += fmt.Sprintf(" and result = $%d", len(args)+1)
		args = append(args, strings.TrimSpace(f.Result))
	}
	query += fmt.Sprintf(" order by created_at desc limit $%d offset $%d", len(args)+1, len(args)+2)
	args = append(args, f.Limit, f.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AdminEventLog
	for rows.Next() {
		var item AdminEventLog
		var metaJSON []byte
		if err := rows.Scan(&item.ID, &item.ActorUserID, &item.TargetUserID, &item.EntityType, &item.EntityID, &item.Action, &item.Result, &item.Message, &metaJSON, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metaJSON, &item.Meta)
		items = append(items, item)
	}
	return items, rows.Err()
}
