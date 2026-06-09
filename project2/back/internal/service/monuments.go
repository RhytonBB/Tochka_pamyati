package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"

	_ "golang.org/x/image/webp"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
	"github.com/tochka-pamyati/tochka-pamyati/internal/moderation"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/storage"
)

type MonumentsService struct {
	monuments     *repo.Monuments
	posts         *repo.Posts
	photos        *repo.Photos
	audit         *repo.AuditLog
	signals       *repo.Signals
	notifications *repo.Notifications
	adminLogs     *repo.AdminEventLogs
	sanctions     *SanctionsService
	trust         *TrustService
	geo           *GeographyService

	textChecker  moderation.TextChecker
	imageChecker moderation.ImageChecker
	uploader     storage.Uploader
}

type MonumentsDeps struct {
	Monuments     *repo.Monuments
	Posts         *repo.Posts
	Photos        *repo.Photos
	Audit         *repo.AuditLog
	Signals       *repo.Signals
	Notifications *repo.Notifications
	AdminLogs     *repo.AdminEventLogs
	Sanctions     *SanctionsService
	Trust         *TrustService
	Geography     *GeographyService

	TextChecker  moderation.TextChecker
	ImageChecker moderation.ImageChecker
	Uploader     storage.Uploader
}

func NewMonumentsService(deps MonumentsDeps) *MonumentsService {
	return &MonumentsService{
		monuments:     deps.Monuments,
		posts:         deps.Posts,
		photos:        deps.Photos,
		audit:         deps.Audit,
		signals:       deps.Signals,
		notifications: deps.Notifications,
		adminLogs:     deps.AdminLogs,
		sanctions:     deps.Sanctions,
		trust:         deps.Trust,
		geo:           deps.Geography,
		textChecker:   deps.TextChecker,
		imageChecker:  deps.ImageChecker,
		uploader:      deps.Uploader,
	}
}

type CreateMonumentInput struct {
	AuthorID    uuid.UUID
	Name        string
	Lon         float64
	Lat         float64
	Properties  map[string]any
	Description string
	Photos      []*multipart.FileHeader
	ContentAck  bool
	CreatePost  bool
}

type CreateMonumentOutput struct {
	MonumentID uuid.UUID `json:"monument_id"`
	PostID     uuid.UUID `json:"post_id"`
	HighRisk   bool      `json:"high_risk"`
}

func (s *MonumentsService) ValidateCreateMonument(ctx context.Context, in CreateMonumentInput) (ContentValidationResult, []repo.NearbyMonument, error) {
	in.Description = normalizeUserText(in.Description)
	fields := map[string]string{}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		fields["name"] = "required"
	}
	if in.Lon < -180 || in.Lon > 180 {
		fields["lon"] = "invalid"
	}
	if in.Lat < -90 || in.Lat > 90 {
		fields["lat"] = "invalid"
	}
	if len(in.Photos) > 10 {
		fields["photos"] = "max_10"
	}
	if in.CreatePost && len(in.Photos) == 0 && strings.TrimSpace(in.Description) == "" {
		fields["description"] = "required_or_photos"
	}
	if len(fields) > 0 {
		return ContentValidationResult{
			RequiresAck: false,
			Reasons:     []string{"invalid_input"},
			Fields:      fields,
			HighRisk:    false,
		}, nil, nil
	}

	dup, err := s.monuments.FindDuplicates(ctx, in.Name, in.Lon, in.Lat)
	if err != nil {
		return ContentValidationResult{}, nil, err
	}

	var reasons []string
	fields = map[string]string{}
	if len(dup) > 0 {
		fields["name"] = "possible_duplicate"
		reasons = append(reasons, "possible_duplicate")
	}

	var textScore *float64
	var textFlags []string
	var textUnavailable bool
	var imgUnavailableAny bool
	if in.CreatePost {
		textScore, textFlags, textUnavailable = s.checkText(ctx, in.Description)
		if len(textFlags) > 0 {
			fields["description"] = "flagged"
			reasons = append(reasons, "text_flagged")
		}
		if textUnavailable {
			reasons = append(reasons, "text_filter_unavailable")
		}

		_, imageIssues, imageUnavailable, err := s.checkImages(ctx, in.Photos)
		if err != nil {
			return ContentValidationResult{}, nil, err
		}
		for k, v := range imageIssues {
			fields[k] = v
			reasons = append(reasons, "image_flagged")
		}
		imgUnavailableAny = imageUnavailable
		if imgUnavailableAny {
			reasons = append(reasons, "image_filter_unavailable")
		}
		if textScore != nil && *textScore >= 0.7 && !containsReason(reasons, "text_flagged") {
			reasons = append(reasons, "text_flagged")
		}
	}

	highRisk := len(fields) > 0 || textUnavailable || imgUnavailableAny
	return ContentValidationResult{
		RequiresAck: len(fields) > 0,
		Reasons:     uniqueReasons(reasons),
		Fields:      fields,
		HighRisk:    highRisk,
	}, dup, nil
}

func (s *MonumentsService) CreateMonumentWithFirstPost(ctx context.Context, in CreateMonumentInput) (CreateMonumentOutput, error) {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeContentCreate); err != nil {
			return CreateMonumentOutput{}, err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentCreate(ctx, in.AuthorID); err != nil {
			return CreateMonumentOutput{}, err
		}
	}
	in.Description = normalizeUserText(in.Description)
	validation, dup, err := s.ValidateCreateMonument(ctx, in)
	if err != nil {
		return CreateMonumentOutput{}, err
	}
	if containsReason(validation.Reasons, "invalid_input") {
		return CreateMonumentOutput{}, apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: validation.Fields}
	}
	if validation.RequiresAck && !in.ContentAck {
		return CreateMonumentOutput{}, apierr.Error{
			Code:    "content_requires_ack",
			Message: "content requires confirmation",
			Fields:  validation.Fields,
			Data: map[string]any{
				"requires_ack": true,
				"reasons":      validation.Reasons,
			},
		}
	}

	var textScore *float64
	var textFlags []string
	var textUnavailable bool
	imageResults := []map[string]any{}
	var imgUnavailableAny bool
	if in.CreatePost {
		textScore, textFlags, textUnavailable = s.checkText(ctx, in.Description)
		imageResults, _, imgUnavailableAny, err = s.checkImages(ctx, in.Photos)
		if err != nil {
			return CreateMonumentOutput{}, err
		}
	}

	aiFlags := map[string]any{
		"reasons":           validation.Reasons,
		"text_toxic_score":  textScore,
		"text_categories":   textFlags,
		"image_results":     imageResults,
		"duplicates":        dup,
		"content_ack":       in.ContentAck,
		"filters_available": map[string]any{"text": !textUnavailable, "image": !imgUnavailableAny},
	}
	if s.trust != nil {
		if err := s.trust.ApplySubmissionTrustFlags(ctx, in.AuthorID, &validation.HighRisk, &validation.Reasons, aiFlags); err != nil {
			return CreateMonumentOutput{}, err
		}
	}

	// Добавляем описание в свойства памятника для отображения в списках и модерации
	if in.Properties == nil {
		in.Properties = make(map[string]any)
	}

	region := s.geo.GetRegionByCoords(in.Lon, in.Lat)

	monumentID, err := s.monuments.Create(ctx, in.Name, in.Lon, in.Lat, region, in.Properties, "pending", &in.AuthorID, validation.HighRisk, aiFlags)
	if err != nil {
		return CreateMonumentOutput{}, err
	}

	if !in.CreatePost {
		monID := monumentID
		s.logEvent(ctx, &in.AuthorID, &in.AuthorID, "monument", &monID, "создание_точки", "Пользователь создал новую точку без первого поста", map[string]any{
			"name":      in.Name,
			"region":    region,
			"high_risk": validation.HighRisk,
		})
		return CreateMonumentOutput{MonumentID: monumentID, HighRisk: validation.HighRisk}, nil
	}

	var toxicScorePtr *float64
	if textScore != nil {
		toxicScorePtr = textScore
	}

	postID, err := s.posts.Create(ctx, monumentID, in.AuthorID, &in.Description, "pending", nil, toxicScorePtr, validation.HighRisk, aiFlags)
	if err != nil {
		return CreateMonumentOutput{}, err
	}

	if err := s.savePhotos(ctx, postID, in.Photos, imageResults); err != nil {
		return CreateMonumentOutput{}, err
	}

	// Post-check: GPS mismatch
	photos, _ := s.photos.ListByPost(ctx, postID)
	for _, p := range photos {
		exifLat, okLat := p.ExifData["gps_lat"].(float64)
		exifLon, okLon := p.ExifData["gps_lon"].(float64)
		if okLat && okLon {
			dist := haversine(exifLat, exifLon, in.Lat, in.Lon)
			if dist > 1.0 { // 1 km
				validation.Reasons = append(validation.Reasons, fmt.Sprintf("gps_mismatch: photo %s is %.2f km away", p.ID, dist))
				validation.HighRisk = true
			}
		}
	}

	if validation.HighRisk {
		aiFlags["reasons"] = uniqueReasons(validation.Reasons)
		_ = s.monuments.UpdateAIFlags(ctx, monumentID, validation.HighRisk, aiFlags)
		_ = s.posts.UpdateAIFlags(ctx, postID, validation.HighRisk, aiFlags)
	}

	monID := monumentID
	postIDCopy := postID
	s.logEvent(ctx, &in.AuthorID, &in.AuthorID, "monument", &monID, "создание_точки_с_первым_постом", "Пользователь создал точку и сразу добавил первый пост", map[string]any{
		"name":      in.Name,
		"region":    region,
		"post_id":   postID.String(),
		"high_risk": validation.HighRisk,
	})
	s.logEvent(ctx, &in.AuthorID, &in.AuthorID, "post", &postIDCopy, "создание_поста", "Пользователь создал первый пост к новой точке", map[string]any{
		"monument_id": monumentID.String(),
		"high_risk":   validation.HighRisk,
	})

	return CreateMonumentOutput{MonumentID: monumentID, PostID: postID, HighRisk: validation.HighRisk}, nil
}

