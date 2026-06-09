package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type MonumentsHandler struct {
	auth *service.Auth
	svc  *service.MonumentsService
}

func NewMonumentsHandler(auth *service.Auth, svc *service.MonumentsService) *MonumentsHandler {
	return &MonumentsHandler{auth: auth, svc: svc}
}

func (h *MonumentsHandler) CreateMonument(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}

	name := firstFormValue(form.Value, "name")
	lon, _ := strconv.ParseFloat(firstFormValue(form.Value, "lon"), 64)
	lat, _ := strconv.ParseFloat(firstFormValue(form.Value, "lat"), 64)
	propsRaw := firstFormValue(form.Value, "properties")
	desc := firstFormValue(form.Value, "description")
	contentAck := parseBool(firstFormValue(form.Value, "content_ack"))
	createPostWithMonument := parseBool(firstFormValue(form.Value, "create_post_with_monument"))

	props, err := service.ParsePropertiesJSON(propsRaw)
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid properties", Fields: map[string]string{"properties": "invalid_json"}}
	}

	files := form.File["photos"]
	out, err := h.svc.CreateMonumentWithFirstPost(c.Request().Context(), service.CreateMonumentInput{
		AuthorID:    user.ID,
		Name:        name,
		Lon:         lon,
		Lat:         lat,
		Properties:  props,
		Description: desc,
		Photos:      files,
		ContentAck:  contentAck,
		CreatePost:  createPostWithMonument,
	})
	if err != nil {
		fmt.Printf("CreateMonument Error: %#v\n", err)
		return err
	}
	return c.JSON(http.StatusCreated, out)
}

func (h *MonumentsHandler) ValidateMonument(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}

	props, err := service.ParsePropertiesJSON(firstFormValue(form.Value, "properties"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid properties", Fields: map[string]string{"properties": "invalid_json"}}
	}
	result, dup, err := h.svc.ValidateCreateMonument(c.Request().Context(), service.CreateMonumentInput{
		AuthorID:    user.ID,
		Name:        firstFormValue(form.Value, "name"),
		Lon:         parseMultipartFloatOrZero(firstFormValue(form.Value, "lon")),
		Lat:         parseMultipartFloatOrZero(firstFormValue(form.Value, "lat")),
		Properties:  props,
		Description: firstFormValue(form.Value, "description"),
		Photos:      form.File["photos"],
		CreatePost:  parseBool(firstFormValue(form.Value, "create_post_with_monument")),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{
		"requires_ack": result.RequiresAck,
		"reasons":      result.Reasons,
		"fields":       result.Fields,
		"high_risk":    result.HighRisk,
		"duplicates":   dup,
	})
}

func (h *MonumentsHandler) AddPost(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	monumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
	}

	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}

	desc := firstFormValue(form.Value, "description")
	contentAck := parseBool(firstFormValue(form.Value, "content_ack"))
	files := form.File["photos"]

	out, err := h.svc.AddPost(c.Request().Context(), service.AddPostInput{
		AuthorID:    user.ID,
		MonumentID:  monumentID,
		Description: desc,
		Photos:      files,
		ContentAck:  contentAck,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, out)
}

func (h *MonumentsHandler) ValidatePost(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	monumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
	}

	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}
	result, err := h.svc.ValidateAddPost(c.Request().Context(), service.AddPostInput{
		AuthorID:    user.ID,
		MonumentID:  monumentID,
		Description: firstFormValue(form.Value, "description"),
		Photos:      form.File["photos"],
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, result)
}

type UpdatePostRequest struct {
	Description string `json:"description"`
	ContentAck  bool   `json:"content_ack"`
}

func (h *MonumentsHandler) UpdatePost(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid post id", Fields: map[string]string{"post_id": "invalid"}}
	}

	var req UpdatePostRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}

	if err := h.svc.UpdatePostText(c.Request().Context(), service.UpdatePostInput{
		AuthorID:    user.ID,
		PostID:      postID,
		Description: req.Description,
		ContentAck:  req.ContentAck,
	}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *MonumentsHandler) UpdatePostSubmission(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("postId"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid post id", Fields: map[string]string{"post_id": "invalid"}}
	}
	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}

	out, err := h.svc.UpdatePostSubmission(c.Request().Context(), service.UpdatePostSubmissionInput{
		AuthorID:       user.ID,
		PostID:         postID,
		Description:    firstFormValue(form.Value, "description"),
		Photos:         form.File["photos"],
		RemovePhotoIDs: parseUUIDList(form.Value["remove_photo_ids"]),
		ContentAck:     parseBool(firstFormValue(form.Value, "content_ack")),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *MonumentsHandler) ValidatePostEdit(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid post id", Fields: map[string]string{"post_id": "invalid"}}
	}
	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}
	result, err := h.svc.ValidateUpdatePostSubmission(c.Request().Context(), service.UpdatePostSubmissionInput{
		AuthorID:       user.ID,
		PostID:         postID,
		Description:    firstFormValue(form.Value, "description"),
		Photos:         form.File["photos"],
		RemovePhotoIDs: parseUUIDList(form.Value["remove_photo_ids"]),
		ContentAck:     parseBool(firstFormValue(form.Value, "content_ack")),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, result)
}

