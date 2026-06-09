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

type SignalComments struct {
	db *pgxpool.Pool
}

func NewSignalComments(db *pgxpool.Pool) *SignalComments {
	return &SignalComments{db: db}
}

type SignalComment struct {
	ID         uuid.UUID  `json:"id"`
	SignalID   uuid.UUID  `json:"signal_id"`
	AuthorID   uuid.UUID  `json:"author_id"`
	AuthorName string     `json:"author_name"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
	Content    string     `json:"content"`
	IsHidden   bool       `json:"is_hidden"`
	ToxicScore *float64   `json:"toxic_score,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	EditedAt   *time.Time `json:"edited_at,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func (c *SignalComments) Create(ctx context.Context, signalID, authorID uuid.UUID, parentID *uuid.UUID, content string, isHidden bool, toxicScore *float64) (uuid.UUID, error) {
	id := ids.NewV7()
	err := c.db.QueryRow(ctx, `
		insert into comments (id, signal_id, author_id, parent_id, content, is_hidden, toxic_score)
		values ($1,$2,$3,$4,$5,$6,$7)
		returning id
	`, id, signalID, authorID, parentID, content, isHidden, toxicScore).Scan(&id)
	return id, err
}

func (c *SignalComments) ListBySignal(ctx context.Context, signalID uuid.UUID) ([]SignalComment, error) {
	rows, err := c.db.Query(ctx, `
		select c.id, c.signal_id, c.author_id, coalesce(u.username, ''), c.parent_id, c.content, c.is_hidden, c.toxic_score, c.created_at, c.edited_at, c.deleted_at
		from comments c
		left join users u on c.author_id = u.id
		where c.signal_id=$1
		  and c.deleted_at is null
		order by c.created_at asc
	`, signalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SignalComment
	for rows.Next() {
		var r SignalComment
		var toxicScore *float64
		var parentID *uuid.UUID
		if err := rows.Scan(&r.ID, &r.SignalID, &r.AuthorID, &r.AuthorName, &parentID, &r.Content, &r.IsHidden, &toxicScore, &r.CreatedAt, &r.EditedAt, &r.DeletedAt); err != nil {
			return nil, err
		}
		r.ParentID = parentID
		r.ToxicScore = toxicScore
		out = append(out, r)
	}
	return out, rows.Err()
}

func (c *SignalComments) GetByID(ctx context.Context, id uuid.UUID) (SignalComment, error) {
	var r SignalComment
	var toxicScore *float64
	var parentID *uuid.UUID
	err := c.db.QueryRow(ctx, `
		select c.id, c.signal_id, c.author_id, coalesce(u.username, ''), c.parent_id, c.content, c.is_hidden, c.toxic_score, c.created_at, c.edited_at, c.deleted_at
		from comments c
		left join users u on c.author_id = u.id
		where c.id=$1
	`, id).Scan(&r.ID, &r.SignalID, &r.AuthorID, &r.AuthorName, &parentID, &r.Content, &r.IsHidden, &toxicScore, &r.CreatedAt, &r.EditedAt, &r.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SignalComment{}, ErrNotFound
	}
	if err != nil {
		return SignalComment{}, err
	}
	r.ParentID = parentID
	r.ToxicScore = toxicScore
	return r, nil
}

func (c *SignalComments) SetHidden(ctx context.Context, commentID uuid.UUID, isHidden bool) error {
	ct, err := c.db.Exec(ctx, `update comments set is_hidden=$2 where id=$1 and deleted_at is null`, commentID, isHidden)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (c *SignalComments) Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID, reason string) error {
	ct, err := c.db.Exec(ctx, `
		update comments
		set deleted_at=now(), deleted_by=$2, deleted_reason=$3
		where id=$1 and deleted_at is null
	`, id, deletedBy, nullIfEmpty(reason))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (c *SignalComments) UpdateContent(ctx context.Context, id uuid.UUID, content string, isHidden bool, toxicScore *float64) error {
	ct, err := c.db.Exec(ctx, `
		update comments
		set content=$2, is_hidden=$3, toxic_score=$4, edited_at=now()
		where id=$1 and deleted_at is null
	`, id, content, isHidden, toxicScore)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (c *SignalComments) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]SignalComment, error) {
	rows, err := c.db.Query(ctx, `
		select c.id, c.signal_id, c.author_id, coalesce(u.username, ''), c.parent_id, c.content, c.is_hidden, c.toxic_score, c.created_at, c.edited_at, c.deleted_at
		from comments c
		left join users u on c.author_id = u.id
		where c.author_id=$1
		order by c.created_at desc
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SignalComment
	for rows.Next() {
		var item SignalComment
		if err := rows.Scan(&item.ID, &item.SignalID, &item.AuthorID, &item.AuthorName, &item.ParentID, &item.Content, &item.IsHidden, &item.ToxicScore, &item.CreatedAt, &item.EditedAt, &item.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (c *SignalComments) CountCreatedByAuthorSince(ctx context.Context, authorID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := c.db.QueryRow(ctx, `
		select count(*)
		from comments
		where author_id=$1 and created_at >= $2
	`, authorID, since).Scan(&count)
	return count, err
}