type AddPostInput struct {
	AuthorID    uuid.UUID
	MonumentID  uuid.UUID
	Description string
	Photos      []*multipart.FileHeader
	ContentAck  bool
}

type AddPostOutput struct {
	PostID   uuid.UUID `json:"post_id"`
	HighRisk bool      `json:"high_risk"`
}

func (s *MonumentsService) ValidateAddPost(ctx context.Context, in AddPostInput) (ContentValidationResult, error) {
	in.Description = normalizeUserText(in.Description)
	if _, err := s.monuments.GetByID(ctx, in.MonumentID); err != nil {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "monument not found", Fields: map[string]string{"monument_id": "not_found"}}
	}

	if len(in.Photos) == 0 && in.Description == "" {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "empty post", Fields: map[string]string{"description": "required_or_photos"}}
	}
	if len(in.Photos) > 10 {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "too many photos", Fields: map[string]string{"photos": "max_10"}}
	}

	var reasons []string
	fields := map[string]string{}

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	if len(textFlags) > 0 {
		fields["description"] = "flagged"
		reasons = append(reasons, "text_flagged")
	}
	if textUnavailable {
		reasons = append(reasons, "text_filter_unavailable")
	}

	_, imageIssues, imgUnavailableAny, err := s.checkImages(ctx, in.Photos)
	if err != nil {
		return ContentValidationResult{}, err
	}
	for k, v := range imageIssues {
		fields[k] = v
		reasons = append(reasons, "image_flagged")
	}
	if imgUnavailableAny {
		reasons = append(reasons, "image_filter_unavailable")
	}
	if textScore != nil && *textScore >= 0.7 && !containsReason(reasons, "text_flagged") {
		reasons = append(reasons, "text_flagged")
	}

	highRisk := len(fields) > 0 || textUnavailable || imgUnavailableAny
	return ContentValidationResult{
		RequiresAck: len(fields) > 0,
		Reasons:     uniqueReasons(reasons),
		Fields:      fields,
		HighRisk:    highRisk,
	}, nil
}

func (s *MonumentsService) AddPost(ctx context.Context, in AddPostInput) (AddPostOutput, error) {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeContentCreate); err != nil {
			return AddPostOutput{}, err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentCreate(ctx, in.AuthorID); err != nil {
			return AddPostOutput{}, err
		}
	}
	in.Description = normalizeUserText(in.Description)
	validation, err := s.ValidateAddPost(ctx, in)
	if err != nil {
		return AddPostOutput{}, err
	}
	if validation.RequiresAck && !in.ContentAck {
		return AddPostOutput{}, apierr.Error{
			Code:    "content_requires_ack",
			Message: "content requires confirmation",
			Fields:  validation.Fields,
			Data: map[string]any{
				"requires_ack": true,
				"reasons":      validation.Reasons,
			},
		}
	}

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	imageResults, _, imgUnavailableAny, err := s.checkImages(ctx, in.Photos)
	if err != nil {
		return AddPostOutput{}, err
	}

	aiFlags := map[string]any{
		"reasons":           validation.Reasons,
		"text_toxic_score":  textScore,
		"text_categories":   textFlags,
		"image_results":     imageResults,
		"content_ack":       in.ContentAck,
		"filters_available": map[string]any{"text": !textUnavailable, "image": !imgUnavailableAny},
	}
	if s.trust != nil {
		if err := s.trust.ApplySubmissionTrustFlags(ctx, in.AuthorID, &validation.HighRisk, &validation.Reasons, aiFlags); err != nil {
			return AddPostOutput{}, err
		}
	}

	var toxicScorePtr *float64
	if textScore != nil {
		toxicScorePtr = textScore
	}
	postID, err := s.posts.Create(ctx, in.MonumentID, in.AuthorID, &in.Description, "pending", nil, toxicScorePtr, validation.HighRisk, aiFlags)
	if err != nil {
		return AddPostOutput{}, err
	}

	if err := s.savePhotos(ctx, postID, in.Photos, imageResults); err != nil {
		return AddPostOutput{}, err
	}

	// Post-check: GPS mismatch
	mon, _ := s.monuments.GetByID(ctx, in.MonumentID)
	photos, _ := s.photos.ListByPost(ctx, postID)
	for _, p := range photos {
		exifLat, okLat := p.ExifData["gps_lat"].(float64)
		exifLon, okLon := p.ExifData["gps_lon"].(float64)
		if okLat && okLon {
			dist := haversine(exifLat, exifLon, mon.Lat, mon.Lon)
			if dist > 1.0 { // 1 km
				validation.Reasons = append(validation.Reasons, fmt.Sprintf("gps_mismatch: photo %s is %.2f km away", p.ID, dist))
				validation.HighRisk = true
			}
		}
	}

	if validation.HighRisk {
		aiFlags["reasons"] = uniqueReasons(validation.Reasons)
		_ = s.posts.UpdateAIFlags(ctx, postID, validation.HighRisk, aiFlags)
	}

	postIDCopy := postID
	s.logEvent(ctx, &in.AuthorID, &in.AuthorID, "post", &postIDCopy, "создание_поста", "Пользователь добавил новый пост к существующей точке", map[string]any{
		"monument_id": in.MonumentID.String(),
		"high_risk":   validation.HighRisk,
	})

	return AddPostOutput{PostID: postID, HighRisk: validation.HighRisk}, nil
}

type UpdatePostInput struct {
	AuthorID    uuid.UUID
	PostID      uuid.UUID
	Description string
	ContentAck  bool
}

type UpdatePostSubmissionInput struct {
	AuthorID       uuid.UUID
	PostID         uuid.UUID
	Description    string
	Photos         []*multipart.FileHeader
	RemovePhotoIDs []uuid.UUID
	ContentAck     bool
}

type UpdateMonumentSubmissionInput struct {
	AuthorID       uuid.UUID
	MonumentID     uuid.UUID
	Name           string
	Lon            float64
	Lat            float64
	Description    string
	Photos         []*multipart.FileHeader
	RemovePhotoIDs []uuid.UUID
	ContentAck     bool
}

