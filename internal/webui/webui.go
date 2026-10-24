// Package webui serves the statically built SvelteKit client that is embedded in the binary.
package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// dist is filled by the client build; the repo only holds a placeholder.
//
//go:embed all:dist
var dist embed.FS

func init() {
	// Without this type some browsers do not offer to install the PWA.
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// Dist returns the embedded client files.
func Dist() fs.FS {
	root, _ := fs.Sub(dist, "dist")
	return root
}

// Handler serves files from root; unknown paths get the app shell.
func Handler(root fs.FS) http.Handler {
	files := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if st, err := fs.Stat(root, name); err == nil && !st.IsDir() {
				if strings.HasPrefix(name, "_app/immutable/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		// Unknown paths return the app shell so that client-side routing takes over.
		index, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.Error(w, "Client ist nicht gebaut. Bitte zuerst den Client bauen.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index)
	})
}
