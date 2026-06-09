package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type SignalsHandler struct {
	auth  *service.Auth
	roles *repo.Roles
	svc   *service.SignalsService
}

func NewSignalsHandler(auth *service.Auth, roles *repo.Roles, svc *service.SignalsService) *SignalsHandler {
	return &SignalsHandler{auth: auth, roles: roles, svc: svc}
}

func (h *SignalsHandler) List(c echo.Context) error {
	status := strings.TrimSpace(c.QueryParam("status"))
	signalType := strings.TrimSpace(c.QueryParam("type"))
	urgency := strings.TrimSpace(c.QueryParam("urgency"))
	region := strings.TrimSpace(c.QueryParam("region"))
	excludeRegion := strings.TrimSpace(c.QueryParam("exclude_region"))
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	if status == "" {
		status = "confirmed"
	} else if status != "confirmed" && status != "resolved" {
		user, ok := h.optionalUser(c)
		if !ok {
			return apierr.Error{Code: "forbidden", Message: "forbidden"}
		}
		if err := h.requireModerator(c.Request().Context(), user.RoleID); err != nil {
			return err
		}
	}

	var userID *uuid.UUID
	if user, ok := h.optionalUser(c); ok {
		userID = &user.ID
	}

	items, err := h.svc.List(c.Request().Context(), repo.ListSignalsFilter{
		Status:     status,
		SignalType: signalType,
		Urgency:    urgency,
		Region:     region,
		ExcludeRegion: excludeRegion,
		Limit:      limit,
		Offset:     offset,
		UserID:     userID,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"items": items,
		"page":  page,
		"limit": limit,
	})
}

func (h *SignalsHandler) GetDetail(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}
	var userID *uuid.UUID
	if user, ok := h.optionalUser(c); ok {
		userID = &user.ID
	}

	out, err := h.svc.GetDetail(c.Request().Context(), id, userID)
	if err != nil {
		return err
	}

	if out.Signal.Status != "confirmed" && out.Signal.Status != "resolved" {
		user, ok := h.optionalUser(c)
		if !ok {
			return apierr.Error{Code: "validation_failed", Message: "signal not found", Fields: map[string]string{"signal_id": "not_found"}}
		}
		if out.Signal.AuthorID != nil && *out.Signal.AuthorID == user.ID {
			return c.JSON(http.StatusOK, out)
		}
		if err := h.requireModerator(c.Request().Context(), user.RoleID); err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, out)
}

func (h *SignalsHandler) Validate(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}

	monumentIDRaw := firstFormValue(form.Value, "monument_id")
	var monumentID *uuid.UUID
	if strings.TrimSpace(monumentIDRaw) != "" {
		id, err := uuid.Parse(monumentIDRaw)
		if err != nil {
			return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
		}
		monumentID = &id
	}

	result, err := h.svc.ValidateCreate(c.Request().Context(), service.CreateSignalInput{
		AuthorID:        user.ID,
		MonumentID:      monumentID,
		NewMonumentName: firstFormValue(form.Value, "monument_name"),
		NewLon:          parseSignalFloatOrZero(firstFormValue(form.Value, "lon")),
		NewLat:          parseSignalFloatOrZero(firstFormValue(form.Value, "lat")),
		SignalType:      firstFormValue(form.Value, "signal_type"),
		Urgency:         firstFormValue(form.Value, "urgency"),
		Description:     firstFormValue(form.Value, "description"),
		Photos:          form.File["photos"],
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, result)
}

func (h *SignalsHandler) ConfirmedMapPoints(c echo.Context) error {
	points, err := h.svc.ConfirmedMapPoints(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, points)
}

func (h *SignalsHandler) Create(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	form, err := service.RequireMultipartForm(c.Request())
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid multipart form"}
	}

	monumentIDRaw := firstFormValue(form.Value, "monument_id")
	var monumentID *uuid.UUID
	if strings.TrimSpace(monumentIDRaw) != "" {
		id, err := uuid.Parse(monumentIDRaw)
		if err != nil {
			return apierr.Error{Code: "validation_failed", Message: "invalid monument id", Fields: map[string]string{"monument_id": "invalid"}}
		}
		monumentID = &id
	}

	name := firstFormValue(form.Value, "monument_name")
	lon, _ := strconv.ParseFloat(firstFormValue(form.Value, "lon"), 64)
	lat, _ := strconv.ParseFloat(firstFormValue(form.Value, "lat"), 64)
	signalType := firstFormValue(form.Value, "signal_type")
	urgency := firstFormValue(form.Value, "urgency")
	desc := firstFormValue(form.Value, "description")
	contentAck := parseBool(firstFormValue(form.Value, "content_ack"))
	files := form.File["photos"]

	out, err := h.svc.Create(c.Request().Context(), service.CreateSignalInput{
		AuthorID:        user.ID,
		MonumentID:      monumentID,
		NewMonumentName: name,
		NewLon:          lon,
		NewLat:          lat,
		SignalType:      signalType,
		Urgency:         urgency,
		Description:     desc,
		Photos:          files,
		ContentAck:      contentAck,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, out)
}

