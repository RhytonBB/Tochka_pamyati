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

type Monuments struct {
	db *pgxpool.Pool
}

func NewMonuments(db *pgxpool.Pool) *Monuments {
	return &Monuments{db: db}
}

type Monument struct {
	ID                uuid.UUID      `json:"id"`
	Name              string         `json:"name"`
	Lat               float64        `json:"lat"`
	Lon               float64        `json:"lon"`
	Region            string         `json:"region"`
	Properties        map[string]any `json:"properties"`
	Status            string         `json:"status"`
	AuthorID          *uuid.UUID     `json:"author_id,omitempty"`
	AuthorName        string         `json:"author_name,omitempty"`
	Thumbnail         string         `json:"thumbnail,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	ModerationComment *string        `json:"moderation_comment,omitempty"`
	HighRisk          bool           `json:"high_risk"`
	AIFlags           map[string]any `json:"ai_flags"`
	Photos            []Photo        `json:"photos,omitempty"`
	IsEditRequest     bool           `json:"is_edit_request,omitempty"`
	IsOrphaned        bool           `json:"is_orphaned"`
	OrphanedAt        *time.Time     `json:"orphaned_at,omitempty"`
	OrphanedByUserID  *uuid.UUID     `json:"orphaned_by_user_id,omitempty"`
}

func (m *Monuments) Create(ctx context.Context, name string, lon, lat float64, region string, properties map[string]any, status string, authorID *uuid.UUID, highRisk bool, aiFlags map[string]any) (uuid.UUID, error) {
	props, err := json.Marshal(properties)
	if err != nil {
		return uuid.Nil, err
	}
	flags, err := json.Marshal(aiFlags)
	if err != nil {
		return uuid.Nil, err
	}

	id := ids.NewV7()
	err = m.db.QueryRow(ctx, `
		insert into monuments (id, name, geom, region, properties, status, author_id, high_risk, ai_flags)
		values ($1, $2, ST_SetSRID(ST_Point($3, $4), 4326), $5, $6, $7, $8, $9, $10)
		returning id
	`, id, name, lon, lat, region, props, status, authorID, highRisk, flags).Scan(&id)
	return id, err
}

type NearbyMonument struct {
	ID   uuid.UUID
	Name string
	Dist float64
}

func (m *Monuments) FindDuplicates(ctx context.Context, name string, lon, lat float64) ([]NearbyMonument, error) {
	rows, err := m.db.Query(ctx, `
		select id, name, ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint($2,$3),4326)::geography) as dist_m
		from monuments
		where ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint($2,$3),4326)::geography, 100)
		  and similarity(lower(name), lower($1)) > 0.4
		order by dist_m asc
		limit 5
	`, strings.TrimSpace(name), lon, lat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []NearbyMonument
	for rows.Next() {
		var r NearbyMonument
		if err := rows.Scan(&r.ID, &r.Name, &r.Dist); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (m *Monuments) GetByID(ctx context.Context, id uuid.UUID) (Monument, error) {
	var out Monument
	var propsJSON []byte
	var flagsJSON []byte
	var authorID *uuid.UUID
	var lon float64
	var lat float64
	var postDesc string

	var photosJSON []byte
	err := m.db.QueryRow(ctx, `
		select
			m.id, m.name, ST_X(m.geom), ST_Y(m.geom), coalesce(m.region, ''), m.properties, m.status, m.author_id, m.created_at, m.updated_at, m.high_risk, m.ai_flags,
			m.is_orphaned, m.orphaned_at, m.orphaned_by_user_id,
			coalesce(u.username, ''),
			coalesce((select ph.thumbnail_path from photos ph join posts po on ph.post_id=po.id where po.monument_id=m.id order by ph.uploaded_at asc limit 1), '') as thumbnail,
			coalesce((select po.description from posts po where po.monument_id=m.id order by po.created_at asc limit 1), '') as post_desc,
			coalesce((select json_agg(ph.*) from (select ph.* from photos ph join posts po on ph.post_id=po.id where po.monument_id=m.id order by ph.uploaded_at asc) ph), '[]'::json) as photos
		from monuments m
		left join users u on m.author_id = u.id
		where m.id=$1
	`, id).Scan(
		&out.ID,
		&out.Name,
		&lon,
		&lat,
		&out.Region,
		&propsJSON,
		&out.Status,
		&authorID,
		&out.CreatedAt,
		&out.UpdatedAt,
		&out.HighRisk,
		&flagsJSON,
		&out.IsOrphaned,
		&out.OrphanedAt,
		&out.OrphanedByUserID,
		&out.AuthorName,
		&out.Thumbnail,
		&postDesc,
		&photosJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		fmt.Printf("GetByID ErrNoRows: %v\n", id)
		return Monument{}, ErrNotFound
	}
	if err != nil {
		fmt.Printf("GetByID OtherError: %#v\n", err)
		return Monument{}, err
	}
	out.Lon = lon
	out.Lat = lat
	out.AuthorID = authorID
	_ = json.Unmarshal(propsJSON, &out.Properties)
	_ = json.Unmarshal(flagsJSON, &out.AIFlags)
	_ = json.Unmarshal(photosJSON, &out.Photos)

	// Fallback для описания, если его нет в properties
	if out.Properties == nil {
		out.Properties = make(map[string]any)
	}
	if _, ok := out.Properties["description"]; !ok && postDesc != "" {
		out.Properties["description"] = postDesc
	}
	return out, nil
}

func (m *Monuments) List(ctx context.Context, status string, limit, offset int) ([]Monument, error) {
	query := `
		select m.id, m.name, ST_X(m.geom), ST_Y(m.geom), m.properties, m.status, m.author_id, m.created_at, m.updated_at, m.high_risk, m.ai_flags,
		       m.is_orphaned, m.orphaned_at, m.orphaned_by_user_id,
		       coalesce(u.username, ''),
		       coalesce((select ph.thumbnail_path from photos ph join posts po on ph.post_id=po.id where po.monument_id=m.id order by ph.uploaded_at asc limit 1), ''),
		       coalesce((select po.description from posts po where po.monument_id=m.id order by po.created_at asc limit 1), ''),
		       coalesce((select json_agg(ph.*) from (select ph.* from photos ph join posts po on ph.post_id=po.id where po.monument_id=m.id order by ph.uploaded_at asc) ph), '[]'::json) as photos
		from monuments m
		left join users u on m.author_id = u.id
		where 1=1
	`
	args := []any{}
	if status != "" {
		query += " and m.status=$1"
		args = append(args, status)
	}
	query += fmt.Sprintf(" order by m.created_at desc limit $%d offset $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := m.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Monument
	for rows.Next() {
		var r Monument
		var propsJSON []byte
		var flagsJSON []byte
		var authorID *uuid.UUID
		var lon float64
		var lat float64
		var photosJSON []byte
		var postDesc string
		if err := rows.Scan(&r.ID, &r.Name, &lon, &lat, &propsJSON, &r.Status, &authorID, &r.CreatedAt, &r.UpdatedAt, &r.HighRisk, &flagsJSON, &r.IsOrphaned, &r.OrphanedAt, &r.OrphanedByUserID, &r.AuthorName, &r.Thumbnail, &postDesc, &photosJSON); err != nil {
			return nil, err
		}
		r.Lon = lon
		r.Lat = lat
		r.AuthorID = authorID
		_ = json.Unmarshal(propsJSON, &r.Properties)
		_ = json.Unmarshal(flagsJSON, &r.AIFlags)
		_ = json.Unmarshal(photosJSON, &r.Photos)

		if r.Properties == nil {
			r.Properties = make(map[string]any)
		}
		if _, ok := r.Properties["description"]; !ok && postDesc != "" {
			r.Properties["description"] = postDesc
		}

		out = append(out, r)
	}
	return out, rows.Err()
}

func (m *Monuments) SetStatus(ctx context.Context, id uuid.UUID, status string, comment *string) error {
	_, err := m.db.Exec(ctx, `update monuments set status=$2, moderation_comment=$3, updated_at=now() where id=$1`, id, status, comment)
	return err
}

func (m *Monuments) StripToNameOnly(ctx context.Context, id uuid.UUID, comment *string) error {
	_, err := m.db.Exec(ctx, `
		update monuments
		set properties='{}'::jsonb,
		    moderation_comment=$2,
		    updated_at=now()
		where id=$1
	`, id, comment)
	return err
}

func (m *Monuments) DeleteByAuthor(ctx context.Context, id, authorID uuid.UUID) error {
	ct, err := m.db.Exec(ctx, `delete from monuments where id=$1 and author_id=$2`, id, authorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Monuments) DeleteByID(ctx context.Context, id uuid.UUID) error {
	ct, err := m.db.Exec(ctx, `delete from monuments where id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Monuments) MarkOrphaned(ctx context.Context, id uuid.UUID, orphanedBy *uuid.UUID, status string, comment *string) error {
	ct, err := m.db.Exec(ctx, `
		update monuments
		set author_id = null,
		    is_orphaned = true,
		    orphaned_at = now(),
		    orphaned_by_user_id = $2,
		    status = $3,
		    properties = '{}'::jsonb,
		    moderation_comment = $4,
		    updated_at = now()
		where id = $1
	`, id, orphanedBy, status, comment)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Monuments) UpdateOrphanState(ctx context.Context, id uuid.UUID, status string, comment *string) error {
	ct, err := m.db.Exec(ctx, `
		update monuments
		set status = $2,
		    moderation_comment = $3,
		    updated_at = now()
		where id = $1
	`, id, status, comment)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Monuments) UpdateCoreFields(ctx context.Context, id uuid.UUID, name *string, lon, lat *float64) error {
	ct, err := m.db.Exec(ctx, `
		update monuments
		set
			name = coalesce($2, name),
			geom = case
				when $3::double precision is not null and $4::double precision is not null
					then ST_SetSRID(ST_Point($3, $4), 4326)
				else geom
			end,
			updated_at = now()
		where id=$1
	`, id, nullIfEmptyStrPtr(name), lon, lat)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Monuments) GetStats(ctx context.Context) (map[string]any, error) {
	rows, err := m.db.Query(ctx, `
		select status, count(*)
		from monuments
		group by status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byStatus := make(map[string]int64)
	var total int64
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		byStatus[status] = count
		total += count
	}

	regionRows, err := m.db.Query(ctx, `
		select coalesce(nullif(trim(region), ''), 'Не указан'), count(*)
		from monuments
		group by coalesce(nullif(trim(region), ''), 'Не указан')
		order by count(*) desc, coalesce(nullif(trim(region), ''), 'Не указан') asc
	`)
	if err != nil {
		return nil, err
	}
	defer regionRows.Close()

	byRegion := make([]map[string]any, 0)
	for regionRows.Next() {
		var region string
		var count int64
		if err := regionRows.Scan(&region, &count); err != nil {
			return nil, err
		}
		byRegion = append(byRegion, map[string]any{
			"region": region,
			"count":  count,
		})
	}

	return map[string]any{
		"total":     total,
		"by_status": byStatus,
		"by_region": byRegion,
	}, nil
}

func (m *Monuments) CountCreatedByAuthorSince(ctx context.Context, authorID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := m.db.QueryRow(ctx, `
		select count(*)
		from monuments
		where author_id=$1 and created_at >= $2
	`, authorID, since).Scan(&count)
	return count, err
}

func (m *Monuments) GetDynamics(ctx context.Context, days int) ([]map[string]any, error) {
	rows, err := m.db.Query(ctx, `
		select date_trunc('day', created_at) as day, count(*)
		from monuments
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

func (m *Monuments) UpdateAIFlags(ctx context.Context, id uuid.UUID, highRisk bool, aiFlags map[string]any) error {
	flags, err := json.Marshal(aiFlags)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(ctx, `update monuments set high_risk=$2, ai_flags=$3 where id=$1`, id, highRisk, flags)
	return err
}

func (m *Monuments) UpdateSubmission(ctx context.Context, id uuid.UUID, authorID uuid.UUID, name string, lon, lat float64, region string, properties map[string]any, highRisk bool, aiFlags map[string]any) error {
	props, err := json.Marshal(properties)
	if err != nil {
		return err
	}
	flags, err := json.Marshal(aiFlags)
	if err != nil {
		return err
	}
	ct, err := m.db.Exec(ctx, `
		update monuments
		set name=$3,
		    geom=ST_SetSRID(ST_Point($4, $5), 4326),
		    region=$6,
		    properties=$7,
		    status='pending',
		    moderation_comment=null,
		    high_risk=$8,
		    ai_flags=$9,
		    updated_at=now()
		where id=$1 and author_id=$2
	`, id, authorID, strings.TrimSpace(name), lon, lat, region, props, highRisk, flags)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Monuments) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]Monument, error) {
	rows, err := m.db.Query(ctx, `
		select m.id, m.name, ST_X(m.geom), ST_Y(m.geom), coalesce(m.region, ''), m.properties, m.status, m.author_id, m.created_at, m.updated_at, m.moderation_comment, m.high_risk, m.ai_flags,
		       m.is_orphaned, m.orphaned_at, m.orphaned_by_user_id,
		       coalesce((select ph.thumbnail_path from photos ph join posts po on ph.post_id=po.id where po.monument_id=m.id and po.author_id = m.author_id order by ph.uploaded_at asc limit 1), '') as thumbnail,
		       coalesce((select json_agg(ph.*) from (select ph.* from photos ph join posts po on ph.post_id=po.id where po.monument_id=m.id and po.author_id = m.author_id order by ph.uploaded_at asc) ph), '[]'::json) as photos
		from monuments m
		where m.author_id=$1
		order by m.created_at desc
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Monument
	for rows.Next() {
		var r Monument
		var lon float64
		var lat float64
		var moderationComment *string
		var propsJSON []byte
		var flagsJSON []byte
		var photosJSON []byte
		if err := rows.Scan(&r.ID, &r.Name, &lon, &lat, &r.Region, &propsJSON, &r.Status, &r.AuthorID, &r.CreatedAt, &r.UpdatedAt, &moderationComment, &r.HighRisk, &flagsJSON, &r.IsOrphaned, &r.OrphanedAt, &r.OrphanedByUserID, &r.Thumbnail, &photosJSON); err != nil {
			return nil, err
		}
		r.Lon = lon
		r.Lat = lat
		r.ModerationComment = moderationComment
		_ = json.Unmarshal(propsJSON, &r.Properties)
		_ = json.Unmarshal(flagsJSON, &r.AIFlags)
		_ = json.Unmarshal(photosJSON, &r.Photos)
		out = append(out, r)
	}
	return out, rows.Err()
}
