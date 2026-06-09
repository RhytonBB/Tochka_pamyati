package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
)

type Photos struct {
	db *pgxpool.Pool
}

func NewPhotos(db *pgxpool.Pool) *Photos {
	return &Photos{db: db}
}

type Photo struct {
	ID             uuid.UUID      `json:"id"`
	PostID         uuid.UUID      `json:"post_id"`
	FilePath       string         `json:"file_path"`
	ThumbnailPath  string         `json:"thumbnail_path"`
	PreviewPath    string         `json:"preview_path"`
	ExifData       map[string]any `json:"exif_data"`
	RelevanceScore *float64       `json:"relevance_score,omitempty"`
	AIFlags        map[string]any `json:"ai_flags"`
	IsHidden       bool           `json:"is_hidden"`
	UploadedAt     time.Time      `json:"uploaded_at"`
}

func (p *Photos) Create(ctx context.Context, postID uuid.UUID, filePath, thumbnailPath, previewPath string, exifData map[string]any, relevanceScore *float64, aiFlags map[string]any) (uuid.UUID, error) {
	exifJSON, err := json.Marshal(exifData)
	if err != nil {
		return uuid.Nil, err
	}
	flagsJSON, err := json.Marshal(aiFlags)
	if err != nil {
		return uuid.Nil, err
	}

	id := ids.NewV7()
	err = p.db.QueryRow(ctx, `
		insert into photos (id, post_id, file_path, thumbnail_path, preview_path, exif_data, relevance_score, ai_flags, is_hidden)
		values ($1,$2,$3,$4,$5,$6,$7,$8,false)
		returning id
	`, id, postID, filePath, thumbnailPath, previewPath, exifJSON, relevanceScore, flagsJSON).Scan(&id)
	return id, err
}

func (p *Photos) ListByMonument(ctx context.Context, monumentID uuid.UUID) ([]Photo, error) {
	rows, err := p.db.Query(ctx, `
		select ph.id, ph.post_id, ph.file_path, ph.thumbnail_path, ph.preview_path, ph.exif_data, ph.relevance_score, ph.ai_flags, ph.is_hidden, ph.uploaded_at
		from photos ph
		join posts po on po.id = ph.post_id
		where po.monument_id=$1
		  and po.status='approved'
		  and po.is_hidden=false
		  and coalesce(po.is_archived,false)=false
		  and ph.is_hidden=false
		order by ph.uploaded_at desc
	`, monumentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Photo
	for rows.Next() {
		var p1 Photo
		var exifJSON []byte
		var flagsJSON []byte
		if err := rows.Scan(&p1.ID, &p1.PostID, &p1.FilePath, &p1.ThumbnailPath, &p1.PreviewPath, &exifJSON, &p1.RelevanceScore, &flagsJSON, &p1.IsHidden, &p1.UploadedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(exifJSON, &p1.ExifData)
		_ = json.Unmarshal(flagsJSON, &p1.AIFlags)
		out = append(out, p1)
	}
	return out, rows.Err()
}

func (p *Photos) ListByPost(ctx context.Context, postID uuid.UUID) ([]Photo, error) {
	rows, err := p.db.Query(ctx, `
		select id, post_id, file_path, thumbnail_path, preview_path, exif_data, relevance_score, ai_flags, is_hidden, uploaded_at
		from photos
		where post_id=$1
		order by uploaded_at desc
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Photo
	for rows.Next() {
		var p1 Photo
		var exifJSON []byte
		var flagsJSON []byte
		if err := rows.Scan(&p1.ID, &p1.PostID, &p1.FilePath, &p1.ThumbnailPath, &p1.PreviewPath, &exifJSON, &p1.RelevanceScore, &flagsJSON, &p1.IsHidden, &p1.UploadedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(exifJSON, &p1.ExifData)
		_ = json.Unmarshal(flagsJSON, &p1.AIFlags)
		out = append(out, p1)
	}
	return out, rows.Err()
}

func (p *Photos) GetByID(ctx context.Context, id uuid.UUID) (Photo, error) {
	var out Photo
	var exifJSON []byte
	var flagsJSON []byte
	err := p.db.QueryRow(ctx, `
		select id, post_id, file_path, thumbnail_path, preview_path, exif_data, relevance_score, ai_flags, is_hidden, uploaded_at
		from photos
		where id=$1
	`, id).Scan(&out.ID, &out.PostID, &out.FilePath, &out.ThumbnailPath, &out.PreviewPath, &exifJSON, &out.RelevanceScore, &flagsJSON, &out.IsHidden, &out.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Photo{}, ErrNotFound
	}
	if err != nil {
		return Photo{}, err
	}
	_ = json.Unmarshal(exifJSON, &out.ExifData)
	_ = json.Unmarshal(flagsJSON, &out.AIFlags)
	return out, nil
}

func (p *Photos) SetHidden(ctx context.Context, photoID uuid.UUID, isHidden bool) error {
	ct, err := p.db.Exec(ctx, `update photos set is_hidden=$2 where id=$1`, photoID, isHidden)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Photos) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := p.db.Exec(ctx, `delete from photos where id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
