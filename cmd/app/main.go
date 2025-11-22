package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xddprog/avito-test-task/internal/config"
	"github.com/xddprog/avito-test-task/internal/handler"
	"github.com/xddprog/avito-test-task/internal/logger"
	"github.com/xddprog/avito-test-task/internal/middleware"
	"github.com/xddprog/avito-test-task/internal/repository"
	"github.com/xddprog/avito-test-task/internal/service"
	db "github.com/xddprog/avito-test-task/pkg/db/migration"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.Log.Level, cfg.Log.Format)

	if err := db.RunMigrations("file://migrations", cfg.Postgres.DSN()); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	config, err := pgxpool.ParseConfig(cfg.Postgres.DSN())
	if err != nil {
		slog.Error("failed to parse db config", "error", err)
		os.Exit(1)
	}

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		slog.Error("failed to create db connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	workDir, err := os.Getwd()
	if err != nil {
		slog.Error("failed to determine working directory", "error", err)
		os.Exit(1)
	}

	userRepository := repository.NewUserRepository(db)
	pullRequestRepository := repository.NewPullRequestRepository(db)
	teamRepository := repository.NewTeamRepository(db)
	statsRepository := repository.NewStatsRepository(db)

	userService := service.NewUserService(userRepository)
	pullRequestService := service.NewPullRequestService(pullRequestRepository, userRepository)
	teamService := service.NewTeamService(teamRepository, pullRequestRepository, pullRequestService, userRepository)
	statsService := service.NewStatsService(statsRepository)

	userHandler := handler.NewUserHandler(userService)
	pullRequestHandler := handler.NewPullRequestHandler(pullRequestService)
	teamHandler := handler.NewTeamHandler(teamService)
	statsHandler := handler.NewStatsHandler(statsService)
	healthHandler := handler.NewHealthHandler()

	openAPISpecPath := filepath.Join(workDir, "api", "openapi.yml")
	mux := handler.NewRouter(userHandler, teamHandler, pullRequestHandler, statsHandler, healthHandler, openAPISpecPath)

	handlerWithLogging := middleware.LoggingMiddleware(mux)

	srv := &http.Server{
		Addr:         cfg.HTTP.Address(),
		Handler:      handlerWithLogging,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	slog.Info("starting HTTP server", "address", cfg.HTTP.Address())
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
