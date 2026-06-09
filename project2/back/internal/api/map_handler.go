package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type MapHandler struct {
	tiles   *repo.MapTiles
	signals *service.SignalsService
}

func NewMapHandler(tiles *repo.MapTiles, signals *service.SignalsService) *MapHandler {
	return &MapHandler{tiles: tiles, signals: signals}
}

func (h *MapHandler) MonumentTile(c echo.Context) error {
	z, err := strconv.Atoi(c.Param("z"))
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	x, err := strconv.Atoi(c.Param("x"))
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	yParam := c.Param("y")
	yParam = strings.TrimSuffix(yParam, ".mvt")
	yParam = strings.TrimSuffix(yParam, ".pbf")
	y, err := strconv.Atoi(yParam)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	tile, err := h.tiles.MonumentTile(c.Request().Context(), z, x, y)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Type", "application/x-protobuf")
	c.Response().Header().Set("Content-Encoding", "identity")
	return c.Blob(http.StatusOK, "application/x-protobuf", tile)
}
