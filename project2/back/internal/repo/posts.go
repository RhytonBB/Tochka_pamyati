package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type Posts struct {
	db *pgxpool.Pool
}

func NewPosts(db *pgxpool.Pool) *Posts {
	return &Posts{db: db}
}

type Post struct {
	ID                    uuid.UUID      `json:"id"`
	MonumentID            uuid.UUID      `json:"monument_id"`
	MonumentName          string         `json:"monument_name,omitempty"`
	MonumentIsOrphaned    bool           `json:"monument_is_orphaned,omitempty"`
	AuthorID              uuid.UUID      `json:"author_id"`
	AuthorName            string         `json:"author_name,omitempty"`
	Thumbnail             string         `json:"thumbnail,omitempty"`
	Description           string         `json:"description"`
	Status                string         `json:"status"`
	IsHidden              bool           `json:"is_hidden"`
	EditedAt              *time.Time     `json:"edited_at,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	ModerationComment     *string        `json:"moderation_comment,omitempty"`
	ToxicScore            *float64       `json:"toxic_score,omitempty"`
	HighRisk              bool           `json:"high_risk"`
	AIFlags               map[string]any `json:"ai_flags"`
	Photos                []Photo        `json:"photos,omitempty"`
	IsEditRequest         bool           `json:"is_edit_request,omitempty"`
	IsArchived            bool           `json:"is_archived"`
	ArchiveReason         string         `json:"archive_reason,omitempty"`
	RestoreDecisionStatus string         `json:"restore_decision_status,omitempty"`
	ArchivedAt            *time.Time     `json:"archived_at,omitempty"`
	RestoredAt            *time.Time     `json:"restored_at,omitempty"`
}

func (p *Posts) Create(ctx context.Context, monumentID, authorID uuid.UUID, description *string, status string, editedAt *time.Time, toxicScore *float64, highRisk bool, aiFlags map[string]any) (uuid.UUID, error) {
	flags, err := json.Marshal(aiFlags)
	if err != nil {
		return uuid.Nil, err
	}
	id := ids.NewV7()
	err = p.db.QueryRow(ctx, `
		insert into posts (id, monument_id, author_id, description, status, edited_at, toxic_score, high_risk, ai_flags, is_hidden)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,false)
		returning id
	`, id, monumentID, authorID, nullIfEmptyStrPtr(description), status, editedAt, toxicScore, highRisk, flags).Scan(&id)
	return id, err
}

func (p *Posts) GetByID(ctx context.Context, id uuid.UUID) (Post, error) {
	var out Post
	var desc *string
	var modComment *string
	var toxicScore *float64
	var flagsJSON []byte
	var photosJSON []byte
	err := p.db.QueryRow(ctx, `
		select p.id, p.monument_id, p.author_id, p.description, p.status, p.is_hidden, p.edited_at, p.created_at, p.moderation_comment, p.toxic_score, p.high_risk, p.ai_flags,
		       p.is_archived, coalesce(p.archive_reason, ''), coalesce(p.restore_decision_status, 'none'), p.archived_at, p.restored_at,
		       coalesce(u.username, ''),
		       coalesce((select thumbnail_path from photos where post_id=p.id order by uploaded_at asc limit 1), ''),
		       coalesce((select json_agg(ph.*) from (select ph.* from photos ph where ph.post_id=p.id order by ph.uploaded_at asc) ph), '[]'::json) as photos,
		       coalesce(m.name, '') as monument_name,
		       coalesce(m.is_orphaned, false) as monument_is_orphaned
		from posts p
		left join users u on p.author_id = u.id
		left join monuments m on p.monument_id = m.id
		where p.id=$1
	`, id).Scan(
		&out.ID,
		&out.MonumentID,
		&out.AuthorID,
		&desc,
		&out.Status,
		&out.IsHidden,
		&out.EditedAt,
		&out.CreatedAt,
		&modComment,
		&toxicScore,
		&out.HighRisk,
		&flagsJSON,
		&out.IsArchived,
		&out.ArchiveReason,
		&out.RestoreDecisionStatus,
		&out.ArchivedAt,
		&out.RestoredAt,
		&out.AuthorName,
		&out.Thumbnail,
		&photosJSON,
		&out.MonumentName,
		&out.MonumentIsOrphaned,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, err
	}
	if desc != nil {
		out.Description = *desc
	}
	out.ModerationComment = modComment
	out.ToxicScore = toxicScore
	_ = json.Unmarshal(flagsJSON, &out.AIFlags)
	_ = json.Unmarshal(photosJSON, &out.Photos)
	return out, nil
}

func (p *Posts) UpdateText(ctx context.Context, postID uuid.UUID, newText *string, editedAt time.Time, status string, toxicScore *float64, highRisk bool, aiFlags map[string]any) error {
	flags, err := json.Marshal(aiFlags)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(ctx, `
		update posts
		set description=$2, edited_at=$3, status=$4, toxic_score=$5, high_risk=$6, ai_flags=$7, moderation_comment=null
		where id=$1
	`, postID, nullIfEmptyStrPtr(newText), editedAt, status, toxicScore, highRisk, flags)
	return err
}

func (p *Posts) SetHidden(ctx context.Context, postID uuid.UUID, isHidden bool) error {
	ct, err := p.db.Exec(ctx, `update posts set is_hidden=$2 where id=$1`, postID, isHidden)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Posts) Delete(ctx context.Context, postID uuid.UUID, authorID uuid.UUID) error {
	ct, err := p.db.Exec(ctx, `delete from posts where id=$1 and author_id=$2`, postID, authorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Posts) CountCreatedByAuthorSince(ctx context.Context, authorID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := p.db.QueryRow(ctx, `
		select count(*)
		from posts
		where author_id=$1 and created_at >= $2
	`, authorID, since).Scan(&count)
	return count, err
}

func (p *Posts) ListByMonument(ctx context.Context, monumentID uuid.UUID) ([]Post, error) {
	rows, err := p.db.Query(ctx, `
		select p.id, p.monument_id, p.author_id, p.description, p.status, p.is_hidden, p.edited_at, p.created_at, p.moderation_comment, p.toxic_score, p.high_risk, p.ai_flags,
		       p.is_archived, coalesce(p.archive_reason, ''), coalesce(p.restore_decision_status, 'none'), p.archived_at, p.restored_at,
		       coalesce(u.username, ''),
		       coalesce((select thumbnail_path from photos where post_id=p.id order by uploaded_at asc limit 1), ''),
		       coalesce((select json_agg(ph.*) from (select ph.* from photos ph where ph.post_id=p.id order by ph.uploaded_at asc) ph), '[]'::json) as photos,
		       coalesce(m.name, '') as monument_name,
		       coalesce(m.is_orphaned, false) as monument_is_orphaned
		from posts p
		left join users u on p.author_id = u.id
		left join monuments m on p.monument_id = m.id
		where p.monument_id=$1
		order by p.created_at desc
	`, monumentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var r Post
		var desc *string
		var modComment *string
		var toxicScore *float64
		var flagsJSON []byte
		var photosJSON []byte
		if err := rows.Scan(&r.ID, &r.MonumentID, &r.AuthorID, &desc, &r.Status, &r.IsHidden, &r.EditedAt, &r.CreatedAt, &modComment, &toxicScore, &r.HighRisk, &flagsJSON, &r.IsArchived, &r.ArchiveReason, &r.RestoreDecisionStatus, &r.ArchivedAt, &r.RestoredAt, &r.AuthorName, &r.Thumbnail, &photosJSON, &r.MonumentName, &r.MonumentIsOrphaned); err != nil {
			return nil, err
		}
		if desc != nil {
			r.Description = *desc
		}
		r.ModerationComment = modComment
		r.ToxicScore = toxicScore
		_ = json.Unmarshal(flagsJSON, &r.AIFlags)
		_ = json.Unmarshal(photosJSON, &r.Photos)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *Posts) List(ctx context.Context, status string, limit, offset int) ([]Post, error) {
	query := `
		select p.id, p.monument_id, p.author_id, p.description, p.status, p.is_hidden, p.edited_at, p.created_at, p.moderation_comment, p.toxic_score, p.high_risk, p.ai_flags,
		       p.is_archived, coalesce(p.archive_reason, ''), coalesce(p.restore_decision_status, 'none'), p.archived_at, p.restored_at,
		       coalesce(u.username, ''),
		       coalesce((select thumbnail_path from photos where post_id=p.id order by uploaded_at asc limit 1), ''),
		       coalesce((select json_agg(ph.*) from (select ph.* from photos ph where ph.post_id=p.id order by ph.uploaded_at asc) ph), '[]'::json) as photos,
		       coalesce(m.name, '') as monument_name,
		       coalesce(m.is_orphaned, false) as monument_is_orphaned
		from posts p
		left join monuments m on p.monument_id = m.id
		left join users u on p.author_id = u.id
		where 1=1
	`
	args := []any{}
	if status != "" {
		query += " and p.status=$1"
		args = append(args, status)
		if status == "pending" {
			// Исключаем посты, чей памятник еще сам не одобрен (чтобы не дублировать в модерации)
			query += " and (m.status is null or m.status = 'approved')"
		}
	}
	query += fmt.Sprintf(" order by p.created_at desc limit $%d offset $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var p1 Post
		var desc *string
		var modComment *string
		var toxicScore *float64
		var flagsJSON []byte
		var photosJSON []byte
		if err := rows.Scan(&p1.ID, &p1.MonumentID, &p1.AuthorID, &desc, &p1.Status, &p1.IsHidden, &p1.EditedAt, &p1.CreatedAt, &modComment, &toxicScore, &p1.HighRisk, &flagsJSON, &p1.IsArchived, &p1.ArchiveReason, &p1.RestoreDecisionStatus, &p1.ArchivedAt, &p1.RestoredAt, &p1.AuthorName, &p1.Thumbnail, &photosJSON, &p1.MonumentName, &p1.MonumentIsOrphaned); err != nil {
			return nil, err
		}
		if desc != nil {
			p1.Description = *desc
		}
		p1.ModerationComment = modComment
		p1.ToxicScore = toxicScore
		_ = json.Unmarshal(flagsJSON, &p1.AIFlags)
		_ = json.Unmarshal(photosJSON, &p1.Photos)
		out = append(out, p1)
	}
	return out, rows.Err()
}

func (p *Posts) SetStatus(ctx context.Context, id uuid.UUID, status string, comment *string) error {
	_, err := p.db.Exec(ctx, `update posts set status=$2, moderation_comment=$3 where id=$1`, id, status, comment)
	return err
}

func (p *Posts) Archive(ctx context.Context, postID uuid.UUID, reason string, decisionStatus string) error {
	if strings.TrimSpace(decisionStatus) == "" {
		decisionStatus = "pending"
	}
	ct, err := p.db.Exec(ctx, `
		update posts
		set is_archived = true,
		    archive_reason = $2,
		    restore_decision_status = $3,
		    archived_at = now(),
		    restored_at = null
		where id = $1
	`, postID, nullIfEmpty(reason), decisionStatus)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Posts) RestoreFromArchive(ctx context.Context, postID uuid.UUID, status string) error {
	if strings.TrimSpace(status) == "" {
		status = "approved"
	}
	ct, err := p.db.Exec(ctx, `
		update posts
		set is_archived = false,
		    archive_reason = null,
		    restore_decision_status = 'accepted',
		    archived_at = null,
		    restored_at = now(),
		    status = $2,
		    is_hidden = false
		where id = $1
	`, postID, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Posts) SetRestoreDecision(ctx context.Context, postID uuid.UUID, decisionStatus string) error {
	ct, err := p.db.Exec(ctx, `
		update posts
		set restore_decision_status = $2
		where id = $1
	`, postID, decisionStatus)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Posts) DeleteByID(ctx context.Context, postID uuid.UUID) error {
	ct, err := p.db.Exec(ctx, `delete from posts where id=$1`, postID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Posts) GetStats(ctx context.Context) (map[string]int64, error) {
	rows, err := p.db.Query(ctx, `
		select status, count(*)
		from posts
		group by status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	var total int64
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
		total += count
	}
	stats["total"] = total
	return stats, nil
}