func (h *MonumentsHandler) ValidateMonumentEdit(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	monumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
	}
	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}
	result, err := h.svc.ValidateUpdateMonumentSubmission(c.Request().Context(), service.UpdateMonumentSubmissionInput{
		AuthorID:       user.ID,
		MonumentID:     monumentID,
		Name:           firstFormValue(form.Value, "name"),
		Lon:            parseMultipartFloatOrZero(firstFormValue(form.Value, "lon")),
		Lat:            parseMultipartFloatOrZero(firstFormValue(form.Value, "lat")),
		Description:    firstFormValue(form.Value, "description"),
		Photos:         form.File["photos"],
		RemovePhotoIDs: parseUUIDList(form.Value["remove_photo_ids"]),
		ContentAck:     parseBool(firstFormValue(form.Value, "content_ack")),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, result)
}

func (h *MonumentsHandler) UpdateMonument(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	monumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
	}
	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}
	out, err := h.svc.UpdateMonumentSubmission(c.Request().Context(), service.UpdateMonumentSubmissionInput{
		AuthorID:       user.ID,
		MonumentID:     monumentID,
		Name:           firstFormValue(form.Value, "name"),
		Lon:            parseMultipartFloatOrZero(firstFormValue(form.Value, "lon")),
		Lat:            parseMultipartFloatOrZero(firstFormValue(form.Value, "lat")),
		Description:    firstFormValue(form.Value, "description"),
		Photos:         form.File["photos"],
		RemovePhotoIDs: parseUUIDList(form.Value["remove_photo_ids"]),
		ContentAck:     parseBool(firstFormValue(form.Value, "content_ack")),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *MonumentsHandler) DeletePost(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid post id", Fields: map[string]string{"post_id": "invalid"}}
	}

	if err := h.svc.DeletePost(c.Request().Context(), user.ID, postID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *MonumentsHandler) RestoreArchivedPost(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid post id", Fields: map[string]string{"post_id": "invalid"}}
	}
	var req struct {
		Publish bool `json:"publish"`
	}
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.svc.RestoreArchivedPost(c.Request().Context(), user.ID, postID, req.Publish); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *MonumentsHandler) DeleteMonument(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	monumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
	}
	if err := h.svc.DeleteMonument(c.Request().Context(), user.ID, monumentID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *MonumentsHandler) GetMonumentDetail(c echo.Context) error {
	monumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
	}

	val := c.Get(UserContextKey)
	var userID *uuid.UUID
	var isMod = false
	if val != nil {
		if u, ok := val.(service.PublicUser); ok {
			userID = &u.ID
			if u.RoleName == "moderator" || u.RoleName == "admin" {
				isMod = true
			}
		}
	}

	out, err := h.svc.GetMonumentDetail(c.Request().Context(), monumentID, userID, isMod)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *MonumentsHandler) GetMonumentSummary(c echo.Context) error {
	monumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
	}

	out, err := h.svc.GetMonumentSummary(c.Request().Context(), monumentID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *MonumentsHandler) currentUser(c echo.Context) (service.PublicUser, error) {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || strings.TrimSpace(accessCookie.Value) == "" {
		return service.PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "missing access token"}
	}
	return h.auth.Me(c.Request().Context(), accessCookie.Value)
}

func firstFormValue(values map[string][]string, key string) string {
	if values == nil {
		return ""
	}
	v := values[key]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

func parseBool(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes" || v == "y" || v == "on"
}

func parseMultipartFloatOrZero(v string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f
}

func parseUUIDList(values []string) []uuid.UUID {
	var out []uuid.UUID
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if id, err := uuid.Parse(part); err == nil {
				out = append(out, id)
			}
		}
	}
	return out
}
