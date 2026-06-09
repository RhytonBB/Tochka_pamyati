package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MapTiles struct {
	db *pgxpool.Pool
}

func NewMapTiles(db *pgxpool.Pool) *MapTiles {
	return &MapTiles{db: db}
}

func (m *MapTiles) MonumentTile(ctx context.Context, z, x, y int) ([]byte, error) {
	var tile []byte
	err := m.db.QueryRow(ctx, `
		with bounds as (
			select ST_TileEnvelope($1, $2, $3) as geom
		),
		mvtgeom as (
			select
				m.id,
				m.name,
				ST_AsMVTGeom(ST_Transform(m.geom, 3857), b.geom, 4096, 64, true) as geom
			from monuments m, bounds b
			where m.status='approved'
			  and m.geom && ST_Transform(b.geom, 4326)
		)
		select ST_AsMVT(mvtgeom, 'monuments', 4096, 'geom') from mvtgeom;
	`, z, x, y).Scan(&tile)
	return tile, err
}
