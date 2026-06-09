package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type NotificationsHandler struct {
	auth *service.Auth
	svc  *service.NotificationsService
}

func NewNotificationsHandler(auth *service.Auth, svc *service.NotificationsService) *NotificationsHandler {
	return &NotificationsHandler{auth: auth, svc: svc}
}

func (h *NotificationsHandler) List(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, err := h.svc.ListLatest(c.Request().Context(), user.ID, limit)
	if err != nil {
		return err
	}
	unreadCount, err := h.svc.CountUnread(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{
		"items":        items,
		"unread_count": unreadCount,
	})
}

func (h *NotificationsHandler) UnreadCount(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	cnt, err := h.svc.CountUnread(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"count": cnt})
}

func (h *NotificationsHandler) MarkRead(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid notification id", Fields: map[string]string{"id": "invalid"}}
	}
	if err := h.svc.MarkRead(c.Request().Context(), user.ID, id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationsHandler) MarkAllRead(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	if err := h.svc.MarkAllRead(c.Request().Context(), user.ID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationsHandler) currentUser(c echo.Context) (service.PublicUser, error) {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || strings.TrimSpace(accessCookie.Value) == "" {
		return service.PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "missing access token"}
	}
	return h.auth.Me(c.Request().Context(), accessCookie.Value)
}
