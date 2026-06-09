package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type AdminHandler struct {
	analSvc   *service.AnalyticsService
	usersSvc  *service.UsersService
	monSvc    *service.MonumentsService
	signalSvc *service.SignalsService
	exportSvc *service.ExportService
}

func NewAdminHandler(analSvc *service.AnalyticsService, usersSvc *service.UsersService, monSvc *service.MonumentsService, signalSvc *service.SignalsService, exportSvc *service.ExportService) *AdminHandler {
	return &AdminHandler{
		analSvc:   analSvc,
		usersSvc:  usersSvc,
		monSvc:    monSvc,
		signalSvc: signalSvc,
		exportSvc: exportSvc,
	}
}

func (h *AdminHandler) GetGlobalStats(c echo.Context) error {
	stats, err := h.analSvc.GetGlobalStats(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, stats)
}

func (h *AdminHandler) GetDynamics(c echo.Context) error {
	days, _ := strconv.Atoi(c.QueryParam("days"))
	if days <= 0 {
		days = 30
	}
	dynamics, err := h.analSvc.GetDynamics(c.Request().Context(), days)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dynamics)
}

func (h *AdminHandler) GetTopAuthors(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 10
	}
	top, err := h.analSvc.GetTopAuthors(c.Request().Context(), limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, top)
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}

	filter := repo.ListUsersFilter{
		Username: c.QueryParam("username"),
		Email:    c.QueryParam("email"),
		City:     c.QueryParam("city"),
		Limit:    limit,
		Offset:   offset,
	}

	if roleIDStr := c.QueryParam("role_id"); roleIDStr != "" {
		if id, err := uuid.Parse(roleIDStr); err == nil {
			filter.RoleID = &id
		}
	}
	if activeStr := c.QueryParam("is_active"); activeStr != "" {
		b := activeStr == "true"
		filter.IsActive = &b
	}
	if blockedStr := c.QueryParam("is_blocked"); blockedStr != "" {
		b := blockedStr == "true"
		filter.IsBlocked = &b
	}

	out, err := h.usersSvc.ListUsers(c.Request().Context(), filter)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *AdminHandler) UpdateUserRole(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid user id"}
	}
	var req struct {
		RoleID   uuid.UUID `json:"role_id"`
		RoleName string    `json:"role_name"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.RoleID == uuid.Nil && req.RoleName != "" {
		roleID, err := h.usersSvc.ResolveRoleID(c.Request().Context(), req.RoleName)
		if err != nil {
			return err
		}
		req.RoleID = roleID
	}
	if err := h.usersSvc.UpdateRole(c.Request().Context(), id, req.RoleID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) GetUser(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid user id"}
	}
	out, err := h.usersSvc.GetUserAdminDetail(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *AdminHandler) SetUserBlocked(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid user id"}
	}
	var req struct {
		Blocked bool `json:"blocked"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.usersSvc.SetBlocked(c.Request().Context(), id, req.Blocked); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) DeleteUser(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid user id"}
	}
	if err := h.usersSvc.Delete(c.Request().Context(), id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) CreateSanction(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid user id"}
	}
	actor := c.Get(UserContextKey).(service.PublicUser)
	var req struct {
		Kind              string         `json:"kind"`
		Source            string         `json:"source"`
		ReasonCode        string         `json:"reason_code"`
		ReasonText        string         `json:"reason_text"`
		Scopes            []string       `json:"scopes"`
		EndsAt            *time.Time     `json:"ends_at"`
		RelatedEntityType string         `json:"related_entity_type"`
		RelatedEntityID   *uuid.UUID     `json:"related_entity_id"`
		Meta              map[string]any `json:"meta"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	item, err := h.usersSvc.CreateSanction(c.Request().Context(), repo.CreateUserSanctionParams{
		UserID:            userID,
		Kind:              req.Kind,
		Source:            req.Source,
		ReasonCode:        req.ReasonCode,
		ReasonText:        req.ReasonText,
		Scopes:            req.Scopes,
		StartsAt:          time.Now(),
		EndsAt:            req.EndsAt,
		Status:            "active",
		CreatedBy:         &actor.ID,
		RelatedEntityType: req.RelatedEntityType,
		RelatedEntityID:   req.RelatedEntityID,
		Meta:              req.Meta,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, item)
}

func (h *AdminHandler) ListSanctions(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid user id"}
	}
	items, err := h.usersSvc.ListSanctions(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *AdminHandler) RevokeSanction(c echo.Context) error {
	sanctionID, err := uuid.Parse(c.Param("sanctionId"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid sanction id"}
	}
	actor := c.Get(UserContextKey).(service.PublicUser)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)
	if err := h.usersSvc.RevokeSanction(c.Request().Context(), sanctionID, actor.ID, req.Reason); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) UpdateSanction(c echo.Context) error {
	sanctionID, err := uuid.Parse(c.Param("sanctionId"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid sanction id"}
	}
	var req struct {
		Scopes     []string       `json:"scopes"`
		EndsAt     *time.Time     `json:"ends_at"`
		ReasonText string         `json:"reason_text"`
		Meta       map[string]any `json:"meta"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.usersSvc.UpdateSanction(c.Request().Context(), sanctionID, req.Scopes, req.EndsAt, req.ReasonText, req.Meta); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) ExportMonumentsCSV(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=monuments.csv")
	return h.exportSvc.ExportMonumentsCSV(c.Request().Context(), c.Response().Writer)
}