func (s *MonumentsService) UpdatePostText(ctx context.Context, in UpdatePostInput) error {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeContentEdit); err != nil {
			return err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, in.AuthorID); err != nil {
			return err
		}
	}
	in.Description = normalizeUserText(in.Description)
	post, err := s.posts.GetByID(ctx, in.PostID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "post not found", Fields: map[string]string{"post_id": "not_found"}}
	}
	if post.AuthorID != in.AuthorID {
		return apierr.Error{Code: "invalid_credentials", Message: "forbidden"}
	}

	var reasons []string
	fields := map[string]string{}

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	if len(textFlags) > 0 {
		fields["description"] = "flagged"
		reasons = append(reasons, "text_flagged")
	}
	if textUnavailable {
		reasons = append(reasons, "text_filter_unavailable")
	}

	hasBlockingIssues := len(fields) > 0
	highRisk := hasBlockingIssues || textUnavailable

	if hasBlockingIssues && !in.ContentAck {
		return apierr.Error{
			Code:    "content_requires_ack",
			Message: "content requires confirmation",
			Fields:  fields,
			Data: map[string]any{
				"requires_ack": true,
				"reasons":      reasons,
			},
		}
	}

	aiFlags := map[string]any{
		"reasons":           reasons,
		"text_toxic_score":  textScore,
		"text_categories":   textFlags,
		"content_ack":       in.ContentAck,
		"filters_available": map[string]any{"text": !textUnavailable},
	}
	if s.trust != nil {
		if err := s.trust.ApplySubmissionTrustFlags(ctx, in.AuthorID, &highRisk, &reasons, aiFlags); err != nil {
			return err
		}
	}

	old := normalizeUserText(post.Description)
	newV := in.Description
	if old != newV {
		oldPtr := &old
		newPtr := &newV
		authorID := in.AuthorID
		_ = s.audit.Add(ctx, "post", post.ID, "description", oldPtr, newPtr, &authorID, "pending")
	}

	now := time.Now()
	var toxicScorePtr *float64
	if textScore != nil {
		toxicScorePtr = textScore
	}
	return s.posts.UpdateText(ctx, post.ID, &in.Description, now, "pending", toxicScorePtr, highRisk, aiFlags)
}

func (s *MonumentsService) ValidateUpdatePostSubmission(ctx context.Context, in UpdatePostSubmissionInput) (ContentValidationResult, error) {
	in.Description = normalizeUserText(in.Description)
	post, err := s.posts.GetByID(ctx, in.PostID)
	if err != nil {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "post not found", Fields: map[string]string{"post_id": "not_found"}}
	}
	if post.AuthorID != in.AuthorID {
		return ContentValidationResult{}, apierr.Error{Code: "invalid_credentials", Message: "forbidden"}
	}

	retainedPhotos, retainedFields, retainedReasons, err := s.collectRetainedPhotoIssues(ctx, in.PostID, in.RemovePhotoIDs)
	if err != nil {
		return ContentValidationResult{}, err
	}
	if len(retainedPhotos)+len(in.Photos) == 0 && in.Description == "" {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "empty post", Fields: map[string]string{"description": "required_or_photos"}}
	}
	if len(retainedPhotos)+len(in.Photos) > 10 {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "too many photos", Fields: map[string]string{"photos": "max_10"}}
	}

	fields := retainedFields
	reasons := append([]string{}, retainedReasons...)

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	if len(textFlags) > 0 {
		fields["description"] = "flagged"
		reasons = append(reasons, "text_flagged")
	}
	if textUnavailable {
		reasons = append(reasons, "text_filter_unavailable")
	}

	_, imageIssues, imgUnavailableAny, err := s.checkImages(ctx, in.Photos)
	if err != nil {
		return ContentValidationResult{}, err
	}
	for k, v := range imageIssues {
		fields[k] = v
		reasons = append(reasons, "image_flagged")
	}
	if imgUnavailableAny {
		reasons = append(reasons, "image_filter_unavailable")
	}
	if textScore != nil && *textScore >= 0.7 && !containsReason(reasons, "text_flagged") {
		reasons = append(reasons, "text_flagged")
	}

	highRisk := len(fields) > 0 || textUnavailable || imgUnavailableAny
	return ContentValidationResult{
		RequiresAck: len(fields) > 0,
		Reasons:     uniqueReasons(reasons),
		Fields:      fields,
		HighRisk:    highRisk,
	}, nil
}

func (s *MonumentsService) UpdatePostSubmission(ctx context.Context, in UpdatePostSubmissionInput) (AddPostOutput, error) {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeContentEdit); err != nil {
			return AddPostOutput{}, err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, in.AuthorID); err != nil {
			return AddPostOutput{}, err
		}
	}
	in.Description = normalizeUserText(in.Description)
	validation, err := s.ValidateUpdatePostSubmission(ctx, in)
	if err != nil {
		return AddPostOutput{}, err
	}
	if validation.RequiresAck && !in.ContentAck {
		return AddPostOutput{}, apierr.Error{
			Code:    "content_requires_ack",
			Message: "content requires confirmation",
			Fields:  validation.Fields,
			Data: map[string]any{
				"requires_ack": true,
				"reasons":      validation.Reasons,
			},
		}
	}

	post, err := s.posts.GetByID(ctx, in.PostID)
	if err != nil {
		return AddPostOutput{}, err
	}
	if post.AuthorID != in.AuthorID {
		return AddPostOutput{}, apierr.Error{Code: "invalid_credentials", Message: "forbidden"}
	}
	_ = s.audit.UpdateStatusByEntity(ctx, "post", post.ID, "superseded", nil)
	beforePhotos, _ := s.photos.ListByPost(ctx, post.ID)
	beforePhotoIDs := make(map[uuid.UUID]struct{}, len(beforePhotos))
	for _, photo := range beforePhotos {
		beforePhotoIDs[photo.ID] = struct{}{}
	}

	for _, photoID := range in.RemovePhotoIDs {
		photo, err := s.photos.GetByID(ctx, photoID)
		if err != nil || photo.PostID != in.PostID {
			continue
		}
		oldPath := photo.PreviewPath
		if strings.TrimSpace(oldPath) == "" {
			oldPath = photo.ThumbnailPath
		}
		if strings.TrimSpace(oldPath) != "" {
			oldValue := oldPath
			authorID := in.AuthorID
			_ = s.audit.Add(ctx, "post", post.ID, "photo_removed", &oldValue, nil, &authorID, "pending")
		}
		_ = s.photos.Delete(ctx, photoID)
	}

	textScore, textFlags, textUnavailable := s.checkText(ctx, in.Description)
	imageResults, _, imgUnavailableAny, err := s.checkImages(ctx, in.Photos)
	if err != nil {
		return AddPostOutput{}, err
	}
	aiFlags := map[string]any{
		"reasons":           validation.Reasons,
		"text_toxic_score":  textScore,
		"text_categories":   textFlags,
		"image_results":     imageResults,
		"content_ack":       in.ContentAck,
		"filters_available": map[string]any{"text": !textUnavailable, "image": !imgUnavailableAny},
	}
	if s.trust != nil {
		if err := s.trust.ApplySubmissionTrustFlags(ctx, in.AuthorID, &validation.HighRisk, &validation.Reasons, aiFlags); err != nil {
			return AddPostOutput{}, err
		}
	}

	now := time.Now()
	var toxicScorePtr *float64
	if textScore != nil {
		toxicScorePtr = textScore
	}
	if err := s.posts.UpdateText(ctx, post.ID, &in.Description, now, "pending", toxicScorePtr, validation.HighRisk, aiFlags); err != nil {
		return AddPostOutput{}, err
	}
	oldDescription := normalizeUserText(post.Description)
	newDescription := in.Description
	if oldDescription != newDescription {
		oldPtr := &oldDescription
		newPtr := &newDescription
		authorID := in.AuthorID
		_ = s.audit.Add(ctx, "post", post.ID, "description", oldPtr, newPtr, &authorID, "pending")
	}
	if len(in.Photos) > 0 {
		if err := s.savePhotos(ctx, post.ID, in.Photos, imageResults); err != nil {
			return AddPostOutput{}, err
		}
		afterPhotos, _ := s.photos.ListByPost(ctx, post.ID)
		for _, photo := range afterPhotos {
			if _, exists := beforePhotoIDs[photo.ID]; exists {
				continue
			}
			newPath := photo.PreviewPath
			if strings.TrimSpace(newPath) == "" {
				newPath = photo.ThumbnailPath
			}
			if strings.TrimSpace(newPath) == "" {
				continue
			}
			newValue := newPath
			authorID := in.AuthorID
			_ = s.audit.Add(ctx, "post", post.ID, "photo_added", nil, &newValue, &authorID, "pending")
		}
	}
	return AddPostOutput{PostID: post.ID, HighRisk: validation.HighRisk}, nil
}

