package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
)

type ExportService struct {
	monuments *repo.Monuments
	signals   *repo.Signals
}

func NewExportService(monuments *repo.Monuments, signals *repo.Signals) *ExportService {
	return &ExportService{monuments: monuments, signals: signals}
}

func (s *ExportService) ExportMonumentsCSV(ctx context.Context, w io.Writer) error {
	monuments, err := s.monuments.List(ctx, "approved", 1000000, 0)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{"ID", "Name", "Longitude", "Latitude", "Status", "CreatedAt"}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for _, m := range monuments {
		record := []string{
			m.ID.String(),
			m.Name,
			fmt.Sprintf("%f", m.Lon),
			fmt.Sprintf("%f", m.Lat),
			m.Status,
			m.CreatedAt.String(),
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func (s *ExportService) ExportMonumentsGeoJSON(ctx context.Context, w io.Writer) error {
	monuments, err := s.monuments.List(ctx, "approved", 1000000, 0)
	if err != nil {
		return err
	}

	type Feature struct {
		Type       string         `json:"type"`
		Geometry   map[string]any `json:"geometry"`
		Properties map[string]any `json:"properties"`
	}
	type FeatureCollection struct {
		Type     string    `json:"type"`
		Features []Feature `json:"features"`
	}

	fc := FeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0, len(monuments)),
	}

	for _, m := range monuments {
		f := Feature{
			Type: "Feature",
			Geometry: map[string]any{
				"type":        "Point",
				"coordinates": []float64{m.Lon, m.Lat},
			},
			Properties: map[string]any{
				"id":         m.ID,
				"name":       m.Name,
				"status":     m.Status,
				"created_at": m.CreatedAt,
			},
		}
		for k, v := range m.Properties {
			f.Properties[k] = v
		}
		fc.Features = append(fc.Features, f)
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(fc)
}

func (s *ExportService) ExportSignalsCSV(ctx context.Context, w io.Writer) error {
	signals, err := s.signals.List(ctx, repo.ListSignalsFilter{
		Limit: 1000000,
	})
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{"ID", "MonumentID", "Status", "Type", "Urgency", "CreatedAt"}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for _, s := range signals {
		mID := ""
		if s.MonumentID != nil {
			mID = s.MonumentID.String()
		}
		record := []string{
			s.ID.String(),
			mID,
			s.Status,
			s.SignalType,
			s.Urgency,
			s.CreatedAt.String(),
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}
	return nil
}
