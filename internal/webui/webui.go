// Package webui liefert den statisch gebauten SvelteKit-Client aus, der ins Binary eingebettet ist.
package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// dist wird vom Client-Build befüllt; im Repo liegt nur ein Platzhalter.
//
//go:embed all:dist
var dist embed.FS

func init() {
	// Ohne diesen Typ bieten manche Browser die Installation der PWA nicht an.
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// Dist liefert die eingebetteten Client-Dateien.
func Dist() fs.FS {
	root, _ := fs.Sub(dist, "dist")
	return root
}

// Handler liefert Dateien aus root aus; unbekannte Pfade bekommen die App-Hülle.
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
		// Unbekannte Pfade liefern die App-Hülle, damit das Client-Routing greift.
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
