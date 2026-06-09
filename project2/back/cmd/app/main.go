package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/tochka-pamyati/tochka-pamyati/internal/api"
	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/config"
	"github.com/tochka-pamyati/tochka-pamyati/internal/db"
	"github.com/tochka-pamyati/tochka-pamyati/internal/mailer"
	"github.com/tochka-pamyati/tochka-pamyati/internal/moderation"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
	"github.com/tochka-pamyati/tochka-pamyati/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := autoMigrate(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	mailQueue := mailer.NewQueue(cfg.EmailWorkerCount)
	defer mailQueue.Close()

	var mailSender mailer.Sender = mailer.NoopSender{}
	if cfg.EmailEnabled {
		mailSender = mailer.NewSMTP(cfg.SMTP)
	}

	userRepo := repo.NewUsers(pool)
	roleRepo := repo.NewRoles(pool)
	verificationRepo := repo.NewEmailVerifications(pool)
	sessionRepo := repo.NewSessions(pool)
	monumentRepo := repo.NewMonuments(pool)
	postRepo := repo.NewPosts(pool)
	photoRepo := repo.NewPhotos(pool)
	auditRepo := repo.NewAuditLog(pool)
	mapTilesRepo := repo.NewMapTiles(pool)
	searchRepo := repo.NewSearch(pool)
	signalsRepo := repo.NewSignals(pool)
	signalPhotosRepo := repo.NewSignalPhotos(pool)
	signalCommentsRepo := repo.NewSignalComments(pool)
	reportsRepo := repo.NewReports(pool)
	notificationsRepo := repo.NewNotifications(pool)
	userSanctionsRepo := repo.NewUserSanctions(pool)
	commentIncidentsRepo := repo.NewCommentAIIncidents(pool)
	trustEventsRepo := repo.NewTrustEvents(pool)
	adminEventLogsRepo := repo.NewAdminEventLogs(pool)

	geoSvc, err := service.NewGeographyService(cfg.RegionsGeoJSONPath)
	if err != nil {
		log.Fatalf("regions: %v", err)
	}

	trustSvc := service.NewTrustService(service.TrustDeps{
		Users:         userRepo,
		Events:        trustEventsRepo,
		Notifications: notificationsRepo,
		Monuments:     monumentRepo,
		Posts:         postRepo,
		Signals:       signalsRepo,
		Comments:      signalCommentsRepo,
	})

	sanctionsSvc := service.NewSanctionsService(service.SanctionsDeps{
		Sanctions:     userSanctionsRepo,
		Incidents:     commentIncidentsRepo,
		Sessions:      sessionRepo,
		Users:         userRepo,
		Trust:         trustSvc,
		Notifications: notificationsRepo,
	})

	authSvc := service.NewAuth(service.AuthDeps{
		Config:             cfg.Auth,
		Users:              userRepo,
		Roles:              roleRepo,
		EmailVerifications: verificationRepo,
		Sessions:           sessionRepo,
		AdminLogs:          adminEventLogsRepo,
		Sanctions:          sanctionsSvc,
		Trust:              trustSvc,
		Mailer:             mailSender,
		MailQueue:          mailQueue,
	})

	textChecker := moderation.NewHTTPTextChecker(cfg.TextFilterURL)
	imageChecker := moderation.NewHTTPImageChecker(cfg.ImageFilterURL)
	uploader := storage.NewLocalFS(cfg.UploadsDir)

	monumentsSvc := service.NewMonumentsService(service.MonumentsDeps{
		Monuments:     monumentRepo,
		Posts:         postRepo,
		Photos:        photoRepo,
		Audit:         auditRepo,
		Signals:       signalsRepo,
		Notifications: notificationsRepo,
		AdminLogs:     adminEventLogsRepo,
		Sanctions:     sanctionsSvc,
		Trust:         trustSvc,
		Geography:     geoSvc,
		TextChecker:   textChecker,
		ImageChecker:  imageChecker,
		Uploader:      uploader,
	})

	signalsSvc := service.NewSignalsService(service.SignalsDeps{
		Signals:       signalsRepo,
		SignalPhotos:  signalPhotosRepo,
		Comments:      signalCommentsRepo,
		Monuments:     monumentRepo,
		Audit:         auditRepo,
		Notifications: notificationsRepo,
		AdminLogs:     adminEventLogsRepo,
		Users:         userRepo,
		Sanctions:     sanctionsSvc,
		Trust:         trustSvc,
		Geography:     geoSvc,
		TextChecker:   textChecker,
		ImageChecker:  imageChecker,
		Uploader:      uploader,
	})

	notificationsSvc := service.NewNotificationsService(notificationsRepo)
	reportsSvc := service.NewReportsService(service.ReportsDeps{
		Reports:       reportsRepo,
		Notifications: notificationsRepo,
		Audit:         auditRepo,
		AdminLogs:     adminEventLogsRepo,
		Users:         userRepo,
		Trust:         trustSvc,
		Monuments:     monumentRepo,
		Posts:         postRepo,
		Photos:        photoRepo,
		SignalPhotos:  signalPhotosRepo,
		Signals:       signalsRepo,
		Comments:      signalCommentsRepo,
		MonumentSvc:   monumentsSvc,
	})

	analyticsSvc := service.NewAnalyticsService(monumentRepo, postRepo, userRepo, signalsRepo, reportsRepo)
	usersSvc := service.NewUsersService(service.UsersDeps{
		Users:     userRepo,
		Roles:     roleRepo,
		Sanctions: sanctionsSvc,
		Trust:     trustSvc,
		Incidents: commentIncidentsRepo,
		Sessions:  sessionRepo,
		Monuments: monumentRepo,
		Posts:     postRepo,
		Signals:   signalsRepo,
		AdminLogs: adminEventLogsRepo,
	})
	exportSvc := service.NewExportService(monumentRepo, signalsRepo)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = api.ErrorHandler

	e.Use(middleware.Recover())
	e.Use(api.RequestLogger())
	e.Use(api.BodyLimit(cfg.MaxBodyBytes))
	e.Use(api.CORS(cfg.CORS))
	e.Use(api.RateLimitGlobal(cfg.RateLimit))

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		code := http.StatusInternalServerError
		var resp any = map[string]string{"message": "Внутренняя ошибка сервера"}

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			resp = he.Message
		} else if ae, ok := err.(apierr.Error); ok {
			code = apierr.Status(ae.Code)
			resp = ae
		} else {
			// Логируем системные ошибки, которые мы скрываем от пользователя
			log.Printf("Internal Server Error: %v", err)
		}

		if err := c.JSON(code, resp); err != nil {
			e.Logger.Error(err)
		}
	}

	// Serve uploaded files statically
	e.Static("/uploads", cfg.UploadsDir)
	e.Static("/thumbnails", filepath.Join(cfg.UploadsDir, "thumbnails"))
	e.Static("/previews", filepath.Join(cfg.UploadsDir, "previews"))
	e.Static("/originals", filepath.Join(cfg.UploadsDir, "originals"))
	e.Static("/signal_thumbnails", filepath.Join(cfg.UploadsDir, "signal_thumbnails"))
	e.Static("/signal_previews", filepath.Join(cfg.UploadsDir, "signal_previews"))
	e.Static("/signal_originals", filepath.Join(cfg.UploadsDir, "signal_originals"))

	api.RegisterRoutes(e, api.Deps{
		Config:        cfg,
		Pool:          pool,
		Auth:          authSvc,
		Roles:         roleRepo,
		Monuments:     monumentsSvc,
		Signals:       signalsSvc,
		Geography:     geoSvc,
		MapTiles:      mapTilesRepo,
		Search:        searchRepo,
		Notifications: notificationsSvc,
		Reports:       reportsSvc,
		Analytics:     analyticsSvc,
		Users:         usersSvc,
		Export:        exportSvc,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           e,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("Server started on http://localhost%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func autoMigrate(databaseURL string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	migrationsDir := filepath.Join(cwd, "migrations")
	if _, err := os.Stat(migrationsDir); err != nil {
		alt2 := filepath.Join(cwd, "back", "migrations")
		if _, alt2Err := os.Stat(alt2); alt2Err == nil {
			migrationsDir = alt2
		} else {
			exe, exeErr := os.Executable()
			if exeErr == nil {
				alt := filepath.Join(filepath.Dir(exe), "migrations")
				if _, altErr := os.Stat(alt); altErr == nil {
					migrationsDir = alt
				}
			}
		}
	}
	migrationsDir = filepath.ToSlash(migrationsDir)

	// Replace postgresql:// with pgx5:// for golang-migrate
	dbURL := strings.Replace(databaseURL, "postgresql://", "pgx5://", 1)
	dbURL = strings.Replace(dbURL, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://"+migrationsDir, dbURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
