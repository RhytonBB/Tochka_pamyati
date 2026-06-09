package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type AuditLog struct {
	db *pgxpool.Pool
}

func NewAuditLog(db *pgxpool.Pool) *AuditLog {
	return &AuditLog{db: db}
}

func (a *AuditLog) Add(ctx context.Context, entityType string, entityID uuid.UUID, fieldName string, oldValue, newValue *string, authorID *uuid.UUID, status string) error {
	id := ids.NewV7()
	_, err := a.db.Exec(ctx, `
		insert into audit_log (id, entity_type, entity_id, field_name, old_value, new_value, author_id, status)
		values ($1,$2,$3,$4,$5,$6,$7,$8)
	`, id, entityType, entityID, fieldName, oldValue, newValue, authorID, status)
	return err
}

type AuditEntry struct {
	ID          uuid.UUID  `json:"id"`
	EntityType  string     `json:"entity_type"`
	EntityID    uuid.UUID  `json:"entity_id"`
	FieldName   string     `json:"field_name"`
	OldValue    *string    `json:"old_value"`
	NewValue    *string    `json:"new_value"`
	AuthorID    *uuid.UUID `json:"author_id,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ModeratedAt *time.Time `json:"moderated_at,omitempty"`
	ModeratedBy *uuid.UUID `json:"moderated_by,omitempty"`
}

func (a *AuditLog) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]AuditEntry, error) {
	rows, err := a.db.Query(ctx, `
		select id, entity_type, entity_id, field_name, old_value, new_value, author_id, status, created_at, moderated_at, moderated_by
		from audit_log
		where entity_type=$1 and entity_id=$2
		order by created_at desc
	`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.FieldName, &e.OldValue, &e.NewValue, &e.AuthorID, &e.Status, &e.CreatedAt, &e.ModeratedAt, &e.ModeratedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (a *AuditLog) ListPending(ctx context.Context, limit, offset int) ([]AuditEntry, error) {
	rows, err := a.db.Query(ctx, `
		select id, entity_type, entity_id, field_name, old_value, new_value, author_id, status, created_at, moderated_at, moderated_by
		from audit_log
		where status='pending'
		order by created_at asc
		limit $1 offset $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.FieldName, &e.OldValue, &e.NewValue, &e.AuthorID, &e.Status, &e.CreatedAt, &e.ModeratedAt, &e.ModeratedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (a *AuditLog) GetByID(ctx context.Context, id uuid.UUID) (AuditEntry, error) {
	var e AuditEntry
	err := a.db.QueryRow(ctx, `
		select id, entity_type, entity_id, field_name, old_value, new_value, author_id, status, created_at, moderated_at, moderated_by
		from audit_log
		where id=$1
	`, id).Scan(&e.ID, &e.EntityType, &e.EntityID, &e.FieldName, &e.OldValue, &e.NewValue, &e.AuthorID, &e.Status, &e.CreatedAt, &e.ModeratedAt, &e.ModeratedBy)
	if err != nil {
		return AuditEntry{}, err
	}
	return e, nil
}

func (a *AuditLog) SetStatus(ctx context.Context, id uuid.UUID, status string, moderatedBy *uuid.UUID) error {
	_, err := a.db.Exec(ctx, `
		update audit_log
		set status=$2, moderated_at=now(), moderated_by=$3
		where id=$1
	`, id, status, moderatedBy)
	return err
}

func (a *AuditLog) UpdateStatusByEntity(ctx context.Context, entityType string, entityID uuid.UUID, status string, moderatedBy *uuid.UUID) error {
	_, err := a.db.Exec(ctx, `
		update audit_log
		set status=$3, moderated_at=now(), moderated_by=$4
		where entity_type=$1 and entity_id=$2 and status='pending'
	`, entityType, entityID, status, moderatedBy)
	return err
}
