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

type ReportsHandler struct {
	auth  *service.Auth
	roles *repo.Roles
	svc   *service.ReportsService
}

func NewReportsHandler(auth *service.Auth, roles *repo.Roles, svc *service.ReportsService) *ReportsHandler {
	return &ReportsHandler{auth: auth, roles: roles, svc: svc}
}

type CreateReportRequest struct {
	EntityType        string   `json:"entity_type"`
	EntityID          string   `json:"entity_id"`
	ReasonCode        string   `json:"reason_code"`
	Comment           string   `json:"comment"`
	SuggestedTitle    string   `json:"suggested_title"`
	SuggestedLon      *float64 `json:"suggested_lon"`
	SuggestedLat      *float64 `json:"suggested_lat"`
	DuplicateTargetID *string  `json:"duplicate_target_id"`
}

func (h *ReportsHandler) Create(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	var req CreateReportRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "Некорректный формат запроса"}
	}
	entityID, err := uuid.Parse(strings.TrimSpace(req.EntityID))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "Некорректный entity_id", Fields: map[string]string{"entity_id": "invalid"}}
	}

	out, err := h.svc.Create(c.Request().Context(), service.CreateReportInput{
		ReporterID:        user.ID,
		EntityType:        req.EntityType,
		EntityID:          entityID,
		ReasonCode:        req.ReasonCode,
		Comment:           req.Comment,
		SuggestedTitle:    req.SuggestedTitle,
		SuggestedLon:      req.SuggestedLon,
		SuggestedLat:      req.SuggestedLat,
		DuplicateTargetID: parseUUIDPtr(req.DuplicateTargetID),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, out)
}

func (h *ReportsHandler) List(c echo.Context) error {
	user, ok := h.optionalUser(c)
	if !ok {
		return apierr.Error{Code: "forbidden", Message: "Доступ запрещен"}
	}
	if err := h.requireModerator(c.Request().Context(), user.RoleID); err != nil {
		return err
	}

	status := strings.TrimSpace(c.QueryParam("status"))
	entityType := strings.TrimSpace(c.QueryParam("entity_type"))
	reasonCode := strings.TrimSpace(c.QueryParam("reason_code"))
	category := strings.TrimSpace(c.QueryParam("category"))
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, p, l, err := h.svc.ListCases(c.Request().Context(), status, entityType, reasonCode, category, page, limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items, "page": p, "limit": l})
}

type ModerateReportRequest struct {
	Action            string         `json:"action"`
	ModeratorComment  string         `json:"moderator_comment"`
	EditedContent     string         `json:"edited_content"`
	ReturnStatus      string         `json:"return_status"`
	FixPayload        map[string]any `json:"fix_payload"`
	TargetPartType    string         `json:"target_part_type"`
	TargetPartID      *uuid.UUID     `json:"target_part_id"`
	DuplicateTargetID *uuid.UUID     `json:"duplicate_target_id"`
}

func (h *ReportsHandler) Moderate(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	if err := h.requireModerator(c.Request().Context(), user.RoleID); err != nil {
		return err
	}

	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "Некорректный id жалобы", Fields: map[string]string{"id": "invalid"}}
	}

	var req ModerateReportRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "Некорректный формат запроса"}
	}

	if err := h.svc.ModerateCase(c.Request().Context(), service.ModerateReportCaseInput{
		CaseID:           reportID,
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

func (h *ReportsHandler) MyReports(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	status := strings.TrimSpace(c.QueryParam("status"))
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, p, l, err := h.svc.ListMyReports(c.Request().Context(), user.ID, status, page, limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items, "page": p, "limit": l})
}

func parseUUIDPtr(raw *string) *uuid.UUID {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	id, err := uuid.Parse(strings.TrimSpace(*raw))
	if err != nil {
		return nil
	}
	return &id
}

func (h *ReportsHandler) currentUser(c echo.Context) (service.PublicUser, error) {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || strings.TrimSpace(accessCookie.Value) == "" {
		return service.PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "Отсутствует токен доступа"}
	}
	return h.auth.Me(c.Request().Context(), accessCookie.Value)
}

func (h *ReportsHandler) optionalUser(c echo.Context) (service.PublicUser, bool) {
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

func (h *ReportsHandler) requireModerator(ctx context.Context, roleID uuid.UUID) error {
	if h.roles == nil {
		return apierr.Error{Code: "forbidden", Message: "Доступ запрещен"}
	}
	name, err := h.roles.GetNameByID(ctx, roleID)
	if err != nil {
		return apierr.Error{Code: "forbidden", Message: "Доступ запрещен"}
	}
	if name == "moderator" || name == "admin" {
		return nil
	}
	return apierr.Error{Code: "forbidden", Message: "Доступ запрещен"}
}
