package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type SearchHandler struct {
	search *repo.Search
	auth   *service.Auth
	roles  *repo.Roles
}

func NewSearchHandler(search *repo.Search, auth *service.Auth, roles *repo.Roles) *SearchHandler {
	return &SearchHandler{search: search, auth: auth, roles: roles}
}

func (h *SearchHandler) Suggest(c echo.Context) error {
	q := strings.TrimSpace(c.QueryParam("q"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	out, err := h.search.SuggestMonuments(c.Request().Context(), q, limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, out)
}

func (h *SearchHandler) Search(c echo.Context) error {
	q := strings.TrimSpace(c.QueryParam("q"))
	status := strings.TrimSpace(c.QueryParam("status"))
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	if q == "" {
		return apierr.Error{Code: "validation_failed", Message: "missing query", Fields: map[string]string{"q": "required"}}
	}

	if status != "" && status != "approved" {
		user, ok := h.optionalUser(c)
		if !ok {
			return apierr.Error{Code: "forbidden", Message: "forbidden"}
		}
		if err := h.requireModerator(c.Request().Context(), user.RoleID); err != nil {
			return err
		}
	}

	var authorID *uuid.UUID
	if v := strings.TrimSpace(c.QueryParam("author_id")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return apierr.Error{Code: "validation_failed", Message: "invalid author_id", Fields: map[string]string{"author_id": "invalid"}}
		}
		authorID = &id
	}
	city := strings.TrimSpace(c.QueryParam("city"))

	var dateFrom *time.Time
	if v := strings.TrimSpace(c.QueryParam("from")); v != "" {
		tm, err := parseTime(v)
		if err != nil {
			return apierr.Error{Code: "validation_failed", Message: "invalid from", Fields: map[string]string{"from": "invalid"}}
		}
		dateFrom = &tm
	}
	var dateTo *time.Time
	if v := strings.TrimSpace(c.QueryParam("to")); v != "" {
		tm, err := parseTime(v)
		if err != nil {
			return apierr.Error{Code: "validation_failed", Message: "invalid to", Fields: map[string]string{"to": "invalid"}}
		}
		dateTo = &tm
	}

	items, err := h.search.SearchMonuments(c.Request().Context(), repo.MonumentSearchFilter{
		Query:    q,
		Status:   status,
		AuthorID: authorID,
		City:     city,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Limit:    limit,
		Offset:   offset,
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

func parseTime(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}

func (h *SearchHandler) optionalUser(c echo.Context) (service.PublicUser, bool) {
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

func (h *SearchHandler) requireModerator(ctx context.Context, roleID uuid.UUID) error {
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
