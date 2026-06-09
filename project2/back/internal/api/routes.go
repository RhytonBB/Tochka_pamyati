package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/config"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type Deps struct {
	Config        config.Config
	Pool          *pgxpool.Pool
	Auth          *service.Auth
	Roles         *repo.Roles
	Monuments     *service.MonumentsService
	Signals       *service.SignalsService
	Geography     *service.GeographyService
	MapTiles      *repo.MapTiles
	Search        *repo.Search
	Notifications *service.NotificationsService
	Reports       *service.ReportsService
	Analytics     *service.AnalyticsService
	Users         *service.UsersService
	Export        *service.ExportService
}

func RegisterRoutes(e *echo.Echo, deps Deps) {
	e.GET("/health", func(c echo.Context) error {
		if err := deps.Pool.Ping(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "db_down"})
		}
		return c.JSON(http.StatusOK, map[string]any{"status": "ok"})
	})

	apiV1 := e.Group("/api/v1")
	apiV1.Use(AuthMiddleware(deps.Auth))

	adminHandler := NewAdminHandler(deps.Analytics, deps.Users, deps.Monuments, deps.Signals, deps.Export)

	authHandler := NewAuthHandler(deps.Auth)
	auth := apiV1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/verify-email", authHandler.VerifyEmail)
	auth.POST("/resend-code", authHandler.ResendCode)
	auth.POST("/password/forgot", authHandler.RequestPasswordReset)
	auth.POST("/password/reset", authHandler.ResetPassword)
	auth.POST("/password/change", authHandler.ChangePassword)
	auth.POST("/login", authHandler.Login, RateLimitLogin(5, 15*time.Minute))
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)

	apiV1.GET("/me", authHandler.Me)
	apiV1.PUT("/profile", authHandler.UpdateProfile)

	monumentsHandler := NewMonumentsHandler(deps.Auth, deps.Monuments)
	apiV1.GET("/monuments/:id", monumentsHandler.GetMonumentDetail)
	apiV1.GET("/monuments/:id/summary", monumentsHandler.GetMonumentSummary)
	apiV1.POST("/monuments/validate", monumentsHandler.ValidateMonument)
	apiV1.POST("/monuments", monumentsHandler.CreateMonument)
	apiV1.POST("/monuments/:id/validate", monumentsHandler.ValidateMonumentEdit)
	apiV1.PUT("/monuments/:id", monumentsHandler.UpdateMonument)
	apiV1.DELETE("/monuments/:id", monumentsHandler.DeleteMonument)
	apiV1.POST("/monuments/:id/posts/validate", monumentsHandler.ValidatePost)
	apiV1.POST("/monuments/:id/posts", monumentsHandler.AddPost)
	apiV1.PUT("/monuments/:id/posts/:postId", monumentsHandler.UpdatePostSubmission)
	apiV1.POST("/posts/:id/validate", monumentsHandler.ValidatePostEdit)
	apiV1.PUT("/posts/:id", monumentsHandler.UpdatePost)
	apiV1.DELETE("/posts/:id", monumentsHandler.DeletePost)
	apiV1.POST("/posts/:id/restore", monumentsHandler.RestoreArchivedPost)

	mapHandler := NewMapHandler(deps.MapTiles, deps.Signals)
	apiV1.GET("/tiles/monuments/:z/:x/:y", mapHandler.MonumentTile)

	searchHandler := NewSearchHandler(deps.Search, deps.Auth, deps.Roles)
	apiV1.GET("/search/suggest", searchHandler.Suggest)
	apiV1.GET("/search", searchHandler.Search)

	regionsHandler := NewRegionsHandler(deps.Geography)
	apiV1.GET("/regions", regionsHandler.List)

	signalsHandler := NewSignalsHandler(deps.Auth, deps.Roles, deps.Signals)
	apiV1.GET("/signals", signalsHandler.List)
	apiV1.POST("/signals/validate", signalsHandler.Validate)
	apiV1.POST("/signals", signalsHandler.Create)
	apiV1.GET("/signals/:id", signalsHandler.GetDetail)
	apiV1.PUT("/signals/:id", signalsHandler.UpdateOwn)
	apiV1.DELETE("/signals/:id", signalsHandler.DeleteOwn)
	apiV1.POST("/signals/:id/status", signalsHandler.SetOwnResolved)
	apiV1.POST("/signals/:id/comments", signalsHandler.AddComment)
	apiV1.PUT("/signals/:id/comments/:commentId", signalsHandler.EditComment)
	apiV1.DELETE("/signals/:id/comments/:commentId", signalsHandler.DeleteComment)
	apiV1.POST("/signals/:id/support", signalsHandler.Support)
	apiV1.DELETE("/signals/:id/support", signalsHandler.Unsupport)
	apiV1.GET("/signals/map/confirmed", signalsHandler.ConfirmedMapPoints)
	apiV1.GET("/stats", adminHandler.GetGlobalStats)

	mod := apiV1.Group("/moderation")
	mod.Use(RequireRole(deps.Roles, "moderator", "admin"))

	modHandler := NewModerationHandler(deps.Monuments, deps.Signals, deps.Reports)
	mod.GET("/monuments", modHandler.ListMonuments)
	mod.POST("/monuments/:id/action", modHandler.ModerateMonument)
	mod.GET("/posts", modHandler.ListPosts)
	mod.GET("/posts/:id", modHandler.GetPostDetail)
	mod.POST("/posts/:id/action", modHandler.ModeratePost)
	mod.DELETE("/photos/:id", modHandler.DeletePhoto)
	mod.DELETE("/comments/:id", modHandler.DeleteComment)
	mod.GET("/edits", modHandler.ListEdits)
	mod.GET("/edits/:id", modHandler.GetEditDetail)
	mod.POST("/edits/:id/action", modHandler.ModerateEdit)
	mod.GET("/signals", modHandler.ListSignals)
	mod.POST("/signals/:id/action", modHandler.ModerateSignal)
	mod.GET("/reports", modHandler.ListReports)
	mod.GET("/reports/:id", modHandler.GetReportCase)
	mod.POST("/reports/:id/action", modHandler.ModerateReport)
	mod.POST("/undo", modHandler.UndoModeration)

	mod.GET("/stats", adminHandler.GetGlobalStats)

	admin := apiV1.Group("/admin")
	admin.Use(RequireRole(deps.Roles, "admin"))

	admin.GET("/stats", adminHandler.GetGlobalStats)
	admin.GET("/dynamics", adminHandler.GetDynamics)
	admin.GET("/top-authors", adminHandler.GetTopAuthors)
	admin.GET("/users", adminHandler.ListUsers)
	admin.GET("/users/:id", adminHandler.GetUser)
	admin.POST("/users/:id/role", adminHandler.UpdateUserRole)
	admin.POST("/users/:id/block", adminHandler.SetUserBlocked)
	admin.GET("/users/:id/sanctions", adminHandler.ListSanctions)
	admin.POST("/users/:id/sanctions", adminHandler.CreateSanction)
	admin.POST("/users/:id/sanctions/:sanctionId/revoke", adminHandler.RevokeSanction)
	admin.POST("/users/:id/sanctions/:sanctionId/update", adminHandler.UpdateSanction)
	admin.DELETE("/users/:id", adminHandler.DeleteUser)
	admin.GET("/logs", adminHandler.ListLogs)
	admin.POST("/monuments/:id/delete", adminHandler.DeleteMonument)
	admin.POST("/posts/:id/delete", adminHandler.DeletePost)
	admin.POST("/signals/:id/delete", adminHandler.DeleteSignal)
	admin.GET("/export/monuments/csv", adminHandler.ExportMonumentsCSV)
	admin.GET("/export/monuments/geojson", adminHandler.ExportMonumentsGeoJSON)
	admin.GET("/export/signals/csv", adminHandler.ExportSignalsCSV)

	reportsHandler := NewReportsHandler(deps.Auth, deps.Roles, deps.Reports)
	apiV1.POST("/reports", reportsHandler.Create)
	apiV1.GET("/reports/my", reportsHandler.MyReports)

	notificationsHandler := NewNotificationsHandler(deps.Auth, deps.Notifications)
	apiV1.GET("/notifications", notificationsHandler.List)
	apiV1.GET("/notifications/unread-count", notificationsHandler.UnreadCount)
	apiV1.POST("/notifications/:id/read", notificationsHandler.MarkRead)
	apiV1.POST("/notifications/read-all", notificationsHandler.MarkAllRead)

	meHandler := NewMeHandler(deps.Auth, deps.Monuments, deps.Signals)
	apiV1.GET("/me/monuments", meHandler.Monuments)
	apiV1.GET("/me/posts", meHandler.Posts)
	apiV1.GET("/me/signals", meHandler.Signals)
	apiV1.GET("/me/comments", meHandler.Comments)
}