func (p *Posts) GetDynamics(ctx context.Context, days int) ([]map[string]any, error) {
	rows, err := p.db.Query(ctx, `
		select date_trunc('day', created_at) as day, count(*)
		from posts
		where created_at > now() - $1 * interval '1 day'
		group by day
		order by day asc
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var day time.Time
		var count int64
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"date":  day.Format("2006-01-02"),
			"count": count,
		})
	}
	return out, nil
}

func (p *Posts) GetTopAuthors(ctx context.Context, limit int) ([]map[string]any, error) {
	rows, err := p.db.Query(ctx, `
		select u.id, u.username, count(p.id) as approved_posts
		from users u
		join posts p on p.author_id = u.id
		where p.status = 'approved'
		group by u.id, u.username
		order by approved_posts desc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var username string
		var count int64
		if err := rows.Scan(&id, &username, &count); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":             id,
			"username":       username,
			"approved_posts": count,
		})
	}
	return out, nil
}

func (p *Posts) UpdateAIFlags(ctx context.Context, id uuid.UUID, highRisk bool, aiFlags map[string]any) error {
	flags, err := json.Marshal(aiFlags)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(ctx, `update posts set high_risk=$2, ai_flags=$3 where id=$1`, id, highRisk, flags)
	return err
}

func (p *Posts) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]Post, error) {
	rows, err := p.db.Query(ctx, `
		select p.id, p.monument_id, p.author_id, p.description, p.status, p.is_hidden, p.edited_at, p.created_at, p.moderation_comment, p.toxic_score, p.high_risk, p.ai_flags,
		       p.is_archived, coalesce(p.archive_reason, ''), coalesce(p.restore_decision_status, 'none'), p.archived_at, p.restored_at,
		       coalesce((select thumbnail_path from photos where post_id=p.id order by uploaded_at asc limit 1), ''),
		       coalesce((select json_agg(ph.*) from (select ph.* from photos ph where ph.post_id=p.id order by ph.uploaded_at asc) ph), '[]'::json) as photos,
		       coalesce(m.name, '') as monument_name,
		       coalesce(m.is_orphaned, false) as monument_is_orphaned
		from posts p
		left join monuments m on p.monument_id = m.id
		where p.author_id=$1
		order by p.created_at desc
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var r Post
		var desc *string
		var modComment *string
		var toxicScore *float64
		var flagsJSON []byte
		var photosJSON []byte
		if err := rows.Scan(&r.ID, &r.MonumentID, &r.AuthorID, &desc, &r.Status, &r.IsHidden, &r.EditedAt, &r.CreatedAt, &modComment, &toxicScore, &r.HighRisk, &flagsJSON, &r.IsArchived, &r.ArchiveReason, &r.RestoreDecisionStatus, &r.ArchivedAt, &r.RestoredAt, &r.Thumbnail, &photosJSON, &r.MonumentName, &r.MonumentIsOrphaned); err != nil {
			return nil, err
		}
		if desc != nil {
			r.Description = *desc
		}
		r.ModerationComment = modComment
		r.ToxicScore = toxicScore
		_ = json.Unmarshal(flagsJSON, &r.AIFlags)
		_ = json.Unmarshal(photosJSON, &r.Photos)
		out = append(out, r)
	}
	return out, rows.Err()
}

func nullIfEmptyStrPtr(s *string) any {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return v
}