func (s *MonumentsService) ValidateUpdateMonumentSubmission(ctx context.Context, in UpdateMonumentSubmissionInput) (ContentValidationResult, error) {
	in.Description = normalizeUserText(in.Description)
	mon, err := s.monuments.GetByID(ctx, in.MonumentID)
	if err != nil {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "monument not found", Fields: map[string]string{"monument_id": "not_found"}}
	}
	if mon.AuthorID == nil || *mon.AuthorID != in.AuthorID {
		return ContentValidationResult{}, apierr.Error{Code: "invalid_credentials", Message: "forbidden"}
	}

	posts, err := s.posts.ListByMonument(ctx, in.MonumentID)
	if err != nil || len(posts) == 0 {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "source post not found"}
	}
	sourcePost, ok := selectAuthorSourcePost(posts, in.AuthorID)
	if !ok {
		return ContentValidationResult{}, apierr.Error{Code: "validation_failed", Message: "source post not found"}
	}
	return s.ValidateUpdatePostSubmission(ctx, UpdatePostSubmissionInput{
		AuthorID:       in.AuthorID,
		PostID:         sourcePost.ID,
		Description:    in.Description,
		Photos:         in.Photos,
		RemovePhotoIDs: in.RemovePhotoIDs,
		ContentAck:     in.ContentAck,
	})
}

func (s *MonumentsService) UpdateMonumentSubmission(ctx context.Context, in UpdateMonumentSubmissionInput) (CreateMonumentOutput, error) {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, in.AuthorID, SanctionScopeContentEdit); err != nil {
			return CreateMonumentOutput{}, err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, in.AuthorID); err != nil {
			return CreateMonumentOutput{}, err
		}
	}
	in.Description = normalizeUserText(in.Description)
	validation, err := s.ValidateUpdateMonumentSubmission(ctx, in)
	if err != nil {
		return CreateMonumentOutput{}, err
	}
	if validation.RequiresAck && !in.ContentAck {
		return CreateMonumentOutput{}, apierr.Error{
			Code:    "content_requires_ack",
			Message: "content requires confirmation",
			Fields:  validation.Fields,
			Data: map[string]any{
				"requires_ack": true,
				"reasons":      validation.Reasons,
			},
		}
	}

	mon, err := s.monuments.GetByID(ctx, in.MonumentID)
	if err != nil {
		return CreateMonumentOutput{}, err
	}
	_ = s.audit.UpdateStatusByEntity(ctx, "monument", in.MonumentID, "superseded", nil)
	posts, err := s.posts.ListByMonument(ctx, in.MonumentID)
	if err != nil || len(posts) == 0 {
		return CreateMonumentOutput{}, apierr.Error{Code: "validation_failed", Message: "source post not found"}
	}
	sourcePost, ok := selectAuthorSourcePost(posts, in.AuthorID)
	if !ok {
		return CreateMonumentOutput{}, apierr.Error{Code: "validation_failed", Message: "source post not found"}
	}

	postResult, err := s.UpdatePostSubmission(ctx, UpdatePostSubmissionInput{
		AuthorID:       in.AuthorID,
		PostID:         sourcePost.ID,
		Description:    in.Description,
		Photos:         in.Photos,
		RemovePhotoIDs: in.RemovePhotoIDs,
		ContentAck:     true,
	})
	if err != nil {
		return CreateMonumentOutput{}, err
	}

	properties := mon.Properties
	if properties == nil {
		properties = map[string]any{}
	}
	properties["description"] = in.Description

	aiFlags := sourcePost.AIFlags
	if aiFlags == nil {
		aiFlags = map[string]any{}
	}
	aiFlags["reasons"] = validation.Reasons
	region := s.geo.GetRegionByCoords(in.Lon, in.Lat)
	if err := s.monuments.UpdateSubmission(ctx, in.MonumentID, in.AuthorID, in.Name, in.Lon, in.Lat, region, properties, validation.HighRisk, aiFlags); err != nil {
		return CreateMonumentOutput{}, err
	}
	authorID := in.AuthorID
	oldName := strings.TrimSpace(mon.Name)
	newName := strings.TrimSpace(in.Name)
	if oldName != newName {
		oldPtr := &oldName
		newPtr := &newName
		_ = s.audit.Add(ctx, "monument", in.MonumentID, "name", oldPtr, newPtr, &authorID, "pending")
	}
	if mon.Lon != in.Lon {
		oldLon := fmt.Sprintf("%.6f", mon.Lon)
		newLon := fmt.Sprintf("%.6f", in.Lon)
		_ = s.audit.Add(ctx, "monument", in.MonumentID, "lon", &oldLon, &newLon, &authorID, "pending")
	}
	if mon.Lat != in.Lat {
		oldLat := fmt.Sprintf("%.6f", mon.Lat)
		newLat := fmt.Sprintf("%.6f", in.Lat)
		_ = s.audit.Add(ctx, "monument", in.MonumentID, "lat", &oldLat, &newLat, &authorID, "pending")
	}

	return CreateMonumentOutput{
		MonumentID: in.MonumentID,
		PostID:     postResult.PostID,
		HighRisk:   validation.HighRisk,
	}, nil
}

func (s *MonumentsService) DeletePost(ctx context.Context, authorID uuid.UUID, postID uuid.UUID) error {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, authorID, SanctionScopeContentEdit); err != nil {
			return err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, authorID); err != nil {
			return err
		}
	}
	preDeletePost, _ := s.posts.GetByID(ctx, postID)
	photos, _ := s.photos.ListByPost(ctx, postID)
	if err := s.posts.Delete(ctx, postID, authorID); err != nil {
		return err
	}
	for _, p := range photos {
		_ = s.uploader.Delete(ctx, p.FilePath)
		_ = s.uploader.Delete(ctx, p.PreviewPath)
		_ = s.uploader.Delete(ctx, p.ThumbnailPath)
	}
	post, err := s.posts.GetByID(ctx, postID)
	if err == nil {
		s.ensureOrphanMonumentVisibility(ctx, post.MonumentID)
	}
	if preDeletePost.AuthorID == authorID {
		postIDCopy := postID
		s.logEvent(ctx, &authorID, &authorID, "post", &postIDCopy, "удаление_поста_автором", "Пользователь удалил свой пост", map[string]any{
			"monument_id": preDeletePost.MonumentID.String(),
		})
	}
	return nil
}

func (s *MonumentsService) logEvent(ctx context.Context, actorID, targetUserID *uuid.UUID, entityType string, entityID *uuid.UUID, action, message string, meta map[string]any) {
	if s.adminLogs == nil {
		return
	}
	if _, err := s.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
		ActorUserID:  actorID,
		TargetUserID: targetUserID,
		EntityType:   entityType,
		EntityID:     entityID,
		Action:       action,
		Result:       "success",
		Message:      message,
		Meta:         meta,
	}); err != nil {
		log.Printf("[ADMIN_LOG] не удалось записать событие %s: %v", action, err)
	}
}

