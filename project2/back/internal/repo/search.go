package repo

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Search struct {
	db *pgxpool.Pool
}

func NewSearch(db *pgxpool.Pool) *Search {
	return &Search{db: db}
}

type MonumentSuggestion struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Lon  float64   `json:"lon"`
	Lat  float64   `json:"lat"`
}

func (s *Search) SuggestMonuments(ctx context.Context, q string, limit int) ([]MonumentSuggestion, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []MonumentSuggestion{}, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	tsQuery := buildPrefixTSQuery(q)

	rows, err := s.db.Query(ctx, `
		select id, name, ST_X(geom), ST_Y(geom)
		from monuments
		where status='approved'
		  and (
			name ilike '%' || $1 || '%'
			or ($2 <> '' and search_tsv @@ to_tsquery('russian', $2))
			or similarity(lower(name), lower($1)) > 0.3
		  )
		order by
			(case when name ilike $1 || '%' then 0 else 1 end),
			(case when name ilike '% ' || $1 || '%' or name ilike '%-' || $1 || '%' then 0 else 1 end),
			(case when $2 <> '' and search_tsv @@ to_tsquery('russian', $2) then 0 else 1 end),
			(case when position(lower($1) in lower(name)) > 0 then position(lower($1) in lower(name)) else 9999 end),
			similarity(lower(name), lower($1)) desc,
			name asc
		limit $3
	`, q, tsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MonumentSuggestion
	for rows.Next() {
		var r MonumentSuggestion
		if err := rows.Scan(&r.ID, &r.Name, &r.Lon, &r.Lat); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type MonumentSearchResult struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Lon  float64   `json:"lon"`
	Lat  float64   `json:"lat"`
}

type MonumentSearchFilter struct {
	Query    string
	Status   string
	AuthorID *uuid.UUID
	City     string
	DateFrom *time.Time
	DateTo   *time.Time
	Limit    int
	Offset   int
}

func (s *Search) SearchMonuments(ctx context.Context, f MonumentSearchFilter) ([]MonumentSearchResult, error) {
	f.Query = strings.TrimSpace(f.Query)
	if f.Query == "" {
		return []MonumentSearchResult{}, nil
	}
	if f.Limit <= 0 || f.Limit > 50 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	tsQuery := buildPrefixTSQuery(f.Query)
	args := []any{f.Query, f.Limit, f.Offset, tsQuery}
	where := []string{
		"(($4 <> '' and (m.search_tsv @@ to_tsquery('russian', $4) or p.search_tsv @@ to_tsquery('russian', $4))) or m.name ilike '%' || $1 || '%')",
	}

	if strings.TrimSpace(f.Status) == "" {
		where = append(where, "m.status='approved'")
	} else {
		args = append(args, strings.TrimSpace(f.Status))
		where = append(where, "m.status=$"+itoa(len(args)))
	}

	if f.AuthorID != nil {
		args = append(args, *f.AuthorID)
		where = append(where, "p.author_id=$"+itoa(len(args)))
	}
	if strings.TrimSpace(f.City) != "" {
		args = append(args, strings.TrimSpace(f.City))
		where = append(where, "u.city=$"+itoa(len(args)))
	}
	if f.DateFrom != nil {
		args = append(args, *f.DateFrom)
		where = append(where, "p.created_at >= $"+itoa(len(args)))
	}
	if f.DateTo != nil {
		args = append(args, *f.DateTo)
		where = append(where, "p.created_at <= $"+itoa(len(args)))
	}

	rows, err := s.db.Query(ctx, `
		select distinct m.id, m.name, ST_X(m.geom), ST_Y(m.geom),
		       case
		       	when $4 <> '' then ts_rank(m.search_tsv, to_tsquery('russian', $4))
		       	else 0
		       end as rank
		from monuments m
		left join posts p on p.monument_id=m.id and p.status='approved' and p.is_hidden=false
		left join users u on u.id=p.author_id
		where `+strings.Join(where, " and ")+`
		order by
			(case when m.name ilike $1 || '%' then 0 else 1 end),
			(case when position(lower($1) in lower(m.name)) > 0 then position(lower($1) in lower(m.name)) else 9999 end),
			rank desc,
			m.name asc
		limit $2 offset $3
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MonumentSearchResult
	for rows.Next() {
		var r MonumentSearchResult
		var rank float32
		if err := rows.Scan(&r.ID, &r.Name, &r.Lon, &r.Lat, &rank); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func buildPrefixTSQuery(q string) string {
	tokens := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(q)), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
	if len(tokens) == 0 {
		return ""
	}

	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		parts = append(parts, token+":*")
	}
	return strings.Join(parts, " & ")
}
