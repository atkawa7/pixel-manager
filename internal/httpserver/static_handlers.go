package httpserver

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			serveFileFromFS(staticFS, w, "/index.html")
			return
		}

		if cleanPath == "index.html" {
			serveFileFromFS(staticFS, w, "/index.html")
			return
		}

		if _, err := fs.Stat(staticFS, cleanPath); err == nil {
			if strings.HasPrefix(cleanPath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		if path.Ext(cleanPath) == "" {
			serveFileFromFS(staticFS, w, "/index.html")
			return
		}

		http.NotFound(w, r)
	})
}

func serveFileFromFS(staticFS fs.FS, w http.ResponseWriter, assetPath string) {
	normalized := strings.TrimPrefix(assetPath, "/")
	data, err := fs.ReadFile(staticFS, normalized)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if normalized == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	}

	if ct := mime.TypeByExtension(path.Ext(normalized)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	_, _ = w.Write(data)
}
