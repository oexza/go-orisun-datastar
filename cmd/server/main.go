package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/hono-event-starter-go/internal/appdb"
	"github.com/example/hono-event-starter-go/internal/auth"
	"github.com/example/hono-event-starter-go/internal/config"
	"github.com/example/hono-event-starter-go/internal/email"
	"github.com/example/hono-event-starter-go/internal/eventcatalog"
	"github.com/example/hono-event-starter-go/internal/eventstore"
	"github.com/example/hono-event-starter-go/internal/features/profile"
	"github.com/example/hono-event-starter-go/internal/features/todo"
	"github.com/example/hono-event-starter-go/internal/httpui"
	"github.com/example/hono-event-starter-go/internal/natsbus"
	"github.com/example/hono-event-starter-go/internal/storage"
	"github.com/example/hono-event-starter-go/internal/viewstore"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	seedOnly := flag.Bool("seed-only", false, "run seed tasks and exit")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := appdb.Open(ctx, cfg.SQLitePath)
	if err != nil {
		logger.Error("open sqlite", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("close sqlite", "err", err)
		}
	}()

	if *migrateOnly {
		logger.Info("sqlite migrations complete", "path", cfg.SQLitePath)
		return
	}
	if *seedOnly {
		logger.Info("seed requested; no seed tasks are currently defined")
		return
	}

	orisunStore, err := eventstore.StartEmbeddedOrisun(ctx, eventstore.EmbeddedConfig{
		Boundary:     cfg.OrisunBoundary,
		SQLiteDir:    cfg.OrisunSQLiteDir,
		NATSStoreDir: eventstore.PollingStoreDir(),
		LogLevel:     "info",
	})
	if err != nil {
		logger.Error("start embedded orisun", "err", err)
		os.Exit(1)
	}
	defer orisunStore.Close(context.Background())
	if err := orisunStore.EnsureBoundaryIndexes(ctx, eventcatalog.BoundaryIndexes()); err != nil {
		logger.Error("ensure orisun indexes", "err", err)
		os.Exit(1)
	}

	bus, err := natsbus.Connect(cfg.NATSURL)
	if err != nil {
		logger.Error("connect nats", "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	var viewStore viewstore.Store
	viewStore, err = viewstore.NewNATSStore(bus.Conn(), "go-starter-view-state", 5*time.Minute)
	if err != nil {
		logger.Warn("using in-memory view store fallback", "err", err)
		viewStore = viewstore.NewMemoryStore()
	}

	storageProvider := storage.NewLocalProvider(cfg.UploadDir, cfg.UploadBaseURL)
	emailSender := email.LogSender{Logger: logger}
	authService := auth.NewService(db, orisunStore, orisunStore, !cfg.DevelopmentCookie)
	todoReadModel := todo.NewReadModel(db)
	todoService := todo.NewService(todoReadModel, orisunStore, orisunStore, bus)
	profileReadModel := profile.NewReadModel(db)
	profileService := profile.NewService(orisunStore, storageProvider)

	checkpointer := eventstore.NewSQLiteCheckpointer(db)
	registrationOTPEventHandler, err := auth.NewRegistrationOTPToBeGeneratedEventHandler(orisunStore, checkpointer, authService, logger)
	if err != nil {
		logger.Error("create registration OTP event handler", "err", err)
		os.Exit(1)
	}
	if err := registrationOTPEventHandler.StartSubscribing(ctx); err != nil {
		logger.Error("start registration OTP event handler", "err", err)
		os.Exit(1)
	}
	defer registrationOTPEventHandler.StopSubscribing()

	emailValidationOTPEventHandler, err := auth.NewEmailValidationOTPToBeSentEventHandler(orisunStore, checkpointer, orisunStore, orisunStore, emailSender, logger)
	if err != nil {
		logger.Error("create email validation OTP event handler", "err", err)
		os.Exit(1)
	}
	if err := emailValidationOTPEventHandler.StartSubscribing(ctx); err != nil {
		logger.Error("start email validation OTP event handler", "err", err)
		os.Exit(1)
	}
	defer emailValidationOTPEventHandler.StopSubscribing()

	passwordResetEmailEventHandler, err := auth.NewPasswordResetEmailToBeSentEventHandler(orisunStore, checkpointer, orisunStore, orisunStore, emailSender, cfg.AppURL, logger)
	if err != nil {
		logger.Error("create password reset email event handler", "err", err)
		os.Exit(1)
	}
	if err := passwordResetEmailEventHandler.StartSubscribing(ctx); err != nil {
		logger.Error("start password reset email event handler", "err", err)
		os.Exit(1)
	}
	defer passwordResetEmailEventHandler.StopSubscribing()

	authUserProjectionEventHandler, err := auth.NewAuthUserProjectionEventHandler(orisunStore, checkpointer, orisunStore, authService, logger)
	if err != nil {
		logger.Error("create auth user projection event handler", "err", err)
		os.Exit(1)
	}
	if err := authUserProjectionEventHandler.StartSubscribing(ctx); err != nil {
		logger.Error("start auth user projection event handler", "err", err)
		os.Exit(1)
	}
	defer authUserProjectionEventHandler.StopSubscribing()

	profileReadModelEventHandler, err := profile.NewReadModelEventHandler(orisunStore, checkpointer, profileReadModel, logger)
	if err != nil {
		logger.Error("create profile read model event handler", "err", err)
		os.Exit(1)
	}
	if err := profileReadModelEventHandler.StartSubscribing(ctx); err != nil {
		logger.Error("start profile read model event handler", "err", err)
		os.Exit(1)
	}
	defer profileReadModelEventHandler.StopSubscribing()

	profileImageUploadedAuthUserEventHandler, err := profile.NewProfileImageUploadedAuthUserEventHandler(orisunStore, checkpointer, authService, logger)
	if err != nil {
		logger.Error("create profile image uploaded auth user event handler", "err", err)
		os.Exit(1)
	}
	if err := profileImageUploadedAuthUserEventHandler.StartSubscribing(ctx); err != nil {
		logger.Error("start profile image uploaded auth user event handler", "err", err)
		os.Exit(1)
	}
	defer profileImageUploadedAuthUserEventHandler.StopSubscribing()

	todoReadModelEventHandler, err := todo.NewTodoReadModelEventHandler(orisunStore, checkpointer, todoReadModel, bus, logger)
	if err != nil {
		logger.Error("create todo read model event handler", "err", err)
		os.Exit(1)
	}
	if err := todoReadModelEventHandler.StartSubscribing(ctx); err != nil {
		logger.Error("start todo read model event handler", "err", err)
		os.Exit(1)
	}
	defer todoReadModelEventHandler.StopSubscribing()

	app := httpui.Server{Auth: authService, Todos: todoService, Profile: profileService, Subscriber: bus, ViewStore: viewStore, Development: cfg.DevelopmentCookie}
	server := &http.Server{Addr: ":" + cfg.Port, Handler: app.Routes(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("starting server", "addr", "http://localhost:"+cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
