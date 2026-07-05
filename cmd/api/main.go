package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	authjwt "finish-line/internal/auth/adapters/jwt"
	authmiddleware "finish-line/internal/auth/adapters/middleware"
	authpostgres "finish-line/internal/auth/adapters/postgres"
	authrest "finish-line/internal/auth/adapters/rest"
	authservice "finish-line/internal/auth/service"
	"finish-line/internal/common/config"
	"finish-line/internal/common/email"
	"finish-line/internal/common/postgres"
	"finish-line/internal/common/ratelimit"
	"finish-line/internal/common/security"
	"finish-line/internal/common/server"
	participantnotification "finish-line/internal/participant/adapters/notification"
	participantpostgres "finish-line/internal/participant/adapters/postgres"
	participantrest "finish-line/internal/participant/adapters/rest"
	participantservice "finish-line/internal/participant/service"
	racepostgres "finish-line/internal/race/adapters/postgres"
	racerest "finish-line/internal/race/adapters/rest"
	raceservice "finish-line/internal/race/service"
	userpostgres "finish-line/internal/user/adapters/postgres"
	userrest "finish-line/internal/user/adapters/rest"
	userservice "finish-line/internal/user/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := newLogger(cfg.Env)
	slog.SetDefault(logger)

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := postgres.Connect(cfg.DB.DSN())
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	// Shared infrastructure used across modules.
	hasher := security.NewBcryptHasher()
	userRepo := userpostgres.NewRepository(db)

	// Auth module reuses the user repository (as a UserFinder) and the hasher
	// (as a PasswordVerifier) — the narrow interfaces it actually needs.
	authSvc := authservice.New(
		userRepo,
		hasher,
		authjwt.New(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL),
		authpostgres.NewRepository(db),
		cfg.Auth.RefreshTTL,
	)

	// The user service uses the auth service as its SessionRevoker so that a
	// password change can end every session. auth depends on the user repo,
	// the user service depends on auth — no cycle, since they are different
	// objects.
	userSvc := userservice.New(userRepo, hasher, authSvc)

	// AutoMigrate is a development convenience; production schema changes
	// must ship as explicit, reviewed migrations. Register each module's
	// migration here in dependency order (users before refresh_tokens).
	if !cfg.IsProduction() {
		if err := postgres.RunMigrations(db,
			userpostgres.Migrate,
			authpostgres.Migrate,
			racepostgres.Migrate,
			participantpostgres.Migrate,
		); err != nil {
			return fmt.Errorf("running dev migrations: %w", err)
		}
		if err := userSvc.EnsureAdmin(ctx, "Admin", "admin@finishline.dev", "admin.123"); err != nil {
			return fmt.Errorf("seeding admin user: %w", err)
		}
		logger.Info("dev migrations applied and admin ensured")
	}

	// Throttle login attempts per IP to blunt brute-force password guessing.
	loginLimiter := ratelimit.PerIP(rate.Every(time.Second), 5)

	raceSvc := raceservice.New(racepostgres.NewRepository(db))

	// Email: use Resend when configured, otherwise log messages so local dev
	// needs no API key.
	var emailSender email.Sender
	if cfg.Email.ResendAPIKey != "" {
		emailSender = email.NewResendSender(cfg.Email.ResendAPIKey, cfg.Email.From)
	} else {
		logger.Warn("RESEND_API_KEY not set — emails will be logged, not sent")
		emailSender = email.NewLogSender()
	}

	// Participant module reuses the race service (as a RaceFinder) and sends
	// confirmations through the notification adapter.
	participantSvc := participantservice.New(
		participantpostgres.NewParticipantRepository(db),
		participantpostgres.NewRegistrationRepository(db),
		raceSvc,
		participantnotification.NewConfirmationNotifier(emailSender),
	)

	authMW := authmiddleware.RequireAuth(authSvc)

	userModule := userrest.NewHandler(userSvc)
	authModule := authrest.NewHandler(authSvc, cfg.Auth.RefreshTTL, cfg.IsProduction(), loginLimiter)
	// The Strapi webhook authenticates with its own shared secret; the race
	// handler guards its own admin-only /races route with the auth middleware,
	// so the module stays in the public group.
	raceModule := racerest.NewHandler(raceSvc, cfg.Strapi.WebhookSecret, authMW)
	// Registration is public; the participant handler guards its own admin
	// report route with the auth middleware.
	participantModule := participantrest.NewHandler(participantSvc, authMW)

	srv := &http.Server{
		Addr: ":" + cfg.AppPort,
		Handler: server.New(logger, db, authMW, server.Modules{
			Public:    []server.RouteRegistrar{authModule, raceModule, participantModule},
			Protected: []server.RouteRegistrar{userModule},
		}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}

	return nil
}

func newLogger(env string) *slog.Logger {
	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
