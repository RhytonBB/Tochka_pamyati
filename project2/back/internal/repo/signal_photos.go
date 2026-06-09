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

type SignalPhotos struct {
	db *pgxpool.Pool
}

func NewSignalPhotos(db *pgxpool.Pool) *SignalPhotos {
	return &SignalPhotos{db: db}
}

type SignalPhoto struct {
	ID             uuid.UUID      `json:"id"`
	SignalID       uuid.UUID      `json:"signal_id"`
	FilePath       string         `json:"file_path"`
	ThumbnailPath  string         `json:"thumbnail_path"`
	PreviewPath    string         `json:"preview_path"`
	ExifData       map[string]any `json:"exif_data"`
	RelevanceScore *float64       `json:"relevance_score,omitempty"`
	AIFlags        map[string]any `json:"ai_flags"`
	IsHidden       bool           `json:"is_hidden"`
	UploadedAt     time.Time      `json:"uploaded_at"`
}

func (p *SignalPhotos) Create(ctx context.Context, signalID uuid.UUID, filePath, thumbnailPath, previewPath string, exifData map[string]any, relevanceScore *float64, aiFlags map[string]any) (uuid.UUID, error) {
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
		insert into signal_photos (id, signal_id, file_path, thumbnail_path, preview_path, exif_data, relevance_score, ai_flags, is_hidden)
		values ($1,$2,$3,$4,$5,$6,$7,$8,false)
		returning id
	`, id, signalID, filePath, thumbnailPath, previewPath, exifJSON, relevanceScore, flagsJSON).Scan(&id)
	return id, err
}

func (p *SignalPhotos) ListBySignal(ctx context.Context, signalID uuid.UUID) ([]SignalPhoto, error) {
	rows, err := p.db.Query(ctx, `
		select id, signal_id, file_path, thumbnail_path, preview_path, exif_data, relevance_score, ai_flags, is_hidden, uploaded_at
		from signal_photos
		where signal_id=$1 and is_hidden=false
		order by uploaded_at desc
	`, signalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SignalPhoto
	for rows.Next() {
		var sp SignalPhoto
		var exifJSON []byte
		var flagsJSON []byte
		if err := rows.Scan(&sp.ID, &sp.SignalID, &sp.FilePath, &sp.ThumbnailPath, &sp.PreviewPath, &exifJSON, &sp.RelevanceScore, &flagsJSON, &sp.IsHidden, &sp.UploadedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(exifJSON, &sp.ExifData)
		_ = json.Unmarshal(flagsJSON, &sp.AIFlags)
		out = append(out, sp)
	}
	return out, rows.Err()
}

func (p *SignalPhotos) ListBySignalIncludeHidden(ctx context.Context, signalID uuid.UUID) ([]SignalPhoto, error) {
	rows, err := p.db.Query(ctx, `
		select id, signal_id, file_path, thumbnail_path, preview_path, exif_data, relevance_score, ai_flags, is_hidden, uploaded_at
		from signal_photos
		where signal_id=$1
		order by uploaded_at desc
	`, signalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SignalPhoto
	for rows.Next() {
		var sp SignalPhoto
		var exifJSON []byte
		var flagsJSON []byte
		if err := rows.Scan(&sp.ID, &sp.SignalID, &sp.FilePath, &sp.ThumbnailPath, &sp.PreviewPath, &exifJSON, &sp.RelevanceScore, &flagsJSON, &sp.IsHidden, &sp.UploadedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(exifJSON, &sp.ExifData)
		_ = json.Unmarshal(flagsJSON, &sp.AIFlags)
		out = append(out, sp)
	}
	return out, rows.Err()
}

func (p *SignalPhotos) ListPathsBySignal(ctx context.Context, signalID uuid.UUID) ([]SignalPhoto, error) {
	rows, err := p.db.Query(ctx, `
		select id, signal_id, file_path, thumbnail_path, preview_path, '{}'::jsonb, null, '{}'::jsonb, is_hidden, uploaded_at
		from signal_photos
		where signal_id=$1 and is_hidden=false
	`, signalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SignalPhoto
	for rows.Next() {
		var sp SignalPhoto
		var exifJSON []byte
		var flagsJSON []byte
		if err := rows.Scan(&sp.ID, &sp.SignalID, &sp.FilePath, &sp.ThumbnailPath, &sp.PreviewPath, &exifJSON, &sp.RelevanceScore, &flagsJSON, &sp.IsHidden, &sp.UploadedAt); err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

func (p *SignalPhotos) GetByID(ctx context.Context, id uuid.UUID) (SignalPhoto, error) {
	var out SignalPhoto
	var exifJSON []byte
	var flagsJSON []byte
	err := p.db.QueryRow(ctx, `
		select id, signal_id, file_path, thumbnail_path, preview_path, exif_data, relevance_score, ai_flags, is_hidden, uploaded_at
		from signal_photos
		where id=$1
	`, id).Scan(&out.ID, &out.SignalID, &out.FilePath, &out.ThumbnailPath, &out.PreviewPath, &exifJSON, &out.RelevanceScore, &flagsJSON, &out.IsHidden, &out.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SignalPhoto{}, ErrNotFound
	}
	if err != nil {
		return SignalPhoto{}, err
	}
	_ = json.Unmarshal(exifJSON, &out.ExifData)
	_ = json.Unmarshal(flagsJSON, &out.AIFlags)
	return out, nil
}

func (p *SignalPhotos) SetHidden(ctx context.Context, photoID uuid.UUID, isHidden bool) error {
	ct, err := p.db.Exec(ctx, `update signal_photos set is_hidden=$2 where id=$1`, photoID, isHidden)
	if err != nil {
		return err
	}
	rows := ct.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
