package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type RegionsHandler struct {
	geo *service.GeographyService
}

func NewRegionsHandler(geo *service.GeographyService) *RegionsHandler {
	return &RegionsHandler{geo: geo}
}

func (h *RegionsHandler) List(c echo.Context) error {
	items := []string{}
	if h.geo != nil {
		items = h.geo.GetAllRegions()
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}
