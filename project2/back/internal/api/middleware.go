package api

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/config"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

const UserContextKey = "user"

func AuthMiddleware(authSvc *service.Auth) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(service.AccessCookieName)
			if err != nil {
				return next(c) // No cookie, proceed as guest
			}

			user, err := authSvc.Me(c.Request().Context(), cookie.Value)
			if err != nil {
				return next(c) // Invalid token, proceed as guest
			}

			c.Set(UserContextKey, user)
			return next(c)
		}
	}
}

func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(UserContextKey)
			if user == nil {
				return apierr.Error{Code: "unauthorized", Message: "authentication required"}
			}
			return next(c)
		}
	}
}

func RequireRole(rolesRepo *repo.Roles, allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			val := c.Get(UserContextKey)
			if val == nil {
				return apierr.Error{Code: "unauthorized", Message: "authentication required"}
			}

			user, ok := val.(service.PublicUser)
			if !ok {
				return apierr.Error{Code: "internal_error", Message: "failed to get user from context"}
			}

			roleName, err := rolesRepo.GetNameByID(c.Request().Context(), user.RoleID)
			if err != nil {
				return apierr.Error{Code: "internal_error", Message: "failed to get role"}
			}

			for _, r := range allowedRoles {
				if roleName == r {
					return next(c)
				}
			}

			return apierr.Error{Code: "forbidden", Message: "insufficient permissions"}
		}
	}
}

func RequestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogMethod:   true,
		LogRemoteIP: true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			c.Logger().Infof("%s %s status=%d ip=%s latency=%s", v.Method, v.URI, v.Status, v.RemoteIP, v.Latency)
			return nil
		},
	})
}

func BodyLimit(maxBytes string) echo.MiddlewareFunc {
	maxBytes = strings.TrimSpace(maxBytes)
	if maxBytes == "" {
		maxBytes = "10M"
	}
	return middleware.BodyLimit(maxBytes)
}

func CORS(cfg config.CORSConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			enabled := cfg.AddHeaders(c.Response(), c.Request())
			if c.Request().Method == http.MethodOptions && enabled {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}

func RateLimitGlobal(cfg config.RateLimitConfig) echo.MiddlewareFunc {
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(cfg.GlobalRPS),
		Burst:     int(cfg.GlobalRPS * 2),
		ExpiresIn: 2 * time.Minute,
	})

	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: store,
		IdentifierExtractor: func(c echo.Context) (string, error) {
			ip := c.RealIP()
			if ip == "" {
				ip = "unknown"
			}
			return ip, nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return apierr.Error{Code: "rate_limited", Message: "too many requests"}
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			return apierr.Error{Code: "rate_limited", Message: "too many requests"}
		},
	})
}

func RateLimitLogin(maxAttempts int, window time.Duration) echo.MiddlewareFunc {
	type rec struct {
		count int
		reset time.Time
	}

	var mu sync.Mutex
	byIP := map[string]rec{}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if ip == "" {
				ip = "unknown"
			}

			now := time.Now()
			mu.Lock()
			r := byIP[ip]
			if r.reset.IsZero() || now.After(r.reset) {
				r = rec{count: 0, reset: now.Add(window)}
			}
			r.count++
			byIP[ip] = r

			for k, v := range byIP {
				if now.After(v.reset.Add(window)) {
					delete(byIP, k)
				}
			}
			mu.Unlock()

			if r.count > maxAttempts {
				return apierr.Error{Code: "rate_limited", Message: "too many login attempts"}
			}
			return next(c)
		}
	}
}