func (s *MonumentsService) deletePostAssetsAndRecord(ctx context.Context, post repo.Post) {
	postPhotos, _ := s.photos.ListByPost(ctx, post.ID)
	for _, photo := range postPhotos {
		_ = s.photos.Delete(ctx, photo.ID)
		_ = s.uploader.Delete(ctx, photo.FilePath)
		_ = s.uploader.Delete(ctx, photo.PreviewPath)
		_ = s.uploader.Delete(ctx, photo.ThumbnailPath)
	}
	_ = s.posts.DeleteByID(ctx, post.ID)
}

func (s *MonumentsService) countVisiblePosts(posts []repo.Post) int {
	count := 0
	for _, post := range posts {
		if post.Status == "approved" && !post.IsHidden && !post.IsArchived {
			count++
		}
	}
	return count
}

func (s *MonumentsService) ensureOrphanMonumentVisibility(ctx context.Context, monumentID uuid.UUID) {
	mon, err := s.monuments.GetByID(ctx, monumentID)
	if err != nil || !mon.IsOrphaned {
		return
	}
	posts, err := s.posts.ListByMonument(ctx, monumentID)
	if err != nil {
		return
	}
	if s.countVisiblePosts(posts) > 0 {
		comment := "Точка оставлена на карте без исходного автора"
		_ = s.monuments.UpdateOrphanState(ctx, monumentID, "approved", &comment)
		return
	}
	comment := "Точка скрыта до появления хотя бы одного согласованного поста"
	_ = s.monuments.UpdateOrphanState(ctx, monumentID, "orphaned_hidden", &comment)
}

func (s *MonumentsService) notifyArchivedPostAuthors(ctx context.Context, monument repo.Monument, actorID uuid.UUID, posts []repo.Post, adminAction bool) {
	if s.notifications == nil {
		return
	}
	link := "/profile"
	title := "Точка переведена в режим без автора"
	if adminAction {
		title = "Исходная точка снята администратором"
	}
	seen := map[uuid.UUID]struct{}{}
	for _, post := range posts {
		if _, ok := seen[post.AuthorID]; ok {
			continue
		}
		seen[post.AuthorID] = struct{}{}
		content := fmt.Sprintf("Точка «%s» больше не связана с исходным автором. Пост можно вернуть на карту или оставить в архиве профиля.", monument.Name)
		_, _ = s.notifications.Create(ctx, post.AuthorID, "monument_orphaned", title, content, &link)
		target := post.AuthorID
		postID := post.ID
		s.logEvent(ctx, &actorID, &target, "post", &postID, "уведомление_автору_архивного_поста", "Автору отправлено уведомление о точке без автора", map[string]any{
			"monument_id":   monument.ID.String(),
			"monument_name": monument.Name,
		})
	}
}

func (s *MonumentsService) handleMonumentRemoval(ctx context.Context, actorID uuid.UUID, monumentID uuid.UUID, adminAction bool) error {
	mon, err := s.monuments.GetByID(ctx, monumentID)
	if err != nil {
		return err
	}
	posts, err := s.posts.ListByMonument(ctx, monumentID)
	if err != nil {
		return err
	}

	ownerID := uuid.Nil
	if mon.AuthorID != nil {
		ownerID = *mon.AuthorID
	}
	var foreignPosts []repo.Post
	for _, post := range posts {
		if ownerID != uuid.Nil && post.AuthorID == ownerID {
			s.deletePostAssetsAndRecord(ctx, post)
			continue
		}
		foreignPosts = append(foreignPosts, post)
	}

	if len(foreignPosts) == 0 {
		return s.monuments.DeleteByID(ctx, monumentID)
	}

	comment := "Исходная точка удалена, чужие посты переведены в архив до решения авторов"
	var orphanedBy *uuid.UUID
	if actorID != uuid.Nil {
		orphanedBy = &actorID
	}
	if err := s.monuments.MarkOrphaned(ctx, monumentID, orphanedBy, "orphaned_hidden", &comment); err != nil {
		return err
	}
	for _, post := range foreignPosts {
		if err := s.posts.Archive(ctx, post.ID, "monument_removed_by_owner", "pending"); err != nil {
			return err
		}
	}
	s.notifyArchivedPostAuthors(ctx, mon, actorID, foreignPosts, adminAction)
	return nil
}

func (s *MonumentsService) DeleteMonument(ctx context.Context, authorID, monumentID uuid.UUID) error {
	if s.sanctions != nil {
		if err := s.sanctions.Check(ctx, authorID, SanctionScopeContentEdit); err != nil {
			return err
		}
	}
	if s.trust != nil {
		if err := s.trust.CheckContentEdit(ctx, authorID); err != nil {
			return err
		}
	}
	mon, err := s.monuments.GetByID(ctx, monumentID)
	if err != nil {
		return err
	}
	if mon.AuthorID == nil || *mon.AuthorID != authorID {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}
	if err := s.handleMonumentRemoval(ctx, authorID, monumentID, false); err != nil {
		return err
	}
	monID := monumentID
	s.logEvent(ctx, &authorID, nil, "monument", &monID, "удаление_точки_автором", "Автор удалил исходную точку", nil)
	return nil
	posts, err := s.posts.ListByMonument(ctx, monumentID)
	if err != nil {
		return err
	}
	hasOtherAuthors := false
	for _, post := range posts {
		if post.AuthorID != authorID {
			hasOtherAuthors = true
			break
		}
	}
	if !hasOtherAuthors {
		for _, post := range posts {
			postPhotos, _ := s.photos.ListByPost(ctx, post.ID)
			for _, photo := range postPhotos {
				_ = s.photos.Delete(ctx, photo.ID)
				_ = s.uploader.Delete(ctx, photo.FilePath)
				_ = s.uploader.Delete(ctx, photo.PreviewPath)
				_ = s.uploader.Delete(ctx, photo.ThumbnailPath)
			}
			_ = s.posts.Delete(ctx, post.ID, post.AuthorID)
		}
		return s.monuments.DeleteByAuthor(ctx, monumentID, authorID)
	}

	comment := "Контент автора точки удален, другие публикации сохранены"
	if err := s.monuments.StripToNameOnly(ctx, monumentID, &comment); err != nil {
		return err
	}
	for _, post := range posts {
		if post.AuthorID != authorID {
			continue
		}
		_ = s.posts.SetHidden(ctx, post.ID, true)
		postPhotos, _ := s.photos.ListByPost(ctx, post.ID)
		for _, photo := range postPhotos {
			_ = s.photos.SetHidden(ctx, photo.ID, true)
		}
	}
	return nil
}

func (s *MonumentsService) PreserveMonumentButHideInitialContent(ctx context.Context, monumentID uuid.UUID, reason string) error {
	return s.handleMonumentRemoval(ctx, uuid.Nil, monumentID, true)
	comment := reason
	if strings.TrimSpace(comment) == "" {
		comment = "Контент скрыт после проверки жалобы"
	}
	if err := s.monuments.StripToNameOnly(ctx, monumentID, &comment); err != nil {
		return err
	}
	mon, err := s.monuments.GetByID(ctx, monumentID)
	if err != nil {
		return err
	}
	posts, err := s.posts.ListByMonument(ctx, monumentID)
	if err != nil {
		return err
	}
	for _, post := range posts {
		if mon.AuthorID != nil && post.AuthorID == *mon.AuthorID {
			_ = s.posts.SetHidden(ctx, post.ID, true)
			postPhotos, _ := s.photos.ListByPost(ctx, post.ID)
			for _, photo := range postPhotos {
				_ = s.photos.SetHidden(ctx, photo.ID, true)
			}
		}
	}
	return nil
}

