package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type Signals struct {
	db *pgxpool.Pool
}

func NewSignals(db *pgxpool.Pool) *Signals {
	return &Signals{db: db}
}

type Signal struct {
	ID                uuid.UUID      `json:"id"`
	MonumentID        *uuid.UUID     `json:"monument_id,omitempty"`
	MonumentName      *string        `json:"monument_name,omitempty"`
	Lon               *float64       `json:"lon,omitempty"`
	Lat               *float64       `json:"lat,omitempty"`
	Region            string         `json:"region"`
	SignalType        string         `json:"signal_type"`
	Urgency           string         `json:"urgency"`
	Description       string         `json:"description"`
	AuthorID          *uuid.UUID     `json:"author_id,omitempty"`
	AuthorName        string         `json:"author_name,omitempty"`
	Thumbnail         string         `json:"thumbnail,omitempty"`
	Status            string         `json:"status"`
	OfficialResponse  *string        `json:"official_response,omitempty"`
	ResolutionKind    *string        `json:"resolution_kind,omitempty"`
	ResolutionComment *string        `json:"resolution_comment,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	ResolvedAt        *time.Time     `json:"resolved_at,omitempty"`
	HighRisk          bool           `json:"high_risk"`
	AIFlags           map[string]any `json:"ai_flags"`
	SupportCount      int            `json:"support_count"`
	IsSupported       bool           `json:"is_supported,omitempty"`
	Photos            []SignalPhoto  `json:"photos,omitempty"`
}

type UpdateOwnSignalParams struct {
	ID          uuid.UUID
	AuthorID    uuid.UUID
	SignalType  string
	Description string
	Urgency     string
	HighRisk    bool
	AIFlags     map[string]any
}

func (s *Signals) Create(ctx context.Context, in Signal) (uuid.UUID, error) {
	flags, err := json.Marshal(in.AIFlags)
	if err != nil {
		return uuid.Nil, err
	}

	id := ids.NewV7()
	err = s.db.QueryRow(ctx, `
		insert into signals (id, monument_id, monument_name, monument_location, region, signal_type, urgency, description, author_id, status, official_response, resolution_kind, resolution_comment, high_risk, ai_flags)
		values ($1, $2, $3, case when ($4::double precision) is not null and ($5::double precision) is not null then ST_SetSRID(ST_Point($4::double precision, $5::double precision), 4326) else null end, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		returning id
	`, id, in.MonumentID, in.MonumentName, in.Lon, in.Lat, in.Region, in.SignalType, in.Urgency, in.Description, in.AuthorID, in.Status, in.OfficialResponse, in.ResolutionKind, in.ResolutionComment, in.HighRisk, flags).Scan(&id)
	return id, err
}

func (s *Signals) GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (Signal, error) {
	var out Signal
	var monumentID *uuid.UUID
	var monumentName *string
	var authorID *uuid.UUID
	var lon *float64
	var lat *float64
	var officialResp *string
	var resolutionKind *string
	var resolutionComment *string
	var resolvedAt *time.Time
	var flagsJSON []byte
	var photosJSON []byte

	userSupportCheck := "false"
	args := []any{id}
	if userID != nil {
		args = append(args, *userID)
		userSupportCheck = "exists(select 1 from signal_supports ss where ss.signal_id=si.id and ss.user_id=$2)"
	}

	err := s.db.QueryRow(ctx, `
		select
			si.id, si.monument_id, coalesce(si.monument_name, m.name),
			ST_X(si.monument_location), ST_Y(si.monument_location),
			si.signal_type, si.urgency, si.description, si.author_id, si.status, si.official_response, si.resolution_kind, si.resolution_comment, si.created_at, si.resolved_at,
			si.region,
			si.high_risk, si.ai_flags,
			(select count(*) from signal_supports ss where ss.signal_id=si.id) as support_count,
			coalesce(u.username, ''),
			coalesce((select thumbnail_path from signal_photos where signal_id=si.id and is_hidden=false order by uploaded_at asc limit 1), ''),
			coalesce((select json_agg(ph.*) from (select ph.* from signal_photos ph where ph.signal_id=si.id and ph.is_hidden=false order by ph.uploaded_at asc) ph), '[]'::json) as photos,
			`+userSupportCheck+` as is_supported
		from signals si
		left join users u on si.author_id = u.id
		left join monuments m on si.monument_id = m.id
		where si.id=$1
	`, args...).Scan(
		&out.ID,
		&monumentID,
		&monumentName,
		&lon,
		&lat,
		&out.SignalType,
		&out.Urgency,
		&out.Description,
		&authorID,
		&out.Status,
		&officialResp,
		&resolutionKind,
		&resolutionComment,
		&out.CreatedAt,
		&resolvedAt,
		&out.Region,
		&out.HighRisk,
		&flagsJSON,
		&out.SupportCount,
		&out.AuthorName,
		&out.Thumbnail,
		&photosJSON,
		&out.IsSupported,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Signal{}, ErrNotFound
	}
	if err != nil {
		return Signal{}, err
	}
	out.MonumentID = monumentID
	out.MonumentName = monumentName
	out.AuthorID = authorID
	out.OfficialResponse = officialResp
	out.ResolutionKind = resolutionKind
	out.ResolutionComment = resolutionComment
	out.ResolvedAt = resolvedAt
	out.Lon = lon
	out.Lat = lat
	_ = json.Unmarshal(flagsJSON, &out.AIFlags)
	_ = json.Unmarshal(photosJSON, &out.Photos)
	return out, nil
}

type ListSignalsFilter struct {
	Status        string
	SignalType    string
	Urgency       string
	Region        string
	ExcludeRegion string
	Limit         int
	Offset        int
	UserID        *uuid.UUID
}

func (s *Signals) List(ctx context.Context, f ListSignalsFilter) ([]Signal, error) {
	if f.Limit <= 0 || f.Limit > 50 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	args := []any{f.Limit, f.Offset}
	where := []string{"1=1"}

	if strings.TrimSpace(f.Status) != "" {
		args = append(args, f.Status)
		where = append(where, "si.status=$"+strconv.Itoa(len(args)))
	}
	if strings.TrimSpace(f.SignalType) != "" {
		args = append(args, f.SignalType)
		where = append(where, "si.signal_type=$"+strconv.Itoa(len(args)))
	}
	if strings.TrimSpace(f.Urgency) != "" {
		args = append(args, f.Urgency)
		where = append(where, "si.urgency=$"+strconv.Itoa(len(args)))
	}
	if strings.TrimSpace(f.Region) != "" {
		args = append(args, f.Region)
		where = append(where, "si.region=$"+strconv.Itoa(len(args)))
	}
	if strings.TrimSpace(f.ExcludeRegion) != "" {
		args = append(args, f.ExcludeRegion)
		where = append(where, "coalesce(si.region, '') <> $"+strconv.Itoa(len(args)))
	}

	userSupportCheck := "false"
	if f.UserID != nil {
		args = append(args, *f.UserID)
		userSupportCheck = "exists(select 1 from signal_supports ss where ss.signal_id=si.id and ss.user_id=$" + strconv.Itoa(len(args)) + ")"
	}

	q := `
		select
			si.id, si.monument_id, coalesce(si.monument_name, m.name),
			ST_X(si.monument_location), ST_Y(si.monument_location),
			si.signal_type, si.urgency, si.description, si.author_id, si.status, si.official_response, si.resolution_kind, si.resolution_comment, si.created_at, si.resolved_at,
			si.region,
			si.high_risk, si.ai_flags,
			(select count(*) from signal_supports ss where ss.signal_id=si.id) as support_count,
			coalesce(u.username, ''),
			coalesce((select thumbnail_path from signal_photos where signal_id=si.id and is_hidden=false order by uploaded_at asc limit 1), ''),
			coalesce((select json_agg(ph.*) from (select ph.* from signal_photos ph where ph.signal_id=si.id and ph.is_hidden=false order by ph.uploaded_at asc) ph), '[]'::json) as photos,
			` + userSupportCheck + ` as is_supported
		from signals si
		left join users u on si.author_id = u.id
		left join monuments m on si.monument_id = m.id
		where ` + strings.Join(where, " and ") + `
		order by si.created_at desc
		limit $1 offset $2
	`

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Signal
	for rows.Next() {
		var r Signal
		var monumentID *uuid.UUID
		var monumentName *string
		var authorID *uuid.UUID
		var lon *float64
		var lat *float64
		var officialResp *string
		var resolutionKind *string
		var resolutionComment *string
		var resolvedAt *time.Time
		var photosJSON []byte
		var flagsJSON []byte
		if err := rows.Scan(
			&r.ID,
			&monumentID,
			&monumentName,
			&lon,
			&lat,
			&r.SignalType,
			&r.Urgency,
			&r.Description,
			&authorID,
			&r.Status,
			&officialResp,
			&resolutionKind,
			&resolutionComment,
			&r.CreatedAt,
			&resolvedAt,
			&r.Region,
			&r.HighRisk,
			&flagsJSON,
			&r.SupportCount,
			&r.AuthorName,
			&r.Thumbnail,
			&photosJSON,
			&r.IsSupported,
		); err != nil {
			return nil, err
		}
		r.MonumentID = monumentID
		r.MonumentName = monumentName
		r.AuthorID = authorID
		r.OfficialResponse = officialResp
		r.ResolutionKind = resolutionKind
		r.ResolutionComment = resolutionComment
		r.ResolvedAt = resolvedAt
		r.Lon = lon
		r.Lat = lat
		_ = json.Unmarshal(flagsJSON, &r.AIFlags)
		_ = json.Unmarshal(photosJSON, &r.Photos)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Signals) ListByMonumentConfirmed(ctx context.Context, monumentID uuid.UUID) ([]Signal, error) {
	rows, err := s.db.Query(ctx, `
		select
			si.id, si.monument_id, si.monument_name,
			null, null,
			si.signal_type, si.urgency, si.description, si.author_id, si.status, si.official_response, si.resolution_kind, si.resolution_comment, si.created_at, si.resolved_at,
			si.region,
			si.high_risk, si.ai_flags,
			(select count(*) from signal_supports ss where ss.signal_id=si.id) as support_count
		from signals si
		where si.monument_id=$1 and si.status in ('confirmed','resolved')
		order by si.created_at desc
	`, monumentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Signal
	for rows.Next() {
		var r Signal
		var monumentID2 *uuid.UUID
		var monumentName *string
		var authorID *uuid.UUID
		var lon *float64
		var lat *float64
		var officialResp *string
		var resolutionKind *string
		var resolutionComment *string
		var resolvedAt *time.Time
		var flagsJSON []byte

		if err := rows.Scan(
			&r.ID,
			&monumentID2,
			&monumentName,
			&lon,
			&lat,
			&r.SignalType,
			&r.Urgency,
			&r.Description,
			&authorID,
			&r.Status,
			&officialResp,
			&resolutionKind,
			&resolutionComment,
			&r.CreatedAt,
			&resolvedAt,
			&r.Region,
			&r.HighRisk,
			&flagsJSON,
			&r.SupportCount,
		); err != nil {
			return nil, err
		}
		r.MonumentID = monumentID2
		r.MonumentName = monumentName
		r.AuthorID = authorID
		r.OfficialResponse = officialResp
		r.ResolutionKind = resolutionKind
		r.ResolutionComment = resolutionComment
		r.ResolvedAt = resolvedAt
		r.Lon = lon
		r.Lat = lat
		_ = json.Unmarshal(flagsJSON, &r.AIFlags)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Signals) AddSupport(ctx context.Context, signalID, userID uuid.UUID) (bool, error) {
	ct, err := s.db.Exec(ctx, `insert into signal_supports (signal_id, user_id) values ($1,$2) on conflict do nothing`, signalID, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (s *Signals) RemoveSupport(ctx context.Context, signalID, userID uuid.UUID) (bool, error) {
	ct, err := s.db.Exec(ctx, `delete from signal_supports where signal_id=$1 and user_id=$2`, signalID, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (s *Signals) HasSupport(ctx context.Context, signalID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `select exists(select 1 from signal_supports where signal_id=$1 and user_id=$2)`, signalID, userID).Scan(&exists)
	return exists, err
}

func (s *Signals) UpdateStatus(ctx context.Context, signalID uuid.UUID, status string, officialResponse *string, resolvedAt *time.Time, urgency *string, resolutionKind *string, resolutionComment *string) error {
	ct, err := s.db.Exec(ctx, `
		update signals
		set status=$2, official_response=$3, resolved_at=$4, urgency=coalesce($5, urgency), resolution_kind=$6, resolution_comment=$7
		where id=$1
	`, signalID, status, officialResponse, resolvedAt, urgency, resolutionKind, resolutionComment)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Signals) UpdateOwn(ctx context.Context, in UpdateOwnSignalParams) error {
	flags, err := json.Marshal(in.AIFlags)
	if err != nil {
		return err
	}
	ct, err := s.db.Exec(ctx, `
		update signals
		set signal_type=$3,
		    description=$4,
		    urgency=$5,
		    status='pending',
		    official_response=null,
		    resolved_at=null,
		    resolution_kind=null,
		    resolution_comment=null,
		    high_risk=$6,
		    ai_flags=$7
		where id=$1 and author_id=$2
	`, in.ID, in.AuthorID, in.SignalType, in.Description, in.Urgency, in.HighRisk, flags)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Signals) DeleteByAuthor(ctx context.Context, signalID, authorID uuid.UUID) error {
	ct, err := s.db.Exec(ctx, `delete from signals where id=$1 and author_id=$2`, signalID, authorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Signals) Delete(ctx context.Context, signalID uuid.UUID) error {
	ct, err := s.db.Exec(ctx, `delete from signals where id=$1`, signalID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type SignalMapPoint struct {
	ID         uuid.UUID  `json:"id"`
	Lon        float64    `json:"lon"`
	Lat        float64    `json:"lat"`
	Urgency    string     `json:"urgency"`
	SignalType string     `json:"signal_type"`
	MonumentID *uuid.UUID `json:"monument_id,omitempty"`
}

func (s *Signals) ConfirmedMapPoints(ctx context.Context) ([]SignalMapPoint, error) {
	rows, err := s.db.Query(ctx, `
		select id, ST_X(monument_location), ST_Y(monument_location), urgency
		from signals
		where status='confirmed' and monument_location is not null
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SignalMapPoint
	for rows.Next() {
		var p SignalMapPoint
		if err := rows.Scan(&p.ID, &p.Lon, &p.Lat, &p.Urgency); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Signals) CountCreatedByAuthorSince(ctx context.Context, authorID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		select count(*)
		from signals
		where author_id=$1 and created_at >= $2
	`, authorID, since).Scan(&count)
	return count, err
}

func (s *Signals) GetStats(ctx context.Context) (map[string]any, error) {
	var total int64
	err := s.db.QueryRow(ctx, `select count(*) from signals`).Scan(&total)
	if err != nil {
		return nil, err
	}

	statusRows, err := s.db.Query(ctx, `select status, count(*) from signals group by status`)
	if err != nil {
		return nil, err
	}
	defer statusRows.Close()
	byStatus := make(map[string]int64)
	for statusRows.Next() {
		var st string
		var c int64
		if err := statusRows.Scan(&st, &c); err != nil {
			return nil, err
		}
		byStatus[st] = c
	}

	urgencyRows, err := s.db.Query(ctx, `select urgency, count(*) from signals group by urgency`)
	if err != nil {
		return nil, err
	}
	defer urgencyRows.Close()
	byUrgency := make(map[string]int64)
	for urgencyRows.Next() {
		var urg string
		var c int64
		if err := urgencyRows.Scan(&urg, &c); err != nil {
			return nil, err
		}
		byUrgency[urg] = c
	}

	typeRows, err := s.db.Query(ctx, `select signal_type, count(*) from signals group by signal_type`)
	if err != nil {
		return nil, err
	}
	defer typeRows.Close()
	byType := make(map[string]int64)
	for typeRows.Next() {
		var t string
		var c int64
		if err := typeRows.Scan(&t, &c); err != nil {
			return nil, err
		}
		byType[t] = c
	}

	resolutionRows, err := s.db.Query(ctx, `
		select coalesce(resolution_kind, ''), count(*)
		from signals
		where status='resolved'
		group by resolution_kind
	`)
	if err != nil {
		return nil, err
	}
	defer resolutionRows.Close()
	byResolution := make(map[string]int64)
	var resolvedTotal int64
	for resolutionRows.Next() {
		var resolutionKind string
		var c int64
		if err := resolutionRows.Scan(&resolutionKind, &c); err != nil {
			return nil, err
		}
		if strings.TrimSpace(resolutionKind) == "" {
			resolutionKind = "unspecified"
		}
		byResolution[resolutionKind] = c
		resolvedTotal += c
	}

	return map[string]any{
		"total":          total,
		"resolved":       byStatus["resolved"],
		"by_status":      byStatus,
		"by_urgency":     byUrgency,
		"by_type":        byType,
		"by_resolution":  byResolution,
		"resolved_total": resolvedTotal,
	}, nil
}

func (s *Signals) GetDynamics(ctx context.Context, days int) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		select date_trunc('day', created_at) as day, count(*)
		from signals
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

func (s *Signals) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]Signal, error) {
	rows, err := s.db.Query(ctx, `
		select si.id, si.monument_id, coalesce(si.monument_name, m.name),
			ST_X(si.monument_location), ST_Y(si.monument_location),
			si.signal_type, si.urgency, si.description, si.author_id, si.status, si.official_response, si.resolution_kind, si.resolution_comment, si.created_at, si.resolved_at,
			si.region, si.high_risk, si.ai_flags,
			(select count(*) from signal_supports ss where ss.signal_id=si.id) as support_count,
			coalesce((select thumbnail_path from signal_photos where signal_id=si.id and is_hidden=false order by uploaded_at asc limit 1), '') as thumbnail,
			coalesce((select json_agg(ph.*) from (select ph.* from signal_photos ph where ph.signal_id=si.id and ph.is_hidden=false order by ph.uploaded_at asc) ph), '[]'::json) as photos
		from signals si
		left join monuments m on si.monument_id = m.id
		where si.author_id=$1
		order by si.created_at desc
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Signal
	for rows.Next() {
		var r Signal
		var monID *uuid.UUID
		var monName *string
		var lon, lat *float64
		var offResp *string
		var resolutionKind *string
		var resolutionComment *string
		var resAt *time.Time
		var region string
		var flagsJSON []byte
		var photosJSON []byte
		if err := rows.Scan(&r.ID, &monID, &monName, &lon, &lat, &r.SignalType, &r.Urgency, &r.Description, &r.AuthorID, &r.Status, &offResp, &resolutionKind, &resolutionComment, &r.CreatedAt, &resAt, &region, &r.HighRisk, &flagsJSON, &r.SupportCount, &r.Thumbnail, &photosJSON); err != nil {
			return nil, err
		}
		r.MonumentID = monID
		r.MonumentName = monName
		r.Lon = lon
		r.Lat = lat
		r.Region = region
		r.OfficialResponse = offResp
		r.ResolutionKind = resolutionKind
		r.ResolutionComment = resolutionComment
		r.ResolvedAt = resAt
		_ = json.Unmarshal(flagsJSON, &r.AIFlags)
		_ = json.Unmarshal(photosJSON, &r.Photos)
		out = append(out, r)
	}
	return out, rows.Err()
}
