package handler

import (
	"net/http"
	"path/filepath"
)

func NewRouter(
	user *UserHandler,
	team *TeamHandler,
	pr *PullRequestHandler,
	stats *StatsHandler,
	health *HealthHandler,
	openAPISpecPath string,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users/setIsActive", user.SetIsActive)
	mux.HandleFunc("GET  /users/getReview", user.GetReview)

	mux.HandleFunc("GET  /team/get", team.GetTeam)
	mux.HandleFunc("POST /team/add", team.AddTeam)
	mux.HandleFunc("POST /team/deactivate", team.DeactivateMembers)

	mux.HandleFunc("POST /pullRequest/create", pr.CreatePullRequest)
	mux.HandleFunc("POST /pullRequest/merge", pr.MergePullRequest)
	mux.HandleFunc("POST /pullRequest/reassign", pr.ReassignPullRequest)
	mux.HandleFunc("GET /stats/summary", stats.Summary)
	mux.HandleFunc("GET /health", health.Check)

	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeFile(w, r, openAPISpecPath)
	})

	swaggerDir := filepath.Join(filepath.Dir(openAPISpecPath), "swagger")
	swaggerHandler := http.StripPrefix("/swagger/", http.FileServer(http.Dir(swaggerDir)))
	mux.Handle("/swagger/", swaggerHandler)

	return mux
}