func (h *AdminHandler) ExportMonumentsGeoJSON(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=monuments.geojson")
	return h.exportSvc.ExportMonumentsGeoJSON(c.Request().Context(), c.Response().Writer)
}

func (h *AdminHandler) ExportSignalsCSV(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=signals.csv")
	return h.exportSvc.ExportSignalsCSV(c.Request().Context(), c.Response().Writer)
}

func (h *AdminHandler) ListLogs(c echo.Context) error {
	filter := repo.ListAdminEventLogsFilter{}
	if actorID := c.QueryParam("actor_user_id"); actorID != "" {
		if parsed, err := uuid.Parse(actorID); err == nil {
			filter.ActorUserID = &parsed
		}
	}
	if targetID := c.QueryParam("target_user_id"); targetID != "" {
		if parsed, err := uuid.Parse(targetID); err == nil {
			filter.TargetUserID = &parsed
		}
	}
	if entityID := c.QueryParam("entity_id"); entityID != "" {
		if parsed, err := uuid.Parse(entityID); err == nil {
			filter.EntityID = &parsed
		}
	}
	filter.EntityType = c.QueryParam("entity_type")
	filter.Action = c.QueryParam("action")
	filter.Result = c.QueryParam("result")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 30
	}
	filter.Limit = limit

	items, err := h.usersSvc.ListAdminLogs(c.Request().Context(), filter)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *AdminHandler) DeleteMonument(c echo.Context) error {
	actor := c.Get(UserContextKey).(service.PublicUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid monument id"}
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if !req.Confirm {
		return apierr.Error{Code: "validation_failed", Message: "confirmation required", Fields: map[string]string{"confirm": "required"}}
	}
	if err := h.monSvc.DeleteMonumentAsAdmin(c.Request().Context(), actor.ID, id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) DeletePost(c echo.Context) error {
	actor := c.Get(UserContextKey).(service.PublicUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid post id"}
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if !req.Confirm {
		return apierr.Error{Code: "validation_failed", Message: "confirmation required", Fields: map[string]string{"confirm": "required"}}
	}
	if err := h.monSvc.DeletePostAsAdmin(c.Request().Context(), actor.ID, id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) DeleteSignal(c echo.Context) error {
	actor := c.Get(UserContextKey).(service.PublicUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid signal id"}
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if !req.Confirm {
		return apierr.Error{Code: "validation_failed", Message: "confirmation required", Fields: map[string]string{"confirm": "required"}}
	}
	if err := h.signalSvc.DeleteSignalAsAdmin(c.Request().Context(), actor.ID, id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
