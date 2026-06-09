package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Roles struct {
	db *pgxpool.Pool
}

func NewRoles(db *pgxpool.Pool) *Roles {
	return &Roles{db: db}
}

func (r *Roles) GetByName(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `select id from roles where name=$1`, name).Scan(&id)
	return id, err
}

func (r *Roles) GetNameByID(ctx context.Context, id uuid.UUID) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `select name from roles where id=$1`, id).Scan(&name)
	return name, err
}