func (s *MonumentsService) RestoreArchivedPost(ctx context.Context, authorID, postID uuid.UUID, publish bool) error {
	post, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post.AuthorID != authorID {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}
	if !post.IsArchived {
		return nil
	}
	if publish {
		status := post.Status
		if strings.TrimSpace(status) == "" || status == "rejected" {
			status = "approved"
		}
		if err := s.posts.RestoreFromArchive(ctx, postID, status); err != nil {
			return err
		}
		s.ensureOrphanMonumentVisibility(ctx, post.MonumentID)
		postIDCopy := post.ID
		target := post.AuthorID
		s.logEvent(ctx, &authorID, &target, "post", &postIDCopy, "повторная_публикация_архивного_поста", "Автор вернул архивный пост на карту", map[string]any{"monument_id": post.MonumentID.String()})
		return nil
	}
	if err := s.posts.SetRestoreDecision(ctx, postID, "declined"); err != nil {
		return err
	}
	postIDCopy := post.ID
	target := post.AuthorID
	s.logEvent(ctx, &authorID, &target, "post", &postIDCopy, "отказ_от_возврата_архивного_поста", "Автор оставил пост в архиве", map[string]any{"monument_id": post.MonumentID.String()})
	s.ensureOrphanMonumentVisibility(ctx, post.MonumentID)
	return nil
}

func (s *MonumentsService) DeleteMonumentAsAdmin(ctx context.Context, actorID, monumentID uuid.UUID) error {
	if err := s.handleMonumentRemoval(ctx, actorID, monumentID, true); err != nil {
		return err
	}
	monID := monumentID
	s.logEvent(ctx, &actorID, nil, "monument", &monID, "удаление_точки_администратором", "Администратор удалил точку с сохранением безопасной логики", nil)
	return nil
}

func (s *MonumentsService) DeletePostAsAdmin(ctx context.Context, actorID, postID uuid.UUID) error {
	post, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	s.deletePostAssetsAndRecord(ctx, post)
	s.ensureOrphanMonumentVisibility(ctx, post.MonumentID)
	postIDCopy := post.ID
	target := post.AuthorID
	s.logEvent(ctx, &actorID, &target, "post", &postIDCopy, "удаление_поста_администратором", "Администратор удалил пользовательский пост", map[string]any{"monument_id": post.MonumentID.String()})
	if s.notifications != nil {
		link := "/profile"
		_, _ = s.notifications.Create(ctx, post.AuthorID, "content_hidden_by_report", "Пост удален администратором", "Один из постов был удален администратором системы.", &link)
	}
	return nil
}

type MonumentDetail struct {
	Monument repo.Monument     `json:"monument"`
	Posts    []repo.Post       `json:"posts"`
	Photos   []repo.Photo      `json:"photos"`
	Signals  []repo.Signal     `json:"signals"`
	Audit    []repo.AuditEntry `json:"audit,omitempty"`
}

func (s *MonumentsService) GetMonumentDetail(ctx context.Context, monumentID uuid.UUID, userID *uuid.UUID, isMod bool) (MonumentDetail, error) {
	m, err := s.monuments.GetByID(ctx, monumentID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return MonumentDetail{}, apierr.Error{Code: "validation_failed", Message: "monument not found", Fields: map[string]string{"monument_id": "not_found"}}
		}
		return MonumentDetail{}, err
	}
	if m.Status != "approved" {
		allowed := isMod
		if !allowed && userID != nil && m.AuthorID != nil && *userID == *m.AuthorID {
			allowed = true
		}
		if !allowed {
			return MonumentDetail{}, apierr.Error{Code: "validation_failed", Message: "monument not found", Fields: map[string]string{"monument_id": "not_found"}}
		}
	}
	posts, err := s.posts.ListByMonument(ctx, monumentID)
	if err != nil {
		return MonumentDetail{}, err
	}
	var visiblePosts []repo.Post
	for _, p := range posts {
		if p.Status == "approved" && !p.IsHidden && !p.IsArchived {
			visiblePosts = append(visiblePosts, p)
		} else {
			allowed := isMod
			if !allowed && userID != nil && p.AuthorID == *userID {
				allowed = true
			}
			if allowed {
				visiblePosts = append(visiblePosts, p)
			}
		}
	}
	photos, err := s.photos.ListByMonument(ctx, monumentID)
	if err != nil {
		return MonumentDetail{}, err
	}
	var signals []repo.Signal
	if s.signals != nil {
		signals, err = s.signals.ListByMonumentConfirmed(ctx, monumentID)
		if err != nil {
			return MonumentDetail{}, err
		}
	}
	var audit []repo.AuditEntry
	if isMod || (userID != nil && m.AuthorID != nil && *userID == *m.AuthorID) {
		audit, _ = s.pendingMonumentAudit(ctx, monumentID, m.AuthorID)
	}
	return MonumentDetail{Monument: m, Posts: visiblePosts, Photos: photos, Signals: signals, Audit: audit}, nil
}

type MonumentSummary struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Lon       float64   `json:"lon"`
	Lat       float64   `json:"lat"`
	Thumbnail string    `json:"thumbnail,omitempty"`
}

func (s *MonumentsService) GetMonumentSummary(ctx context.Context, monumentID uuid.UUID) (MonumentSummary, error) {
	m, err := s.monuments.GetByID(ctx, monumentID)
	if err != nil {
		return MonumentSummary{}, apierr.Error{Code: "validation_failed", Message: "monument not found", Fields: map[string]string{"monument_id": "not_found"}}
	}
	photos, err := s.photos.ListByMonument(ctx, monumentID)
	if err != nil {
		return MonumentSummary{}, err
	}
	thumb := ""
	if len(photos) > 0 {
		thumb = photos[0].ThumbnailPath
	}
	return MonumentSummary{ID: m.ID, Name: m.Name, Lon: m.Lon, Lat: m.Lat, Thumbnail: thumb}, nil
}

type PostDetail struct {
	Post   repo.Post         `json:"post"`
	Photos []repo.Photo      `json:"photos"`
	Audit  []repo.AuditEntry `json:"audit,omitempty"`
}

type EditDetail struct {
	Entry      repo.AuditEntry   `json:"entry"`
	Related    []repo.AuditEntry `json:"related"`
	EntityType string            `json:"entity_type"`
	EntityID   uuid.UUID         `json:"entity_id"`
	Title      string            `json:"title"`
	AuthorName string            `json:"author_name,omitempty"`
	Lat        *float64          `json:"lat,omitempty"`
	Lon        *float64          `json:"lon,omitempty"`
}

type EditQueueItem struct {
	repo.AuditEntry
	Title         string `json:"title"`
	AuthorName    string `json:"author_name,omitempty"`
	MonumentName  string `json:"monument_name,omitempty"`
	Thumbnail     string `json:"thumbnail,omitempty"`
	IsEditRequest bool   `json:"is_edit_request"`
}

func selectAuthorSourcePost(posts []repo.Post, authorID uuid.UUID) (repo.Post, bool) {
	for i := len(posts) - 1; i >= 0; i-- {
		if posts[i].AuthorID == authorID {
			return posts[i], true
		}
	}
	return repo.Post{}, false
}

func (s *MonumentsService) pendingMonumentAudit(ctx context.Context, monumentID uuid.UUID, authorID *uuid.UUID) ([]repo.AuditEntry, error) {
	entries, err := s.audit.GetByEntity(ctx, "monument", monumentID)
	if err != nil {
		return nil, err
	}
	var out []repo.AuditEntry
	for _, entry := range entries {
		if entry.Status == "pending" {
			out = append(out, entry)
		}
	}
	posts, err := s.posts.ListByMonument(ctx, monumentID)
	if err != nil {
		return out, nil
	}
	for _, post := range posts {
		if authorID != nil && post.AuthorID != *authorID {
			continue
		}
		postAudit, auditErr := s.audit.GetByEntity(ctx, "post", post.ID)
		if auditErr != nil {
			continue
		}
		for _, entry := range postAudit {
			if entry.Status == "pending" {
				out = append(out, entry)
			}
		}
	}
	return out, nil
}

func (s *MonumentsService) GetPostDetail(ctx context.Context, postID uuid.UUID, includeAudit bool) (PostDetail, error) {
	p, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return PostDetail{}, apierr.Error{Code: "validation_failed", Message: "post not found", Fields: map[string]string{"post_id": "not_found"}}
	}
	photos, err := s.photos.ListByPost(ctx, postID)
	if err != nil {
		return PostDetail{}, err
	}
	var audit []repo.AuditEntry
	if includeAudit {
		audit, _ = s.audit.GetByEntity(ctx, "post", postID)
	}
	return PostDetail{Post: p, Photos: photos, Audit: audit}, nil
}

