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

type Reports struct {
	db *pgxpool.Pool
}

func NewReports(db *pgxpool.Pool) *Reports {
	return &Reports{db: db}
}

type ReportCase struct {
	ID                     uuid.UUID      `json:"id"`
	EntityType             string         `json:"entity_type"`
	EntityID               uuid.UUID      `json:"entity_id"`
	ReasonCode             string         `json:"reason_code"`
	Category               string         `json:"category"`
	Severity               string         `json:"severity"`
	ReportsCount           int            `json:"reports_count"`
	DistinctReportersCount int            `json:"distinct_reporters_count"`
	Status                 string         `json:"status"`
	EntitySnapshot         map[string]any `json:"entity_snapshot"`
	SuggestedFix           map[string]any `json:"suggested_fix"`
	PriorityScore          int            `json:"priority_score"`
	LastReportedAt         time.Time      `json:"last_reported_at"`
	ResolvedAt             *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy             *uuid.UUID     `json:"resolved_by,omitempty"`
	ResolutionAction       *string        `json:"resolution_action,omitempty"`
	ModeratorComment       *string        `json:"moderator_comment,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
	ActionProfile          map[string]any `json:"action_profile,omitempty"`
	AvailableActions       []string       `json:"available_actions,omitempty"`
	CanEdit                bool           `json:"can_edit,omitempty"`
	CanHidePart            bool           `json:"can_hide_part,omitempty"`
	CanHideEntity          bool           `json:"can_hide_entity,omitempty"`
	CanApplyFix            bool           `json:"can_apply_fix,omitempty"`
	Votes                  []ReportVote   `json:"votes,omitempty"`
}

type ReportVote struct {
	ID             uuid.UUID      `json:"id"`
	CaseID         uuid.UUID      `json:"case_id"`
	EntityType     string         `json:"entity_type"`
	EntityID       uuid.UUID      `json:"entity_id"`
	ReporterID     uuid.UUID      `json:"reporter_id"`
	ReporterName   string         `json:"reporter_name"`
	ReasonCode     string         `json:"reason_code"`
	Comment        *string        `json:"comment,omitempty"`
	Status         string         `json:"status"`
	EntitySnapshot map[string]any `json:"entity_snapshot"`
	SuggestedFix   map[string]any `json:"suggested_fix"`
	CreatedAt      time.Time      `json:"created_at"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy     *uuid.UUID     `json:"resolved_by,omitempty"`
}

type CreateReportVoteParams struct {
	EntityType     string
	EntityID       uuid.UUID
	ReporterID     uuid.UUID
	ReasonCode     string
	Category       string
	Severity       string
	Comment        *string
	EntitySnapshot map[string]any
	SuggestedFix   map[string]any
	PriorityScore  int
}

type ListReportCasesFilter struct {
	Status     string
	EntityType string
	ReasonCode string
	Category   string
	Limit      int
	Offset     int
}

type ListReportVotesFilter struct {
	ReporterID uuid.UUID
	Status     string
	Limit      int
	Offset     int
}

func marshalMap(v map[string]any) ([]byte, error) {
	if v == nil {
		v = map[string]any{}
	}
	return json.Marshal(v)
}

