package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	device "pi_dash/src/device/types"
	views "pi_dash/src/views/pages"
	"pi_dash/src/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ws_server := ws.NewServer()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.RealIP)

	// Create a route along /files that will serve contents from
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "public"))
	FileServer(r, "/public", filesDir)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		device_data, err := device.NewData()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		views.DashPage(device_data).Render(r.Context(), w)
	})

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		_, err := ws.NewClient(ws_server, w, r)
		if err != nil {
			http.Error(w, "Failed handling web socket connection: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	go startWebSocket()

	http.ListenAndServe(":3000", r)
}

func startWebSocket() {
	for {

	}
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
