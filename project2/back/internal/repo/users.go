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

var ErrNotFound = errors.New("not found")

type Users struct {
	db *pgxpool.Pool
}

func NewUsers(db *pgxpool.Pool) *Users {
	return &Users{db: db}
}

type User struct {
	ID                   uuid.UUID
	Username             string
	Email                string
	PasswordHash         string
	RoleID               uuid.UUID
	TrustScore           int
	City                 string
	Region               string
	NotificationSettings map[string]any
	IsActive             bool
	IsBlocked            bool
	CreatedAt            time.Time
	LastLogin            *time.Time
}

func (u *Users) Create(ctx context.Context, user User) (User, error) {
	settingsJSON, err := json.Marshal(user.NotificationSettings)
	if err != nil {
		return User{}, err
	}
	user.ID = ids.NewV7()

	row := u.db.QueryRow(ctx, `
		insert into users (id, username, email, password_hash, role_id, trust_score, city, region, notification_settings, is_active, is_blocked)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning created_at
	`,
		user.ID,
		user.Username,
		strings.ToLower(strings.TrimSpace(user.Email)),
		user.PasswordHash,
		user.RoleID,
		user.TrustScore,
		nullIfEmpty(user.City),
		nullIfEmpty(user.Region),
		settingsJSON,
		user.IsActive,
		user.IsBlocked,
	)

	var createdAt time.Time
	if err := row.Scan(&createdAt); err != nil {
		return User{}, err
	}

	user.CreatedAt = createdAt
	return user, nil
}

