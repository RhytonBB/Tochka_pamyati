package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type ModerationHandler struct {
	monSvc *service.MonumentsService
	sigSvc *service.SignalsService
	repSvc *service.ReportsService
}

func NewModerationHandler(monSvc *service.MonumentsService, sigSvc *service.SignalsService, repSvc *service.ReportsService) *ModerationHandler {
	return &ModerationHandler{monSvc: monSvc, sigSvc: sigSvc, repSvc: repSvc}
}

func (h *ModerationHandler) ListMonuments(c echo.Context) error {
	status := strings.TrimSpace(c.QueryParam("status"))
	if status == "" {
		status = "pending"
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}

	items, err := h.monSvc.ListMonuments(c.Request().Context(), status, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items, "page": page, "limit": limit})
}

func (h *ModerationHandler) ListPosts(c echo.Context) error {
	status := strings.TrimSpace(c.QueryParam("status"))
	if status == "" {
		status = "pending"
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}

	items, err := h.monSvc.ListPosts(c.Request().Context(), status, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items, "page": page, "limit": limit})
}

func (h *ModerationHandler) GetPostDetail(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	out, err := h.monSvc.GetPostDetail(c.Request().Context(), id, true)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *ModerationHandler) ModerateMonument(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	var req struct {
		Action  string `json:"action"`
		Comment string `json:"comment"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	status := ""
	if req.Action == "approve" {
		status = "approved"
	} else if req.Action == "reject" {
		status = "rejected"
		if strings.TrimSpace(req.Comment) == "" {
			return apierr.Error{Code: "validation_failed", Message: "comment required for rejection", Fields: map[string]string{"comment": "required"}}
		}
	} else {
		return apierr.Error{Code: "validation_failed", Message: "invalid action"}
	}

	val := c.Get(UserContextKey)
	if val == nil {
		return apierr.Error{Code: "unauthorized", Message: "unauthorized"}
	}
	user := val.(service.PublicUser)

	if err := h.monSvc.ModerateMonument(c.Request().Context(), id, status, req.Comment, user.ID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ModerationHandler) DeletePhoto(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Reason) == "" {
		return apierr.Error{Code: "validation_failed", Message: "reason required", Fields: map[string]string{"reason": "required"}}
	}

	if err := h.monSvc.DeletePhoto(c.Request().Context(), id, req.Reason); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ModerationHandler) DeleteComment(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Reason) == "" {
		return apierr.Error{Code: "validation_failed", Message: "reason required", Fields: map[string]string{"reason": "required"}}
	}

	if err := h.sigSvc.DeleteComment(c.Request().Context(), id, req.Reason); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ModerationHandler) ModeratePost(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	var req struct {
		Action  string `json:"action"`
		Comment string `json:"comment"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	status := ""
	if req.Action == "approve" {
		status = "approved"
	} else if req.Action == "reject" {
		status = "rejected"
		if strings.TrimSpace(req.Comment) == "" {
			return apierr.Error{Code: "validation_failed", Message: "comment required for rejection", Fields: map[string]string{"comment": "required"}}
		}
	} else {
		return apierr.Error{Code: "validation_failed", Message: "invalid action"}
	}

	val := c.Get(UserContextKey)
	if val == nil {
		return apierr.Error{Code: "unauthorized", Message: "unauthorized"}
	}
	user := val.(service.PublicUser)

	if err := h.monSvc.ModeratePost(c.Request().Context(), id, status, req.Comment, user.ID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ModerationHandler) ListEdits(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}

	items, err := h.monSvc.ListEdits(c.Request().Context(), limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items, "page": page, "limit": limit})
}

func (h *ModerationHandler) GetEditDetail(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	item, err := h.monSvc.GetEditDetail(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, item)
}

func (h *ModerationHandler) ModerateEdit(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	var req struct {
		Action string `json:"action"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	val := c.Get(UserContextKey)
	if val == nil {
		return apierr.Error{Code: "unauthorized", Message: "unauthorized"}
	}
	user := val.(service.PublicUser)

	if err := h.monSvc.ModerateEdit(c.Request().Context(), id, req.Action, user.ID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ModerationHandler) ListSignals(c echo.Context) error {
	status := strings.TrimSpace(c.QueryParam("status"))
	if status == "" {
		status = "pending"
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}

	items, err := h.sigSvc.List(c.Request().Context(), repo.ListSignalsFilter{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items, "page": page, "limit": limit})
}

func (h *ModerationHandler) ModerateSignal(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	var req struct {
		Action   string `json:"action"`
		Response string `json:"official_response"`
		Urgency  string `json:"urgency"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	val := c.Get(UserContextKey)
	if val == nil {
		return apierr.Error{Code: "unauthorized", Message: "unauthorized"}
	}
	user := val.(service.PublicUser)

	if err := h.sigSvc.Moderate(c.Request().Context(), service.ModerateSignalInput{
		SignalID: id,
		ActorID:  user.ID,
		Action:   req.Action,
		Comment:  req.Response,
		Urgency:  req.Urgency,
	}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ModerationHandler) ListReports(c echo.Context) error {
	status := strings.TrimSpace(c.QueryParam("status"))
	if status == "" {
		status = "pending"
	}
	entityType := strings.TrimSpace(c.QueryParam("entity_type"))
	reasonCode := strings.TrimSpace(c.QueryParam("reason_code"))
	category := strings.TrimSpace(c.QueryParam("category"))
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	items, p, l, err := h.repSvc.ListCases(c.Request().Context(), status, entityType, reasonCode, category, page, limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items, "page": p, "limit": l})
}

func (h *ModerationHandler) GetReportCase(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	item, err := h.repSvc.GetCase(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, item)
}

func (h *ModerationHandler) ModerateReport(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid id"}
	}
	var req struct {
		Action            string         `json:"action"`
		ModeratorComment  string         `json:"moderator_comment"`
		EditedContent     string         `json:"edited_content"`
		ReturnStatus      string         `json:"return_status"`
		FixPayload        map[string]any `json:"fix_payload"`
		TargetPartType    string         `json:"target_part_type"`
		TargetPartID      *uuid.UUID     `json:"target_part_id"`
		DuplicateTargetID *uuid.UUID     `json:"duplicate_target_id"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	val := c.Get(UserContextKey)
	if val == nil {
		return apierr.Error{Code: "unauthorized", Message: "unauthorized"}
	}
	user := val.(service.PublicUser)

	if err := h.repSvc.ModerateCase(c.Request().Context(), service.ModerateReportCaseInput{
		CaseID:           id,
		ActorID:          user.ID,
		Action:           req.Action,
		ModeratorComment: req.ModeratorComment,
		EditedContent:    req.EditedContent,
		ReturnStatus:     req.ReturnStatus,
		FixPayload:       req.FixPayload,
		TargetPartType:   req.TargetPartType,
		TargetPartID:     req.TargetPartID,
		DuplicateTargetID: req.DuplicateTargetID,
	}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ModerationHandler) UndoModeration(c echo.Context) error {
	var req struct {
		EntityType string    `json:"entity_type"`
		EntityID   uuid.UUID `json:"entity_id"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.EntityType == "monument" {
		if err := h.monSvc.ModerateMonument(c.Request().Context(), req.EntityID, "pending", "", uuid.Nil); err != nil {
			return err
		}
	} else if req.EntityType == "post" {
		if err := h.monSvc.ModeratePost(c.Request().Context(), req.EntityID, "pending", "", uuid.Nil); err != nil {
			return err
		}
	} else {
		return apierr.Error{Code: "validation_failed", Message: "invalid entity type"}
	}

	return c.NoContent(http.StatusNoContent)
}
