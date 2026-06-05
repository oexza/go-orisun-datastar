package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/auth"
	"github.com/oexza/go-orisun-datastar/internal/config"
	"github.com/oexza/go-orisun-datastar/internal/email"
	"github.com/oexza/go-orisun-datastar/internal/eventcatalog"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/features/profile"
	"github.com/oexza/go-orisun-datastar/internal/features/todo"
	"github.com/oexza/go-orisun-datastar/internal/httpui"
	"github.com/oexza/go-orisun-datastar/internal/natsbus"
	"github.com/oexza/go-orisun-datastar/internal/storage"
	"github.com/oexza/go-orisun-datastar/internal/viewstore"
)

type runOptions struct {
	migrateOnly bool
	seedOnly    bool
}

type appComponents struct {
	authService      *auth.Service
	todoReadModel    *todo.ReadModel
	profileReadModel *profile.ReadModel
	checkpointer     eventstore.Checkpointer
	emailSender      email.Sender
	profileStorage   profile.ObjectStore
}

type eventHandler interface {
	StartSubscribing(context.Context) error
	StopSubscribing()
}

type eventHandlerFactory struct {
	name   string
	create func() (eventHandler, error)
}

func main() {
	opts := parseOptions()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(ctx, stop, config.Load(), opts, logger); err != nil {
		logger.Error("run app", "err", err)
		os.Exit(1)
	}
}

func parseOptions() runOptions {
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	seedOnly := flag.Bool("seed-only", false, "run seed tasks and exit")
	flag.Parse()
	return runOptions{migrateOnly: *migrateOnly, seedOnly: *seedOnly}
}

