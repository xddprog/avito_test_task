package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xddprog/avito-test-task/internal/config"
	"github.com/xddprog/avito-test-task/internal/handler"
	"github.com/xddprog/avito-test-task/internal/repository"
	"github.com/xddprog/avito-test-task/internal/service"
	db "github.com/xddprog/avito-test-task/pkg/db/migration"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load env: %v", err)
	}

	if err := db.RunMigrations("file://migrations", cfg.Postgres.DSN()); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	config, err := pgxpool.ParseConfig(cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("failed to create db connection: %v", err)
	}

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("failed to create db connection: %v", err)
	}

	defer db.Close()

	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to determine working directory: %v", err)
	}

	userRepository := repository.NewUserRepository(db)
	pullRequestRepository := repository.NewPullRequestRepository(db)
	teamRepository := repository.NewTeamRepository(db)

	userService := service.NewUserService(userRepository)
	pullRequestService := service.NewPullRequestService(pullRequestRepository, userRepository)
	teamService := service.NewTeamService(teamRepository)

	userHandler := handler.NewUserHandler(userService)
	pullRequestHandler := handler.NewPullRequestHandler(pullRequestService)
	teamHandler := handler.NewTeamHandler(teamService)

	openAPISpecPath := filepath.Join(workDir, "api", "openapi.yml")
	mux := handler.NewRouter(userHandler, teamHandler, pullRequestHandler, openAPISpecPath)

	srv := &http.Server{
		Addr:         cfg.HTTP.Address(),
		Handler:      mux,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	log.Printf("starting HTTP server at %s", cfg.HTTP.Address())
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
