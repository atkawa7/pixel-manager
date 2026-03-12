package httpserver

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	s.serveOpenAPIFile(w, "openapi.json")
}

func (s *Server) handleOpenAPIAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	relPath := strings.TrimPrefix(r.URL.Path, "/openapi/")
	if strings.HasPrefix(r.URL.Path, "/api/openapi/") {
		relPath = strings.TrimPrefix(r.URL.Path, "/api/openapi/")
	}
	relPath = strings.TrimSpace(relPath)
	relPath = path.Clean(relPath)
	if relPath == "." || relPath == "" || strings.HasPrefix(relPath, "..") {
		http.NotFound(w, r)
		return
	}

	s.serveOpenAPIFile(w, relPath)
}

func (s *Server) serveOpenAPIFile(w http.ResponseWriter, relPath string) {
	data, err := fs.ReadFile(embeddedOpenAPIFiles, path.Join("openapi", relPath))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if ct := mime.TypeByExtension(path.Ext(relPath)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