func (s *MonumentsService) ListMonuments(ctx context.Context, status string, limit, offset int) ([]repo.Monument, error) {
	items, err := s.monuments.List(ctx, status, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range items {
		audit, auditErr := s.pendingMonumentAudit(ctx, items[i].ID, items[i].AuthorID)
		if auditErr != nil {
			continue
		}
		for range audit {
			items[i].IsEditRequest = true
			break
		}
	}
	return items, nil
}

func (s *MonumentsService) ListPosts(ctx context.Context, status string, limit, offset int) ([]repo.Post, error) {
	items, err := s.posts.List(ctx, status, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range items {
		audit, auditErr := s.audit.GetByEntity(ctx, "post", items[i].ID)
		if auditErr != nil {
			continue
		}
		for _, entry := range audit {
			if entry.Status == "pending" {
				items[i].IsEditRequest = true
				break
			}
		}
	}
	return items, nil
}

func (s *MonumentsService) ListMonumentsByAuthor(ctx context.Context, authorID uuid.UUID) ([]repo.Monument, error) {
	return s.monuments.ListByAuthor(ctx, authorID)
}

func (s *MonumentsService) ListPostsByAuthor(ctx context.Context, authorID uuid.UUID) ([]repo.Post, error) {
	return s.posts.ListByAuthor(ctx, authorID)
}

func (s *MonumentsService) ModerateMonument(ctx context.Context, id uuid.UUID, status string, comment string, actorID uuid.UUID) error {
	if status != "approved" && status != "rejected" {
		return apierr.Error{Code: "validation_failed", Message: "invalid status"}
	}
	var commentPtr *string
	if comment != "" {
		commentPtr = &comment
	}

	if err := s.monuments.SetStatus(ctx, id, status, commentPtr); err != nil {
		return err
	}

	// Отправляем уведомление автору
	mon, _ := s.monuments.GetByID(ctx, id)
	if mon.AuthorID != nil {
		msg := "Ваша заявка на добавление памятника «" + mon.Name + "» "
		if status == "approved" {
			msg += "одобрена."
		} else {
			msg += "отклонена."
			if comment != "" {
				msg += " Причина: " + comment
			}
		}
		link := monumentLink(id)
		if status == "rejected" {
			link = profileEditMonumentLink(id)
		}
		if mon.AuthorID != nil && *mon.AuthorID != uuid.Nil {
			_, err := s.notifications.Create(ctx, *mon.AuthorID, "monument_status", "Статус заявки", msg, &link)
			if err != nil {
				log.Printf("[NOTIF] Failed to notify monument author %s: %v", mon.AuthorID, err)
			} else {
				log.Printf("[NOTIF] Sent status update to monument author %s", mon.AuthorID)
			}
		}
	}

	// Одобряем/отклоняем все связанные "ожидающие" посты (первый пост новой точки)
	posts, _ := s.posts.ListByMonument(ctx, id)
	for _, p := range posts {
		if p.Status == "pending" {
			_ = s.posts.SetStatus(ctx, p.ID, status, commentPtr)
			_ = s.audit.UpdateStatusByEntity(ctx, "post", p.ID, status, &actorID)
		}
	}

	// Update audit logs
	_ = s.audit.UpdateStatusByEntity(ctx, "monument", id, status, &actorID)

	return nil
}

func (s *MonumentsService) ModeratePost(ctx context.Context, id uuid.UUID, status string, comment string, actorID uuid.UUID) error {
	if status != "approved" && status != "rejected" {
		return apierr.Error{Code: "validation_failed", Message: "invalid status"}
	}
	var commentPtr *string
	if comment != "" {
		commentPtr = &comment
	}

	if err := s.posts.SetStatus(ctx, id, status, commentPtr); err != nil {
		return err
	}

	// Отправляем уведомление автору
	p1, _ := s.posts.GetByID(ctx, id)
	mon1, _ := s.monuments.GetByID(ctx, p1.MonumentID)
	msg1 := "Ваш пост к памятнику «" + mon1.Name + "» "
	if status == "approved" {
		msg1 += "одобрен."
	} else {
		msg1 += "отклонена."
		if comment != "" {
			msg1 += " Причина: " + comment
		}
	}
	link1 := monumentLink(p1.MonumentID)
	if status == "rejected" {
		link1 = profileEditPostLink(id)
	}
	if p1.AuthorID != uuid.Nil {
		_, err := s.notifications.Create(ctx, p1.AuthorID, "post_status", "Статус поста", msg1, &link1)
		if err != nil {
			log.Printf("[NOTIF] Failed to notify post author %s: %v", p1.AuthorID, err)
		} else {
			log.Printf("[NOTIF] Sent status update to post author %s", p1.AuthorID)
		}
	}

	// Update audit logs
	_ = s.audit.UpdateStatusByEntity(ctx, "post", id, status, &actorID)

	return nil
}

func (s *MonumentsService) ListEdits(ctx context.Context, limit, offset int) ([]EditQueueItem, error) {
	entries, err := s.audit.ListPending(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]EditQueueItem, 0, len(entries))
	for _, entry := range entries {
		item := EditQueueItem{
			AuditEntry:    entry,
			Title:         "Заявка на редактирование",
			IsEditRequest: true,
		}
		switch entry.EntityType {
		case "monument":
			mon, monErr := s.monuments.GetByID(ctx, entry.EntityID)
			if monErr == nil {
				item.Title = mon.Name
				item.AuthorName = mon.AuthorName
				item.Thumbnail = mon.Thumbnail
			}
		case "post":
			post, postErr := s.posts.GetByID(ctx, entry.EntityID)
			if postErr == nil {
				item.Title = post.MonumentName
				item.AuthorName = post.AuthorName
				item.MonumentName = post.MonumentName
				item.Thumbnail = post.Thumbnail
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *MonumentsService) GetEditDetail(ctx context.Context, editID uuid.UUID) (EditDetail, error) {
	entry, err := s.audit.GetByID(ctx, editID)
	if err != nil {
		return EditDetail{}, apierr.Error{Code: "validation_failed", Message: "edit not found"}
	}
	related, _ := s.audit.GetByEntity(ctx, entry.EntityType, entry.EntityID)
	detail := EditDetail{
		Entry:      entry,
		Related:    related,
		EntityType: entry.EntityType,
		EntityID:   entry.EntityID,
	}
	switch entry.EntityType {
	case "monument":
		mon, monErr := s.monuments.GetByID(ctx, entry.EntityID)
		if monErr == nil {
			detail.Title = mon.Name
			detail.AuthorName = mon.AuthorName
			detail.Lat = &mon.Lat
			detail.Lon = &mon.Lon
		}
	case "post":
		post, postErr := s.posts.GetByID(ctx, entry.EntityID)
		if postErr == nil {
			detail.Title = post.MonumentName
			detail.AuthorName = post.AuthorName
		}
	default:
		detail.Title = "Правка"
	}
	return detail, nil
}

func (s *MonumentsService) ModerateEdit(ctx context.Context, editID uuid.UUID, action string, actorID uuid.UUID) error {
	edit, err := s.audit.GetByID(ctx, editID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "edit not found"}
	}

	if action == "approve" {
		if edit.EntityType == "post" && edit.FieldName == "description" && edit.NewValue != nil {
			if err := s.posts.UpdateText(ctx, edit.EntityID, edit.NewValue, time.Now(), "approved", nil, false, nil); err != nil {
				return err
			}
		}
		// TODO: handle other entity/field types if added
		return s.audit.SetStatus(ctx, editID, "approved", &actorID)
	} else if action == "reject" {
		return s.audit.SetStatus(ctx, editID, "rejected", &actorID)
	}
	return apierr.Error{Code: "validation_failed", Message: "invalid action"}
}

func (s *MonumentsService) DeletePhoto(ctx context.Context, photoID uuid.UUID, reason string) error {
	photo, err := s.photos.GetByID(ctx, photoID)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "photo not found"}
	}

	if err := s.photos.Delete(ctx, photoID); err != nil {
		return err
	}

	// Отправляем уведомление автору поста
	postPh, _ := s.posts.GetByID(ctx, photo.PostID)
	monPh, _ := s.monuments.GetByID(ctx, postPh.MonumentID)
	msgPh := "Ваша фотография из поста к памятнику «" + monPh.Name + "» была удалена модератором. Причина: " + reason
	linkPh := profileEditPostLink(postPh.ID)
	_, _ = s.notifications.Create(ctx, postPh.AuthorID, "photo_deleted", "Фотография удалена", msgPh, &linkPh)

	_ = s.uploader.Delete(ctx, photo.FilePath)
	_ = s.uploader.Delete(ctx, photo.PreviewPath)
	_ = s.uploader.Delete(ctx, photo.ThumbnailPath)

	// Notify author
	post, err := s.posts.GetByID(ctx, photo.PostID)
	if err == nil && s.notifications != nil {
		link := profileEditPostLink(post.ID)
		_, _ = s.notifications.Create(ctx, post.AuthorID, "photo_deleted", "Фотография удалена", "Причина: "+reason, &link)
	}

	return nil
}

func (s *MonumentsService) checkText(ctx context.Context, text string) (*float64, []string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil, false
	}

	if s.textChecker == nil {
		return nil, nil, true
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := s.textChecker.Check(ctx, text)
	if err != nil {
		log.Printf("AI Text Filter unavailable: %v", err)
		return nil, nil, true
	}

	var toxicScore *float64
	if res.Scores != nil {
		if v, ok := res.Scores["toxicity"]; ok {
			toxicScore = &v
		}
	}

	var flags []string
	for _, c := range res.Categories {
		if c == "profanity" || c == "toxicity" {
			flags = append(flags, c)
		}
	}
	return toxicScore, flags, false
}

func (s *MonumentsService) checkImages(ctx context.Context, photos []*multipart.FileHeader) ([]map[string]any, map[string]string, bool, error) {
	var results []map[string]any
	issues := map[string]string{}
	anyUnavailable := false

	for i, fh := range photos {
		file, err := fh.Open()
		if err != nil {
			return nil, nil, false, err
		}
		b, err := io.ReadAll(io.LimitReader(file, 20<<20))
		_ = file.Close()
		if err != nil {
			return nil, nil, false, err
		}

		if s.imageChecker == nil {
			anyUnavailable = true
			results = append(results, map[string]any{"i": i, "unavailable": true})
			continue
		}

		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		res, err := s.imageChecker.Check(ctx2, fh.Filename, b)
		cancel()
		if err != nil {
			log.Printf("AI Image Filter unavailable (photo %d): %v", i, err)
			anyUnavailable = true
			results = append(results, map[string]any{"i": i, "unavailable": true, "err": err.Error()})
			continue
		}

		results = append(results, map[string]any{"i": i, "status": res.Status, "confidence": res.Confidence})
		if strings.EqualFold(res.Status, "garbage") {
			issues[fmt.Sprintf("photos.%d", i)] = "garbage"
		}
	}
	return results, issues, anyUnavailable, nil
}

func (s *MonumentsService) savePhotos(ctx context.Context, postID uuid.UUID, files []*multipart.FileHeader, imageResults []map[string]any) error {
	for i, fh := range files {
		file, err := fh.Open()
		if err != nil {
			return err
		}
		originalBytes, err := io.ReadAll(io.LimitReader(file, 20<<20))
		_ = file.Close()
		if err != nil {
			return err
		}

		var imageResult map[string]any
		if i < len(imageResults) {
			imageResult = imageResults[i]
		}
		assets, err := buildPreparedImageAssets(originalBytes, fh.Filename, imageResult, sanitize, 95, 95, 95, encodeJPG)
		if err != nil {
			return apierr.Error{Code: "validation_failed", Message: "invalid image", Fields: map[string]string{fmt.Sprintf("photos.%d", i): "invalid_image"}}
		}

		fileID := ids.NewV7()
		fileName := fileID.String() + assets.Ext

		origPath := filepath.ToSlash(filepath.Join("originals", fileName))
		prevPath := filepath.ToSlash(filepath.Join("previews", fileName))
		thumbPath := filepath.ToSlash(filepath.Join("thumbnails", fileName))

		if err := s.uploader.Save(ctx, origPath, bytes.NewReader(assets.Original)); err != nil {
			return err
		}
		if err := s.uploader.Save(ctx, prevPath, bytes.NewReader(assets.Preview)); err != nil {
			_ = s.uploader.Delete(ctx, origPath)
			return err
		}
		if err := s.uploader.Save(ctx, thumbPath, bytes.NewReader(assets.Thumb)); err != nil {
			_ = s.uploader.Delete(ctx, origPath)
			_ = s.uploader.Delete(ctx, prevPath)
			return err
		}

		var relevanceScore *float64
		aiFlags := map[string]any{}
		if i < len(imageResults) {
			aiFlags["image_filter"] = imageResults[i]
			if v, ok := imageResults[i]["confidence"].(float64); ok {
				relevanceScore = &v
			}
		}

		exif := map[string]any{
			"original_name": fh.Filename,
		}
		for k, v := range extractExifData(originalBytes) {
			exif[k] = v
		}

		if _, err := s.photos.Create(ctx, postID, origPath, thumbPath, prevPath, exif, relevanceScore, aiFlags); err != nil {
			_ = s.uploader.Delete(ctx, origPath)
			_ = s.uploader.Delete(ctx, prevPath)
			_ = s.uploader.Delete(ctx, thumbPath)
			return err
		}
	}
	return nil
}

func sanitize(img image.Image, max int) image.Image {
	dst := img
	if img.Bounds().Dx() > max || img.Bounds().Dy() > max {
		dst = imaging.Fit(img, max, max, imaging.Lanczos)
	}
	return dst
}

func encodeJPG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParsePropertiesJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func RequireMultipartForm(r *http.Request) (*multipart.Form, error) {
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		return nil, err
	}
	if r.MultipartForm == nil {
		return nil, errors.New("missing multipart form")
	}
	return r.MultipartForm, nil
}

