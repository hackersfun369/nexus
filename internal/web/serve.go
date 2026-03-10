package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var staticFiles embed.FS

//go:embed landing.html
var landingHTML []byte

// Handler returns an http.Handler that serves:
//   / and /index.html  → landing page
//   /app               → React SPA
//   /app/*             → React SPA (client-side routing)
//   /assets/*          → static assets
func Handler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Landing page
		if path == "/" || path == "/index.html" || path == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(landingHTML)
			return
		}

		// React app routes
		if path == "/app" || strings.HasPrefix(path, "/app/") {
			// Strip /app prefix and serve from static
			r2 := r.Clone(r.Context())
			if path == "/app" || path == "/app/" {
				r2.URL.Path = "/"
			} else {
				r2.URL.Path = strings.TrimPrefix(path, "/app")
			}
			// Always serve index.html for React client-side routing
			if !strings.Contains(r2.URL.Path, ".") {
				r2.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r2)
			return
		}

		// Static assets (js, css, fonts etc.)
		fileServer.ServeHTTP(w, r)
	})
}
