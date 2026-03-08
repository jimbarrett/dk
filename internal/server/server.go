package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/barrett/dk/internal/docker"
)

// Start launches the web server on the given port using the provided filesystem for static assets.
func Start(port string, webFS fs.FS) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/list", handleList)
	mux.HandleFunc("/api/action", handleAction)
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	addr := ":" + port
	fmt.Printf("dk web UI running on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	grouped, err := docker.ListGrouped(true)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(grouped)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	action := r.URL.Query().Get("action")
	container := r.URL.Query().Get("container")

	if container == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "no container specified"})
		return
	}

	var result string
	var err error

	switch action {
	case "start":
		result, err = docker.Start(container)
	case "stop":
		result, err = docker.Stop(container)
	case "restart":
		result, err = docker.Restart(container)
	case "remove":
		result, err = docker.Remove(container, false)
	default:
		json.NewEncoder(w).Encode(map[string]string{"error": "unknown action"})
		return
	}

	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "result": result})
}