func extractExifData(originalBytes []byte) map[string]any {
	x, err := exif.Decode(bytes.NewReader(originalBytes))
	if err != nil {
		return map[string]any{}
	}

	out := map[string]any{}
	if tm, err := x.DateTime(); err == nil {
		out["datetime"] = tm.UTC().Format(time.RFC3339)
	}

	if tag, err := x.Get(exif.Model); err == nil {
		if s, err := tag.StringVal(); err == nil {
			out["model"] = s
		}
	}

	lat, lon, err := x.LatLong()
	if err == nil {
		out["gps_lat"] = lat
		out["gps_lon"] = lon
	}

	return out
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func (s *MonumentsService) collectRetainedPhotoIssues(ctx context.Context, postID uuid.UUID, removePhotoIDs []uuid.UUID) ([]repo.Photo, map[string]string, []string, error) {
	allPhotos, err := s.photos.ListByPost(ctx, postID)
	if err != nil {
		return nil, nil, nil, err
	}

	removeSet := make(map[uuid.UUID]struct{}, len(removePhotoIDs))
	for _, photoID := range removePhotoIDs {
		removeSet[photoID] = struct{}{}
	}

	fields := map[string]string{}
	var reasons []string
	var retained []repo.Photo

	for _, photo := range allPhotos {
		if _, skip := removeSet[photo.ID]; skip {
			continue
		}
		retained = append(retained, photo)
		if imageFilter, ok := photo.AIFlags["image_filter"].(map[string]any); ok {
			if status, ok := imageFilter["status"].(string); ok && strings.EqualFold(status, "garbage") {
				fields["existing_photos."+photo.ID.String()] = "garbage"
				reasons = append(reasons, "image_flagged")
				continue
			}
		}
		if photo.RelevanceScore != nil && *photo.RelevanceScore < 0.5 {
			fields["existing_photos."+photo.ID.String()] = "low_relevance"
			reasons = append(reasons, "image_flagged")
		}
	}

	return retained, fields, uniqueReasons(reasons), nil
}
