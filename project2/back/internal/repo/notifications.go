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

type Notifications struct {
	db *pgxpool.Pool
}

func NewNotifications(db *pgxpool.Pool) *Notifications {
	return &Notifications{db: db}
}

type Notification struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Link      *string   `json:"link,omitempty"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

func (n *Notifications) Create(ctx context.Context, userID uuid.UUID, typ, title, content string, link *string) (uuid.UUID, error) {
	typ = strings.TrimSpace(typ)
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	id := ids.NewV7()
	err := n.db.QueryRow(ctx, `
		insert into notifications (id, user_id, type, title, content, link)
		values ($1,$2,$3,$4,$5,$6)
		returning id
	`, id, userID, typ, title, content, nullIfEmptyStrPtr(link)).Scan(&id)
	return id, err
}

func (n *Notifications) CreateForRoleNames(ctx context.Context, roleNames []string, typ, title, content string, link *string) error {
	rows, err := n.db.Query(ctx, `
		select u.id
		from users u
		join roles r on r.id = u.role_id
		where r.name = any($1::text[])
		  and u.is_active=true
		  and u.is_blocked=false
	`, roleNames)
	if err != nil {
		return err
	}
	defer rows.Close()

	typ = strings.TrimSpace(typ)
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	normalizedLink := nullIfEmptyStrPtr(link)

	var batch pgx.Batch
	count := 0
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return err
		}
		count++
		batch.Queue(`
			insert into notifications (id, user_id, type, title, content, link)
			values ($1,$2,$3,$4,$5,$6)
		`, ids.NewV7(), userID, typ, title, content, normalizedLink)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	br := n.db.SendBatch(ctx, &batch)
	defer br.Close()
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (n *Notifications) ListLatest(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := n.db.Query(ctx, `
		select id, user_id, type, title, content, link, is_read, created_at
		from notifications
		where user_id=$1
		order by created_at desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var it Notification
		var link *string
		if err := rows.Scan(&it.ID, &it.UserID, &it.Type, &it.Title, &it.Content, &link, &it.IsRead, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.Link = link
		out = append(out, it)
	}
	return out, rows.Err()
}

func (n *Notifications) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var cnt int
	err := n.db.QueryRow(ctx, `select count(*) from notifications where user_id=$1 and is_read=false`, userID).Scan(&cnt)
	return cnt, err
}

func (n *Notifications) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	ct, err := n.db.Exec(ctx, `
		update notifications
		set is_read=true
		where id=$1 and user_id=$2
	`, notificationID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (n *Notifications) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := n.db.Exec(ctx, `update notifications set is_read=true where user_id=$1 and is_read=false`, userID)
	return err
}

func (n *Notifications) GetByID(ctx context.Context, id uuid.UUID) (Notification, error) {
	var out Notification
	var link *string
	err := n.db.QueryRow(ctx, `
		select id, user_id, type, title, content, link, is_read, created_at
		from notifications
		where id=$1
	`, id).Scan(&out.ID, &out.UserID, &out.Type, &out.Title, &out.Content, &link, &out.IsRead, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	if err != nil {
		return Notification{}, err
	}
	out.Link = link
	return out, nil
}
