package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"pr-reviewer/internal/adapters/db"
	"pr-reviewer/internal/adapters/rest"
	"pr-reviewer/internal/closers"
	"pr-reviewer/internal/config"
	"pr-reviewer/internal/core"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.MustLoad(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := mustSetupLogger(cfg.LogLevel)

	if err := run(cfg, logger); err != nil {
		logger.Error("service failed", "error", err)
		os.Exit(1)
	}

}

func run(cfg *config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug message are enabled")

	// database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}

	defer closers.CloseOrLog(log, storage)

	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %v", err)
	}

	service := core.NewService(storage.Team, storage.User, storage.PR)

	mux := http.NewServeMux()
	mux.Handle("POST /team/add", rest.CreateTeamHandler(log, service))
	mux.Handle("GET /team/get", rest.GetTeamHandler(log, service))
	mux.Handle("POST /users/setIsActive", rest.SetUserActiveHandler(log, service))
	mux.Handle("POST /pullRequest/create", rest.CreatePRHandler(log, service))
	mux.Handle("POST /pullRequest/merge", rest.MergePRHandler(log, service))
	mux.Handle("POST /pullRequest/reassign", rest.ReassignReviewerHandler(log, service))
	mux.Handle("GET /users/getReview", rest.GetUserReviewsHandler(log, service))
	mux.Handle("GET /statistics", rest.GetStatisticsHandler(log, service))

	handler := rest.LoggingMiddleware(log)(mux)

	server := &http.Server{
		Addr:         cfg.HTTPConfig.Address,
		ReadTimeout:  cfg.HTTPConfig.Timeout,
		WriteTimeout: cfg.HTTPConfig.Timeout,
		IdleTimeout:  cfg.HTTPConfig.Timeout,
		Handler:      handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server closed unexpectedly: %v", err)
		}
	}
	return nil
}

func mustSetupLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}
