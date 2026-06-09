package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
)

func ErrorHandler(err error, c echo.Context) {
	var apiErr apierr.Error
	if errors.As(err, &apiErr) {
		_ = c.JSON(apierr.Status(apiErr.Code), apiErr)
		return
	}

	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		_ = c.JSON(httpErr.Code, apierr.Error{Code: "http_error", Message: "request failed"})
		return
	}

	_ = c.JSON(http.StatusInternalServerError, apierr.Error{Code: "internal_error", Message: "internal error"})
}