type UpdateSignalRequest struct {
	SignalType  string `json:"signal_type"`
	Urgency     string `json:"urgency"`
	Description string `json:"description"`
}

type UpdateSignalStatusRequest struct {
	Resolved          bool   `json:"resolved"`
	ResolutionKind    string `json:"resolution_kind"`
	ResolutionComment string `json:"resolution_comment"`
}

func (h *SignalsHandler) UpdateOwn(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}
	var req UpdateSignalRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.svc.UpdateOwnSignal(c.Request().Context(), service.UpdateOwnSignalInput{
		AuthorID:    user.ID,
		SignalID:    signalID,
		SignalType:  req.SignalType,
		Urgency:     req.Urgency,
		Description: req.Description,
	}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SignalsHandler) DeleteOwn(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}
	if err := h.svc.DeleteOwnSignal(c.Request().Context(), user.ID, signalID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SignalsHandler) SetOwnResolved(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}
	var req UpdateSignalStatusRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.svc.SetOwnResolved(c.Request().Context(), user.ID, signalID, req.Resolved, req.ResolutionKind, req.ResolutionComment); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type AddCommentRequest struct {
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	Content  string     `json:"content"`
}

func (h *SignalsHandler) AddComment(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}

	var req AddCommentRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}

	out, err := h.svc.AddComment(c.Request().Context(), service.AddCommentInput{
		AuthorID: user.ID,
		SignalID: signalID,
		ParentID: req.ParentID,
		Content:  req.Content,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, out)
}

func (h *SignalsHandler) EditComment(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid comment id", Fields: map[string]string{"comment_id": "invalid"}}
	}
	var req AddCommentRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	out, err := h.svc.EditOwnComment(c.Request().Context(), user.ID, commentID, req.Content)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *SignalsHandler) DeleteComment(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid comment id", Fields: map[string]string{"comment_id": "invalid"}}
	}
	if err := h.svc.DeleteOwnComment(c.Request().Context(), user.ID, commentID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SignalsHandler) Support(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}
	if err := h.svc.Support(c.Request().Context(), signalID, user.ID, true); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SignalsHandler) Unsupport(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}
	if err := h.svc.Support(c.Request().Context(), signalID, user.ID, false); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type ModerateSignalRequest struct {
	Action  string `json:"action"`
	Comment string `json:"comment"`
	Urgency string `json:"urgency"`
}

func (h *SignalsHandler) Moderate(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	if err := h.requireModerator(c.Request().Context(), user.RoleID); err != nil {
		return err
	}

	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id", Fields: map[string]string{"signal_id": "invalid"}}
	}

	var req ModerateSignalRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}

	if err := h.svc.Moderate(c.Request().Context(), service.ModerateSignalInput{
		SignalID: signalID,
		ActorID:  user.ID,
		Action:   req.Action,
		Comment:  req.Comment,
		Urgency:  req.Urgency,
	}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SignalsHandler) currentUser(c echo.Context) (service.PublicUser, error) {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || strings.TrimSpace(accessCookie.Value) == "" {
		return service.PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "missing access token"}
	}
	return h.auth.Me(c.Request().Context(), accessCookie.Value)
}

func parseSignalFloatOrZero(v string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f
}

func (h *SignalsHandler) optionalUser(c echo.Context) (service.PublicUser, bool) {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || strings.TrimSpace(accessCookie.Value) == "" {
		return service.PublicUser{}, false
	}
	u, err := h.auth.Me(c.Request().Context(), accessCookie.Value)
	if err != nil {
		return service.PublicUser{}, false
	}
	return u, true
}

func (h *SignalsHandler) requireModerator(ctx context.Context, roleID uuid.UUID) error {
	if h.roles == nil {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}
	name, err := h.roles.GetNameByID(ctx, roleID)
	if err != nil {
		return apierr.Error{Code: "forbidden", Message: "forbidden"}
	}
	if name == "moderator" || name == "admin" {
		return nil
	}
	return apierr.Error{Code: "forbidden", Message: "forbidden"}
}
