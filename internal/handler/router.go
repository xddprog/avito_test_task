package handler

import (
	"embed"
	"net/http"
)


var swaggerUI embed.FS

func NewRouter(
	user *UserHandler,
	team *TeamHandler,
	pr *PullRequestHandler,
	openAPISpecPath string,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users/setIsActive", user.SetIsActive)
	mux.HandleFunc("GET  /users/getReview", user.GetReview)

	mux.HandleFunc("GET  /team/get", team.GetTeam)
	mux.HandleFunc("POST /team/add", team.AddTeam)

	mux.HandleFunc("POST /pullRequest/create", pr.CreatePullRequest)
	mux.HandleFunc("POST /pullRequest/merge", pr.MergePullRequest)
	mux.HandleFunc("POST /pullRequest/reassign", pr.ReassignPullRequest)

	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeFile(w, r, openAPISpecPath)
	})

	swaggerHandler := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerUI)))
	mux.Handle("/swagger/", swaggerHandler)

	return mux
}