func parseMap(raw []byte) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func normalizeLimitOffset(limit, offset int) (int, int) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (r *Reports) CreateVote(ctx context.Context, in CreateReportVoteParams) (ReportCase, ReportVote, error) {
	snapshotJSON, err := marshalMap(in.EntitySnapshot)
	if err != nil {
		return ReportCase{}, ReportVote{}, err
	}
	suggestedFixJSON, err := marshalMap(in.SuggestedFix)
	if err != nil {
		return ReportCase{}, ReportVote{}, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ReportCase{}, ReportVote{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var caseID uuid.UUID
	err = tx.QueryRow(ctx, `
		select id
		from report_cases
		where entity_type=$1 and entity_id=$2 and reason_code=$3 and status in ('pending','auto_hidden')
		order by created_at desc
		limit 1
	`, in.EntityType, in.EntityID, in.ReasonCode).Scan(&caseID)
	if errors.Is(err, pgx.ErrNoRows) {
		caseID = ids.NewV7()
		err = tx.QueryRow(ctx, `
			insert into report_cases (
				id, entity_type, entity_id, reason_code, category, severity, status,
				entity_snapshot, suggested_fix, priority_score, reports_count,
				distinct_reporters_count, last_reported_at
			)
			values ($1,$2,$3,$4,$5,$6,'pending',$7,$8,$9,0,0,now())
			returning id
		`, caseID, in.EntityType, in.EntityID, in.ReasonCode, in.Category, in.Severity, snapshotJSON, suggestedFixJSON, in.PriorityScore).Scan(&caseID)
		if err != nil {
			return ReportCase{}, ReportVote{}, err
		}
	} else if err != nil {
		return ReportCase{}, ReportVote{}, err
	} else {
		if _, err := tx.Exec(ctx, `
			update report_cases
			set
				entity_snapshot=$2,
				suggested_fix = case
					when suggested_fix = '{}'::jsonb and $3 <> '{}'::jsonb then $3
					else suggested_fix
				end,
				priority_score=greatest(priority_score, $4),
				last_reported_at=now()
			where id=$1
		`, caseID, snapshotJSON, suggestedFixJSON, in.PriorityScore); err != nil {
			return ReportCase{}, ReportVote{}, err
		}
	}

	voteID := ids.NewV7()
	err = tx.QueryRow(ctx, `
		insert into report_votes (
			id, case_id, entity_type, entity_id, reporter_id, reason_code,
			comment, status, entity_snapshot, suggested_fix
		)
		values ($1,$2,$3,$4,$5,$6,$7,'pending',$8,$9)
		returning id
	`, voteID, caseID, in.EntityType, in.EntityID, in.ReporterID, in.ReasonCode, nullIfEmptyStrPtr(in.Comment), snapshotJSON, suggestedFixJSON).Scan(&voteID)
	if err != nil {
		return ReportCase{}, ReportVote{}, err
	}

	if _, err := tx.Exec(ctx, `
		update report_cases rc
		set
			reports_count = stats.reports_count,
			distinct_reporters_count = stats.distinct_reporters_count,
			last_reported_at = now()
		from (
			select
				case_id,
				count(*)::int as reports_count,
				count(distinct reporter_id)::int as distinct_reporters_count
			from report_votes
			where case_id=$1 and status='pending'
			group by case_id
		) stats
		where rc.id = stats.case_id
	`, caseID); err != nil {
		return ReportCase{}, ReportVote{}, err
	}

	reportCase, err := r.getCaseByIDQuerier(ctx, tx, caseID)
	if err != nil {
		return ReportCase{}, ReportVote{}, err
	}
	vote, err := r.getVoteByIDQuerier(ctx, tx, voteID)
	if err != nil {
		return ReportCase{}, ReportVote{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ReportCase{}, ReportVote{}, err
	}
	return reportCase, vote, nil
}

func (r *Reports) GetOpenCase(ctx context.Context, entityType string, entityID uuid.UUID, reasonCode string) (ReportCase, error) {
	return r.getCaseQuery(ctx, r.db, `
		select id, entity_type, entity_id, reason_code, category, severity, reports_count,
		       distinct_reporters_count, status, entity_snapshot, suggested_fix, priority_score,
		       last_reported_at, resolved_at, resolved_by, resolution_action, moderator_comment, created_at
		from report_cases
		where entity_type=$1 and entity_id=$2 and reason_code=$3 and status in ('pending','auto_hidden')
		order by created_at desc
		limit 1
	`, strings.TrimSpace(entityType), entityID, strings.TrimSpace(reasonCode))
}

func (r *Reports) HasActiveVote(ctx context.Context, reporterID uuid.UUID, entityType string, entityID uuid.UUID, reasonCode string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		select exists(
			select 1
			from report_votes
			where reporter_id=$1 and entity_type=$2 and entity_id=$3 and reason_code=$4 and status='pending'
		)
	`, reporterID, strings.TrimSpace(entityType), entityID, strings.TrimSpace(reasonCode)).Scan(&exists)
	return exists, err
}

func (r *Reports) GetCaseByID(ctx context.Context, id uuid.UUID) (ReportCase, error) {
	out, err := r.getCaseByIDQuerier(ctx, r.db, id)
	if err != nil {
		return ReportCase{}, err
	}
	votes, err := r.listVotesByCaseID(ctx, r.db, id)
	if err != nil {
		return ReportCase{}, err
	}
	out.Votes = votes
	return out, nil
}

func (r *Reports) getCaseByIDQuerier(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id uuid.UUID) (ReportCase, error) {
	return r.getCaseQuery(ctx, q, `
		select id, entity_type, entity_id, reason_code, category, severity, reports_count,
		       distinct_reporters_count, status, entity_snapshot, suggested_fix, priority_score,
		       last_reported_at, resolved_at, resolved_by, resolution_action, moderator_comment, created_at
		from report_cases
		where id=$1
	`, id)
}

func (r *Reports) getCaseQuery(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, query string, args ...any) (ReportCase, error) {
	var out ReportCase
	var snapshotJSON []byte
	var suggestedFixJSON []byte
	var resolvedAt *time.Time
	var resolvedBy *uuid.UUID
	var resolutionAction *string
	var moderatorComment *string
	err := q.QueryRow(ctx, query, args...).Scan(
		&out.ID,
		&out.EntityType,
		&out.EntityID,
		&out.ReasonCode,
		&out.Category,
		&out.Severity,
		&out.ReportsCount,
		&out.DistinctReportersCount,
		&out.Status,
		&snapshotJSON,
		&suggestedFixJSON,
		&out.PriorityScore,
		&out.LastReportedAt,
		&resolvedAt,
		&resolvedBy,
		&resolutionAction,
		&moderatorComment,
		&out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportCase{}, ErrNotFound
	}
	if err != nil {
		return ReportCase{}, err
	}
	out.EntitySnapshot = parseMap(snapshotJSON)
	out.SuggestedFix = parseMap(suggestedFixJSON)
	out.ResolvedAt = resolvedAt
	out.ResolvedBy = resolvedBy
	out.ResolutionAction = resolutionAction
	out.ModeratorComment = moderatorComment
	return out, nil
}

func (r *Reports) ListCases(ctx context.Context, f ListReportCasesFilter) ([]ReportCase, error) {
	f.Limit, f.Offset = normalizeLimitOffset(f.Limit, f.Offset)
	args := []any{f.Limit, f.Offset}
	where := []string{"1=1"}

	if s := strings.TrimSpace(f.Status); s != "" {
		args = append(args, s)
		where = append(where, "status=$"+strconv.Itoa(len(args)))
	}
	if s := strings.TrimSpace(f.EntityType); s != "" {
		args = append(args, s)
		where = append(where, "entity_type=$"+strconv.Itoa(len(args)))
	}
	if s := strings.TrimSpace(f.ReasonCode); s != "" {
		args = append(args, s)
		where = append(where, "reason_code=$"+strconv.Itoa(len(args)))
	}
	if s := strings.TrimSpace(f.Category); s != "" {
		args = append(args, s)
		where = append(where, "category=$"+strconv.Itoa(len(args)))
	}

	rows, err := r.db.Query(ctx, `
		select id, entity_type, entity_id, reason_code, category, severity, reports_count,
		       distinct_reporters_count, status, entity_snapshot, suggested_fix, priority_score,
		       last_reported_at, resolved_at, resolved_by, resolution_action, moderator_comment, created_at
		from report_cases
		where `+strings.Join(where, " and ")+`
		order by priority_score desc, last_reported_at desc
		limit $1 offset $2
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReportCase
	for rows.Next() {
		var item ReportCase
		var snapshotJSON []byte
		var suggestedFixJSON []byte
		var resolvedAt *time.Time
		var resolvedBy *uuid.UUID
		var resolutionAction *string
		var moderatorComment *string
		if err := rows.Scan(
			&item.ID,
			&item.EntityType,
			&item.EntityID,
			&item.ReasonCode,
			&item.Category,
			&item.Severity,
			&item.ReportsCount,
			&item.DistinctReportersCount,
			&item.Status,
			&snapshotJSON,
			&suggestedFixJSON,
			&item.PriorityScore,
			&item.LastReportedAt,
			&resolvedAt,
			&resolvedBy,
			&resolutionAction,
			&moderatorComment,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.EntitySnapshot = parseMap(snapshotJSON)
		item.SuggestedFix = parseMap(suggestedFixJSON)
		item.ResolvedAt = resolvedAt
		item.ResolvedBy = resolvedBy
		item.ResolutionAction = resolutionAction
		item.ModeratorComment = moderatorComment
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Reports) ListVotesByReporter(ctx context.Context, f ListReportVotesFilter) ([]ReportVote, error) {
	f.Limit, f.Offset = normalizeLimitOffset(f.Limit, f.Offset)
	args := []any{f.ReporterID, f.Limit, f.Offset}
	where := []string{"reporter_id=$1"}
	if s := strings.TrimSpace(f.Status); s != "" {
		args = append(args, s)
		where = append(where, "status=$"+strconv.Itoa(len(args)))
	}

	rows, err := r.db.Query(ctx, `
		select rv.id, rv.case_id, rv.entity_type, rv.entity_id, rv.reporter_id, rv.reason_code, rv.comment, rv.status,
		       rv.entity_snapshot, rv.suggested_fix, rv.created_at, rv.resolved_at, rv.resolved_by,
		       coalesce(u.username, '')
		from report_votes rv
		left join users u on u.id = rv.reporter_id
		where `+strings.Join(where, " and ")+`
		order by created_at desc
		limit $2 offset $3
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReportVote
	for rows.Next() {
		item, err := scanVote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanVote(scanner interface {
	Scan(dest ...any) error
}) (ReportVote, error) {
	var out ReportVote
	var comment *string
	var snapshotJSON []byte
	var suggestedFixJSON []byte
	var resolvedAt *time.Time
	var resolvedBy *uuid.UUID
	var reporterName string
	if err := scanner.Scan(
		&out.ID,
		&out.CaseID,
		&out.EntityType,
		&out.EntityID,
		&out.ReporterID,
		&out.ReasonCode,
		&comment,
		&out.Status,
		&snapshotJSON,
		&suggestedFixJSON,
		&out.CreatedAt,
		&resolvedAt,
		&resolvedBy,
		&reporterName,
	); err != nil {
		return ReportVote{}, err
	}
	out.Comment = comment
	out.EntitySnapshot = parseMap(snapshotJSON)
	out.SuggestedFix = parseMap(suggestedFixJSON)
	out.ResolvedAt = resolvedAt
	out.ResolvedBy = resolvedBy
	out.ReporterName = reporterName
	return out, nil
}

func (r *Reports) listVotesByCaseID(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, caseID uuid.UUID) ([]ReportVote, error) {
	rows, err := q.Query(ctx, `
		select rv.id, rv.case_id, rv.entity_type, rv.entity_id, rv.reporter_id, rv.reason_code, rv.comment, rv.status,
		       rv.entity_snapshot, rv.suggested_fix, rv.created_at, rv.resolved_at, rv.resolved_by,
		       coalesce(u.username, '')
		from report_votes rv
		left join users u on u.id = rv.reporter_id
		where rv.case_id=$1
		order by created_at desc
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReportVote
	for rows.Next() {
		item, err := scanVote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Reports) getVoteByIDQuerier(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id uuid.UUID) (ReportVote, error) {
	row := q.QueryRow(ctx, `
		select rv.id, rv.case_id, rv.entity_type, rv.entity_id, rv.reporter_id, rv.reason_code, rv.comment, rv.status,
		       rv.entity_snapshot, rv.suggested_fix, rv.created_at, rv.resolved_at, rv.resolved_by,
		       coalesce(u.username, '')
		from report_votes rv
		left join users u on u.id = rv.reporter_id
		where rv.id=$1
	`, id)
	return scanVote(row)
}

func (r *Reports) ResolveCase(ctx context.Context, caseID, actorID uuid.UUID, status, resolutionAction string, moderatorComment *string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()
	ct, err := tx.Exec(ctx, `
		update report_cases
		set status=$2, resolved_at=$3, resolved_by=$4, resolution_action=$5, moderator_comment=$6
		where id=$1
	`, caseID, status, now, actorID, nullIfEmptyStrPtr(&resolutionAction), nullIfEmptyStrPtr(moderatorComment))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	voteStatus := "resolved"
	if status == "rejected" {
		voteStatus = "rejected"
	}
	if _, err := tx.Exec(ctx, `
		update report_votes
		set status=$2, resolved_at=$3, resolved_by=$4
		where case_id=$1 and status='pending'
	`, caseID, voteStatus, now, actorID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Reports) SetCaseStatus(ctx context.Context, caseID uuid.UUID, status string) error {
	ct, err := r.db.Exec(ctx, `update report_cases set status=$2 where id=$1`, caseID, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Reports) CountCaseStats(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.Query(ctx, `
		select status, count(*)
		from report_cases
		group by status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, rows.Err()
}