func (u *Users) GetByEmail(ctx context.Context, email string) (User, error) {
	var out User
	var city *string
	var region *string
	var settingsJSON []byte
	var lastLogin *time.Time

	err := u.db.QueryRow(ctx, `
		select id, username, email, password_hash, role_id, trust_score, city, region, notification_settings, is_active, is_blocked, created_at, last_login
		from users
		where email=$1
	`, strings.ToLower(strings.TrimSpace(email))).Scan(
		&out.ID,
		&out.Username,
		&out.Email,
		&out.PasswordHash,
		&out.RoleID,
		&out.TrustScore,
		&city,
		&region,
		&settingsJSON,
		&out.IsActive,
		&out.IsBlocked,
		&out.CreatedAt,
		&lastLogin,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if city != nil {
		out.City = *city
	}
	if region != nil {
		out.Region = *region
	}
	out.LastLogin = lastLogin
	_ = json.Unmarshal(settingsJSON, &out.NotificationSettings)
	return out, nil
}

func (u *Users) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	var out User
	var city *string
	var region *string
	var settingsJSON []byte
	var lastLogin *time.Time

	err := u.db.QueryRow(ctx, `
		select id, username, email, password_hash, role_id, trust_score, city, region, notification_settings, is_active, is_blocked, created_at, last_login
		from users
		where id=$1
	`, id).Scan(
		&out.ID,
		&out.Username,
		&out.Email,
		&out.PasswordHash,
		&out.RoleID,
		&out.TrustScore,
		&city,
		&region,
		&settingsJSON,
		&out.IsActive,
		&out.IsBlocked,
		&out.CreatedAt,
		&lastLogin,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if city != nil {
		out.City = *city
	}
	if region != nil {
		out.Region = *region
	}
	out.LastLogin = lastLogin
	_ = json.Unmarshal(settingsJSON, &out.NotificationSettings)
	return out, nil
}

func (u *Users) Activate(ctx context.Context, userID uuid.UUID) error {
	ct, err := u.db.Exec(ctx, `update users set is_active=true where id=$1`, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (u *Users) UpdateProfile(ctx context.Context, userID uuid.UUID, username, city, region string, settings map[string]any) (User, error) {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return User{}, err
	}

	_, err = u.db.Exec(ctx, `
		update users
		set username=$2, city=$3, region=$4, notification_settings=$5
		where id=$1
	`, userID, strings.TrimSpace(username), nullIfEmpty(strings.TrimSpace(city)), nullIfEmpty(strings.TrimSpace(region)), settingsJSON)
	if err != nil {
		return User{}, err
	}

	return u.GetByID(ctx, userID)
}

func (u *Users) SetLastLogin(ctx context.Context, userID uuid.UUID, t time.Time) error {
	_, err := u.db.Exec(ctx, `update users set last_login=$2 where id=$1`, userID, t)
	return err
}

func (u *Users) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	ct, err := u.db.Exec(ctx, `update users set password_hash=$2 where id=$1`, userID, passwordHash)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (u *Users) AdjustTrustScore(ctx context.Context, userID uuid.UUID, delta int) error {
	_, err := u.db.Exec(ctx, `update users set trust_score=trust_score+$2 where id=$1`, userID, delta)
	return err
}

func (u *Users) GetStats(ctx context.Context) (map[string]int64, error) {
	var total, active, blocked int64
	err := u.db.QueryRow(ctx, `
		select
			count(*),
			count(*) filter (where is_active = true),
			count(*) filter (where is_blocked = true)
		from users
	`).Scan(&total, &active, &blocked)
	if err != nil {
		return nil, err
	}

	return map[string]int64{
		"total":   total,
		"active":  active,
		"blocked": blocked,
	}, nil
}

func (u *Users) GetDynamics(ctx context.Context, days int) ([]map[string]any, error) {
	rows, err := u.db.Query(ctx, `
		select date_trunc('day', created_at) as day, count(*)
		from users
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

type ListUsersFilter struct {
	Username  string
	Email     string
	City      string
	Region    string
	RoleID    *uuid.UUID
	IsActive  *bool
	IsBlocked *bool
	Limit     int
	Offset    int
}

func (u *Users) List(ctx context.Context, f ListUsersFilter) ([]User, int64, error) {
	query := `
		select id, username, email, role_id, trust_score, city, region, notification_settings, is_active, is_blocked, created_at, last_login, count(*) over()
		from users
		where 1=1
	`
	args := []any{}
	if f.Username != "" {
		query += fmt.Sprintf(" and username ilike $%d", len(args)+1)
		args = append(args, "%"+f.Username+"%")
	}
	if f.Email != "" {
		query += fmt.Sprintf(" and email ilike $%d", len(args)+1)
		args = append(args, "%"+f.Email+"%")
	}
	if f.City != "" {
		query += fmt.Sprintf(" and city ilike $%d", len(args)+1)
		args = append(args, "%"+f.City+"%")
	}
	if f.Region != "" {
		query += fmt.Sprintf(" and region ilike $%d", len(args)+1)
		args = append(args, "%"+f.Region+"%")
	}
	if f.RoleID != nil {
		query += fmt.Sprintf(" and role_id = $%d", len(args)+1)
		args = append(args, *f.RoleID)
	}
	if f.IsActive != nil {
		query += fmt.Sprintf(" and is_active = $%d", len(args)+1)
		args = append(args, *f.IsActive)
	}
	if f.IsBlocked != nil {
		query += fmt.Sprintf(" and is_blocked = $%d", len(args)+1)
		args = append(args, *f.IsBlocked)
	}

	query += fmt.Sprintf(" order by created_at desc limit $%d offset $%d", len(args)+1, len(args)+2)
	args = append(args, f.Limit, f.Offset)

	rows, err := u.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []User
	var total int64
	for rows.Next() {
		var r User
		var city *string
		var region *string
		var settingsJSON []byte
		var lastLogin *time.Time
		if err := rows.Scan(&r.ID, &r.Username, &r.Email, &r.RoleID, &r.TrustScore, &city, &region, &settingsJSON, &r.IsActive, &r.IsBlocked, &r.CreatedAt, &lastLogin, &total); err != nil {
			return nil, 0, err
		}
		if city != nil {
			r.City = *city
		}
		if region != nil {
			r.Region = *region
		}
		r.LastLogin = lastLogin
		_ = json.Unmarshal(settingsJSON, &r.NotificationSettings)
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func (u *Users) UpdateRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := u.db.Exec(ctx, `update users set role_id=$2 where id=$1`, userID, roleID)
	return err
}

func (u *Users) SetBlocked(ctx context.Context, userID uuid.UUID, blocked bool) error {
	_, err := u.db.Exec(ctx, `update users set is_blocked=$2 where id=$1`, userID, blocked)
	return err
}

func (u *Users) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := u.db.Exec(ctx, `delete from users where id=$1`, userID)
	return err
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