func run(ctx context.Context, stop context.CancelFunc, cfg config.Config, opts runOptions, logger *slog.Logger) error {
	db, err := appdb.Open(ctx, cfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer closeDB(db, logger)

	if opts.migrateOnly {
		logger.Info("sqlite migrations complete", "path", cfg.SQLitePath)
		return nil
	}
	if opts.seedOnly {
		logger.Info("seed requested; no seed tasks are currently defined")
		return nil
	}

	orisunStore, err := startEventStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer orisunStore.Close(context.Background())

	bus, err := natsbus.FromConn(orisunStore.NATSConnection())
	if err != nil {
		return fmt.Errorf("borrow embedded orisun nats: %w", err)
	}
	defer bus.Close()

	viewStore := newViewStore(bus, logger)
	components := newAppComponents(db, orisunStore, cfg, logger)
	handlers, err := startEventHandlers(ctx, eventHandlerFactories(orisunStore, bus, cfg, components, logger))
	if err != nil {
		return err
	}
	defer stopEventHandlers(handlers)

	app := httpui.Server{
		Auth:           components.authService,
		Todos:          components.todoReadModel,
		EventSaver:     orisunStore,
		EventRetriever: orisunStore,
		ProfileStorage: components.profileStorage,
		Subscriber:     bus,
		ViewStore:      viewStore,
		Development:    cfg.DevelopmentCookie,
	}
	return serveHTTP(ctx, stop, cfg.Port, app.Routes(), logger)
}

func closeDB(db *appdb.DB, logger *slog.Logger) {
	if err := db.Close(); err != nil {
		logger.Error("close sqlite", "err", err)
	}
}

func startEventStore(ctx context.Context, cfg config.Config) (*eventstore.EmbeddedOrisun, error) {
	store, err := eventstore.StartEmbeddedOrisun(ctx, eventstore.EmbeddedConfig{
		Boundary:     cfg.OrisunBoundary,
		SQLiteDir:    cfg.OrisunSQLiteDir,
		NATSStoreDir: eventstore.PollingStoreDir(),
		LogLevel:     "info",
	})
	if err != nil {
		return nil, fmt.Errorf("start embedded orisun: %w", err)
	}
	if err := store.EnsureBoundaryIndexes(ctx, eventcatalog.BoundaryIndexes()); err != nil {
		store.Close(context.Background())
		return nil, fmt.Errorf("ensure orisun indexes: %w", err)
	}
	return store, nil
}

func newViewStore(bus *natsbus.Bus, logger *slog.Logger) viewstore.Store {
	store, err := viewstore.NewNATSStore(bus.Conn(), "go-starter-view-state", 5*time.Minute)
	if err != nil {
		logger.Warn("using in-memory view store fallback", "err", err)
		return viewstore.NewMemoryStore()
	}
	return store
}

func newAppComponents(db *appdb.DB, store *eventstore.EmbeddedOrisun, cfg config.Config, logger *slog.Logger) appComponents {
	authService := auth.NewService(db, store, store, !cfg.DevelopmentCookie)
	todoReadModel := todo.NewReadModel(db)
	profileReadModel := profile.NewReadModel(db)
	profileStorage := storage.NewLocalProvider(cfg.UploadDir, cfg.UploadBaseURL)

	return appComponents{
		authService:      authService,
		todoReadModel:    todoReadModel,
		profileReadModel: profileReadModel,
		checkpointer:     eventstore.NewSQLiteCheckpointer(db),
		emailSender:      email.LogSender{Logger: logger},
		profileStorage:   profileStorage,
	}
}

func eventHandlerFactories(store *eventstore.EmbeddedOrisun, bus *natsbus.Bus, cfg config.Config, components appComponents, logger *slog.Logger) []eventHandlerFactory {
	return []eventHandlerFactory{
		{
			name: "registration OTP",
			create: func() (eventHandler, error) {
				return auth.NewRegistrationOTPToBeGeneratedEventHandler(store, components.checkpointer, components.authService, logger)
			},
		},
		{
			name: "email validation OTP",
			create: func() (eventHandler, error) {
				return auth.NewEmailValidationOTPToBeSentEventHandler(store, components.checkpointer, store, store, components.emailSender, logger)
			},
		},
		{
			name: "password reset email",
			create: func() (eventHandler, error) {
				return auth.NewPasswordResetEmailToBeSentEventHandler(store, components.checkpointer, store, store, components.emailSender, cfg.AppURL, logger)
			},
		},
		{
			name: "auth user projection",
			create: func() (eventHandler, error) {
				return auth.NewAuthUserProjectionEventHandler(store, components.checkpointer, store, components.authService, logger)
			},
		},
		{
			name: "profile read model",
			create: func() (eventHandler, error) {
				return profile.NewReadModelEventHandler(store, components.checkpointer, components.profileReadModel, logger)
			},
		},
		{
			name: "profile image auth user bridge",
			create: func() (eventHandler, error) {
				return profile.NewProfileImageUploadedAuthUserEventHandler(store, components.checkpointer, components.authService, logger)
			},
		},
		{
			name: "todo read model",
			create: func() (eventHandler, error) {
				return todo.NewTodoReadModelEventHandler(store, components.checkpointer, components.todoReadModel, bus, logger)
			},
		},
	}
}

func startEventHandlers(ctx context.Context, factories []eventHandlerFactory) ([]eventHandler, error) {
	handlers := make([]eventHandler, 0, len(factories))
	for _, factory := range factories {
		handler, err := factory.create()
		if err != nil {
			stopEventHandlers(handlers)
			return nil, fmt.Errorf("create %s event handler: %w", factory.name, err)
		}
		if err := handler.StartSubscribing(ctx); err != nil {
			stopEventHandlers(append(handlers, handler))
			return nil, fmt.Errorf("start %s event handler: %w", factory.name, err)
		}
		handlers = append(handlers, handler)
	}
	return handlers, nil
}

func stopEventHandlers(handlers []eventHandler) {
	for i := len(handlers) - 1; i >= 0; i-- {
		handlers[i].StopSubscribing()
	}
}

func serveHTTP(ctx context.Context, stop context.CancelFunc, port string, handler http.Handler, logger *slog.Logger) error {
	server := &http.Server{Addr: ":" + port, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("starting server", "addr", "http://localhost:"+port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	return nil
}
