// Package webassets serves the browser bundle embedded at build time.
package webassets

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var bundle embed.FS

// Handler returns static assets and only explicitly supported HTML routes.
func Handler() http.Handler {
	assets, err := fs.Sub(bundle, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if response.Header().Get("Content-Security-Policy") == "" {
			response.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		}
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Referrer-Policy", "strict-origin")
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.Header().Set("Allow", "GET, HEAD")
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if request.URL.Path == "/" || request.URL.Path == "/fixture/login" {
			response.Header().Set("Cache-Control", "no-store")
			page, readErr := fs.ReadFile(assets, "index.html")
			if readErr != nil {
				http.Error(response, "assets unavailable", 503)
				return
			}
			response.Header().Set("Content-Type", "text/html; charset=utf-8")
			if request.Method == http.MethodGet {
				_, _ = response.Write(page)
			}
			return
		}
		if strings.HasPrefix(request.URL.Path, "/assets/") {
			response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			files.ServeHTTP(response, request)
			return
		}
		http.NotFound(response, request)
	})
}
