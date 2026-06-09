package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type MeHandler struct {
	auth *service.Auth
	mon  *service.MonumentsService
	sig  *service.SignalsService
}

func NewMeHandler(auth *service.Auth, mon *service.MonumentsService, sig *service.SignalsService) *MeHandler {
	return &MeHandler{auth: auth, mon: mon, sig: sig}
}

func (h *MeHandler) Monuments(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	items, err := h.mon.ListMonumentsByAuthor(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *MeHandler) Posts(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	items, err := h.mon.ListPostsByAuthor(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *MeHandler) Signals(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	items, err := h.sig.ListByAuthor(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *MeHandler) Comments(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	items, err := h.sig.ListCommentsByAuthor(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *MeHandler) currentUser(c echo.Context) (service.PublicUser, error) {
	val := c.Get(UserContextKey)
	if val == nil {
		return service.PublicUser{}, apierr.Error{Code: "unauthorized", Message: "unauthorized"}
	}
	return val.(service.PublicUser), nil
}
